package wipe

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	"github.com/itchio/wharf/archiver"
	"github.com/itchio/wharf/state"
)

var args = struct {
	path *string
}{}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("wipe", "Completely remove a directory (rm -rf)").Hidden()
	args.path = cmd.Arg("path", "Path to completely remove, including its contents").Required().String()
	ctx.Register(cmd, do)
}

func do(ctx *mansion.Context) {
	ctx.Must(Do(ctx, comm.NewStateConsumer(), *args.path))
}

func Do(ctx *mansion.Context, consumer *state.Consumer, path string) error {
	// why have retry logic built into wipe? sometimes when uninstalling
	// games on windows, the os will randomly return I/O errors, retrying
	// usually helps.
	attempt := 0
	sleepPatterns := []time.Duration{
		time.Millisecond * 200,
		time.Millisecond * 400,
		time.Millisecond * 800,
		time.Millisecond * 1600,
	}

	for attempt <= len(sleepPatterns) {
		err := Try(ctx, consumer, path)
		if err == nil {
			break
		}

		if attempt == len(sleepPatterns) {
			return fmt.Errorf("Could not wipe %s: %s", path, err.Error())
		}

		consumer.Warnf("While wiping %s: %s", path, err.Error())

		consumer.Infof("Trying to brute-force permissions, who knows...")
		err = tryChmod(ctx, path)
		if err != nil {
			consumer.Warnf("While bruteforcing: %s", err.Error())
		}

		consumer.Infof("Sleeping for a bit before we retry...")
		sleepDuration := sleepPatterns[attempt]
		time.Sleep(sleepDuration)
		attempt++
	}

	return nil
}

func Try(ctx *mansion.Context, consumer *state.Consumer, path string) error {
	consumer.Debugf("rm -rf %s", path)
	return os.RemoveAll(path)
}

func tryChmod(ctx *mansion.Context, path string) error {
	// oh yeah?
	chmodAll := func(childpath string, f os.FileInfo, err error) error {
		if err != nil {
			// ignore walking errors
			return nil
		}

		// don't ignore chmodding errors
		return os.Chmod(childpath, os.FileMode(archiver.LuckyMode))
	}

	return filepath.Walk(path, chmodAll)
}
