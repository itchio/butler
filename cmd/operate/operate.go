package operate

import (
	"context"
	"fmt"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/comm"
)

func Start(ctx context.Context, conn buse.Conn, params *buse.OperationStartParams) (err error) {
	if params.StagingFolder == "" {
		return errors.New("No staging folder specified")
	}

	oc, err := LoadContext(conn, ctx, comm.NewStateConsumer(), params.StagingFolder)
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
		ires, err := Install(oc, meta)
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
