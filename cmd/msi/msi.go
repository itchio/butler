package msi

import (
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
)

var infoArgs = struct {
	msiPath *string
}{}

var productInfoArgs = struct {
	productCode *string
}{}

var installArgs = struct {
	msiPath string
	logPath string
}{}

var uninstallArgs = struct {
	msiPath string
	logPath string
}{}

func Register(ctx *mansion.Context) {
	{
		installCmd := ctx.App.Command("msi-install", "Install or repair an MSI package").Hidden()
		installCmd.Arg("msiPath", "Path to the MSI file").Required().StringVar(&installArgs.msiPath)
		installCmd.Flag("logPath", "Where to write a (very verbose) install log").StringVar(&installArgs.logPath)
		ctx.Register(installCmd, doInstall)
	}

	{
		uninstallCmd := ctx.App.Command("msi-uninstall", "Uninstall an MSI package").Hidden()
		uninstallCmd.Arg("msiPath", "Path to the MSI file").Required().StringVar(&uninstallArgs.msiPath)
		uninstallCmd.Flag("logPath", "Where to write a (very verbose) uninstall log").StringVar(&uninstallArgs.logPath)
		ctx.Register(uninstallCmd, doUninstall)
	}
}

func onMsiError(err MSIWindowsInstallerError) {
	comm.Result(&MSIWindowsInstallerErrorResult{
		Type:  "windowsInstallerError",
		Value: err,
	})
}

func doInstall(ctx *mansion.Context) {
	ctx.Must(Install(comm.NewStateConsumer(), installArgs.msiPath, installArgs.logPath, onMsiError))
}

func doUninstall(ctx *mansion.Context) {
	ctx.Must(Uninstall(comm.NewStateConsumer(), uninstallArgs.msiPath, uninstallArgs.logPath, onMsiError))
}

/**
 * MSIInfoResult describes an MSI package's properties
 */
type MSIInfoResult struct {
	ProductCode     string `json:"productCode"`
	InstallState    string `json:"installState"`
	InstallLocation string `json:"installLocation"`
}

type MSIWindowsInstallerError struct {
	Code int64  `json:"code"`
	Text string `json:"text"`
}

type MSIWindowsInstallerErrorResult struct {
	Type  string                   `json:"type"`
	Value MSIWindowsInstallerError `json:"value"`
}

type MSIErrorCallback func(err MSIWindowsInstallerError)
