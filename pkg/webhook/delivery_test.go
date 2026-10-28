package webhook_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/migrate"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/charmbracelet/soft-serve/pkg/store"
	"github.com/charmbracelet/soft-serve/pkg/store/database"
	"github.com/charmbracelet/soft-serve/pkg/webhook"
	"github.com/matryer/is"
	_ "modernc.org/sqlite"
)

type captured struct {
	header http.Header
	body   string
}

// deliver sends one event to a local httptest server and returns what the
// receiver saw plus the stored delivery.
func deliver(t *testing.T, secret string, contentType webhook.ContentType, respond func(http.ResponseWriter)) (captured, string) {
	t.Helper()
	is := is.New(t)
	ctx := context.Background()
	dp := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.DataPath = dp
	cfg.DB.Driver = "sqlite"
	cfg.DB.DataSource = dp + "/test.db"
	ctx = config.WithContext(ctx, cfg)
	dbx, err := db.Open(ctx, cfg.DB.Driver, cfg.DB.DataSource)
	is.NoErr(err)
	t.Cleanup(func() { dbx.Close() }) //nolint:errcheck
	is.NoErr(migrate.Migrate(ctx, dbx))
	ctx = db.WithContext(ctx, dbx)
	ds := database.New(ctx, dbx)
	ctx = store.WithContext(ctx, ds)
	be := backend.New(ctx, cfg, dbx, ds)
	ctx = backend.WithContext(ctx, be)
	owner, err := be.CreateUser(ctx, "owner", proto.UserOptions{})
	is.NoErr(err)
	repo, err := be.CreateRepository(ctx, "r", owner, proto.RepositoryOptions{})
	is.NoErr(err)

	var got captured
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		got = captured{header: r.Header.Clone(), body: string(b)}
		respond(w)
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(webhook.SetHTTPClientForTest(srv.Client()))

	id, err := ds.CreateWebhook(ctx, dbx, repo.ID(), srv.URL, secret, int(contentType), true)
	is.NoErr(err)
	hooks, err := ds.GetWebhooksByRepoID(ctx, dbx, repo.ID())
	is.NoErr(err)
	var hook = hooks[0]
	is.Equal(hook.ID, id)

	payload := webhook.Common{EventType: webhook.EventPush, Repository: webhook.Repository{ID: repo.ID(), Name: "r"}}
	is.NoErr(webhook.SendWebhook(ctx, hook, webhook.EventPush, payload))

	ds2, err := ds.GetWebhookDeliveriesByWebhookID(ctx, dbx, id)
	is.NoErr(err)
	is.Equal(len(ds2), 1)
	return got, ds2[0].ResponseBody
}

// TestSignatureMatchesBody: the receiver can recompute the HMAC over the exact
// bytes it received, for both content types.
func TestSignatureMatchesBody(t *testing.T) {
	for _, ct := range []webhook.ContentType{webhook.ContentTypeJSON, webhook.ContentTypeForm} {
		got, _ := deliver(t, "s3cret", ct, func(http.ResponseWriter) {})
		mac := hmac.New(sha256.New, []byte("s3cret"))
		mac.Write([]byte(got.body))
		want := "sha256=" + hex.EncodeToString(mac.Sum(nil))
		if got.header.Get("X-SoftServe-Signature") != want {
			t.Errorf("%v: signature %q, want %q", ct, got.header.Get("X-SoftServe-Signature"), want)
		}
	}
}

// TestNoSecretNoSignature: an unsigned hook sends no signature header.
func TestNoSecretNoSignature(t *testing.T) {
	got, _ := deliver(t, "", webhook.ContentTypeJSON, func(http.ResponseWriter) {})
	if v := got.header.Get("X-SoftServe-Signature"); v != "" {
		t.Errorf("unexpected signature %q", v)
	}
}

// TestSecretNotInTransport: the secret itself is never sent.
func TestSecretNotInTransport(t *testing.T) {
	got, _ := deliver(t, "s3cret-value", webhook.ContentTypeJSON, func(http.ResponseWriter) {})
	if strings.Contains(got.body, "s3cret-value") {
		t.Error("secret present in body")
	}
	for k, v := range got.header {
		if strings.Contains(strings.Join(v, ","), "s3cret-value") {
			t.Errorf("secret present in header %s", k)
		}
	}
}

// TestStoredResponseBodyIsBounded: a receiver controls the response size; the
// stored delivery must not grow without limit.
func TestStoredResponseBodyIsBounded(t *testing.T) {
	const sent = 8 << 20
	_, stored := deliver(t, "", webhook.ContentTypeJSON, func(w http.ResponseWriter) {
		w.Write([]byte(strings.Repeat("A", sent))) //nolint:errcheck
	})
	if len(stored) >= sent {
		t.Errorf("stored %d bytes of a %d byte response; no cap", len(stored), sent)
	}
}
