package backend

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/charmbracelet/soft-serve/pkg/lfs"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/matryer/is"
)

// fakeLFSClient records every Download batch and serves content from memory.
type fakeLFSClient struct {
	mu      sync.Mutex
	batches [][]string
	err     error
}

func (c *fakeLFSClient) Download(_ context.Context, objects []lfs.Pointer, cb lfs.DownloadCallback) error {
	c.mu.Lock()
	oids := make([]string, 0, len(objects))
	for _, o := range objects {
		oids = append(oids, o.Oid)
	}
	c.batches = append(c.batches, oids)
	c.mu.Unlock()

	if c.err != nil {
		return c.err
	}

	for _, o := range objects {
		if err := cb(o, io.NopCloser(strings.NewReader(lfsTestContent(o.Oid))), nil); err != nil {
			return err
		}
	}
	return nil
}

func (c *fakeLFSClient) Upload(context.Context, []lfs.Pointer, lfs.UploadCallback) error {
	return errors.New("unexpected upload")
}

// lfsTestContent returns placeholder object content; the fake client does not
// verify it against the oid.
func lfsTestContent(oid string) string { return "content of " + oid }

// commitLFSPointers commits n distinct LFS pointer files to the bare
// repository at repoPath and returns their pointers. Git is run with a
// test-local configuration so the host's config does not influence it.
func commitLFSPointers(t *testing.T, repoPath string, n int) []lfs.Pointer {
	t.Helper()
	is := is.New(t)

	home := t.TempDir()
	env := append(os.Environ(),
		"HOME="+home,
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_SYSTEM=/dev/null",
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_DIR="+repoPath,
		"GIT_INDEX_FILE="+filepath.Join(home, "index"),
		"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@example.com",
	)
	run := func(stdin string, args ...string) string {
		t.Helper()
		cmd := exec.CommandContext(t.Context(), "git", args...)
		cmd.Env = env
		cmd.Stdin = strings.NewReader(stdin)
		var out, errb bytes.Buffer
		cmd.Stdout, cmd.Stderr = &out, &errb
		if err := cmd.Run(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, errb.String())
		}
		return strings.TrimSpace(out.String())
	}

	pointers := make([]lfs.Pointer, 0, n)
	for i := 0; i < n; i++ {
		p, err := lfs.GeneratePointer(strings.NewReader(fmt.Sprintf("object %d", i)))
		is.NoErr(err)
		pointers = append(pointers, p)
		blob := run(p.String(), "hash-object", "-w", "--stdin")
		run("", "update-index", "--add", "--cacheinfo", "100644,"+blob+","+fmt.Sprintf("file%03d", i))
	}

	tree := run("", "write-tree")
	commit := run("", "commit-tree", "-m", "pointers", tree)
	run("", "update-ref", "refs/heads/main", commit)
	return pointers
}

func TestStoreRepoMissingLFSObjectsBatches(t *testing.T) {
	tests := []struct {
		objects int
		batches []int
	}{
		{objects: 1, batches: []int{1}},
		{objects: 19, batches: []int{19}},
		{objects: 20, batches: []int{20}},
		{objects: 21, batches: []int{20, 1}},
		{objects: 45, batches: []int{20, 20, 5}},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("%d objects", tc.objects), func(t *testing.T) {
			is := is.New(t)
			be, cfg := newTestBackend(t)
			ctx := config.WithContext(context.Background(), cfg)

			_, err := be.CreateRepository(ctx, "repo", nil, proto.RepositoryOptions{})
			is.NoErr(err)
			repo, err := be.Repository(ctx, "repo")
			is.NoErr(err)

			pointers := commitLFSPointers(t, be.repoPath("repo"), tc.objects)

			client := &fakeLFSClient{}
			is.NoErr(StoreRepoMissingLFSObjects(ctx, repo, be.db, be.store, client))

			// Batch sizes are deterministic; which objects land in which
			// batch depends on scan order, so only sizes and the union are
			// checked.
			var sizes []int
			seen := map[string]int{}
			for _, b := range client.batches {
				sizes = append(sizes, len(b))
				for _, oid := range b {
					seen[oid]++
				}
			}
			is.Equal(sizes, tc.batches)
			is.Equal(len(seen), tc.objects)

			for _, p := range pointers {
				is.Equal(seen[p.Oid], 1)

				obj, err := be.store.GetLFSObjectByOid(ctx, be.db, repo.ID(), p.Oid)
				is.NoErr(err)
				is.Equal(obj.Size, p.Size)

				data, err := os.ReadFile(filepath.Join(cfg.DataPath, "lfs", fmt.Sprint(repo.ID()), "objects", p.RelativePath()))
				is.NoErr(err)
				is.Equal(string(data), lfsTestContent(p.Oid))
			}
		})
	}
}

func TestStoreRepoMissingLFSObjectsFinalBatchError(t *testing.T) {
	is := is.New(t)
	be, cfg := newTestBackend(t)
	ctx := config.WithContext(context.Background(), cfg)

	_, err := be.CreateRepository(ctx, "repo", nil, proto.RepositoryOptions{})
	is.NoErr(err)
	repo, err := be.Repository(ctx, "repo")
	is.NoErr(err)

	// A single object is only ever downloaded as the final partial batch.
	commitLFSPointers(t, be.repoPath("repo"), 1)

	want := errors.New("download failed")
	client := &fakeLFSClient{err: want}
	err = StoreRepoMissingLFSObjects(ctx, repo, be.db, be.store, client)
	is.True(errors.Is(err, want))
	is.Equal(len(client.batches), 1)
}
