package config

import (
	"errors"

	"github.com/charmbracelet/keygen"
)

var (
	// ErrNilConfig is returned when a nil config is passed to a function.
	ErrNilConfig = errors.New("nil config")

	// ErrEmptySSHKeyPath is returned when the SSH key path is empty.
	ErrEmptySSHKeyPath = errors.New("empty SSH key path")
)

// KeyPair returns the server's SSH key pair.
func KeyPair(cfg *Config) (*keygen.SSHKeyPair, error) {
	if cfg == nil {
		return nil, ErrNilConfig
	}

	if cfg.SSH.KeyPath == "" {
		return nil, ErrEmptySSHKeyPath
	}

	// Persist a newly generated key. Otherwise a missing key is regenerated in
	// memory on every call and never matches the host key the SSH server
	// writes to the same path.
	return keygen.New(cfg.SSH.KeyPath, keygen.WithKeyType(keygen.Ed25519), keygen.WithWrite())
}
