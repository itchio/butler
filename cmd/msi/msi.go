package msi

import "github.com/itchio/butler/mansion"
import "github.com/itchio/butler/comm"

var infoArgs = struct {
	msiPath *string
}{}

var productInfoArgs = struct {
	productCode *string
}{}

var installArgs = struct {
	msiPath *string
	logPath *string
	target  *string
}{}

var uninstallArgs = struct {
	productCode *string
}{}

func Register(ctx *mansion.Context) {
	{
		infoCmd := ctx.App.Command("msi-info", "Show information about an MSI file").Hidden()
		infoArgs.msiPath = infoCmd.Arg("msiPath", "Path to the MSI file").Required().String()
		ctx.Register(infoCmd, doInfo)
	}

	{
		productInfoCmd := ctx.App.Command("msi-product-info", "Show information an installed product").Hidden()
		productInfoArgs.productCode = productInfoCmd.Arg("productCode", "The product code to print info for").Required().String()
		ctx.Register(productInfoCmd, doProductInfo)
	}

	{
		installCmd := ctx.App.Command("msi-install", "Install or repair an MSI package").Hidden()
		installArgs.msiPath = installCmd.Arg("msiPath", "Path to the MSI file").Required().String()
		installArgs.logPath = installCmd.Flag("logPath", "Where to write a (very verbose) install log").String()
		installArgs.target = installCmd.Flag("target", "Where to install the MSI (does not work with all packages)").String()
		ctx.Register(installCmd, doInstall)
	}

	{
		uninstallCmd := ctx.App.Command("msi-uninstall", "Uninstall an MSI package").Hidden()
		uninstallArgs.productCode = uninstallCmd.Arg("productCode", "Product code to uninstall").Required().String()
		ctx.Register(uninstallCmd, doUninstall)
	}
}

func doInfo(ctx *mansion.Context) {
	res, err := Info(comm.NewStateConsumer(), *infoArgs.msiPath)
	ctx.Must(err)

	comm.ResultOrPrint(res, func() {
		comm.Statf("MSI product code: %s", res.ProductCode)
		comm.Statf("Install state: %s", res.InstallState)
	})
}

func doProductInfo(ctx *mansion.Context) {
	res, err := ProductInfo(comm.NewStateConsumer(), *productInfoArgs.productCode)
	ctx.Must(err)

	comm.ResultOrPrint(res, func() {
		comm.Statf("Installed product state: %s", res.InstallState)
	})
}

func onMsiError(err MSIWindowsInstallerError) {
	comm.Result(&MSIWindowsInstallerErrorResult{
		Type:  "windowsInstallerError",
		Value: err,
	})
}

func doInstall(ctx *mansion.Context) {
	ctx.Must(Install(comm.NewStateConsumer(), *installArgs.msiPath, *installArgs.logPath, *installArgs.target, onMsiError))
}

func doUninstall(ctx *mansion.Context) {
	ctx.Must(Uninstall(comm.NewStateConsumer(), *uninstallArgs.productCode, onMsiError))
}

/**
 * MSIInfoResult describes an MSI package's properties
 */
type MSIInfoResult struct {
	ProductCode  string `json:"productCode"`
	InstallState string `json:"installState"`
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
