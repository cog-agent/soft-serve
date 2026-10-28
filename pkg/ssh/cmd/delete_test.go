package cmd

import (
	"errors"
	"testing"

	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/matryer/is"
)

func TestDeleteRestrictedToOwnerOrAdmin(t *testing.T) {
	is := is.New(t)
	ctx, be := newAuthTestContext(t)
	ownerCtx := withUser(t, ctx, be, "owner", false)
	_, err := be.CreateRepository(ownerCtx, "proj", proto.UserFromContext(ownerCtx), proto.RepositoryOptions{})
	is.NoErr(err)

	// Collaborators, even with admin-access, cannot delete.
	collabCtx := withUser(t, ctx, be, "collab", false)
	for _, level := range []access.AccessLevel{access.ReadWriteAccess, access.AdminAccess} {
		_ = be.RemoveCollaborator(ownerCtx, "proj", "collab")
		is.NoErr(be.AddCollaborator(ownerCtx, "proj", "collab", level))
		err = runRepo(t, collabCtx, "delete", "proj")
		is.True(errors.Is(err, proto.ErrUnauthorized))
		_, err = be.Repository(ctx, "proj")
		is.NoErr(err)
	}

	// Anonymous callers cannot delete, even with anonymous admin-access.
	is.NoErr(be.SetAnonAccess(ctx, access.AdminAccess))
	err = runRepo(t, ctx, "delete", "proj")
	is.True(errors.Is(err, proto.ErrUnauthorized))
	_, err = be.Repository(ctx, "proj")
	is.NoErr(err)
	is.NoErr(be.SetAnonAccess(ctx, access.ReadOnlyAccess))

	// The owner may delete.
	is.NoErr(runRepo(t, ownerCtx, "delete", "proj"))
	_, err = be.Repository(ctx, "proj")
	is.True(errors.Is(err, proto.ErrRepoNotFound))

	// A server admin may delete someone else's repository.
	_, err = be.CreateRepository(ownerCtx, "other", proto.UserFromContext(ownerCtx), proto.RepositoryOptions{})
	is.NoErr(err)
	adminCtx := withUser(t, ctx, be, "root", true)
	is.NoErr(runRepo(t, adminCtx, "delete", "other"))
	_, err = be.Repository(ctx, "other")
	is.True(errors.Is(err, proto.ErrRepoNotFound))
}

// TestDeletePrivateAndMissingAnswerAlike verifies non-owners learn nothing
// about private repositories from delete.
func TestDeletePrivateAndMissingAnswerAlike(t *testing.T) {
	ctx, ownerCtx, be := setupSecretRepo(t)
	_ = ownerCtx
	assertSameAnswer(t, ctx, proto.ErrUnauthorized)
	assertSameAnswer(t, withUser(t, ctx, be, "stranger", false), proto.ErrRepoNotFound)
}
