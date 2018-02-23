package mockharness

import (
	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	itchio "github.com/itchio/go-itchio"
	httpmock "gopkg.in/jarcoal/httpmock.v1"
)

// mockHarness

type mockHarness struct {
	ph buse.Harness
}

var _ buse.Harness = (*mockHarness)(nil)

type WithHarnessFunc func(h buse.Harness) error

func With(cb WithHarnessFunc) error {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	th := &mockHarness{buse.NewProductionHarness()}
	err := cb(th)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}

func (mh *mockHarness) ClientFromCredentials(credentials *buse.GameCredentials) (*itchio.Client, error) {
	return mh.ph.ClientFromCredentials(credentials)
}
