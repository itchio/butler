package operate

import (
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/hush"
)

type InstallSubcontextState struct {
	DownloadSessionID   string              `json:"downloadSessionId,omitempty"`
	InstallerInfo       *hush.InstallerInfo `json:"installerInfo,omitempty"`
	IsAvailableLocally  bool                `json:"isAvailableLocally,omitempty"`
	FirstInstallResult  *hush.InstallResult `json:"firstInstallResult,omitempty"`
	SecondInstallerInfo *hush.InstallerInfo `json:"secondInstallerInfo,omitempty"`
	UpgradePath         *itchio.UpgradePath `json:"upgradePath,omitempty"`
	UpgradePathIndex    int                 `json:"upgradePathIndex,omitempty"`
	UsingHealFallback   bool                `json:"usingHealFallback,omitempty"`
	RefreshedGame       bool                `json:"refreshedGame,omitempty"`

	Events []hush.InstallEvent
}

type InstallSubcontext struct {
	Data      *InstallSubcontextState
	eventSink *hush.InstallEventSink
}

var _ Subcontext = (*InstallSubcontext)(nil)

func (isub *InstallSubcontext) Key() string {
	return "install"
}

func (isub *InstallSubcontext) GetData() interface{} {
	return &isub.Data
}

func (isub *InstallSubcontext) EventSink(oc *OperationContext) *hush.InstallEventSink {
	if isub.eventSink == nil {
		isub.eventSink = &hush.InstallEventSink{
			Append: func(event hush.InstallEvent) error {
				isub.Data.Events = append(isub.Data.Events, event)
				return oc.Save(isub)
			},
		}
	}
	return isub.eventSink
}
