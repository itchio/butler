package apply

import (
	"fmt"
	"path"
	"time"

	"github.com/itchio/wharf/eos/option"

	humanize "github.com/dustin/go-humanize"
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	"github.com/itchio/savior/seeksource"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/pwr"
	"github.com/pkg/errors"
)

var args = struct {
	patch *string
	old   *string

	dir       *string
	inplace   *bool
	dryrun    *bool
	signature *string
	wounds    *string
	heal      *string
	stage     *string
}{}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("apply", "(Advanced) Use a patch to patch a directory to a new version")
	args.patch = cmd.Arg("patch", "Patch file (.pwr), previously generated with the `diff` command.").Required().String()
	args.old = cmd.Arg("old", "Directory, archive, or empty directory (/dev/null) to patch").Required().String()

	args.dir = cmd.Flag("dir", "Directory to create newer files in, instead of working in-place").Short('d').String()
	args.inplace = cmd.Flag("inplace", "Apply patch directly to old directory. Required for safety").Bool()
	args.dryrun = cmd.Flag("dryrun", "Don't write the new files anywhere, just apply the patch in memory").Bool()
	args.signature = cmd.Flag("signature", "When given, verify the integrity of touched file using the signature").String()
	args.wounds = cmd.Flag("wounds", "When given, write wounds to this path instead of failing (exclusive with --heal)").String()
	args.heal = cmd.Flag("heal", "When given, heal using specified source instead of failing (exclusive with --wounds)").String()
	args.stage = cmd.Flag("stage", "When given, use that folder for intermediary files when doing in-place ptching").String()
	ctx.Register(cmd, do)
}

func do(ctx *mansion.Context) {
	ctx.Must(Do(&Params{
		Patch:  *args.patch,
		Target: *args.old,

		Output:        *args.dir,
		InPlace:       *args.inplace,
		DryRun:        *args.dryrun,
		SignaturePath: *args.signature,
		WoundsPath:    *args.wounds,
		HealSpec:      *args.heal,
		StagePath:     *args.stage,
	}))
}

type Params struct {
	Patch  string
	Target string

	Output        string
	InPlace       bool
	DryRun        bool
	SignaturePath string
	WoundsPath    string
	HealSpec      string
	StagePath     string
}

func Do(params *Params) error {
	output := params.Output
	target := params.Target
	signaturePath := params.SignaturePath
	patch := params.Patch
	stagePath := params.StagePath
	woundsPath := params.WoundsPath
	healSpec := params.HealSpec

	if !params.DryRun {
		if output == "" {
			output = target
		}

		target = path.Clean(target)
		output = path.Clean(output)
		if output == target {
			if !params.InPlace {
				comm.Dief("Refusing to destructively patch %s without --inplace", output)
			}
		}
	}

	if signaturePath == "" {
		comm.Opf("Patching %s", output)
	} else {
		comm.Opf("Patching %s with validation", output)
	}

	startTime := time.Now()

	patchReader, err := eos.Open(patch, option.WithConsumer(comm.NewStateConsumer()))
	if err != nil {
		return errors.WithStack(err)
	}

	var signature *pwr.SignatureInfo
	if signaturePath != "" {
		sigReader, sigErr := eos.Open(signaturePath, option.WithConsumer(comm.NewStateConsumer()))
		if sigErr != nil {
			return errors.Wrap(sigErr, "opening signature")
		}
		defer sigReader.Close()

		sigSource := seeksource.FromFile(sigReader)
		_, sigErr = sigSource.Resume(nil)
		if sigErr != nil {
			return errors.Wrap(sigErr, "creating source for signature")
		}

		signature, sigErr = pwr.ReadSignature(sigSource)
		if sigErr != nil {
			return errors.Wrap(sigErr, "decoding signature")
		}
	}

	actx := &pwr.ApplyContext{
		TargetPath: target,
		OutputPath: output,
		DryRun:     params.DryRun,
		InPlace:    params.InPlace,
		Signature:  signature,
		WoundsPath: woundsPath,
		StagePath:  stagePath,
		HealPath:   healSpec,

		Consumer: comm.NewStateConsumer(),
	}

	patchSource := seeksource.FromFile(patchReader)

	_, err = patchSource.Resume(nil)
	if err != nil {
		return errors.WithStack(err)
	}

	comm.StartProgressWithTotalBytes(patchSource.Size())
	err = actx.ApplyPatch(patchSource)
	if err != nil {
		return errors.WithStack(err)
	}
	comm.EndProgress()

	container := actx.SourceContainer
	prettySize := humanize.IBytes(uint64(container.Size))
	perSecond := humanize.IBytes(uint64(float64(container.Size) / time.Since(startTime).Seconds()))

	if actx.InPlace {
		statStr := ""
		if actx.Stats.TouchedFiles > 0 {
			statStr += fmt.Sprintf("patched %d, ", actx.Stats.TouchedFiles)
		}
		if actx.Stats.MovedFiles > 0 {
			statStr += fmt.Sprintf("renamed %d, ", actx.Stats.MovedFiles)
		}
		if actx.Stats.DeletedFiles > 0 {
			statStr += fmt.Sprintf("deleted %d, ", actx.Stats.DeletedFiles)
		}

		comm.Statf("%s (%s stage)", statStr, humanize.IBytes(uint64(actx.Stats.StageSize)))
	}
	comm.Statf("%s (%s) @ %s/s\n", prettySize, container.Stats(), perSecond)

	if actx.WoundsConsumer != nil && actx.WoundsConsumer.HasWounds() {
		extra := ""
		if actx.WoundsPath != "" {
			extra = fmt.Sprintf(" (written to %s)", actx.WoundsPath)
		}

		totalCorrupted := actx.WoundsConsumer.TotalCorrupted()

		verb := "has"
		totalHealed := int64(0)
		if healer, ok := actx.WoundsConsumer.(pwr.Healer); ok {
			verb = "had"
			totalHealed = healer.TotalHealed()
		}

		comm.Logf("Result %s wounds, %s corrupted data, %s healed%s", verb, humanize.IBytes(uint64(totalCorrupted)), humanize.IBytes(uint64(totalHealed)), extra)
	}

	return nil
}
