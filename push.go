package main

import (
	"fmt"
	"strings"

	"golang.org/x/crypto/ssh"

	"github.com/dustin/go-humanize"
	"github.com/itchio/butler/comm"
	"github.com/itchio/go-itchio"
	"github.com/itchio/wharf"
	"github.com/itchio/wharf/counter"
	"github.com/itchio/wharf/pwr"
	"github.com/itchio/wharf/sync"
	"github.com/itchio/wharf/tlc"
)

func push(buildPath string, spec string) {
	must(doPush(buildPath, spec))
}

const (
	defaultPort = 22
)

func doPush(buildPath string, spec string) error {
	sourceContainerChan := make(chan *tlc.Container)
	walkErrs := make(chan error)
	go doWalk(buildPath, sourceContainerChan, walkErrs)

	address := *pushArgs.address

	if !strings.Contains(address, ":") {
		address = fmt.Sprintf("%s:%d", address, defaultPort)
	}

	comm.Logf("Connecting to %s", address)
	conn, err := wharf.Connect(address, *pushArgs.identity, "butler", version)
	if err != nil {
		return err
	}
	comm.Logf("Connected to %s", conn.Conn.RemoteAddr())

	go ssh.DiscardRequests(conn.Reqs)

	req := &wharf.AuthenticationRequest{}
	res := &wharf.AuthenticationResponse{}

	err = conn.SendRequest("authenticate", req, res)
	if err != nil {
		return fmt.Errorf("Authentication error; %s", err.Error())
	}

	err = conn.Close()
	if err != nil {
		return err
	}

	// TODO: if buildPath is an archive, warn and unpack it

	client := itchio.ClientWithKey(res.Key)
	client.BaseURL = res.ItchioBaseUrl

	target, channel, err := parseSpec(spec)
	if err != nil {
		return err
	}

	newBuildRes, err := client.CreateBuild(target, channel)
	if err != nil {
		return err
	}

	buildID := newBuildRes.Build.ID
	parentID := newBuildRes.Build.ParentBuild.ID

	var targetSignature []sync.BlockHash
	var targetContainer *tlc.Container

	if parentID == 0 {
		comm.Logf("This is the first build for channel %s", channel)
		targetSignature = make([]sync.BlockHash, 0)
		targetContainer = &tlc.Container{}
	} else {
		comm.Logf("Latest build for channel %s is %d, downloading its signature", channel, parentID)
		buildFiles, err := client.ListBuildFiles(parentID)
		if err != nil {
			return err
		}

		var signatureFileID int64 = 0
		for _, f := range buildFiles.Files {
			if f.Type == itchio.BuildFileType_SIGNATURE {
				signatureFileID = f.ID
				break
			}
		}

		if signatureFileID == 0 {
			comm.Dief("Could not find signature for parent build %d, aborting", parentID)
		}

		signatureReader, err := client.DownloadBuildFile(parentID, signatureFileID)

		targetContainer, targetSignature, err = pwr.ReadSignature(signatureReader)
		if err != nil {
			return err
		}
	}

	done := make(chan bool)
	errs := make(chan error)

	newPatchRes, err := client.CreateBuildFile(buildID, itchio.BuildFileType_PATCH)
	if err != nil {
		return err
	}
	comm.Debugf("Created patch build file: %+v", newPatchRes.File)

	patchWriter, err := newMultipartUpload(newPatchRes.File.UploadURL,
		newPatchRes.File.UploadParams, fmt.Sprintf("%d-%d.pwr", parentID, buildID),
		done, errs)

	if err != nil {
		return err
	}

	newSignatureRes, err := client.CreateBuildFile(buildID, itchio.BuildFileType_SIGNATURE)
	if err != nil {
		return err
	}
	comm.Debugf("Created signature build file: %+v", newPatchRes.File)

	signatureWriter, err := newMultipartUpload(newSignatureRes.File.UploadURL,
		newSignatureRes.File.UploadParams, fmt.Sprintf("%d-%d.pwr", parentID, buildID),
		done, errs)

	if err != nil {
		return err
	}

	comm.Logf("Launching patch & signature channels")

	patchCounter := counter.NewWriter(patchWriter)
	signatureCounter := counter.NewWriter(signatureWriter)

	// we started walking the source container in the beginning,
	// we actually need it now.
	// note that we could actually start diffing before all the file
	// creation & upload setup is done

	var sourceContainer *tlc.Container

	comm.Debugf("Waiting for source container")
	select {
	case err := <-walkErrs:
		return err
	case sourceContainer = <-sourceContainerChan:
		break
	}

	comm.Debugf("Building diff context")
	dctx := &pwr.DiffContext{
		Compression: &pwr.CompressionSettings{
			Algorithm: pwr.CompressionAlgorithm_BROTLI,
			Quality:   1,
		},

		SourceContainer: sourceContainer,
		SourcePath:      buildPath,

		TargetContainer: targetContainer,
		TargetSignature: targetSignature,

		Consumer: comm.NewStateConsumer(),
	}

	comm.Logf("Sending patch and signature computed on-the-fly...")
	comm.StartProgress()
	err = dctx.WritePatch(patchCounter, signatureCounter)
	if err != nil {
		return err
	}
	comm.EndProgress()

	comm.Logf("Done computing patch, waiting for the rest")
	err = patchWriter.Close()
	if err != nil {
		return err
	}

	err = signatureWriter.Close()
	if err != nil {
		return err
	}

	for c := 0; c < 2; c++ {
		select {
		case err := <-errs:
			return err
		case <-done:
		}
	}
	comm.Logf("Everything sent off!")

	{
		prettyPatchSize := humanize.Bytes(uint64(patchCounter.Count()))
		percReused := 100.0 * float64(dctx.ReusedBytes) / float64(dctx.FreshBytes+dctx.ReusedBytes)
		relToNew := 100.0 * float64(patchCounter.Count()) / float64(sourceContainer.Size)
		prettyFreshSize := humanize.Bytes(uint64(dctx.FreshBytes))

		comm.Statf("Re-used %.2f%% of old, added %s fresh data", percReused, prettyFreshSize)
		comm.Statf("%s patch (%.2f%% of the full size)", prettyPatchSize, relToNew)
	}

	comm.Logf("Finalizing patch file (%d bytes total)", patchCounter.Count())
	_, err = client.FinalizeBuildFile(buildID, newPatchRes.File.ID, patchCounter.Count())
	if err != nil {
		return err
	}

	comm.Logf("Finalizing signature file (%d bytes total)", signatureCounter.Count())
	_, err = client.FinalizeBuildFile(buildID, newSignatureRes.File.ID, signatureCounter.Count())
	if err != nil {
		return err
	}

	return nil
}

func doWalk(path string, result chan *tlc.Container, errs chan error) {
	container, err := tlc.Walk(path, filterDirs)
	if err != nil {
		errs <- err
	}

	result <- container
}

func parseSpec(spec string) (string, string, error) {
	tokens := strings.Split(spec, ":")

	if len(tokens) == 1 {
		return "", "", fmt.Errorf("invalid spec: %s, missing channel (examples: %s:windows-32-beta, %s:linux-64)", spec, spec, spec)
	} else if len(tokens) != 2 {
		return "", "", fmt.Errorf("invalid spec: %s, expected something of the form user/page:channel", spec)
	}

	return tokens[0], tokens[1], nil
}

func parseAddress(address string) string {
	if strings.Contains(address, ":") {
		return address
	} else {
		return fmt.Sprintf("%s:%d", address, defaultPort)
	}
}
