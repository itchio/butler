package operate

import (
	"context"
	"fmt"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	"github.com/sourcegraph/jsonrpc2"
)

// ErrCancelled is returned when the client asked for an operation to be cancelled
var ErrCancelled = errors.New("operation was cancelled")

// ErrAborted is returned when the user stopped an operation outside the client's control
// and it should just be stopped.
var ErrAborted = errors.New("operation was aborted")

func Start(ctx context.Context, mansionContext *mansion.Context, conn *jsonrpc2.Conn, params *buse.OperationStartParams) (err error) {
	if params.StagingFolder == "" {
		return errors.New("No staging folder specified")
	}

	oc, err := LoadContext(conn, ctx, mansionContext, comm.NewStateConsumer(), params.StagingFolder)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	consumer := oc.Consumer()

	meta := &MetaSubcontext{
		data: params,
	}

	oc.Load(meta)

	if meta.data.Operation == "" {
		return errors.New("No operation specified")
	}

	oc.Save(meta)

	switch params.Operation {
	case "install":
		ires, err := install(oc, meta)
		if err != nil {
			consumer.Errorf("Install failed: %s", err.Error())
			if se, ok := err.(*errors.Error); ok {
				consumer.Errorf("Full stack trace:\n%s", se.ErrorStack())
			}
			return errors.Wrap(err, 0)
		}

		consumer.Infof("Installed %d files, reporting success", len(ires.Files))

		err = oc.Retire()
		if err != nil {
			return errors.Wrap(err, 0)
		}

		return nil

	case "uninstall":
		err := uninstall(oc, meta)
		if err != nil {
			consumer.Errorf("Uninstall failed: %s", err.Error())
			return errors.Wrap(err, 0)
		}

		err = oc.Retire()
		if err != nil {
			return errors.Wrap(err, 0)
		}

		return nil
	}

	return fmt.Errorf("Unknown operation '%s'", params.Operation)
}

type MetaSubcontext struct {
	data *buse.OperationStartParams
}

var _ Subcontext = (*MetaSubcontext)(nil)

func (mt *MetaSubcontext) Key() string {
	return "meta"
}

func (mt *MetaSubcontext) Data() interface{} {
	return &mt.data
}
