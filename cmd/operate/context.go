package operate

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/dchest/safefile"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/cmd/wipe"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/headway/state"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
)

type OperationContext struct {
	cave        *models.Cave
	rc          *butlerd.RequestContext
	ctx         context.Context
	consumer    *state.Consumer
	stageFolder string
	logFile     *os.File

	root map[string]interface{}

	// keep track of what we've loaded so far
	// loading more than once is not ok
	loaded map[string]struct{}

	pidFilePath string
}

type PidFileContents struct {
	PID int64 `json:"pid"`
}

func LoadContext(ctx context.Context, rc *butlerd.RequestContext, stageFolder string) (*OperationContext, error) {
	parentConsumer := rc.Consumer

	err := os.MkdirAll(stageFolder, 0o755)
	if err != nil {
		return nil, errors.WithMessage(err, "creating staging folder")
	}

	pidFilePath := filepath.Join(stageFolder, "operate-pid.json")
	pidContents := &PidFileContents{
		PID: int64(os.Getpid()),
	}
	pidBytes, err := json.Marshal(pidContents)
	if err != nil {
		return nil, errors.WithMessage(err, "marshalling pid file")
	}
	err = ioutil.WriteFile(pidFilePath, pidBytes, 0o644)
	if err != nil {
		parentConsumer.Warnf("Could not open write pid file: %s", err.Error())
	}

	logFilePath := filepath.Join(stageFolder, "operate-log.json")
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		parentConsumer.Warnf("Could not open operate log: %s", err.Error())
	}

	oc := &OperationContext{
		logFile:     logFile,
		stageFolder: stageFolder,
		ctx:         ctx,
		rc:          rc,
		root:        make(map[string]interface{}),
		loaded:      make(map[string]struct{}),

		pidFilePath: pidFilePath,
	}
	if err != nil {
		return nil, errors.WithStack(err)
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
			Result:           s.GetData(),
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
	oc.root[s.Key()] = s.GetData()

	path := contextPath(oc.stageFolder)

	f, err := safefile.Create(path, 0o644)
	if err != nil {
		return errors.WithStack(err)
	}
	defer f.Close()

	err = json.NewEncoder(f).Encode(&oc.root)
	if err != nil {
		return errors.WithStack(err)
	}

	err = f.Commit()
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (oc *OperationContext) Release() {
	// defensive programming woo
	if oc.pidFilePath != "" {
		os.Remove(oc.pidFilePath)
	}

	oc.logFile.Close()
}

func (oc *OperationContext) Retire() {
	consumer := oc.rc.Consumer
	consumer.Infof("Retiring staging context!")
	oc.Release()
	oc.logFile.Close()

	err := wipe.Do(consumer, oc.StageFolder())
	if err != nil {
		consumer.Warnf("Could not wipe staging folder: %+v", err)
	}
}

func (oc *OperationContext) StageFolder() string {
	return oc.stageFolder
}

func (oc *OperationContext) Consumer() *state.Consumer {
	return oc.consumer
}

func (oc *OperationContext) Ctx() context.Context {
	return oc.ctx
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
	GetData() interface{}
}

func TeeConsumer(c *state.Consumer, logFile io.Writer) *state.Consumer {
	originalConsumer := *c
	newConsumer := *c

	newConsumer.OnMessage = func(level string, msg string) {
		if originalConsumer.OnMessage != nil {
			originalConsumer.OnMessage(level, msg)
		}

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
