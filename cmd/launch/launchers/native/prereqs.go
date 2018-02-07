package native

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/cmd/launch"
	"github.com/itchio/butler/cmd/prereqs"
	"github.com/itchio/butler/redist"
	"github.com/itchio/wharf/eos"
)

func handlePrereqs(params *launch.LauncherParams) error {
	consumer := params.Consumer
	ctx := params.Ctx
	conn := params.Conn

	if runtime.GOOS != "windows" {
		consumer.Infof("Not on windows, ignoring prereqs")
		return nil
	}

	if params.AppManifest == nil {
		consumer.Infof("No manifest, no prereqs")
		return nil
	}

	if len(params.AppManifest.Prereqs) == 0 {
		consumer.Infof("Got manifest but no prereqs requested")
		return nil
	}

	// TODO: store done somewhere
	prereqsDir := params.ParentParams.PrereqsDir

	// TODO: cache maybe
	consumer.Infof("Fetching prereqs registry...")
	beforeFetch := time.Now()

	registry := &redist.RedistRegistry{}

	err := func() error {
		registryURL := fmt.Sprintf("%s/info.json", prereqs.RedistsBaseURL)
		f, err := eos.Open(registryURL)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		dec := json.NewDecoder(f)
		err = dec.Decode(registry)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		return nil
	}()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	registryFetchDuration := time.Since(beforeFetch)
	consumer.Infof("✓ Fetched %d entries in %s", len(registry.Entries), registryFetchDuration)

	var initialNames []string
	for _, p := range params.AppManifest.Prereqs {
		initialNames = append(initialNames, p.Name)
	}

	pa, err := prereqs.AssessPrereqs(consumer, registry, initialNames)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	if len(pa.Done) > 0 {
		consumer.Infof("✓ %d Prereqs already done: %s", len(pa.Done), strings.Join(pa.Done, ", "))
	}

	if len(pa.Todo) == 0 {
		consumer.Infof("Everything done!")
		return nil
	}
	consumer.Infof("→ %d Prereqs to install: %s", len(pa.Todo), strings.Join(pa.Todo, ", "))

	{
		psn := &buse.PrereqsStartedNotification{
			Tasks: make(map[string]*buse.PrereqTask),
		}
		for i, name := range pa.Todo {
			psn.Tasks[name] = &buse.PrereqTask{
				FullName: registry.Entries[name].FullName,
				Order:    i,
			}
		}

		err = conn.Notify(ctx, "PrereqsStarted", psn)
		if err != nil {
			consumer.Warnf(err.Error())
		}
	}

	tsc := &prereqs.TaskStateConsumer{
		OnState: func(state *buse.PrereqsTaskStateNotification) {
			err = conn.Notify(ctx, "PrereqsTaskState", state)
			if err != nil {
				consumer.Warnf(err.Error())
			}
		},
	}

	err = prereqs.FetchPrereqs(consumer, tsc, prereqsDir, registry, pa.Todo)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	plan := &prereqs.PrereqPlan{}

	for _, name := range pa.Todo {
		plan.Tasks = append(plan.Tasks, &prereqs.PrereqTask{
			Name:    name,
			WorkDir: filepath.Join(prereqsDir, name),
			Info:    *registry.Entries[name],
		})
	}

	err = prereqs.ElevatedInstall(consumer, plan, tsc)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	err = conn.Notify(ctx, "PrereqsEnded", &buse.PrereqsEndedNotification{})
	if err != nil {
		consumer.Warnf(err.Error())
	}

	return nil
}
