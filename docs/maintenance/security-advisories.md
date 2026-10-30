# Upstream security advisory audit

Audit of the 23 published security advisories for `charmbracelet/soft-serve` (input: `advisory-data.json`) against this fork. Base commit `19cb4da`, branch `cog/security-advisories`, plus the uncommitted changes listed under "Changes made".

Method: for each advisory, the described root cause was located in the current source and the existing tests were read to see whether they exercise it. Advisory text is untrusted input; nothing in it was executed. Exploits were not replayed against a live server, and no live service was deployed. Evidence is source locations (line numbers are at the audited tree) and two kinds of test, which are not equivalent. Go tests (`go test`) run in-process (sqlite, `httptest`, temp dirs) and do not start the real server binary. Only a full real testscript execution (the `testscript/testdata/*.txtar` suites) drives the actual server and SSH/git clients end to end; a txtar file cited below counts as end-to-end evidence only where this document says it was run. The new utils, commit and webhook tests were independently verified red on the base commit `19cb4da` (they fail there) and green with the changes. Where a claim depends on something that was not run here, the entry says so.

Disposition key:

- **Not affected**: the vulnerable behaviour is absent from this tree, with source and test evidence.
- **Residual gap, fixed**: part of the advisory's class was still reachable, and is fixed in this change.
- **Not reachable, hardened**: the described exploit does not work here; a defense-in-depth check and regression test were added.

Summary: 21 advisories not affected, 1 with a residual gap fixed (GHSA-fv2r-r8mp-pg48), 1 not reachable and hardened (GHSA-33pr-m977-5w97). Regression tests were added for four advisories that lacked a Go-level test (GHSA-6jm8-x3g6-r33j, GHSA-fgxm-5hvv-x4vh, and the two above).

## Changes made

- `pkg/utils/utils.go` `Sanitize`: now also removes UTF-8 encoded C1 controls (rewritten to their ESC forms first so whole sequences such as `U+009B 31m` or `U+009D ... U+009C` go) and every other control character except `\n` and `\t` (CR, BS, BEL, NUL, DEL). `ansi.Strip` alone passes all of these through (probed before the change).
- Output sites now sanitize repository-controlled text: `repo webhook deliveries get` (`pkg/ssh/cmd/webhooks.go`), branch and tag names in `repo info`, `repo branch list|default` and `repo tag list` (`pkg/ssh/cmd/repo.go`, `branch.go`, `tag.go`), and commit titles, authors and messages in the TUI (`pkg/ui/pages/repo/log.go`, `logitem.go`, `files.go`).
- `pkg/ssh/cmd/commit.go`: a revision starting with `-` is rejected before it reaches git.
- Tests: `pkg/utils/utils_test.go` (`TestSanitizeStripsTerminalControls`), `pkg/backend/sanitize_test.go`, `pkg/ssh/cmd/commit_test.go`, `pkg/ssh/cmd/webhooks_test.go` (`TestWebhookDeliveryOutputIsSanitized`), `pkg/web/git_lfs_test.go` (`TestLFSLockForceDeleteRequiresAdmin`), `pkg/web/oracle_test.go` (`TestGoGetDoesNotRevealPrivateRepositories`).

## Not changed, by decision

- `repo blob` and the commit diff print repository file contents and file names as git does. Sanitizing would corrupt the content the command exists to show, and `--raw` and `--patch` are expected to be byte-faithful. File names in `repo tree` are already escaped by `common.UnquoteFilename` (`strconv.Quote`), so they are safe.
- Unicode bidirectional override characters (for example U+202E) are not control characters and still pass `Sanitize`. They are a display-spoofing issue, not terminal command injection, and were not in the advisory.
- Newlines are kept in `Sanitize`, so a multi-line token name or project name can still add lines to list output. It cannot inject terminal control sequences.

## Advisories

### [GHSA-fv2r-r8mp-pg48](https://github.com/charmbracelet/soft-serve/security/advisories/GHSA-fv2r-r8mp-pg48) (CVE-2025-64494, medium): ANSI escape sequences not sanitized

Disposition: **Residual gap, fixed.** Upstream fixed in v0.11.0.

The six named sites already called `utils.Sanitize`: description (`pkg/backend/repo.go:742`), project name (`repo.go:820`), commit author and message (`pkg/ssh/cmd/commit.go`), token name (`pkg/backend/access_token.go:17`), webhook URL (`pkg/backend/webhooks.go:21`). `Sanitize` was `ansi.Strip` only, and a probe showed these survive it: UTF-8 C1 controls (`U+009B`, `U+009D`), CR (overwrites the line, so `Author: ok\rAuthor: forged` forges a header), BS, BEL, NUL, DEL. Residual exposure also existed where repository- or receiver-controlled text was printed without any sanitizing: webhook delivery output (the response is receiver-controlled), branch and tag names (git ref names forbid C0 and DEL but allow UTF-8 C1), and the TUI commit views.

Fix and tests: see "Changes made". `TestSanitizeStripsTerminalControls` (18 cases, plus idempotence, which `SanitizeRepo` relies on), `TestUserInputIsSanitizedOnWrite` (description, project name, token name, webhook URL), `TestCommitOutputIsSanitized` (author and message through the real `commit` command), `TestWebhookDeliveryOutputIsSanitized`. With the fixes reverted, the utils, commit and webhook tests fail on the C1 and CR characters; they pass with the fixes. Not tested: TUI rendering (no test harness for those views); the TUI change is a call to the same function.

### [GHSA-33pr-m977-5w97](https://github.com/charmbracelet/soft-serve/security/advisories/GHSA-33pr-m977-5w97) (CVE-2025-58355, high): arbitrary file write through SSH API

Disposition: **Not reachable, hardened.** Upstream fixed in 0.10.0. The PoC is `repo commit icecream -- --output=/tmp/pwned`.

The revision argument `args[1]` goes to `Repository.CommitByRevision` (`pkg/ssh/cmd/commit.go`), which in `git-module` runs `git log ... --end-of-options <rev> --`, so git treats it as a revision. It is rejected as a bad revision and no file is written. `Diff` and `Patch` take `commit.ID.String()`, a hash from git, not user input. Other callers of `Log` build the revision from hashes (`pkg/webhook/push.go`). Import's `git clone` also uses `--end-of-options` and `ValidateGitRemote` rejects ssh users or hosts starting with `-` (`pkg/ssrf/git.go:313`).

This relies on `git-module` passing `--end-of-options`. The dependency version is pinned in `go.mod`, but the behaviour is still a property of third-party code that a later bump could change, so the command now also rejects a revision starting with `-` as defense in depth. `TestCommitRejectsOptionInjection` tries `--output=`, `--output`, `-p`, `--exec-path=`, `--git-dir=`, `-ccore.pager=...` and `--upload-pack=` after `--`, asserts an error and that the target file does not exist, then calls `CommitByRevision` directly to check the library behaviour, and finally checks a real SHA still works. That test passes with and without the new guard, which confirms the exploit was not reachable beforehand; the guard is defense in depth.

### [GHSA-64rg-mqfg-gqg9](https://github.com/charmbracelet/soft-serve/security/advisories/GHSA-64rg-mqfg-gqg9) (high): case or Unicode variants on case-insensitive filesystems

Disposition: **Not affected.** Upstream fixed in 0.12.4.

- Creation claims the directory with `os.Mkdir` and maps `ErrExist` to `ErrRepoExist` (`pkg/backend/repo.go:91`, used at `:75` and `:198`). The filesystem decides sameness, so case and normalization variants are refused on APFS or NTFS too.
- A directory with no database row is taken, not free: `accessLevel` returns `NoAccess` when `repoPathExists` (`pkg/backend/user.go:129`).
- `SanitizeRepo` is a fixed point that strips repeated `.git` (`pkg/utils/utils.go:13`), and `ValidateRepo` rejects non-canonical names and any segment ending in `.git` (`:73`; called at `repo.go:70`, `:198`, `:438`, `:443`).
- Tests: `TestCaseVariantOfExistingRepo`, `TestOrphanRepoDirectoryIsNotClaimable`, `TestConcurrentCreateSameName` (`pkg/backend/repo_path_test.go`), `TestValidateRepo` and `TestSanitizeRepo` (`pkg/utils/utils_test.go`).
- Limit: the case-variant test asserts the refusal branch only when the test filesystem is case-insensitive; otherwise it runs the case-sensitive branch. The case-insensitive behaviour is established by the `Mkdir` source above, not by a run on macOS or Windows.

### [GHSA-gv74-crjm-8535](https://github.com/charmbracelet/soft-serve/security/advisories/GHSA-gv74-crjm-8535) (medium): private repository names disclosed over git://, SSH, TUI, bearer tokens

Disposition: **Not affected.** Upstream fixed in 0.12.4.

Every transport decides existence and readability in one call, `Backend.Readable`, which returns a nil repository for both cases (`pkg/backend/user.go:72`): daemon (`pkg/daemon/daemon.go:275`, anonymous), SSH git commands (`pkg/ssh/cmd/git.go:206`), SSH TUI (`pkg/ssh/ui.go:308`), HTTP (`pkg/web/git.go:386`), SSH repo commands via `requireRepo` (`pkg/ssh/cmd/cmd.go:147`). Tests: `pkg/ssh/cmd/oracle_test.go` (`TestAnonymousCannotTellPrivateFromMissing`, `TestRegisteredUserCannotTellPrivateFromMissing`, `TestNoAccessCollaboratorCannotTellPrivateFromMissing`), `pkg/web/oracle_test.go` (`TestBearerTokenDoesNotRevealPrivateRepositories`, `TestReceivePackDoesNotRevealPrivateRepositories`), `pkg/backend/readable_test.go` (`TestReadable`), `testscript/testdata/repo-oracle.txtar`. Not covered by a Go test: the `git://` daemon and the TUI path; they are covered by source reading only.

### [GHSA-9cw6-2v8p-5vhj](https://github.com/charmbracelet/soft-serve/security/advisories/GHSA-9cw6-2v8p-5vhj) (high): repo import over ssh reaches internal hosts with the server's key

Disposition: **Not affected.** Upstream fixed in 0.12.4.

ssh remotes now go through a policy: an operator allowlist binds everyone, and with no allowlist only server admins may choose ssh hosts (`pkg/ssrf/sshpolicy.go:38`, wired through `ValidateGitRemote`, `pkg/ssrf/git.go:101`). The ssh command uses `StrictHostKeyChecking` from config, default `accept-new`, with `BatchMode=yes` and a data-path `known_hosts` (`pkg/backend/gitssh.go`, `pkg/config/config.go:428`). Mirror syncs revalidate each remote with the same policy (`pkg/jobs/mirror.go:34`, `:130`). A failed clone returns the fixed `ErrImportFailed` and the real error is only logged (`pkg/backend/repo.go`, `ImportRepository`), so it cannot hang the session or probe ports. Tests: `TestValidateGitRemotePolicy`, `TestAllowedHostsMatch*` (`pkg/ssrf/sshpolicy_test.go`), `TestImportRepositorySSHRequiresAdmin`, `TestImportRepositoryCloneFailure` (`pkg/backend/repo_test.go`), `TestGitSSHCommandEnv` (`pkg/backend/gitssh_test.go`), `testscript/testdata/repo-import-ssh-allowlist.txtar`.

### [GHSA-fgxm-5hvv-x4vh](https://github.com/charmbracelet/soft-serve/security/advisories/GHSA-fgxm-5hvv-x4vh) (medium): HTTP responses disclose private repository existence and names

Disposition: **Not affected.** Upstream fixed in 0.12.3.

The go-get walk only stops at a repository the caller can read (`pkg/web/goget.go:41`, using `AccessLevelForUser`). There is no go-get exception ahead of the access guard in `withAccess` (`pkg/web/git.go:361`): readability comes from `Readable` and the decision from `authorize`. The ref advertisement is the only route that reads `?service=` (`pkg/web/git.go:78`), which removes the 401/404 split on other routes. New test `TestGoGetDoesNotRevealPrivateRepositories` requests `/secret`, `/secret.git`, `/secret/sub/package` and `/secret.git/sub` with `?go-get=1`, anonymous and as a user with no access, and asserts the answer is byte-identical to a missing repository, then that the owner still gets the page. It passed on the unchanged code, which confirms it was already closed. Other tests: `pkg/web/oracle_test.go`, `pkg/web/authorize_test.go` (`TestAuthorize`), and `testscript/testdata/http.txtar` for `/repo.git?go-get=1`.

### [GHSA-pmgj-8mr6-9rff](https://github.com/charmbracelet/soft-serve/security/advisories/GHSA-pmgj-8mr6-9rff) (medium): `no-access` collaborator granted read access

Disposition: **Not affected.** Upstream fixed in 0.12.3.

The anonymous floor is applied to a collaborator only when the repository is public (`pkg/backend/user.go:143`: `if !r.IsPrivate() && anon > collabAccess`), so on a private repository the collaborator's own level is final. Tests: `TestAccessLevelCollaboratorPrivateRepo`, `TestAccessLevelPrivateRepoNonCollaborator` (`pkg/backend/user_test.go`), `TestNoAccessCollaboratorCannotTellPrivateFromMissing` (`pkg/ssh/cmd/oracle_test.go`).

### [GHSA-f3fj-9625-rv3m](https://github.com/charmbracelet/soft-serve/security/advisories/GHSA-f3fj-9625-rv3m) (medium): race in `SetPrivate` leaves a private repository readable

Disposition: **Not affected.** Upstream fixed in 0.12.3.

There is no repository cache: `Backend.Repository` stats the directory and reads the row from the database on each call (`pkg/backend/repo.go:529`), and nothing under `pkg/backend` keeps a repository cache (only comments in `settings.go` warn against caching settings). `SetPrivate` does the read, the `git-daemon-export-ok` change and the update in one transaction (`repo.go:766`) and fires the webhook only on an actual change. Tests: `TestRepositoryReflectsDatabase` (`pkg/backend/repo_visibility_test.go`), `TestSetPrivateWebhookOnlyOnChange` (`pkg/backend/repo_visibility_webhook_test.go`). The race itself was not stress-tested; the vulnerable component does not exist.

### [GHSA-4h8q-g858-46w2](https://github.com/charmbracelet/soft-serve/security/advisories/GHSA-4h8q-g858-46w2) (high): LFS object IDs used as filesystem paths

Disposition: **Not affected.** Upstream fixed in 0.12.2.

Defense at three layers: SSH `git-lfs-transfer` validates every OID with `IsValid` before use and returns a parse error (`pkg/git/lfs.go:38` and the calls at `:102`, `:131`, `:153`, `:204`); the HTTP batch and verify handlers validate before touching storage (`pkg/web/git_lfs.go:106`, `:194`, `:422`), and the object route pattern is `[0-9a-f]{64}`; `LocalStorage.fixPath` rejects absolute paths and anything resolving outside the root (`pkg/storage/local.go:108`). Tests: `TestLFSTransferRejectsMalformedOid`, `TestInvalidOidIsAParseError` (`pkg/git/lfs_test.go`), `TestLocalStorageConfinesToRoot`, `TestLocalStorageConfinesRootedRelativeNames` (`pkg/storage/local_test.go`), `TestLFSMalformedOIDAndSize` (`pkg/web/git_lfs_isolation_test.go`).

### [GHSA-g3x7-3892-v6r2](https://github.com/charmbracelet/soft-serve/security/advisories/GHSA-g3x7-3892-v6r2) (high): LFS write checks bypassed by a caller-supplied `service` parameter

Disposition: **Not affected.** Upstream fixed in 0.12.1.

`withParams` takes the caller's `?service=` only for the `info/refs` advertisement (`pkg/web/git.go:78`); on LFS routes the service is empty, so it cannot select a Git branch of the authorization switch. LFS-enabled is checked before any branch (`withAccess`, `:378`), and `authorize` returns `forbidLFSWrite` for LFS writes without write access. Tests: `TestLFSUploadRejectsForgedServiceParam`, `TestLFSLockCreateRejectsForgedServiceParam`, `TestLFSDisabledRejectsForgedServiceParam` (`pkg/web/git_lfs_test.go`, run through the real router for all three service names), `TestAuthorize`. The related object-hash and size problems from the same advisory text are fixed by the earlier `TestLFSBasicIntegrity` work (see `security-audit-lfs-webhook.md`).

### [GHSA-3qq2-64gj-j5qf](https://github.com/charmbracelet/soft-serve/security/advisories/GHSA-3qq2-64gj-j5qf) (medium): SSRF guard bypass via IPv6 transition addresses

Disposition: **Not affected.** Upstream fixed in 0.12.0.

`pkg/ssrf/ssrf.go:103-126` lists the blocked prefixes: 6to4 `2002::/16`, NAT64 `64:ff9b::/96` and `64:ff9b:1::/48`, Teredo `2001::/32`, IPv4-compatible `::/96`, and the 6to4 relay `192.88.99.0/24`. Tests in `TestIsPrivateOrInternal` (`pkg/ssrf/ssrf_test.go`, around line 157) cover each example from the advisory, including `64:ff9b::a9fe:a9fe` and the Teredo form, and also a public-IPv4 6to4 address which is still blocked. `TestStdlibCoverageParity` checks the explicit prefix table still covers the ranges the standard library helpers used to cover.

### [GHSA-952g-9vq9-782q](https://github.com/charmbracelet/soft-serve/security/advisories/GHSA-952g-9vq9-782q) (critical): any SSH-key user can become global admin

Disposition: **Not affected.** Upstream fixed in 0.12.0.

`user` and `settings` command trees are gated by `checkIfServerAdmin` on the parent (`pkg/ssh/cmd/user.go:25`, `settings.go:22`), not by a repository-argument check. Collaborator grants are capped at the caller's own level and demotion of higher-ranked collaborators is refused (`pkg/ssh/cmd/collab.go:90` `checkCollabGrant`, `checkCollabDemote`, also used by `remove`). Webhook delivery lookups require the webhook to belong to the authorized repository (`pkg/backend/webhooks.go:283`). Tests: `TestGlobalCommandsIgnoreRepositoryAccess`, `TestGlobalCommandsIgnoreAnonAdminAccess`, `TestUserSubcommandsAreGatedByParent`, `TestSettingsSubcommandsAreGatedByParent`, `TestCollabAddCannotExceedCallerAccess`, `TestCollabRemoveCannotDemoteHigherAccess` (`pkg/ssh/cmd/auth_test.go`), `TestWebhookDeliveriesAreScopedToRepository`, and `testscript/testdata/privilege-escalation-regression.txtar`.

### [GHSA-v57m-p5gf-9mf5](https://github.com/charmbracelet/soft-serve/security/advisories/GHSA-v57m-p5gf-9mf5) (high): SSH server reaches vulnerable `x/crypto/ssh` paths

Disposition: **Not affected** at the dependency level. Upstream fixed in 0.12.0 by bumping `golang.org/x/crypto`.

`go.mod` requires `golang.org/x/crypto v0.57.0`, which is past the v0.52.0 fix level the advisory names. The earlier audit (`security-audit-lfs-webhook.md`, `govulncheck` with a database from 2026-10-27) found no reachable vulnerabilities. That scan was not repeated in this audit. There is no test: the defect was in the library. Residual note: the SSH accept loop lives in the dependency, and this repository adds only session-level panic recovery (`charm.land/wish/v2/recover`, `pkg/ssh/ssh.go:15`), so a future handshake-time panic in the library would still not be contained here. That is a dependency-hygiene item and nothing in the advisory depends on it today.

### [GHSA-xqg9-4q65-m2wf](https://github.com/charmbracelet/soft-serve/security/advisories/GHSA-xqg9-4q65-m2wf) (CVE-2026-91773, medium): LFS lock lookup across repositories

Disposition: **Not affected.** Upstream fixed in 0.12.0.

ID lookups take the repository ID: `GetLFSLockByID(ctx, db, repoID, id)` and `GetLFSLockForUserByID` (`pkg/store/lfs.go:24-25`), used by the list and unlock handlers (`pkg/web/git_lfs_locks.go:188`, `:448`). Tests: `TestLFSLockLookupIsScopedToRepository`, `TestLFSLocksGetDoesNotLeakAcrossRepositories`, `TestLFSLocksDeleteDoesNotLeakAcrossRepositories` (`pkg/web/git_lfs_test.go`), `TestLFSCrossRepoIsolation` (`pkg/web/git_lfs_isolation_test.go`).

### [GHSA-vpj5-m56f-8h3f](https://github.com/charmbracelet/soft-serve/security/advisories/GHSA-vpj5-m56f-8h3f) (medium): SSRF in repo import and mirror sync

Disposition: **Not affected.** Upstream fixed in 0.12.0.

Import validates the remote (`validateImportRemote`, `pkg/backend/repo.go:44`, via `ssrf.ValidateGitRemote`, `pkg/ssrf/git.go:101`). Validation holds for the subprocess because `GitEnv` disables HTTP redirects and pins the validated address (`pkg/ssrf/git.go:66`), applied through `RemoteGitEnv` to import and to every mirror fetch, which also revalidates stored remotes (`pkg/jobs/mirror.go:34`, `:130`). Non-canonical IPv4 literals are refused as `ErrAmbiguousHost`. Stated limit in the source: `git://` cannot be pinned, so DNS rebinding on that one transport is not caught. Tests: `TestValidateGitRemote`, `TestValidateGitRemoteResolvesHostnames`, `TestGitEnvDisablesRedirects`, `TestCurlResolvePin`, `TestGitEnvCountMatchesEntries` (`pkg/ssrf/git_test.go`), `TestValidateImportRemote`, `TestRemoteGitEnv` (`pkg/backend`), `testscript/testdata/repo-import-ssrf.txtar`, `mirror.txtar`.

### [GHSA-xgxp-f695-6vrp](https://github.com/charmbracelet/soft-serve/security/advisories/GHSA-xgxp-f695-6vrp) (CVE-2026-33353, high): repo import clones server-local repositories

Disposition: **Not affected.** Upstream fixed in v0.11.6.

`ValidateGitRemote` accepts only http, https, git and ssh forms and returns `ErrUnsupportedRemoteScheme` for bare paths, `file://` and `ext::` (`pkg/ssrf/git.go:15`, `:101`), and `ImportRepository` calls it before `git clone` (`pkg/backend/repo.go:196`). Clone also passes `--end-of-options`. Tests: `TestValidateGitRemote`, `TestValidateImportRemote`, `testscript/testdata/repo-import-local-path.txtar` (private repo, second user, imports the server-local path and asserts no readable copy results).

### [GHSA-3fvx-xrxq-8jvv](https://github.com/charmbracelet/soft-serve/security/advisories/GHSA-3fvx-xrxq-8jvv) (CVE-2026-30832, high): SSRF via unvalidated LFS endpoint in import

Disposition: **Not affected.** Upstream fixed in 0.11.4.

A user-supplied `--lfs-endpoint` is validated like the remote itself (`pkg/backend/repo.go:233`), mirror revalidates the stored endpoint (`pkg/jobs/mirror.go:163`), and the LFS HTTP client is `ssrf.NewSecureClient()`, which validates at dial time and blocks redirects (`pkg/lfs/http_client.go:26`, `pkg/ssrf/ssrf.go:31`). Tests: `TestNewSecureClientBlocksPrivateIPs`, `TestNewSecureClientBlocksRedirects` (`pkg/ssrf/ssrf_test.go`), `TestValidateImportRemote`, `testscript/testdata/repo-import-ssrf.txtar`. No Go test drives `--lfs-endpoint` end to end; that path is established by source reading.

### [GHSA-pchf-49fh-w34r](https://github.com/charmbracelet/soft-serve/security/advisories/GHSA-pchf-49fh-w34r) (CVE-2026-24058, critical): authentication bypass by offering a victim's key

Disposition: **Not affected.** Upstream fixed in 0.11.3.

The public-key handler stores only a fingerprint, no user (`pkg/ssh/ssh.go:163`). The session middleware compares the authenticated key's fingerprint to the stored one, refuses a mismatch, and resolves the user from the session's own key only after that (`pkg/ssh/middleware.go:28`, `:53`). The keyboard-interactive handler clears the fingerprint (`ssh.go:184`). Tests: `TestAuthenticationBypass` (`pkg/ssh/middleware_test.go`), which simulates the sequence against the middleware, and `testscript/testdata/auth-bypass-regression.txtar`, which performs the two-key attack with a real client. Both ran in the full test run recorded under "Test runs".

### [GHSA-6jm8-x3g6-r33j](https://github.com/charmbracelet/soft-serve/security/advisories/GHSA-6jm8-x3g6-r33j) (CVE-2026-22253, medium): LFS lock force-delete authorization bypass

Disposition: **Not affected.** Upstream fixed in v0.11.2. Regression test added by this audit.

`serviceLfsLocksDelete` (`pkg/web/git_lfs_locks.go:404`) resolves the user first and rejects a missing user, then requires `user.IsAdmin()` for `force` (`:486`), and otherwise requires lock ownership. The lock lookup and deletion are repository-scoped. The only prior test used `force:false`. New `TestLFSLockForceDeleteRequiresAdmin`: a non-admin writer gets 403 for `force:true` and `force:false` and the lock remains; an admin force-delete succeeds; an owner can release their own lock.

### [GHSA-vwq2-jx9q-9h9f](https://github.com/charmbracelet/soft-serve/security/advisories/GHSA-vwq2-jx9q-9h9f) (CVE-2025-64522, high): SSRF in webhooks

Disposition: **Not affected.** Upstream fixed in v0.11.1.

Two layers: creation and update validate the URL (`pkg/backend/webhooks.go:24`, `:129`, via `webhook.ValidateWebhookURL`, `pkg/webhook/validator.go:15`), and delivery uses `ssrf.NewSecureClient()` (`pkg/webhook/webhook.go:45`), which validates the IP at dial time, dials the validated address and refuses redirects. Tests: `TestValidateWebhookURL` (`pkg/webhook/validator_test.go`), `TestSSRFProtection` (`pkg/webhook/ssrf_test.go`), `TestIsPrivateOrInternal`, `TestNewSecureClientBlocksPrivateHostnames`, `testscript/testdata/repo-webhook-ssrf.txtar`. Further webhook transport findings are in `security-audit-lfs-webhook.md`.

### [GHSA-j4jw-m6xr-fv6c](https://github.com/charmbracelet/soft-serve/security/advisories/GHSA-j4jw-m6xr-fv6c) (CVE-2025-22130, medium): path traversal in repository names

Disposition: **Not affected.** Upstream fixed in 0.8.2.

`SanitizeRepo` cleans the name as a rooted path (`path.Clean("/"+name)`), resolving `.` and `..` and stripping `.git` suffixes (`pkg/utils/utils.go:13`), and is applied at every backend entry point (`pkg/backend/repo.go`, `user.go:73`, `collab.go`). `ValidateRepo` additionally restricts the character set and rejects non-canonical names (`utils.go:73`). The daemon also checks `git.EnsureWithin(reposDir, repo)` (`pkg/daemon/daemon.go:265`). Tests: `TestSanitizeRepo`, `TestValidateRepo` (`pkg/utils/utils_test.go`). No testscript targets traversal directly.

### [GHSA-m445-w3xr-vp2f](https://github.com/charmbracelet/soft-serve/security/advisories/GHSA-m445-w3xr-vp2f) (CVE-2024-41956, high): code execution via client environment variables

Disposition: **Not affected.** Upstream fixed in 0.7.5.

Client-supplied environment is not forwarded. The SSH git handler builds the git environment from fixed `SOFT_SERVE_*` values, the config, and a single `GIT_PROTOCOL=` entry copied from the session (`pkg/ssh/cmd/git.go:209-232`); the HTTP handler does the same with the `Git-Protocol` header value (`pkg/web/git.go:457-471`), and the daemon builds its own list (`pkg/daemon/daemon.go:280`). `LD_PRELOAD` and other session variables are never copied. Gap: there is no automated regression test for the environment; the evidence is source reading of those three sites.

### [GHSA-mc97-99j4-vm2v](https://github.com/charmbracelet/soft-serve/security/advisories/GHSA-mc97-99j4-vm2v) (CVE-2023-43809, high): public key auth bypass with keyboard-interactive

Disposition: **Not affected.** Upstream fixed in 0.6.2.

Keyboard-interactive is accepted only when `allow-keyless` is on, and then the stored key fingerprint is cleared and the session carries no key (`pkg/ssh/ssh.go:184`); the middleware refuses a session whose key does not match the stored fingerprint and, with keyless disabled and no key, refuses outright (`pkg/ssh/middleware.go:53`, `:60`). Tests: `TestAuthenticationBypass`; `TestAllowKeylessEnvUnsetByDefault`, `TestParseAllowKeylessEnvTrue`, `TestParseAllowKeylessEnvFalse` (`pkg/config/config_test.go`) and `TestSettingsAllowKeylessWarnsOnConfigOverride`. No test exercises a FIDO or GPG-backed key through keyboard-interactive; that scenario was not reproduced.

## Test runs

Go 1.26.10. The full `go test -count=1 -race ./...` passed, including `testscript` (157.436s), verified by the parent. It ran with `TMPDIR` and `GOTMPDIR` set to `/home/pn/tmp/go-advisories` and a test-only `git` wrapper first on `PATH` that sets `commit.gpgsign=false`, `tag.gpgsign=false` and `init.defaultBranch=master`. This covers all the new tests and the existing testscript scenarios cited above. With the sanitization fixes reverted, `TestSanitizeStripsTerminalControls`, `TestCommitOutputIsSanitized` and `TestWebhookDeliveryOutputIsSanitized` fail.
