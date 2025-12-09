
# Troubleshooting

Here are some common issues and how to fix them.

## 'butler' is not recognized as a command

This means butler is not in your PATH. See the [Installing butler](installing.md) page for
instructions on how to add butler to your PATH.

## Incorrect directory path

**Problem:** Butler cannot locate your game files when the path is wrong, causing the upload to fail.

**Solution:** Verify your directory path carefully. Windows paths typically look like
`C:\Users\username\Documents\mygame`, while macOS/Linux use paths like
`/Users/username/Documents/mygame`. Ensure the directory actually contains your game files
and that you're pointing to a folder, not an individual file.

## Spaces in file paths

**Problem:** The command line interprets spaces as argument separators. A path like
`C:\My Game Files\build` gets split into multiple arguments.

**Solution:** Enclose file paths containing spaces in quotation marks:

```
butler push "C:\My Game Files\build" username/game:channel
```

## Invalid target errors

**Problem:** Wrong username or game name will cause errors like:

  * `itch.io API error (400): /wharf/builds: invalid target (bad user)`
  * `itch.io API error (400): /wharf/builds: invalid target (bad game)`

**Solution:** Verify that both your username and game name match exactly what appears
on your itch.io page. Your username is the subdomain of your itch.io page (e.g. for
`leafo.itch.io`, the username is `leafo`). The game name is the URL slug of your game
page (e.g. for `leafo.itch.io/supergame`, the game name is `supergame`).

The format should be: `butler push /path/to/gamefiles username/game:channel`

## Authentication not complete

**Problem:** Skipping the `butler login` command prevents butler from accessing your account.

**Solution:** Run `butler login` and complete the authorization process through your browser
before attempting to push.

## Project page doesn't exist

**Problem:** Butler requires an existing project page on itch.io — it doesn't create one for you.

**Solution:** Create your project page on itch.io before attempting to push. You can create
a new project at <https://itch.io/game/new>.

## Firewall or antivirus blocking uploads

**Problem:** Security software may prevent butler from connecting to itch.io servers.

**Solution:** If you're having connection issues, try temporarily disabling your firewall
or antivirus to test. If that resolves the issue, add butler to your security software's
exception list.
