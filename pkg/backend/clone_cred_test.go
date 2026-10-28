package backend

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/soft-serve/git"
)

// TestCloneErrorDoesNotLeakPassword uses a loopback fixture server that
// always answers 401. The clone error is what ImportRepository logs, so it
// must not contain the password embedded in the remote URL.
func TestCloneErrorDoesNotLeakPassword(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("WWW-Authenticate", `Basic realm="x"`)
		http.Error(w, "no", http.StatusUnauthorized)
	}))
	defer srv.Close()
	remote := strings.Replace(srv.URL, "http://", "http://user:s3cretfixture@", 1) + "/x.git"
	err := git.Clone(remote, filepath.Join(t.TempDir(), "dst.git"), git.CloneOptions{Bare: true, Quiet: true})
	if err == nil {
		t.Fatal("clone unexpectedly succeeded")
	}
	t.Logf("error: %v", err)
	if strings.Contains(err.Error(), "s3cretfixture") {
		t.Errorf("clone error contains the remote password")
	}
	if got := redactRemote(remote); strings.Contains(got, "s3cretfixture") {
		t.Errorf("redactRemote leaked: %s", got)
	}
}
