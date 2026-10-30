package backend

import (
	"context"
	"testing"
	"time"

	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/charmbracelet/soft-serve/pkg/store"
	"github.com/charmbracelet/soft-serve/pkg/webhook"
	"github.com/matryer/is"
)

// TestUserInputIsSanitizedOnWrite covers the stored fields named in
// GHSA-fv2r-r8mp-pg48 (repository description, project name, access token
// name, webhook URL), using escape forms the original fix did not strip:
// UTF-8 encoded C1 controls and bare C0 controls such as CR.
func TestUserInputIsSanitizedOnWrite(t *testing.T) {
	is := is.New(t)
	be, _ := newTestBackend(t)
	ctx := context.Background()
	wctx := store.WithContext(db.WithContext(ctx, be.db), be.store)

	user, err := be.CreateUser(ctx, "alice", proto.UserOptions{})
	is.NoErr(err)
	r, err := be.CreateRepository(ctx, "proj", user, proto.RepositoryOptions{})
	is.NoErr(err)

	const evil = "ok\x1b[31m\x1b]0;t\x07\u009b2J\u009d0;x\u009c\rFAKE\x08"
	const want = "okFAKE"

	is.NoErr(be.SetDescription(ctx, "proj", evil))
	is.NoErr(be.SetProjectName(ctx, "proj", evil))
	r, err = be.Repository(ctx, "proj")
	is.NoErr(err)
	is.Equal(r.Description(), want)
	is.Equal(r.ProjectName(), want)

	_, err = be.CreateAccessToken(ctx, user, evil, time.Time{})
	is.NoErr(err)
	tokens, err := be.ListAccessTokens(ctx, user)
	is.NoErr(err)
	is.Equal(len(tokens), 1)
	is.Equal(tokens[0].Name, want)

	is.NoErr(be.CreateWebhook(wctx, r, "http://93.184.216.34/\x1b[31mhook\u009b2J\r", webhook.ContentTypeJSON, "", []webhook.Event{webhook.EventPush}, true))
	hooks, err := be.ListWebhooks(wctx, r)
	is.NoErr(err)
	is.Equal(len(hooks), 1)
	is.Equal(hooks[0].URL, "http://93.184.216.34/hook")
}
