package operate

import (
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/installer"
	itchio "github.com/itchio/go-itchio"
)

type InstallSubcontextState struct {
	DownloadSessionID   string                   `json:"downloadSessionId,omitempty"`
	InstallerInfo       *installer.InstallerInfo `json:"installerInfo,omitempty"`
	IsAvailableLocally  bool                     `json:"isAvailableLocally,omitempty"`
	FirstInstallResult  *installer.InstallResult `json:"firstInstallResult,omitempty"`
	SecondInstallerInfo *installer.InstallerInfo `json:"secondInstallerInfo,omitempty"`
	UpgradePath         *itchio.UpgradePath      `json:"upgradePath,omitempty"`
	UpgradePathIndex    int                      `json:"upgradePathIndex,omitempty"`
	UsingHealFallback   bool                     `json:"usingHealFallback,omitempty"`
	RefreshedGame       bool                     `json:"refreshedGame,omitempty"`

	Events []butlerd.InstallEvent
}

type InstallSubcontext struct {
	Data      *InstallSubcontextState
	eventSink *installer.InstallEventSink
}

var _ Subcontext = (*InstallSubcontext)(nil)

func (isub *InstallSubcontext) Key() string {
	return "install"
}

func (isub *InstallSubcontext) GetData() interface{} {
	return &isub.Data
}

func (isub *InstallSubcontext) EventSink(oc *OperationContext) *installer.InstallEventSink {
	if isub.eventSink == nil {
		isub.eventSink = &installer.InstallEventSink{
			Append: func(event butlerd.InstallEvent) error {
				isub.Data.Events = append(isub.Data.Events, event)
				return oc.Save(isub)
			},
		}
	}
	return isub.eventSink
}
