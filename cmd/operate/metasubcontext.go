package operate

import (
	"github.com/itchio/butler/butlerd"
	itchio "github.com/itchio/go-itchio"
)

type InstallParams struct {
	StagingFolder string `json:"stagingFolder"`

	Reason butlerd.DownloadReason `json:"reason"`

	CaveID            string `json:"caveId"`
	InstallFolderName string `json:"installFolderName"`
	InstallLocationID string `json:"installLocationID"`

	InstallFolder string `json:"installFolder"`

	NoCave bool `json:"noCave"`

	Game   *itchio.Game   `json:"game"`
	Upload *itchio.Upload `json:"upload"`
	Build  *itchio.Build  `json:"build"`

	IgnoreInstallers bool `json:"ignoreInstallers,omitempty"`

	Credentials *butlerd.GameCredentials `json:"credentials"`
}

type MetaSubcontext struct {
	Data *InstallParams
}

func NewMetaSubcontext() *MetaSubcontext {
	return &MetaSubcontext{
		Data: &InstallParams{},
	}
}

var _ Subcontext = (*MetaSubcontext)(nil)

func (mt *MetaSubcontext) Key() string {
	return "meta"
}

func (mt *MetaSubcontext) GetData() interface{} {
	return &mt.Data
}
