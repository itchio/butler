package mockharness

import (
	"github.com/itchio/butler/butlerd"
	itchio "github.com/itchio/go-itchio"
	"github.com/pkg/errors"
	httpmock "gopkg.in/jarcoal/httpmock.v1"
)

// mockHarness

type mockHarness struct {
	ph butlerd.Harness
}

var _ butlerd.Harness = (*mockHarness)(nil)

type WithHarnessFunc func(h butlerd.Harness) error

func With(cb WithHarnessFunc) error {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	th := &mockHarness{butlerd.NewProductionHarness()}
	err := cb(th)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (mh *mockHarness) ClientFromCredentials(credentials *butlerd.GameCredentials) (*itchio.Client, error) {
	return mh.ph.ClientFromCredentials(credentials)
}
