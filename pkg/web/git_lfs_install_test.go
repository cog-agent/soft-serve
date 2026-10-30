package web

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/soft-serve/pkg/storage"
)

// denyRename simulates a platform that refuses to rename over an existing
// file (Windows returns "Access is denied" during concurrent installs).
type denyRename struct{ *storage.LocalStorage }

func (denyRename) Rename(string, string) error { return os.ErrPermission }

func TestInstallVerifiedObjectRenameDenied(t *testing.T) {
	const body = "installed content"
	oid := sumHex(body)
	final := "objects/ab/cd/" + oid

	setup := func(t *testing.T, existing string, write bool) (denyRename, string) {
		root := t.TempDir()
		s := denyRename{storage.NewLocalStorage(root)}
		if _, err := s.Put("tmp/x", strings.NewReader(body)); err != nil {
			t.Fatal(err)
		}
		if write {
			if _, err := s.Put(final, strings.NewReader(existing)); err != nil {
				t.Fatal(err)
			}
		}
		return s, root
	}

	t.Run("identical existing object is accepted", func(t *testing.T) {
		s, root := setup(t, body, true)
		if err := installVerifiedObject(s, "tmp/x", final, oid); err != nil {
			t.Fatalf("got %v, want nil", err)
		}
		if _, err := os.Stat(filepath.Join(root, "tmp", "x")); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("temp object not removed: %v", err)
		}
	})
	t.Run("mismatching existing object is not accepted", func(t *testing.T) {
		s, _ := setup(t, "corrupt", true)
		if err := installVerifiedObject(s, "tmp/x", final, oid); !errors.Is(err, os.ErrPermission) {
			t.Fatalf("got %v, want rename error", err)
		}
	})
	t.Run("missing destination returns the rename error", func(t *testing.T) {
		s, _ := setup(t, "", false)
		if err := installVerifiedObject(s, "tmp/x", final, oid); !errors.Is(err, os.ErrPermission) {
			t.Fatalf("got %v, want rename error", err)
		}
	})
}
