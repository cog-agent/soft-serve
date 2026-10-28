package git

import (
	"path/filepath"
	"testing"
)

func TestEnsureWithin(t *testing.T) {
	root := filepath.Join(t.TempDir(), "repos")
	for _, tc := range []struct {
		name, repo string
		ok         bool
	}{
		{"plain", "a.git", true},
		{"nested", "a/b.git", true},
		{"dotdot out", "../x.git", false},
		{"sibling sharing prefix", "../repos-evil/x.git", false},
		{"sibling sharing prefix 2", "../reposx", false},
		{"deep dotdot", "a/../../etc/passwd", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := EnsureWithin(root, tc.repo)
			if (err == nil) != tc.ok {
				t.Errorf("EnsureWithin(%q) = %v, want ok=%v", tc.repo, err, tc.ok)
			}
		})
	}
}
