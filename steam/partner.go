package steam

import (
	"context"
	"errors"

	"github.com/itchio/fresh-steamer/partner"
)

// SetPublisherKey verifies the key against the partner API before saving
// it, and returns the apps it controls.
func SetPublisherKey(ctx context.Context, s Store, key string) ([]partner.App, error) {
	if key == "" {
		return nil, errors.New("empty publisher key")
	}
	apps, err := newPartnerClient(key).Apps(ctx)
	if err != nil {
		return nil, wrapPartnerErr(err, "verifying publisher key")
	}
	if _, err := s.Update(func(c *Creds) { c.PublisherKey = key }); err != nil {
		return nil, err
	}
	return apps, nil
}

func RemovePublisherKey(s Store) error {
	c, err := s.Load()
	if err != nil {
		return err
	}
	if !c.HasPublisherKey() {
		return nil
	}
	_, err = s.Update(func(c *Creds) { c.PublisherKey = "" })
	return err
}

// ListApps returns the apps the stored publisher key controls.
func ListApps(ctx context.Context, s Store) ([]partner.App, error) {
	pc, err := s.PartnerClient()
	if err != nil {
		return nil, err
	}
	apps, err := pc.Apps(ctx)
	if err != nil {
		return nil, wrapPartnerErr(err, "listing partner apps")
	}
	return apps, nil
}
