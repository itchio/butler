package operate

import (
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
)

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("operate", "Perform a complex operation: game install, upgrade, etc.").Hidden()
	ctx.Register(cmd, do)
}

func do(ctx *mansion.Context) {
	ctx.Must(Do(ctx))
}

var tr *JSONTransport

func Do(ctx *mansion.Context) error {
	tr = NewJSONTransport()
	tr.Start()

	sigs := make(chan os.Signal)
	signal.Notify(sigs, os.Interrupt)

	go func() {
		for sig := range sigs {
			comm.Logf("Caught %s", sig)
			if sig == os.Interrupt {
				comm.Warnf("Interruted! (%s)", sig)
				comm.Warnf("Will quit in 1 second...")
				time.Sleep(time.Second)
				os.Exit(1)
			}
		}
	}()

	var params OperationParams
	err := readMessage("operation-params", &params)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	if params.StageFolder == "" {
		return errors.New("No stage folder specified")
	}

	oc := LoadContext(ctx, comm.NewStateConsumer(), params.StageFolder)

	meta := &MetaSubcontext{}
	oc.Load(meta)

	meta.MergeParams(&params)

	if meta.data.Operation == "" {
		return errors.New("No operation specified")
	}

	oc.Save(meta)

	switch meta.data.Operation {
	case OperationInstall:
		return install(oc, meta)
	default:
		return fmt.Errorf("Unknown cave command operation '%s'", params.Operation)
	}
}

type MetaSubcontext struct {
	data OperationParams
}

var _ Subcontext = (*MetaSubcontext)(nil)

func (mt *MetaSubcontext) Key() string {
	return "meta"
}

func (mt *MetaSubcontext) Data() interface{} {
	return &mt.data
}

func (mt *MetaSubcontext) MergeParams(params *OperationParams) {
	if mt.data.Operation == "" {
		mt.data.Operation = params.Operation
	}

	if mt.data.InstallParams == nil {
		mt.data.InstallParams = params.InstallParams
	}
}
