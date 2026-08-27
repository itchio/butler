package launchcmd

import (
	"testing"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/jsonrpc2"
)

func callHeadless(t *testing.T, client *headlessClient, method string, params interface{}) interface{} {
	t.Helper()

	raw, err := jsonrpc2.EncodeJSON(params)
	if err != nil {
		t.Fatalf("encoding %s params: %v", method, err)
	}
	res, err := client.HandleRequest(nil, jsonrpc2.Request{Method: method, Params: &raw})
	if err != nil {
		t.Fatalf("handling %s: %v", method, err)
	}
	return res
}

func TestHeadlessLicensePolicy(t *testing.T) {
	t.Run("rejects by default", func(t *testing.T) {
		client := &headlessClient{}
		res := callHeadless(t, client, "AcceptLicense", butlerd.AcceptLicenseParams{Text: "terms"})
		if res.(butlerd.AcceptLicenseResult).Accept {
			t.Fatal("expected license rejection")
		}
		if client.needsApp() == "" {
			t.Fatal("expected rejection to record an app fallback reason")
		}
	})

	t.Run("accepts with explicit policy", func(t *testing.T) {
		client := &headlessClient{acceptLicenses: true}
		res := callHeadless(t, client, "AcceptLicense", butlerd.AcceptLicenseParams{Text: "terms"})
		if !res.(butlerd.AcceptLicenseResult).Accept {
			t.Fatal("expected license acceptance")
		}
		if client.needsApp() != "" {
			t.Fatal("did not expect an app fallback reason")
		}
	})
}

func TestHeadlessPrereqFailurePolicy(t *testing.T) {
	params := butlerd.PrereqsFailedParams{Error: "installer failed"}

	t.Run("aborts by default", func(t *testing.T) {
		client := &headlessClient{}
		res := callHeadless(t, client, "PrereqsFailed", params)
		if res.(butlerd.PrereqsFailedResult).Continue {
			t.Fatal("expected launch to abort after prerequisite failure")
		}
		if client.needsApp() == "" {
			t.Fatal("expected failure to record an app fallback reason")
		}
	})

	t.Run("continues with explicit policy", func(t *testing.T) {
		client := &headlessClient{continueAfterPrereqFailure: true}
		res := callHeadless(t, client, "PrereqsFailed", params)
		if !res.(butlerd.PrereqsFailedResult).Continue {
			t.Fatal("expected launch to continue after prerequisite failure")
		}
		if client.needsApp() != "" {
			t.Fatal("did not expect an app fallback reason")
		}
	})
}
