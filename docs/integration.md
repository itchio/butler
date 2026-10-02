
# Third-party integrations

butler was designed to be integrated into your workflow, no matter which tools
you prefer. Here's a list of existing integrations.

## setup-butler

[setup-butler](https://github.com/remarkablegames/setup-butler) is a GitHub Action
that installs and configures butler for use in GitHub Actions workflows. This makes
it easy to automate pushing builds to itch.io as part of your CI/CD pipeline.

  * [View setup-butler on GitHub Marketplace](https://github.com/marketplace/actions/setup-butler)

## unity-build

[unity-build](https://github.com/Chaser324/unity-build) is a set of a build
scripts by Chase Petit that are meant to be generic, exapandable and
customizable. This includes supporting the generation of AssetBundles as well
as preparing/uploading builds for services like itch.io and Steam.

  * [View unity-build on Github](https://github.com/Chaser324/unity-build)

## godot-butlerexportplugin

[Godot-ButlerExportPlugin](https://github.com/NovaDC/Godot-ButlerExportPlugin) is a simple export plugin for Godot that allows for game exports to be pushed automatically after building.

## itchy-electron

[itchy-electron](https://github.com/erbridge/itchy-electron) is a CLI app
for building and publishing games and tools using Electron to itch.io.

  * [View itchy-electron on Github](https://github.com/erbridge/itchy-electron)

## gradle-butler-plugin

[gradle-butler-plugin](https://github.com/mini2Dx/gradle-butler-plugin) is a gradle
plugin for automatically installing, updating, and running butler as part of your build.

  * [View gradle-butler-plugin on Github](https://github.com/mini2Dx/gradle-butler-plugin)

## Easy Itch Push (Unreal Engine 5)

[Easy Itch Push](https://assets.tessermind.com/easy-itch-push/) is a free Unreal Engine 5
editor plugin that packages your project and pushes it to itch.io with butler in one click,
without leaving the editor. It handles butler install/upgrade, API key or login auth,
channels and `--userversion`.

* [View Easy Itch Push documentation](https://assets.tessermind.com/easy-itch-push/)
* [Get Easy Itch Push on Fab](https://www.fab.com/listings/274e3f81-dd36-484d-a8c7-f516558561bf)

## Other resources

Here's a few articles people have written about integrating butler into their release
pipeline:

  * [Unity build automation](http://smashriot.com/unity-build-automation/) (by Jesse, developer of Dr. Spacezoo) shows how to use an editor script in conjunction with a bash script to push to itch.io and other platforms in a single go.

