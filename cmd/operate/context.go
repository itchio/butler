package operate

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dchest/safefile"
	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/cmd/wipe"
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	"github.com/itchio/butler/pb"
	"github.com/itchio/butler/progress"
	"github.com/itchio/wharf/state"
	"github.com/mitchellh/mapstructure"
	"github.com/sourcegraph/jsonrpc2"
)

type OperationContext struct {
	conn        *jsonrpc2.Conn
	ctx         context.Context
	consumer    *state.Consumer
	stageFolder string
	logFile     *os.File

	counter *progress.Counter

	mansionContext *mansion.Context

	root map[string]interface{}

	// keep track of what we've loaded so far
	// loading more than once is not ok
	loaded map[string]struct{}
}

func LoadContext(conn *jsonrpc2.Conn, ctx context.Context, mansionContext *mansion.Context, consumer *state.Consumer, stageFolder string) *OperationContext {
	err := os.MkdirAll(stageFolder, 0755)
	if err != nil {
		consumer.Warnf("Could not create operate directory: %s", err.Error())
	}

	logFilePath := filepath.Join(stageFolder, "operate-log.txt")
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		consumer.Warnf("Could not open operate log: %s", err.Error())
	}

	// shows percentages, to the 1/100th
	bar := pb.New64(100 * 100)
	bar.AlwaysUpdate = true
	bar.NotPrint = true
	bar.RefreshRate = 250 * time.Millisecond
	bar.Start()

	oc := &OperationContext{
		logFile:        logFile,
		stageFolder:    stageFolder,
		ctx:            ctx,
		conn:           conn,
		mansionContext: mansionContext,
		root:           make(map[string]interface{}),
		loaded:         make(map[string]struct{}),
	}

	subconsumer := &state.Consumer{
		OnMessage: func(level, msg string) {
			if logFile != nil {
				fmt.Fprintf(logFile, "[%s] %s\n", level, msg)
			}
			conn.Notify(ctx, "Log", &buse.LogNotification{
				Level:   level,
				Message: msg,
			})
		},
		OnProgress: func(alpha float64) {
			if oc.counter == nil {
				// skip
				return
			}

			oc.counter.SetProgress(alpha)
			notif := &buse.OperationProgressNotification{
				Progress: alpha,
				ETA:      oc.counter.ETA().Seconds(),
				BPS:      oc.counter.BPS(),
			}
			conn.Notify(ctx, "Operation.Progress", notif)
		},
		OnProgressLabel: func(label string) {
			// muffin
		},
		OnPauseProgress: func() {
			if oc.counter != nil {
				oc.counter.Pause()
			}
		},
		OnResumeProgress: func() {
			if oc.counter != nil {
				oc.counter.Resume()
			}
		},
	}

	oc.consumer = subconsumer

	path := contextPath(stageFolder)

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			// empty context, that's fine
		} else {
			oc.consumer.Warnf("While loading context from %s: %s", path, err.Error())
		}
		return oc
	}
	defer f.Close()

	err = json.NewDecoder(f).Decode(&oc.root)
	if err != nil {
		oc.consumer.Warnf("While decoding context from %s: %s", path, err.Error())
	}

	return oc
}

func (oc *OperationContext) StartProgress() {
	oc.StartProgressWithTotalBytes(0)
}

func (oc *OperationContext) StartProgressWithTotalBytes(totalBytes int64) {
	oc.StartProgressWithInitialAndTotal(0.0, totalBytes)
}

func (oc *OperationContext) StartProgressWithInitialAndTotal(initialProgress float64, totalBytes int64) {
	if oc.counter != nil {
		oc.consumer.Warnf("Asked to start progress but already tracking progress!")
		return
	}

	oc.counter = progress.NewCounter()
	oc.counter.SetSilent(true)
	oc.counter.SetProgress(initialProgress)
	oc.counter.SetTotalBytes(totalBytes)
	oc.counter.Start()
}

func (oc *OperationContext) EndProgress() {
	if oc.counter != nil {
		oc.consumer.Infof("Ending progress...")
		oc.counter.Finish()
		oc.counter = nil
	} else {
		oc.consumer.Warnf("Asked to stop progress but wasn't tracking progress!")
	}
}

func (oc *OperationContext) Load(s Subcontext) {
	if _, ok := oc.loaded[s.Key()]; ok {
		oc.consumer.Warnf("Refusing to load subcontext %s a second time", s.Key())
		return
	}

	// only load if there's actually something there
	if val, ok := oc.root[s.Key()]; ok {
		dec, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
			TagName: "json",
			Result:  s.Data(),
		})
		if err != nil {
			oc.consumer.Warnf("could not load subcontext %s: while configuring decoder, %s", s.Key(), err.Error())
			return
		}

		err = dec.Decode(val)
		if err != nil {
			oc.consumer.Warnf("could not load subcontext %s: while decoding, %s", s.Key(), err.Error())
			return
		}
	}

	oc.loaded[s.Key()] = struct{}{}
}

func (oc *OperationContext) Save(s Subcontext) error {
	oc.root[s.Key()] = s.Data()

	path := contextPath(oc.stageFolder)

	f, err := safefile.Create(path, 0644)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	err = json.NewEncoder(f).Encode(&oc.root)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	err = f.Commit()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	defer f.Close()

	return nil
}

func (oc *OperationContext) Retire() error {
	consumer := oc.Consumer()

	consumer.Infof("Retiring stage folder...")
	err := oc.logFile.Close()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	err = wipe.Do(oc.MansionContext(), comm.NewStateConsumer(), oc.StageFolder())
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}

func (oc *OperationContext) MansionContext() *mansion.Context {
	return oc.mansionContext
}

func (oc *OperationContext) StageFolder() string {
	return oc.stageFolder
}

func (oc *OperationContext) Consumer() *state.Consumer {
	return oc.consumer
}

func contextPath(stageFolder string) string {
	return filepath.Join(stageFolder, "operate-context.json")
}

type Subcontext interface {
	// Key returns a unique string key used for storing
	// something under the context object
	Key() string

	// Data should return a pointer to the underlying struct
	// of the subcontext
	Data() interface{}
}
