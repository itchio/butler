package main

import (
	"archive/zip"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/itchio/butler/comm"
	"github.com/itchio/go-itchio"
	"github.com/itchio/wharf/counter"
	"github.com/itchio/wharf/pwr"
	"github.com/itchio/wharf/sync"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/uploader"
)

func push(buildPath string, spec string, userVersion string, fixPerms bool) {
	go versionCheck()
	must(doPush(buildPath, spec, userVersion, fixPerms))
}

func doPush(buildPath string, spec string, userVersion string, fixPerms bool) error {
	butlerBlockCache := os.Getenv("BUTLER_BLOCK_CACHE")
	useBlockCache := len(butlerBlockCache) > 0
	if useBlockCache {
		comm.Logf("Using block cache (experimental!) @ %s", butlerBlockCache)
	}

	spec = strings.ToLower(spec)

	// start walking source container while waiting on auth flow
	sourceContainerChan := make(chan walkResult)
	walkErrs := make(chan error)
	go doWalk(buildPath, sourceContainerChan, walkErrs, fixPerms)

	target, channel, err := parseSpec(spec)
	if err != nil {
		return err
	}

	client, err := authenticateViaOauth()
	if err != nil {
		return err
	}

	newBuildRes, err := client.CreateBuild(target, channel, userVersion)
	if err != nil {
		return err
	}

	buildID := newBuildRes.Build.ID
	parentID := newBuildRes.Build.ParentBuild.ID

	var targetSignature []sync.BlockHash
	var targetContainer *tlc.Container

	if parentID == 0 {
		comm.Opf("For channel `%s`: pushing first build", channel)
		targetSignature = make([]sync.BlockHash, 0)
		targetContainer = &tlc.Container{}
	} else {
		comm.Opf("For channel `%s`: last build is %d, downloading its signature", channel, parentID)
		var buildFiles itchio.ListBuildFilesResponse
		buildFiles, err = client.ListBuildFiles(parentID)
		if err != nil {
			return err
		}

		var signatureFileID int64
		for _, f := range buildFiles.Files {
			if f.Type == itchio.BuildFileType_SIGNATURE {
				signatureFileID = f.ID
				break
			}
		}

		if signatureFileID == 0 {
			comm.Dief("Could not find signature for parent build %d, aborting", parentID)
		}

		var signatureReader io.Reader
		signatureReader, err = client.DownloadBuildFile(parentID, signatureFileID)
		if err != nil {
			return err
		}

		targetContainer, targetSignature, err = pwr.ReadSignature(signatureReader)
		if err != nil {
			return err
		}
	}

	newPatchRes, newSignatureRes, err := createBothFiles(client, buildID)
	if err != nil {
		return err
	}

	uploadDone := make(chan bool)
	uploadErrs := make(chan error)

	patchWriter, err := uploader.NewResumableUpload(newPatchRes.File.UploadURL,
		uploadDone, uploadErrs, comm.NewStateConsumer())
	if err != nil {
		return err
	}

	signatureWriter, err := uploader.NewResumableUpload(newSignatureRes.File.UploadURL,
		uploadDone, uploadErrs, comm.NewStateConsumer())
	if err != nil {
		return err
	}

	comm.Debugf("Launching patch & signature channels")

	patchCounter := counter.NewWriter(patchWriter)
	signatureCounter := counter.NewWriter(signatureWriter)

	// we started walking the source container in the beginning,
	// we actually need it now.
	// note that we could actually start diffing before all the file
	// creation & upload setup is done

	var sourceContainer *tlc.Container
	var sourcePool sync.FilePool

	comm.Debugf("Waiting for source container")
	select {
	case err := <-walkErrs:
		return err
	case walkies := <-sourceContainerChan:
		comm.Debugf("Got sourceContainer!")
		sourceContainer = walkies.container
		sourcePool = walkies.pool
		break
	}

	comm.Logf("")
	comm.Opf("Pushing %s (%s)", humanize.Bytes(uint64(sourceContainer.Size)), sourceContainer.Stats())

	comm.Debugf("Building diff context")
	var readBytes int64

	bytesPerSec := float64(0)
	lastUploadedBytes := int64(0)
	stopTicking := make(chan struct{})

	updateProgress := func() {
		uploadedBytes := int64(float64(patchWriter.UploadedBytes))

		// input bytes that aren't in output, for esxample:
		//  - bytes that have been compressed away
		//  - bytes that were in old build and were simply reused
		goneBytes := readBytes - patchWriter.TotalBytes

		conservativeTotalBytes := sourceContainer.Size - goneBytes

		leftBytes := conservativeTotalBytes - uploadedBytes
		if leftBytes > 10*1024 {
			netStatus := "- network idle"
			if bytesPerSec > 1 {
				netStatus = fmt.Sprintf("@ %s/s", humanize.Bytes(uint64(bytesPerSec)))
			}
			comm.ProgressLabel(fmt.Sprintf("%s, %s left", netStatus, humanize.Bytes(uint64(leftBytes))))
		} else {
			comm.ProgressLabel(fmt.Sprintf("- almost there"))
		}

		conservativeProgress := float64(uploadedBytes) / float64(conservativeTotalBytes)
		conservativeProgress = min(1.0, conservativeProgress)
		comm.Progress(conservativeProgress)

		comm.ProgressScale(float64(readBytes) / float64(sourceContainer.Size))
	}

	go func() {
		ticker := time.NewTicker(time.Second * time.Duration(2))
		for {
			select {
			case <-ticker.C:
				bytesPerSec = float64(patchWriter.UploadedBytes-lastUploadedBytes) / 2.0
				lastUploadedBytes = patchWriter.UploadedBytes
				updateProgress()
			case <-stopTicking:
				break
			}
		}
	}()

	patchWriter.OnProgress = updateProgress

	stateConsumer := &pwr.StateConsumer{
		OnProgress: func(progress float64) {
			readBytes = int64(float64(sourceContainer.Size) * progress)
			updateProgress()
		},
	}

	dctx := &pwr.DiffContext{
		Compression: &pwr.CompressionSettings{
			Algorithm: pwr.CompressionAlgorithm_BROTLI,
			Quality:   1,
		},

		SourceContainer: sourceContainer,
		FilePool:        sourcePool,

		TargetContainer: targetContainer,
		TargetSignature: targetSignature,

		Consumer: stateConsumer,
	}

	if useBlockCache {
		h256 := sha256.New()
		hbuf := make([]byte, 32)
		client := &http.Client{
			Transport: makeTransport(),
		}

		dataLookup := func(buf []byte) (string, error) {
			h256.Reset()
			_, err := h256.Write(buf)
			if err != nil {
				return "", err
			}
			sum := h256.Sum(hbuf[:0])

			key := fmt.Sprintf("%d/%x", len(buf), sum)
			// return key, nil
			fmt.Sprintf("Should look up %s", key)

			req, err := http.NewRequest("HEAD", fmt.Sprintf("%s/%s", butlerBlockCache, key), nil)
			fmt.Fprintf(os.Stderr, "lookup %s\n", req.RequestURI)
			if err != nil {
				return "", err
			}

			res, err := client.Do(req)
			if err != nil {
				fmt.Fprintf(os.Stderr, "lookup error: %s", err.Error())
				return "", nil
			}

			err = res.Body.Close()
			if err != nil {
				return "", err
			}

			if res.StatusCode != 200 {
				return "", nil
			}

			return key, nil
		}

		dctx.DataLookup = dataLookup
	}

	comm.StartProgress()
	comm.ProgressScale(0.0)
	err = dctx.WritePatch(patchCounter, signatureCounter)
	if err != nil {
		return err
	}

	// close in a goroutine to avoid deadlocking
	doClose := func(c io.Closer, done chan bool, errs chan error) {
		err := c.Close()
		if err != nil {
			errs <- err
			return
		}

		done <- true
	}

	go doClose(patchWriter, uploadDone, uploadErrs)
	go doClose(signatureWriter, uploadDone, uploadErrs)

	for c := 0; c < 4; c++ {
		select {
		case err := <-uploadErrs:
			return err
		case <-uploadDone:
			comm.Debugf(">>>>>>>>>>> woo, got a done")
		}
	}

	close(stopTicking)
	comm.ProgressLabel("finalizing build")

	finalDone := make(chan bool)
	finalErrs := make(chan error)

	doFinalize := func(fileID int64, fileSize int64, done chan bool, errs chan error) {
		_, err = client.FinalizeBuildFile(buildID, fileID, fileSize)
		if err != nil {
			errs <- err
			return
		}

		done <- true
	}

	go doFinalize(newPatchRes.File.ID, patchCounter.Count(), finalDone, finalErrs)
	go doFinalize(newSignatureRes.File.ID, signatureCounter.Count(), finalDone, finalErrs)

	for i := 0; i < 2; i++ {
		select {
		case err := <-finalErrs:
			return err
		case <-finalDone:
		}
	}

	comm.EndProgress()

	{
		prettyPatchSize := humanize.Bytes(uint64(patchCounter.Count()))
		percReused := 100.0 * float64(dctx.ReusedBytes) / float64(dctx.FreshBytes+dctx.ReusedBytes)
		relToNew := 100.0 * float64(patchCounter.Count()) / float64(sourceContainer.Size)
		prettyFreshSize := humanize.Bytes(uint64(dctx.FreshBytes))
		savings := 100.0 - relToNew

		if dctx.ReusedBytes > 0 {
			comm.Statf("Re-used %.2f%% of old, added %s fresh data", percReused, prettyFreshSize)
		} else {
			comm.Statf("Added %s fresh data", prettyFreshSize)
		}

		if savings > 0 && !math.IsNaN(savings) {
			comm.Statf("%s patch (%.2f%% savings)", prettyPatchSize, 100.0-relToNew)
		} else {
			comm.Statf("%s patch (no savings)", prettyPatchSize)
		}
	}
	return nil
}

type fileSlot struct {
	Type     itchio.BuildFileType
	Response itchio.NewBuildFileResponse
}

func createBothFiles(client *itchio.Client, buildID int64) (patch itchio.NewBuildFileResponse, signature itchio.NewBuildFileResponse, err error) {
	createFile := func(buildType itchio.BuildFileType, done chan fileSlot, errs chan error) {
		var res itchio.NewBuildFileResponse
		res, err = client.CreateBuildFile(buildID, buildType, itchio.BuildFileSubType_DEFAULT, itchio.UploadType_RESUMABLE)
		if err != nil {
			errs <- err
		}
		comm.Debugf("Created %s build file: %+v", buildType, res.File)
		done <- fileSlot{buildType, res}
	}

	done := make(chan fileSlot)
	errs := make(chan error)

	go createFile(itchio.BuildFileType_PATCH, done, errs)
	go createFile(itchio.BuildFileType_SIGNATURE, done, errs)

	for i := 0; i < 2; i++ {
		select {
		case err = <-errs:
			return
		case slot := <-done:
			switch slot.Type {
			case itchio.BuildFileType_PATCH:
				patch = slot.Response
			case itchio.BuildFileType_SIGNATURE:
				signature = slot.Response
			}
		}
	}

	return
}

type walkResult struct {
	container *tlc.Container
	pool      sync.FilePool
}

func doWalk(path string, out chan walkResult, errs chan error, fixPerms bool) {
	var result walkResult

	stats, err := os.Lstat(path)
	if err != nil {
		errs <- err
		return
	}

	if stats.IsDir() {
		container, err := tlc.Walk(path, filterPaths)
		if err != nil {
			errs <- err
			return
		}

		result = walkResult{
			container: container,
			pool:      container.NewFilePool(path),
		}
	} else {
		sourceReader, err := os.Open(path)
		if err != nil {
			errs <- err
			return
		}

		zr, err := zip.NewReader(sourceReader, stats.Size())
		if err != nil {
			errs <- err
			return
		}

		container, err := tlc.WalkZip(zr, filterPaths)
		if err != nil {
			errs <- err
			return
		}

		result = walkResult{
			container: container,
			pool:      container.NewZipPool(zr),
		}
	}

	if fixPerms {
		result.container.FixPermissions(result.pool)
	}
	out <- result
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

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func makeTransport() *http.Transport {
	rootPEM := `-----BEGIN CERTIFICATE-----
MIIEkjCCA3qgAwIBAgIQCgFBQgAAAVOFc2oLheynCDANBgkqhkiG9w0BAQsFADA/
MSQwIgYDVQQKExtEaWdpdGFsIFNpZ25hdHVyZSBUcnVzdCBDby4xFzAVBgNVBAMT
DkRTVCBSb290IENBIFgzMB4XDTE2MDMxNzE2NDA0NloXDTIxMDMxNzE2NDA0Nlow
SjELMAkGA1UEBhMCVVMxFjAUBgNVBAoTDUxldCdzIEVuY3J5cHQxIzAhBgNVBAMT
GkxldCdzIEVuY3J5cHQgQXV0aG9yaXR5IFgzMIIBIjANBgkqhkiG9w0BAQEFAAOC
AQ8AMIIBCgKCAQEAnNMM8FrlLke3cl03g7NoYzDq1zUmGSXhvb418XCSL7e4S0EF
q6meNQhY7LEqxGiHC6PjdeTm86dicbp5gWAf15Gan/PQeGdxyGkOlZHP/uaZ6WA8
SMx+yk13EiSdRxta67nsHjcAHJyse6cF6s5K671B5TaYucv9bTyWaN8jKkKQDIZ0
Z8h/pZq4UmEUEz9l6YKHy9v6Dlb2honzhT+Xhq+w3Brvaw2VFn3EK6BlspkENnWA
a6xK8xuQSXgvopZPKiAlKQTGdMDQMc2PMTiVFrqoM7hD8bEfwzB/onkxEz0tNvjj
/PIzark5McWvxI0NHWQWM6r6hCm21AvA2H3DkwIDAQABo4IBfTCCAXkwEgYDVR0T
AQH/BAgwBgEB/wIBADAOBgNVHQ8BAf8EBAMCAYYwfwYIKwYBBQUHAQEEczBxMDIG
CCsGAQUFBzABhiZodHRwOi8vaXNyZy50cnVzdGlkLm9jc3AuaWRlbnRydXN0LmNv
bTA7BggrBgEFBQcwAoYvaHR0cDovL2FwcHMuaWRlbnRydXN0LmNvbS9yb290cy9k
c3Ryb290Y2F4My5wN2MwHwYDVR0jBBgwFoAUxKexpHsscfrb4UuQdf/EFWCFiRAw
VAYDVR0gBE0wSzAIBgZngQwBAgEwPwYLKwYBBAGC3xMBAQEwMDAuBggrBgEFBQcC
ARYiaHR0cDovL2Nwcy5yb290LXgxLmxldHNlbmNyeXB0Lm9yZzA8BgNVHR8ENTAz
MDGgL6AthitodHRwOi8vY3JsLmlkZW50cnVzdC5jb20vRFNUUk9PVENBWDNDUkwu
Y3JsMB0GA1UdDgQWBBSoSmpjBH3duubRObemRWXv86jsoTANBgkqhkiG9w0BAQsF
AAOCAQEA3TPXEfNjWDjdGBX7CVW+dla5cEilaUcne8IkCJLxWh9KEik3JHRRHGJo
uM2VcGfl96S8TihRzZvoroed6ti6WqEBmtzw3Wodatg+VyOeph4EYpr/1wXKtx8/
wApIvJSwtmVi4MFU5aMqrSDE6ea73Mj2tcMyo5jMd6jmeWUHK8so/joWUoHOUgwu
X4Po1QYz+3dszkDqMp4fklxBwXRsW10KXzPMTZ+sOPAveyxindmjkW8lGy+QsRlG
PfZ+G6Z6h7mjem0Y+iWlkYcV4PIWL1iwBi8saCbGS5jN2p8M+X+Q7UNKEkROb3N6
KOqkqm57TH2H3eDJAkSnh6/DNFu0Qg==
-----END CERTIFICATE-----`

	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM([]byte(rootPEM))
	if !ok {
		panic("failed to parse root certificate")
	}

	localPEM := `-----BEGIN CERTIFICATE-----
MIIC/DCCAeSgAwIBAgIQE+edX+67bADw3M3wsd8mmjANBgkqhkiG9w0BAQsFADAS
MRAwDgYDVQQKEwdBY21lIENvMB4XDTE2MDQyNjIyMTc0NloXDTE3MDQyNjIyMTc0
NlowEjEQMA4GA1UEChMHQWNtZSBDbzCCASIwDQYJKoZIhvcNAQEBBQADggEPADCC
AQoCggEBALXwW6U4LFzQ0Q46gBMRU/lVHKCpbDw1vQ23EbpLvbVgWbWFMek9OBl3
hW13S44EVjqufcUWpo9XN32VZBcn3f4NPJEpEwajxwdkRIvEwCoRgftKnkhq23Iq
JzD9YST6iUft0MhODtx9614QeEFnZofMI0im+z11jZcH7pDx88EQeVh0GM2Hc+gd
mT0J4th3sUWJ/KLnErLlBTRcQReKUYY2oU3UTLlg86jtj1RJEcMMyBZkXfVZEKvu
9noa8dDi/u1HjQQiQZKFnHfGv7nTx/sgXobMCszLACDR3oS9xtG6etUx5qhGG6GT
WrwVMC+XaDkELQ29vRDoVIqiIRp6wZcCAwEAAaNOMEwwDgYDVR0PAQH/BAQDAgKk
MBMGA1UdJQQMMAoGCCsGAQUFBwMBMA8GA1UdEwEB/wQFMAMBAf8wFAYDVR0RBA0w
C4IJbG9jYWxob3N0MA0GCSqGSIb3DQEBCwUAA4IBAQAwrMPgE9ESkjD+x0AqkyxJ
1ARs0Mhb6mc2qKffas8PbZJDV1Oi+DlZFDVHVRlOIi3x+8gk/WfXyfgjyded/CJa
YgGFNEKzVkWXieTyczksRtRd6uILbGJy4ZUyONN+cQ+H05kg52Nylt8867dEfAtd
4R/0J4ER/huzyRUgCgs1WuTWuYfsrSyeahjECSEO6Lm1rdKtLiJen0nmno8fG8Pi
GBEZ1z8Co3xui85HQXxtCy3VCaUQ+p3I3ZD/r4gkn3jUX/fTUiIjtvbt0vloU72v
fY0v7gjVB+ud6M4CzotDPzeOy5iBzW+5YDvmk7rPD2+wo3cc7kCsYiQHXw7dNyPi
-----END CERTIFICATE-----`

	ok = roots.AppendCertsFromPEM([]byte(localPEM))
	if !ok {
		panic("failed to parse root certificate")
	}

	tlsConf := &tls.Config{RootCAs: roots}
	tlsConf.BuildNameToCertificate()
	return &http.Transport{TLSClientConfig: tlsConf}
}
