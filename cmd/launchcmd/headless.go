package launchcmd

import (
	"sync"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/jsonrpc2"
	"github.com/itchio/butler/comm"
)

// headlessClient answers the server->client requests a Launch can emit
// with non-interactive defaults. Requests that need a person or a UI are
// refused and recorded so the caller can fall back to the itch app.
type headlessClient struct {
	acceptLicenses             bool
	continueAfterPrereqFailure bool

	mu        sync.Mutex
	appReason string
}

var _ jsonrpc2.Handler = (*headlessClient)(nil)

func (h *headlessClient) needsApp() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.appReason
}

func (h *headlessClient) setNeedsApp(reason string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.appReason = reason
}

func (h *headlessClient) HandleRequest(conn jsonrpc2.Conn, req jsonrpc2.Request) (interface{}, error) {
	switch req.Method {
	case "PickManifestAction":
		// only reached when --target and the cave's launchTarget setting
		// are both unset; --target is the escape hatch for multi-action games
		return butlerd.PickManifestActionResult{Index: 0}, nil
	case "AcceptLicense":
		var params butlerd.AcceptLicenseParams
		if req.Params != nil {
			_ = jsonrpc2.DecodeJSON(*req.Params, &params)
		}
		if h.acceptLicenses {
			comm.Notice("License agreement (accepted via --accept-licenses)", []string{params.Text})
			return butlerd.AcceptLicenseResult{Accept: true}, nil
		}
		// accepting persists a consent marker in the install folder, so
		// refusal is the only honest headless answer; the app shows a dialog
		h.setNeedsApp("license agreement needs acceptance (or pass --accept-licenses)")
		return butlerd.AcceptLicenseResult{Accept: false}, nil
	case "PrereqsFailed":
		if h.continueAfterPrereqFailure {
			return butlerd.PrereqsFailedResult{Continue: true}, nil
		}
		h.setNeedsApp("prerequisite installation failed (or pass --continue-after-prereq-failure)")
		return butlerd.PrereqsFailedResult{Continue: false}, nil
	case "AllowSandboxSetup":
		// the OS shows its own elevation prompt right after
		return butlerd.AllowSandboxSetupResult{Allow: true}, nil
	case "HTMLLaunch", "ShellLaunch", "URLLaunch":
		// backstop: LaunchParams.AllowedStrategies should reject these
		// server-side before any launcher runs
		h.setNeedsApp(req.Method + " requires the itch app")
		return nil, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInternalError,
			Message: req.Method + " requires the itch app",
		}
	}
	return nil, &jsonrpc2.Error{
		Code:    jsonrpc2.CodeMethodNotFound,
		Message: "headless launch cannot answer " + req.Method,
	}
}

func (h *headlessClient) HandleNotification(conn jsonrpc2.Conn, notif jsonrpc2.Notification) {
	decode := func(v interface{}) bool {
		if notif.Params == nil {
			return false
		}
		return jsonrpc2.DecodeJSON(*notif.Params, v) == nil
	}

	switch notif.Method {
	case "Log":
		var params butlerd.LogNotification
		if decode(&params) {
			switch params.Level {
			case butlerd.LogLevelError, butlerd.LogLevelWarning:
				comm.Warnf("%s", params.Message)
			default:
				comm.Logf("%s", params.Message)
			}
		}
	case "LaunchRunning":
		comm.Statf("Game is running")
	case "LaunchExited":
		// exit is reported by the Launch call returning
	case "PrereqsStarted":
		comm.Opf("Installing prerequisites...")
	case "PrereqsEnded":
		comm.Statf("Prerequisites installed")
	}
}
