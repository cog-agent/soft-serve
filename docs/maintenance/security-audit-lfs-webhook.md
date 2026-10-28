# LFS basic transfer and webhook audit

Scope: `pkg/web/git_lfs.go` basic-transfer handlers (download, upload, verify), webhook signing and transport in `pkg/webhook`, and Go dependency vulnerability status. Base commit `d24f034`, branch `cog/security-hardening`. All scenarios use local fixtures only (in-process sqlite, `httptest`, temp dirs). No live host or external target was contacted, apart from fetching the public Go vulnerability database for `govulncheck`.

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

## Limitations

- Findings come from tests and code reading of these handlers only. SSH, git transport, authorization and the other packages were not re-audited.
- govulncheck reflects the build on this machine (linux/amd64); other build tags or platforms were not scanned.
- Tests ran against sqlite and local storage, not a real git-lfs client.
- The existing `locks` handlers were not re-reviewed beyond the existing tests.

## Verification

`go vet` and `go test -count=1` on `./pkg/web ./pkg/lfs ./pkg/webhook ./pkg/ssrf ./pkg/storage` pass.
