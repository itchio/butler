package operate

import (
	"github.com/itchio/butler/buse"
	itchio "github.com/itchio/go-itchio"
)

type InstallParams struct {
	StagingFolder string `json:"stagingFolder"`
	InstallFolder string `json:"installFolder"`

	Game   *itchio.Game   `json:"game"`
	Upload *itchio.Upload `json:"upload"`
	Build  *itchio.Build  `json:"build"`

	IgnoreInstallers bool `json:"ignoreInstallers,omitempty"`

	Credentials *buse.GameCredentials `json:"credentials"`
}

type MetaSubcontext struct {
	data *InstallParams
}

func NewMetaSubcontext() *MetaSubcontext {
	return &MetaSubcontext{
		data: &InstallParams{},
	}
}

var _ Subcontext = (*MetaSubcontext)(nil)

func (mt *MetaSubcontext) Key() string {
	return "meta"
}

func (mt *MetaSubcontext) Data() interface{} {
	return &mt.data
}
