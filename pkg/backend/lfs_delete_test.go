package backend

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/charmbracelet/soft-serve/pkg/store"
	"github.com/matryer/is"
)

// TestDeleteRepositoryRemovesLFSStorage checks that nothing under the
// repository's LFS directory survives deletion: not registered objects, not
// orphans without a database row, and not interrupted-upload temp files. The
// context carries no user, as for an anonymous push-create cleaned up after
// failing; that used to panic while building the delete event.
func TestDeleteRepositoryRemovesLFSStorage(t *testing.T) {
	is := is.New(t)
	be, cfg := newTestBackend(t)
	ctx := config.WithContext(context.Background(), cfg)
	ctx = db.WithContext(ctx, be.db)
	ctx = store.WithContext(ctx, be.store)
	user, err := be.CreateUser(ctx, "alice", proto.UserOptions{})
	is.NoErr(err)
	r, err := be.CreateRepository(ctx, "gone", user, proto.RepositoryOptions{})
	is.NoErr(err)

	dir := filepath.Join(cfg.DataPath, "lfs", strconv.FormatInt(r.ID(), 10))
	for _, f := range []string{"objects/aa/bb/orphan", "tmp/interrupted"} {
		is.NoErr(os.MkdirAll(filepath.Dir(filepath.Join(dir, f)), 0o755))
		is.NoErr(os.WriteFile(filepath.Join(dir, f), []byte("x"), 0o644))
	}

	is.NoErr(be.DeleteRepository(ctx, "gone"))
	if _, err := os.Stat(dir); err == nil {
		t.Errorf("LFS directory %s survived repository deletion", dir)
	}
}
