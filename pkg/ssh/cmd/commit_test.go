package cmd

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/matryer/is"
)

// seedCommit writes a single commit with the given author name and message
// into the bare repository and points HEAD's branch at it.
func seedCommit(t *testing.T, ctx context.Context, repo, author, message string) string {
	t.Helper()
	dir := filepath.Join(config.FromContext(ctx).DataPath, "repos", repo+".git")
	run := func(stdin string, args ...string) string {
		t.Helper()
		c := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
		c.Env = append(os.Environ(),
			"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null",
			"GIT_AUTHOR_NAME="+author, "GIT_AUTHOR_EMAIL=a@example.com",
			"GIT_COMMITTER_NAME=committer", "GIT_COMMITTER_EMAIL=c@example.com",
		)
		c.Stdin = strings.NewReader(stdin)
		out, err := c.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	tree := run("", "mktree")
	sha := run(message, "commit-tree", tree)
	run("", "update-ref", "refs/heads/master", sha)
	return sha
}

// TestCommitRejectsOptionInjection covers GHSA-33pr-m977-5w97:
// `repo commit REPO -- --output=/path` must never reach git as an option.
func TestCommitRejectsOptionInjection(t *testing.T) {
	is := is.New(t)
	ctx, be := newAuthTestContext(t)
	ownerCtx := withUser(t, ctx, be, "owner", false)
	_, err := be.CreateRepository(ownerCtx, "proj", proto.UserFromContext(ownerCtx), proto.RepositoryOptions{})
	is.NoErr(err)
	sha := seedCommit(t, ctx, "proj", "author", "message")

	pwned := filepath.Join(t.TempDir(), "pwned")
	for _, rev := range []string{
		"--output=" + pwned,
		"--output",
		"-p",
		"--exec-path=" + pwned,
		"--git-dir=" + pwned,
		"-c" + "core.pager=touch " + pwned,
		"--upload-pack=touch " + pwned,
	} {
		_, _, err := runRepoOutput(t, ownerCtx, "commit", "proj", "--", rev)
		if err == nil {
			t.Errorf("revision %q was accepted", rev)
		}
		if _, statErr := os.Stat(pwned); statErr == nil {
			t.Fatalf("revision %q created %s", rev, pwned)
		}
	}

	// The library underneath must also treat the revision as data, so a
	// future caller that forgets the check above is still safe.
	rr, err := git.Open(filepath.Join(config.FromContext(ctx).DataPath, "repos", "proj.git"))
	is.NoErr(err)
	_, err = rr.CommitByRevision("--output=" + pwned)
	is.True(err != nil)
	_, statErr := os.Stat(pwned)
	is.True(os.IsNotExist(statErr))

	// A real revision still works.
	stdout, _, err := runRepoOutput(t, ownerCtx, "commit", "proj", sha)
	is.NoErr(err)
	is.True(strings.Contains(stdout, "message"))
}

// TestCommitOutputIsSanitized covers GHSA-fv2r-r8mp-pg48 residuals: author
// names and messages are repository-controlled and must not reach the
// caller's terminal with escape sequences or other control characters.
func TestCommitOutputIsSanitized(t *testing.T) {
	is := is.New(t)
	ctx, be := newAuthTestContext(t)
	ownerCtx := withUser(t, ctx, be, "owner", false)
	_, err := be.CreateRepository(ownerCtx, "proj", proto.UserFromContext(ownerCtx), proto.RepositoryOptions{})
	is.NoErr(err)

	sha := seedCommit(t, ctx, "proj",
		"evil\x1b]0;pwn\x07\u009b2Jname\rforged",
		"subject\x1b[31m red\u009d0;x\u009c\rFORGED\x07\x08\n\nbody\r\nline\x1bc")

	stdout, _, err := runRepoOutput(t, ownerCtx, "commit", "proj", sha)
	is.NoErr(err)
	// The diff of an empty tree is empty, so any control character left in
	// the output came from the author or message.
	for _, r := range stdout {
		if r != '\n' && r != '\t' && (r < 0x20 || (r >= 0x7f && r <= 0x9f)) {
			t.Fatalf("control character %U in output %q", r, stdout)
		}
	}
	is.True(strings.Contains(stdout, "subject"))
}
