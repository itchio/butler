
# Installing butler

## Downloading

You can download stable and bleeding-edge builds of butler from its itch.io page:

  - <https://itchio.itch.io/butler>

You can find installation guides for each platform below:

  - [Installing butler on Windows](#on-windows)
  - [Installing butler on Linux](#on-linux)
  - [Installing butler on macOS](#on-macos)

### Automation-friendly way

For CI/CD pipelines, use broth instead of the itch.io download page. The broth
URLs are fixed and can be safely embedded directly in your scripts, whereas the
itch.io download links are expiring and cannot be reused.

  - <https://broth.itch.zone/butler>

The `-head` channels are bleeding-edge, the other ones are stable.

If you want to get the latest stable, for example, you could curl or wget the following URL:

  - <https://broth.itch.zone/butler/linux-amd64/LATEST/archive/default>

You can substitute `linux-amd64` with any channel listed on broth.

The file served is a .zip file. While the URL redirects to an expiring download
URL internally, the broth URL itself is permanent and will always fetch the
latest version.

**Note:** the .zip file contains the butler executable, along with two dynamic
libraries related to 7-zip, which grant butler extra functionality. They're
not strictly required for `butler push`, however, they shouldn't hurt either.

If you need help integrating butler in your CI pipeline, butler's GitHub issue
tracker is a good place to ask: https://github.com/itchio/butler/issues/

## Installing

Adding butler to your PATH allows you to launch it from anywhere, no matter
which directory you're currently in.

### On Windows

1. Download butler from <https://itchio.itch.io/butler>
2. Extract the downloaded `.zip` file to a folder of your choice (e.g. `C:\butler`)
3. Add that folder to your PATH:
   1. Press `Win + X` and select **System**
   2. Click **Advanced system settings**
   3. Click **Environment Variables**
   4. Under **System variables**, find and select `Path`, then click **Edit**
   5. Click **New** and add the path to the folder where you extracted butler (e.g. `C:\butler`)
   6. Click **OK** to save all changes

You'll need to close and re-open any command prompt windows for the changes to take effect.

#### Verifying the installation

Open a new Command Prompt (`Win + R`, then type `cmd` and press Enter) and run:

```
butler version
```


It should print something like:

```
v15.24.0, built on Dec 12 2024 @ 17:12:35, ref f1203f79c4b65ef2201e95ca81b6b02b7d37cb04
```

Here's how it looks on Windows:

![](images/butler-cmd-exe.png)

If it works, you can proceed to log in:

```
butler login
```

This will open your browser to authenticate with your itch.io account.

*Note: you can also run butler from PowerShell.*

#### Troubleshooting

If you see the error `'butler' is not recognized as an internal or external command`, this means the PATH was not configured correctly. Double-check that:

- You added the correct path to the Environment Variables
- You clicked **OK** to save all the dialogs (there are multiple OK buttons)
- You opened a **new** Command Prompt window after making the changes

For more help, see the [Troubleshooting](troubleshooting.md) page.

*Alternatively, Windows will look into the current working directory when
looking for commands, so you can run butler without adding it to your PATH
if you navigate to the folder containing butler.exe first.*

### On Linux

1. Download butler from <https://itchio.itch.io/butler>
2. Extract the downloaded `.zip` file to a folder of your choice (e.g. `~/bin`)
3. Mark butler as executable:

```sh
chmod +x ~/bin/butler
```

(Replacing `~/bin` with the folder you actually extracted butler to)

4. Add the folder to your PATH by editing `~/.bashrc` and adding this line at the end:

```sh
export PATH="$PATH:$HOME/bin"
```

(Again, replacing `$HOME/bin` as appropriate)

5. Close and start a new terminal to apply the changes.

#### Verifying the installation

Run:

```sh
butler version
```


It should print something like:

```
v15.24.0, built on Dec 12 2024 @ 17:12:35, ref f1203f79c4b65ef2201e95ca81b6b02b7d37cb04
```

If it works, you can proceed to log in:

```sh
butler login
```

This will open your browser to authenticate with your itch.io account.

### On macOS

1. Download butler from <https://itchio.itch.io/butler>
2. Extract the downloaded `.zip` file to a folder of your choice (e.g. `~/bin`)
3. Mark butler as executable:

```sh
chmod +x ~/bin/butler
```

(Replacing `~/bin` with the folder you actually extracted butler to)

4. Add the folder to your PATH by editing `~/.bash_profile` and adding this line at the end:

```sh
export PATH="$PATH:$HOME/bin"
```

(Again, replacing `$HOME/bin` as appropriate)

You may have to create the `~/.bash_profile` file if it doesn't exist yet.

5. Close and start a new terminal to apply the changes.

#### Verifying the installation

Run:

```sh
butler version
```


It should print something like:

```
v15.24.0, built on Dec 12 2024 @ 17:12:35, ref f1203f79c4b65ef2201e95ca81b6b02b7d37cb04
```

If it works, you can proceed to log in:

```sh
butler login
```

This will open your browser to authenticate with your itch.io account.

## Appendix: Finding butler

If you ever forget where you put your butler.exe, the `butler which` command
will print its complete path.

## Appendix: Using butler from the itch app

The itch desktop app includes its own copy of butler. If you already have the app installed, you can use this bundled version directly instead of downloading it separately.

### Paths by platform

| Platform | Path                                                                      |
|----------|---------------------------------------------------------------------------|
| Windows  | `%APPDATA%\itch\broth\butler\versions\{version}\butler.exe`               |
| macOS    | `~/Library/Application Support/itch/broth/butler/versions/{version}/butler` |
| Linux    | `~/.config/itch/broth/butler/versions/{version}/butler`                   |

### Finding the current version

The active version is stored in a `.chosen-version` file:

| Platform | Path                                                            |
|----------|-----------------------------------------------------------------|
| Windows  | `%APPDATA%\itch\broth\butler\.chosen-version`                   |
| macOS    | `~/Library/Application Support/itch/broth/butler/.chosen-version` |
| Linux    | `~/.config/itch/broth/butler/.chosen-version`                   |

This file contains the version string (e.g., `15.21.0`) which corresponds to the subdirectory name under `versions/`.

