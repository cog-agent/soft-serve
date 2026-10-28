package web

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/lfs"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/matryer/is"
)

func sumHex(b string) string {
	s := sha256.Sum256([]byte(b))
	return hex.EncodeToString(s[:])
}

// TestLFSBasicIntegrity reproduces, with local fixtures only, the integrity
// properties of the basic-transfer handlers. Each subtest names the scenario.
func TestLFSBasicIntegrity(t *testing.T) {
	is := is.New(t)
	ctx, be, datastore := newLFSTestContext(t)
	cfg := config.FromContext(ctx)
	dbx := db.FromContext(ctx)
	owner, err := be.CreateUser(ctx, "owner", proto.UserOptions{})
	is.NoErr(err)
	repo, err := be.CreateRepository(ctx, "r", owner, proto.RepositoryOptions{})
	is.NoErr(err)
	token, err := be.CreateAccessToken(ctx, owner, "t", time.Time{})
	is.NoErr(err)
	router := lfsTestRouter(ctx)
	base := "/r.git/info/lfs/objects/basic/"
	objPath := func(oid string) string {
		return filepath.Join(cfg.DataPath, "lfs", strconv.FormatInt(repo.ID(), 10), "objects", oid[0:2], oid[2:4], oid)
	}
	put := func(oid, body string, chunked bool) *httptest.ResponseRecorder {
		req := httptest.NewRequestWithContext(ctx, http.MethodPut, base+oid, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/octet-stream")
		req.Header.Set("Authorization", "token "+token)
		if chunked {
			req.ContentLength = -1
			req.Header.Del("Content-Length")
		} else {
			req.Header.Set("Content-Length", strconv.Itoa(len(body)))
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}

	t.Run("upload with content not matching oid is rejected and not stored", func(t *testing.T) {
		oid := strings.Repeat("c", 64)
		w := put(oid, "not the preimage", false)
		if w.Code == http.StatusOK {
			t.Errorf("status 200 for content whose sha256 != oid")
		}
		if _, err := os.Stat(objPath(oid)); err == nil {
			t.Errorf("mismatching object left on disk")
		}
		if _, err := datastore.GetLFSObjectByOid(ctx, dbx, repo.ID(), oid); err == nil {
			t.Errorf("mismatching object registered in database")
		}
	})

	t.Run("upload without Content-Length leaves no orphan file", func(t *testing.T) {
		body := "chunked body"
		oid := sumHex(body)
		w := put(oid, body, true)
		_, statErr := os.Stat(objPath(oid))
		_, dbErr := datastore.GetLFSObjectByOid(ctx, dbx, repo.ID(), oid)
		if w.Code != http.StatusOK && statErr == nil {
			t.Errorf("status %d but file left on disk with no database row", w.Code)
		}
		if (statErr == nil) != (dbErr == nil) {
			t.Errorf("disk/database disagree: stat=%v db=%v", statErr, dbErr)
		}
	})

	t.Run("recorded size is the stored size, not the header", func(t *testing.T) {
		body := "real body"
		oid := sumHex(body)
		req := httptest.NewRequestWithContext(ctx, http.MethodPut, base+oid, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/octet-stream")
		req.Header.Set("Authorization", "token "+token)
		req.Header.Set("Content-Length", "1") // lie; httptest does not enforce it
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if obj, err := datastore.GetLFSObjectByOid(ctx, dbx, repo.ID(), oid); err == nil && obj.Size != int64(len(body)) {
			t.Errorf("database size %d, stored %d", obj.Size, len(body))
		}
	})

	t.Run("verify answers every request explicitly on size mismatch with stored file", func(t *testing.T) {
		body := "verify me"
		oid := sumHex(body)
		is.Equal(put(oid, body, false).Code, http.StatusOK)
		// Make the file on disk disagree with the database row.
		is.NoErr(os.WriteFile(objPath(oid), []byte("x"), 0o644))
		req := httptest.NewRequestWithContext(ctx, http.MethodPost, base+"verify",
			strings.NewReader(`{"oid":"`+oid+`","size":`+strconv.Itoa(len(body))+`}`))
		req.Header.Set("Content-Type", lfs.MediaType)
		req.Header.Set("Accept", lfs.MediaType)
		req.Header.Set("Authorization", "token "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		// Implicit fallthrough yields an empty 200 even though stored size differs.
		if w.Code == http.StatusOK {
			t.Errorf("verify returned 200 although stored size (1) != claimed size (%d)", len(body))
		}
	})

	t.Run("download of object without database row", func(t *testing.T) {
		body := "orphan"
		oid := sumHex(body)
		is.NoErr(os.MkdirAll(filepath.Dir(objPath(oid)), 0o755))
		is.NoErr(os.WriteFile(objPath(oid), []byte(body), 0o644))
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, base+oid, nil)
		req.Header.Set("Authorization", "token "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Header().Get("Content-Length") != strconv.Itoa(len(body)) {
			t.Errorf("Content-Length %q for %d byte body (status %d)", w.Header().Get("Content-Length"), len(body), w.Code)
		}
	})
}
