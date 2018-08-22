package operate

import (
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
