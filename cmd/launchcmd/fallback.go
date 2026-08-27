package launchcmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// openItchURL hands the launch to the itch app via its URL handler,
// booting the app if needed. Note the process returns immediately, so a
// Steam shortcut going through this path loses accurate playtime state.
func openItchURL(gameID int64, uploadID int64) error {
	url := fmt.Sprintf("itch://install?game_id=%d&launch", gameID)
	if uploadID != 0 {
		url = fmt.Sprintf("itch://install?game_id=%d&upload_id=%d&launch", gameID, uploadID)
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	cmd.Env = envWithoutOverlayPreload()
	return cmd.Run()
}

// Steam preloads its overlay into every descendant of a shortcut process,
// which deadlocks the app's Chromium startup. The itch-mode shortcut strips
// it with LD_PRELOAD="" %command%; this fallback has to do the same.
func envWithoutOverlayPreload() []string {
	var env []string
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "LD_PRELOAD=") ||
			strings.HasPrefix(kv, "DYLD_INSERT_LIBRARIES=") {
			continue
		}
		env = append(env, kv)
	}
	return env
}
