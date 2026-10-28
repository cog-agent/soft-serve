# LFS basic transfer and webhook audit

Scope (first pass): `pkg/web/git_lfs.go` basic-transfer handlers (download, upload, verify), webhook signing and transport in `pkg/webhook`, and Go dependency vulnerability status. Base commit `d24f034`, branch `cog/security-hardening`. All scenarios use local fixtures only (in-process sqlite, `httptest`, temp dirs). No live host or external target was contacted, apart from fetching the public Go vulnerability database for `govulncheck`.

The findings were first tested on unmodified code and failed as described below. The fixes in this change then made them pass. The earlier run's LFS claims were not available to this audit, so every item here is independently derived.

## LFS basic transfer

Test: `TestLFSBasicIntegrity` in `pkg/web/git_lfs_integrity_test.go`. Callers hold a write token for the repository, so these are integrity and robustness defects inside the write-trust boundary, not privilege escalation.

| # | Scenario | Result before | Fix |
|---|----------|---------------|-----|
| 1 | PUT body whose SHA-256 differs from the OID in the URL | 200; file stored and DB row created under the wrong OID. The pre-existing `TestLFSWriteStillWorksForAuthorizedUser` uploaded `"payload"` under `bbbb…` and encoded the flaw. | Upload streams to `tmp/<uuid>`, hashes while writing, and renames to the content-addressed path only if the digest matches. Otherwise 422 and the temp file is removed. That test now uses the real digest. |
| 2 | PUT with no Content-Length (chunked) | 400 after the body was already written: file on disk, no DB row | The size now comes from bytes actually stored, and Content-Length is no longer parsed. The file is never left behind on failure. |
| 3 | DB size comes from the request header | A lying header gave a wrong row size, but only under `httptest`. Go's real server bounds the body by Content-Length, so this is not shown to be reachable over a real socket. | The stored byte count is used. |
| 4 | Verify when the on-disk size differs from the claimed size (DB row matches) | Fell off the end of the handler and sent an implicit empty 200 | Explicit 422. |
| 5 | GET of an object on disk without a DB row | `Content-Length: 0` while a body is written, so the response is malformed | Content-Length comes from the opened file's stat. The unused DB lookup is dropped. |

Checked and not a flaw: the object route constrains `{oid}` to `[0-9a-f]{64}`, `Pointer.IsValid` guards the batch and verify handlers, and `LocalStorage.fixPath` rejects traversal. No path traversal was found.

Not changed, and needing a policy decision: there is no upload size limit, and concurrent uploads of the same OID are not serialized (last rename wins, and both carry identical verified content after fix #1).

## Webhooks

Tests: `pkg/webhook/delivery_test.go`, using a test-only client swap in `export_test.go`.

- **Signing**: `X-SoftServe-Signature` is `sha256=` plus HMAC-SHA256 over the exact request body. `TestSignatureMatchesBody` recomputes it for the JSON and form content types and it matches. With no secret, no header is sent. The secret never appears in the body or headers. Limitation: there is no timestamp or nonce in the signed data, so a captured delivery can be replayed. `X-SoftServe-Delivery` is a UUID the receiver can dedupe on, but it is not signed. Changing this alters the wire format and was not done.
- **Transport**: the SSRF guard (`pkg/ssrf`) validates the IP at dial time, dials the validated IP, disables redirects, and blocks the IPv6 transition ranges. Existing tests pass. DNS resolution picks `ips[0]` only, so a hostname with a public first record and a private second record is permitted; this is harmless because only the validated IP is dialed. Plain `http://` to public hosts is allowed, so the body and signature travel in cleartext. That is a policy question and was left unchanged.
- **Finding, fixed**: the receiver's response body was read with `io.ReadAll` and stored in the delivery row. An 8 MiB response was stored whole (`TestStoredResponseBodyIsBounded` failed). The read is now capped at 1 MiB. The target is configured by repo admins, so the impact is resource exhaustion by an admin or a hostile receiver, not remote exploitation.
- **Not changed**: no SSRF allowlist or other policy change was made (see upstream PR #945 in `upstream-triage.md`).

## Dependencies

`govulncheck` v1.9.0, vulnerability database updated 2026-10-27, run with `./...` (symbol level), with Go 1.26.10.

- Reachable vulnerabilities: **0**.
- Imported but not called: GO-2026-6866 and GO-2026-6860 in `golang.org/x/net` v0.60.0 (HTML parsing), fixed in v0.62.0.
- Required module with no fix available: GO-2026-5932 (`golang.org/x/crypto/openpgp`, unmaintained). The package is not imported and there is no fixed version.
- Suggestion: bumping `golang.org/x/net` to v0.62.0 is hygiene only, as nothing reachable depends on it. `go.mod` and `go.sum` were not modified.

## Follow-up pass

Same local-fixture rules. Each defect below was reproduced with a failing test before its fix.

| # | Area | Finding | Fix | Test |
|---|------|---------|-----|------|
| 1 | LFS concurrency | Concurrent uploads of one OID: the loser's `CreateLFSObject` hit a duplicate key and returned 500 although the verified object was stored. Introduced by the previous commit's flow, where the existence check precedes the insert. | A duplicate row counts as success, since the content is hash-verified. | `TestLFSConcurrentUploadSameOID` (16 writers, `-race`, run 3 times) |
| 2 | Lifecycle | `DeleteRepository` removed only objects with database rows. Orphan objects and `tmp/` files stayed in `lfs/<repoID>/`. Repo IDs are AUTOINCREMENT, so reuse by a new repo was not shown, but the files stayed on disk. | The whole `lfs/<repoID>` directory is removed. | `TestDeleteRepositoryRemovesLFSStorage` |
| 3 | Lifecycle | `webhook.NewRepositoryEvent` dereferenced a nil user, so `DeleteRepository` panicked with no user in the context. Callers reach this with no user in the push-create cleanup path (`pkg/ssh/cmd/git.go`) when anonymous write is enabled. Reachability through a real SSH session was not exercised. | Nil user gives an empty sender. | Same test (the context has no user) |
| 4 | Path boundary | `git.EnsureWithin` used a bare `HasPrefix`, so a sibling such as `repos-evil` passed. Not reachable now, because `SanitizeRepo` strips `..`. This is defence in depth. | Compare whole path elements. | `TestEnsureWithin` |
| 5 | Test fixture | The shared LFS test DSN lacked the `busy_timeout` and `foreign_keys` pragmas that the production default sets. Without them the concurrency test showed SQLITE_BUSY, which production would not. | The fixture now uses the production pragmas. | n/a |

Checked, no defect found (tests kept as regressions):
- **Cross-repo and private isolation**, `TestLFSCrossRepoIsolation`: a private repo's object is not readable by another user or anonymously, not through another repo's GET, batch or verify, and the same OID uploaded to two repos is stored independently.
- **Malformed OID and size**, `TestLFSMalformedOIDAndSize`: wrong case, wrong length, traversal strings, control characters, and negative, overflowing, string and null sizes on batch, verify, GET and PUT. No 5xx, no actions issued, no 200 for invalid input.
- **Credential leakage**, `TestCloneErrorDoesNotLeakPassword`: a loopback 401 server with a password in the clone URL. The git error and `redactRemote` do not contain the password.
- **Existing coverage re-read, not re-derived**: `pkg/ssrf/git_test.go` already covers option-looking remotes (`-oProxyCommand=...`), `ext::`, `file://`, non-canonical IPs, and userinfo parsing. The SSH git command path sanitizes the repo name before use and does not pass caller arguments to the git binary (source reading only).
- The `lint` finding in `pkg/webhook/delivery_test.go` (gofumpt, from the previous commit) was fixed.

Dependency check repeated: govulncheck v1.9.0, database 2026-10-27, Go 1.26.10. Reachable (symbol-level) vulnerabilities: 0. Module-level only: GO-2026-6860 and GO-2026-6866 (`golang.org/x/net` v0.60.0, fixed in v0.62.0) and GO-2026-5932 (`golang.org/x/crypto`, no fix). `go.mod` and `go.sum` are unchanged.

## Limitations and untested categories

- **Not tested**: SSH authentication and key-handling paths beyond reading `gitRunE`. A live SSH session against git commands with hostile arguments. Symlink escape inside repository or LFS directories (it needs filesystem access the HTTP/SSH user does not have, and `LocalStorage.fixPath` is lexical only; a symlink planted by an operator would be followed). Real git-lfs client interop. HTTP basic and token authentication beyond the existing tests. Import over a live remote and the mirror job's sync. Hostname-based SSRF of the import path, which relies on the existing resolver unit tests. DNS rebinding of `git://` remotes (documented in code). Non-sqlite (Postgres) backends. Windows and macOS runtime behaviour: only cross-compiled.
- **Webhook sender**: `push.go`, `branch_tag.go` and `collaborator.go` also dereference `user` and were not changed. Whether they can receive a nil user was not checked.
- **Open policy items, unchanged**: no LFS upload size limit, no stale `tmp/` sweeper after a crash, no signed timestamp on webhooks.
- Findings come from tests and code reading of the listed areas only. govulncheck reflects linux/amd64 only. The remainder of the original limitations still apply.

## Verification

- Targeted: `go test -count=1 -race -run TestLFS ./pkg/web` (3 runs); `./pkg/git`; `./pkg/backend`; `./pkg/webhook`.
- Broader: `go vet ./...`; `go test -count=1 -race ./...`: no failures; `golangci-lint run ./...`: 0 issues.
- Cross-build with `CGO_ENABLED=0`: linux/amd64, linux/arm64, darwin/arm64. With cgo, windows/amd64 builds; the arm64 and darwin cgo builds fail on the host toolchain, not on the code.
