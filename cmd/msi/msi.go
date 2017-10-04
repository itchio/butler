package msi

import "github.com/itchio/butler/butler"

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

func Register(ctx *butler.Context) {
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

func doInfo(ctx *butler.Context) {
	ctx.Must(Info(ctx, *infoArgs.msiPath))
}

func doProductInfo(ctx *butler.Context) {
	ctx.Must(Info(ctx, *productInfoArgs.productCode))
}

func doInstall(ctx *butler.Context) {
	ctx.Must(Install(ctx, *installArgs.msiPath, *installArgs.logPath, *installArgs.target))
}

func doUninstall(ctx *butler.Context) {
	ctx.Must(Uninstall(ctx, *uninstallArgs.productCode))
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
