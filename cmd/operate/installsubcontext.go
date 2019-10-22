package operate

import (
	"fmt"
	"time"

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
	Data *InstallSubcontextState
}

var _ Subcontext = (*InstallSubcontext)(nil)

func (mt *InstallSubcontext) Key() string {
	return "install"
}

func (mt *InstallSubcontext) GetData() interface{} {
	return &mt.Data
}

func (mt *InstallSubcontext) PostEvent(oc *OperationContext, event butlerd.InstallEvent) error {
	event.Timestamp = time.Now()

	mt.Data.Events = append(mt.Data.Events, event)
	return oc.Save(mt)
}

func (mt *InstallSubcontext) PostProblem(oc *OperationContext, err error) error {
	return mt.PostEvent(oc, butlerd.InstallEvent{
		Type: butlerd.InstallEventProblem,
		Problem: &butlerd.ProblemInstallEvent{
			Error:      fmt.Sprintf("%v", err),
			ErrorStack: fmt.Sprintf("%+v", err),
		},
	})
}
