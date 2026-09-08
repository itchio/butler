package steam

import (
	"path/filepath"
	"testing"
)

func TestStoreLayers(t *testing.T) {
	s := StoreFor(filepath.Join(t.TempDir(), "butler_creds"))
	if err := s.Save(&Creds{AccountName: "disk", RefreshToken: "disk-token", PublisherKey: "disk-key"}); err != nil {
		t.Fatal(err)
	}

	c, err := s.Load()
	if err != nil || c.RefreshToken != "disk-token" || c.PublisherKey != "disk-key" {
		t.Fatalf("file layer: %+v %v", c, err)
	}

	t.Setenv(EnvRefreshToken, "env-token")
	t.Setenv(EnvAccountName, "env")
	c, _ = s.Load()
	if c.RefreshToken != "env-token" || c.AccountName != "env" || c.PublisherKey != "disk-key" {
		t.Fatalf("env should override the login but keep the file's key: %+v", c)
	}

	s.Override = &Creds{PublisherKey: "flag-key"}
	c, _ = s.Load()
	if c.RefreshToken != "env-token" || c.PublisherKey != "flag-key" {
		t.Fatalf("override applies field by field: %+v", c)
	}

	// Token from one of env and flags, account name from the other.
	s.Override = &Creds{AccountName: "flag"}
	c, _ = s.Load()
	if c.RefreshToken != "env-token" || c.AccountName != "flag" {
		t.Fatalf("env token with flag name: %+v", c)
	}
	t.Setenv(EnvRefreshToken, "")
	s.Override = &Creds{RefreshToken: "flag-token"}
	c, _ = s.Load()
	if c.RefreshToken != "flag-token" || c.AccountName != "env" {
		t.Fatalf("flag token with env name: %+v", c)
	}

	// A token with no name anywhere but the file is refused rather than
	// paired with the file's name.
	t.Setenv(EnvAccountName, "")
	if _, err := s.Load(); err == nil {
		t.Fatal("token without account name should be an error")
	}
	t.Setenv(EnvRefreshToken, "env-token")
	t.Setenv(EnvAccountName, "env")
	s.Override = nil

	// Saving through Update must not carry layered state to disk.
	if _, err := s.Update(func(c *Creds) { c.PublisherKey = "new-key" }); err != nil {
		t.Fatal(err)
	}
	disk, _ := s.Persisted()
	if disk.RefreshToken != "disk-token" || disk.PublisherKey != "new-key" {
		t.Fatalf("update leaked layered state to disk: %+v", disk)
	}

	if err := s.Logout(); err != nil {
		t.Fatal(err)
	}
	disk, _ = s.Persisted()
	if disk.LoggedIn() || disk.HasPublisherKey() {
		t.Fatalf("logout should clear the file: %+v", disk)
	}
}
