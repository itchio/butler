package operate

import (
	"encoding/json"
	"fmt"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	"github.com/sourcegraph/jsonrpc2"
)

func Start(ctx *mansion.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (*buse.OperationResult, error) {
	params := &buse.OperationStartParams{}
	err := json.Unmarshal(*req.Params, params)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	if params.StagingFolder == "" {
		return nil, errors.New("No staging folder specified")
	}

	oc := LoadContext(conn, ctx, comm.NewStateConsumer(), params.StagingFolder)

	meta := &MetaSubcontext{
		data: params,
	}

	if meta.data.Operation == "" {
		return nil, errors.New("No operation specified")
	}

	oc.Save(meta)

	switch params.Operation {
	case "install":
		ires, err := install(oc, meta)
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}

		return &buse.OperationResult{
			Success:       true,
			InstallResult: ires,
		}, nil
	}

	return nil, fmt.Errorf("Unknown operation '%s'", params.Operation)
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
