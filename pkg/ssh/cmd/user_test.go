package cmd

import (
	"crypto/ed25519"
	"crypto/rand"
	"strings"
	"testing"

	"github.com/charmbracelet/soft-serve/pkg/sshutils"
	"github.com/matryer/is"
	gossh "golang.org/x/crypto/ssh"
)

func TestUserCreateKeyArgs(t *testing.T) {
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	spk, err := gossh.NewPublicKey(pub)
	if err != nil {
		t.Fatal(err)
	}
	full := sshutils.MarshalAuthorizedKey(spk)
	fields := strings.Fields(full)

	tests := []struct {
		name    string
		args    []string
		wantKey bool
		wantErr bool
	}{
		{"quoted key", []string{"create", "u1", "-k", full}, true, false},
		{"quoted key with comment", []string{"create", "u1", "-k", full + " me@host"}, true, false},
		{"split key", []string{"create", "u1", "-k", fields[0], fields[1]}, true, false},
		{"split key with comment", []string{"create", "u1", "-k", fields[0], fields[1], "me@host"}, true, false},
		{"split key, flags before username", []string{"create", "-a", "-k", fields[0], "u1", fields[1]}, true, false},
		{"no key", []string{"create", "u1"}, false, false},
		{"extra arg without key", []string{"create", "u1", "extra"}, false, true},
		{"split key invalid", []string{"create", "u1", "-k", fields[0], "notbase64!"}, false, true},
		{"missing username", []string{"create"}, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := is.New(t)
			ctx, be := newAuthTestContext(t)
			ctx = withUser(t, ctx, be, "root", true)

			err := runUser(t, ctx, tt.args...)
			if tt.wantErr {
				is.True(err != nil)
				_, uerr := be.User(ctx, "u1")
				is.True(uerr != nil) // user must not be created
				return
			}
			is.NoErr(err)

			u, err := be.User(ctx, "u1")
			is.NoErr(err)
			got, err := be.UserByPublicKey(ctx, spk)
			if tt.wantKey {
				is.NoErr(err)
				is.Equal(got.Username(), u.Username())
			} else {
				is.True(err != nil)
			}
		})
	}
}
