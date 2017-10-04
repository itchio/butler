package cave

type InstallerType string

const (
	InstallerTypeNaked   InstallerType = "naked"
	InstallerTypeArchive               = "archive"
	InstallerTypeDMG                   = "dmg"
	InstallerTypeInno                  = "inno"
	InstallerTypeNsis                  = "nsis"
	InstallerTypeMSI                   = "msi"
	InstallerTypeUnknown               = "unknown"
)

func getInstallerType(target string) InstallerType {
	// TODO: implement
	return InstallerTypeUnknown
}
