package operate

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/dchest/safefile"
	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/cmd/wipe"
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/butler/pb"
	"github.com/itchio/wharf/state"
	"github.com/mitchellh/mapstructure"
)

type OperationContext struct {
	cave        *models.Cave
	rc          *buse.RequestContext
	ctx         context.Context
	consumer    *state.Consumer
	stageFolder string
	logFile     *os.File

	root map[string]interface{}

	// keep track of what we've loaded so far
	// loading more than once is not ok
	loaded map[string]struct{}
}

func LoadContext(ctx context.Context, rc *buse.RequestContext, stageFolder string) (*OperationContext, error) {
	parentConsumer := rc.Consumer

	err := os.MkdirAll(stageFolder, 0755)
	if err != nil {
		parentConsumer.Warnf("Could not create operate directory: %s", err.Error())
	}

	logFilePath := filepath.Join(stageFolder, "operate-log.json")
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		parentConsumer.Warnf("Could not open operate log: %s", err.Error())
	}

	// shows percentages, to the 1/100th
	bar := pb.New64(100 * 100)
	bar.AlwaysUpdate = true
	bar.NotPrint = true
	bar.RefreshRate = 250 * time.Millisecond
	bar.Start()

	oc := &OperationContext{
		logFile:     logFile,
		stageFolder: stageFolder,
		ctx:         ctx,
		rc:          rc,
		root:        make(map[string]interface{}),
		loaded:      make(map[string]struct{}),
	}
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	consumer := TeeConsumer(parentConsumer, logFile)
	oc.consumer = consumer
	path := contextPath(stageFolder)

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			// empty context, that's fine
		} else {
			consumer.Warnf("While loading context from %s: %s", path, err.Error())
		}
		return oc, nil
	}
	defer f.Close()

	err = json.NewDecoder(f).Decode(&oc.root)
	if err != nil {
		consumer.Warnf("While decoding context from %s: %s", path, err.Error())
	}

	return oc, nil
}

func (oc *OperationContext) Load(s Subcontext) {
	if _, ok := oc.loaded[s.Key()]; ok {
		oc.consumer.Warnf("Refusing to load subcontext %s a second time", s.Key())
		return
	}

	// only load if there's actually something there
	if val, ok := oc.root[s.Key()]; ok {
		dec, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
			TagName:          "json",
			Result:           s.Data(),
			WeaklyTypedInput: true,
			DecodeHook:       mapstructure.StringToTimeHookFunc(time.RFC3339Nano),
		})
		if err != nil {
			oc.consumer.Warnf("Could not load subcontext %s: while configuring decoder, %s", s.Key(), err.Error())
			return
		}

		err = dec.Decode(val)
		if err != nil {
			oc.consumer.Warnf("Could not load subcontext %s: while decoding, %s", s.Key(), err.Error())
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

	err = wipe.Do(comm.NewStateConsumer(), oc.StageFolder())
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
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

func TeeConsumer(c *state.Consumer, logFile io.Writer) *state.Consumer {
	originalConsumer := *c
	newConsumer := *c

	newConsumer.OnMessage = func(level string, msg string) {
		originalConsumer.OnMessage(level, msg)

		payload, err := json.Marshal(map[string]interface{}{
			"time":  currentTimeMillis(),
			"name":  "butler",
			"level": butlerLevelToItchLevel(level),
			"msg":   msg,
		})
		if err == nil {
			fmt.Fprintf(logFile, "%s\n", string(payload))
		} else {
			fmt.Fprintf(logFile, "could not marshal json log entry: %s\n", err.Error())
		}
	}
	return &newConsumer
}

func butlerLevelToItchLevel(level string) int {
	switch level {
	case "fatal":
		return 60
	case "error":
		return 50
	case "warning":
		return 40
	case "info":
		return 30
	case "debug":
		return 20
	case "trace":
		return 10
	default:
		return 30 // default
	}
}

func currentTimeMillis() int64 {
	timeUtc := time.Now().UTC()
	nanos := timeUtc.Nanosecond()
	millis := timeUtc.Unix() * 1000
	millis += int64(nanos) / 1000000
	return millis
}
