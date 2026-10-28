package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBadSSHKeyPair(t *testing.T) {
	for _, cfg := range []*Config{
		nil,
		{},
	} {
		if _, err := KeyPair(cfg); err == nil {
			t.Errorf("cfg.SSH.KeyPair() => _, nil, want non-nil error")
		}
	}
}

func TestGoodSSHKeyPair(t *testing.T) {
	cfg := &Config{
		SSH: SSHConfig{
			KeyPath: filepath.Join(t.TempDir(), "ssh_host_ed25519_key"),
		},
	}

	if _, err := KeyPair(cfg); err != nil {
		t.Errorf("cfg.SSH.KeyPair() => _, %v, want nil error", err)
	}
}

// A missing host key must be written on first use so later calls, and the
// SSH server, see the same identity.
func TestKeyPairPersistsGeneratedKey(t *testing.T) {
	cfg := &Config{
		SSH: SSHConfig{
			KeyPath: filepath.Join(t.TempDir(), "ssh", "host_ed25519"),
		},
	}

	first, err := KeyPair(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(cfg.SSH.KeyPath); err != nil {
		t.Fatalf("key was not written: %v", err)
	}

	second, err := KeyPair(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if first.AuthorizedKey() != second.AuthorizedKey() {
		t.Error("KeyPair returned a different key on the second call")
	}
}
