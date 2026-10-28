package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/charmbracelet/soft-serve/pkg/lfs"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/matryer/is"
)

// TestLFSConcurrentUploadSameOID uploads one object from many writers at once.
// Every request carries identical, verified content, so every request must
// succeed; a loser of the race must not see a database error.
func TestLFSConcurrentUploadSameOID(t *testing.T) {
	is := is.New(t)
	ctx, be, _ := newLFSTestContext(t)
	owner, err := be.CreateUser(ctx, "owner", proto.UserOptions{})
	is.NoErr(err)
	_, err = be.CreateRepository(ctx, "r", owner, proto.RepositoryOptions{})
	is.NoErr(err)
	token, err := be.CreateAccessToken(ctx, owner, "t", time.Time{})
	is.NoErr(err)
	router := lfsTestRouter(ctx)

	body := strings.Repeat("concurrent ", 1000)
	oid := sumHex(body)
	const n = 16
	codes := make([]int, n)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequestWithContext(ctx, http.MethodPut, "/r.git/info/lfs/objects/basic/"+oid, strings.NewReader(body))
			req.Header.Set("Content-Type", "application/octet-stream")
			req.Header.Set("Authorization", "token "+token)
			w := httptest.NewRecorder()
			<-start
			router.ServeHTTP(w, req)
			codes[i] = w.Code
		}()
	}
	close(start)
	wg.Wait()
	for i, c := range codes {
		if c != http.StatusOK {
			t.Errorf("upload %d: status %d, want 200", i, c)
		}
	}
}

// TestLFSCrossRepoIsolation checks that an object stored in a private
// repository is invisible through any other repository and to non-readers.
func TestLFSCrossRepoIsolation(t *testing.T) {
	is := is.New(t)
	ctx, be, _ := newLFSTestContext(t)
	alice, err := be.CreateUser(ctx, "alice", proto.UserOptions{})
	is.NoErr(err)
	bob, err := be.CreateUser(ctx, "bob", proto.UserOptions{})
	is.NoErr(err)
	_, err = be.CreateRepository(ctx, "secret", alice, proto.RepositoryOptions{Private: true})
	is.NoErr(err)
	_, err = be.CreateRepository(ctx, "pub", bob, proto.RepositoryOptions{})
	is.NoErr(err)
	aTok, err := be.CreateAccessToken(ctx, alice, "t", time.Time{})
	is.NoErr(err)
	bTok, err := be.CreateAccessToken(ctx, bob, "t", time.Time{})
	is.NoErr(err)
	router := lfsTestRouter(ctx)

	body := "private-fixture-content"
	oid := sumHex(body)
	do := func(method, target, tok, ct, payload string) *httptest.ResponseRecorder {
		req := httptest.NewRequestWithContext(ctx, method, target, strings.NewReader(payload))
		if ct != "" {
			req.Header.Set("Content-Type", ct)
			req.Header.Set("Accept", lfs.MediaType)
		}
		if tok != "" {
			req.Header.Set("Authorization", "token "+tok)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}
	is.Equal(do(http.MethodPut, "/secret.git/info/lfs/objects/basic/"+oid, aTok, "application/octet-stream", body).Code, http.StatusOK)

	ptr := `{"oid":"` + oid + `","size":` + strconv.Itoa(len(body)) + `}`
	batch := `{"operation":"download","transfers":["basic"],"objects":[` + ptr + `]}`

	t.Run("owner can download", func(t *testing.T) {
		w := do(http.MethodGet, "/secret.git/info/lfs/objects/basic/"+oid, aTok, "", "")
		if w.Code != http.StatusOK || w.Body.String() != body {
			t.Errorf("owner download: %d", w.Code)
		}
	})
	t.Run("other user cannot read private repo objects", func(t *testing.T) {
		for _, tok := range []string{bTok, ""} {
			if w := do(http.MethodGet, "/secret.git/info/lfs/objects/basic/"+oid, tok, "", ""); w.Code == http.StatusOK || strings.Contains(w.Body.String(), body) {
				t.Errorf("token %q: status %d leaked object", tok, w.Code)
			}
			if w := do(http.MethodPost, "/secret.git/info/lfs/objects/batch", tok, lfs.MediaType, batch); w.Code == http.StatusOK {
				t.Errorf("token %q: batch on private repo: %d", tok, w.Code)
			}
		}
	})
	t.Run("object is not reachable through another repository", func(t *testing.T) {
		if w := do(http.MethodGet, "/pub.git/info/lfs/objects/basic/"+oid, bTok, "", ""); w.Code == http.StatusOK {
			t.Errorf("download via pub: %d", w.Code)
		}
		w := do(http.MethodPost, "/pub.git/info/lfs/objects/batch", bTok, lfs.MediaType, batch)
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"error"`) || strings.Contains(w.Body.String(), "download") {
			t.Errorf("batch via pub should report not found: %d %s", w.Code, w.Body.String())
		}
		if w := do(http.MethodPost, "/pub.git/info/lfs/objects/basic/verify", bTok, lfs.MediaType, ptr); w.Code == http.StatusOK {
			t.Errorf("verify via pub: %d", w.Code)
		}
	})
	t.Run("upload of same oid to another repo is stored independently", func(t *testing.T) {
		is.Equal(do(http.MethodPut, "/pub.git/info/lfs/objects/basic/"+oid, bTok, "application/octet-stream", body).Code, http.StatusOK)
	})
}

// TestLFSMalformedOIDAndSize feeds malformed identifiers and sizes to every
// LFS object endpoint. None may succeed, panic, or produce a 5xx.
func TestLFSMalformedOIDAndSize(t *testing.T) {
	is := is.New(t)
	ctx, be, _ := newLFSTestContext(t)
	owner, err := be.CreateUser(ctx, "owner", proto.UserOptions{})
	is.NoErr(err)
	_, err = be.CreateRepository(ctx, "r", owner, proto.RepositoryOptions{})
	is.NoErr(err)
	tok, err := be.CreateAccessToken(ctx, owner, "t", time.Time{})
	is.NoErr(err)
	router := lfsTestRouter(ctx)
	do := func(method, target, ct, payload string) *httptest.ResponseRecorder {
		req := httptest.NewRequestWithContext(ctx, method, target, strings.NewReader(payload))
		req.Header.Set("Authorization", "token "+tok)
		if ct != "" {
			req.Header.Set("Content-Type", ct)
			req.Header.Set("Accept", lfs.MediaType)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}
	good := strings.Repeat("a", 64)
	oids := []string{
		"", "a", strings.Repeat("A", 64), strings.Repeat("a", 63), strings.Repeat("a", 65),
		"../../../../etc/passwd", strings.Repeat("g", 64), good + "\n", "aa/" + good[3:],
		"..%2f" + strings.Repeat("a", 59), strings.Repeat("a", 62) + "\x00a",
	}
	sizes := []string{"-1", "-9223372036854775808", "9223372036854775807", "1e3", `"1"`, "null", "0"}
	for _, oid := range oids {
		js := strconv.Quote(oid)
		for _, size := range sizes {
			ptr := `{"oid":` + js + `,"size":` + size + `}`
			for _, op := range []string{"download", "upload"} {
				w := do(http.MethodPost, "/r.git/info/lfs/objects/batch", lfs.MediaType, `{"operation":"`+op+`","objects":[`+ptr+`]}`)
				if w.Code >= 500 {
					t.Errorf("batch %s %q size %s: %d", op, oid, size, w.Code)
				}
				if w.Code == http.StatusOK && strings.Contains(w.Body.String(), `"actions"`) && !isGoodPtr(oid, size) {
					t.Errorf("batch %s issued actions for oid %q size %s", op, oid, size)
				}
			}
			w := do(http.MethodPost, "/r.git/info/lfs/objects/basic/verify", lfs.MediaType, ptr)
			if w.Code == http.StatusOK || w.Code >= 500 {
				t.Errorf("verify %q size %s: %d", oid, size, w.Code)
			}
		}
		for _, m := range []string{http.MethodGet, http.MethodPut} {
			w := do(m, "/r.git/info/lfs/objects/basic/"+url.PathEscape(oid), "application/octet-stream", "x")
			if w.Code == http.StatusOK || w.Code >= 500 {
				t.Errorf("%s oid %q: %d", m, oid, w.Code)
			}
		}
	}
}

func isGoodPtr(oid, size string) bool {
	p := lfs.Pointer{Oid: oid}
	n, err := strconv.ParseInt(size, 10, 64)
	p.Size = n
	return err == nil && p.IsValid()
}
