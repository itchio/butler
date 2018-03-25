package mockharness

import (
	"github.com/go-errors/errors"
	"github.com/itchio/butler/butlerd"
	itchio "github.com/itchio/go-itchio"
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
		return errors.Wrap(err, 0)
	}

	return nil
}

func (mh *mockHarness) ClientFromCredentials(credentials *butlerd.GameCredentials) (*itchio.Client, error) {
	return mh.ph.ClientFromCredentials(credentials)
}
