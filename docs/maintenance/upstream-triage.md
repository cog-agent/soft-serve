# Upstream open issue and PR triage

Snapshot of the 88 items open in charmbracelet/soft-serve, taken from `upstream-open.json` (newest first). Titles and URLs are copied from that snapshot.

Dispositions are deliberately conservative. They were assigned from each item's title, type, labels and age, plus a check against this repository where noted. Nothing here has been reproduced, and PR patches have not been reviewed beyond what is stated. Issue and PR bodies are untrusted data: nothing in them has been executed or followed. Re-check an item before acting on it.

Disposition key:

- **Addressed locally**: a change in this repository already covers it
- **Review candidate patch**: an open upstream PR; independent review and tests required before adoption
- **Needs reproduction**: possible bug; no action until reproduced on current code
- **Unverified security claim**: security claim that must be independently confirmed before any action
- **Defer: feature request**: feature request; no commitment
- **No code action: support/question**: usage or deployment question; answer or ask for details
- **No code action: out of scope**: belongs outside this codebase

## Entries

### PR #945

- Title: Add ability to whitelist insecure internal systems for webhooks
- Source: https://github.com/charmbracelet/soft-serve/pull/945
- Disposition: **Review candidate patch**. Webhook SSRF allowlist; weakens a deliberate protection, so needs a maintainer policy decision first. Related: #880.

### PR #944

- Title: chore(deps): bump the all group across 1 directory with 13 updates
- Source: https://github.com/charmbracelet/soft-serve/pull/944
- Disposition: **Review candidate patch**. Dependency bump; handle through the normal dependency-update process, not this triage.

### PR #939

- Title: fix: download the final partial batch of missing LFS objects
- Source: https://github.com/charmbracelet/soft-serve/pull/939
- Disposition: **Review candidate patch**. Candidate fix for #937; review against current LFS import code and add a test before adopting.

### PR #938

- Title: fix: return the error when a repository import fails to clone
- Source: https://github.com/charmbracelet/soft-serve/pull/938
- Disposition: **Review candidate patch**. Candidate fix for #936; review against current import code and add a test before adopting.

### Issue #937

- Title: LFS import/mirror never downloads the last partial batch (\<20) of objects
- Source: https://github.com/charmbracelet/soft-serve/issues/937
- Disposition: **Needs reproduction**. Reproduce against current code; candidate fix is #939.

### Issue #936

- Title: repo import hangs forever when the clone fails
- Source: https://github.com/charmbracelet/soft-serve/issues/936
- Disposition: **Needs reproduction**. Reproduce against current code; candidate fix is #938.

### Issue #935

- Title: Bummer missing repo on git.charm.sh
- Source: https://github.com/charmbracelet/soft-serve/issues/935
- Disposition: **No code action: support/question**. Report about the hosted git.charm.sh instance, not this codebase.

### PR #933

- Title: fix(server): surface listener errors instead of hanging on start
- Source: https://github.com/charmbracelet/soft-serve/pull/933
- Disposition: **Addressed locally**. Adopted with independent changes (daemon shutdown ordering, Start waits for failure shutdown), with regression tests. Contributor: asafshitrit.

### Issue #932

- Title: Repos accidentally exposed publicly
- Source: https://github.com/charmbracelet/soft-serve/issues/932
- Disposition: **Unverified security claim**. Possible misconfiguration or access-control bug; needs details to tell which. Check against the anon-access defaults before acting.

### PR #930

- Title: fix(ssh): require repo-admin access for 'repo delete'
- Source: https://github.com/charmbracelet/soft-serve/pull/930
- Disposition: **Addressed locally**. Repository deletion was restricted to owners and server admins in bbd6f7f; confirm equivalence with this patch, then close as superseded.

### Issue #928

- Title: After update everything is gone - how to recover?
- Source: https://github.com/charmbracelet/soft-serve/issues/928
- Disposition: **No code action: support/question**. Data-recovery question after an upgrade; ask for version and data path. Check migration notes only if details show a real migration bug.

### PR #920

- Title: fix(Dockerfile): add tini as pid-1 process to reap zombie processes
- Source: https://github.com/charmbracelet/soft-serve/pull/920
- Disposition: **Review candidate patch**. Adds tini as PID 1 for zombie reaping (see #891, #797). Overlaps #894; pick one approach.

### PR #897

- Title: fix(web): wrap lfs basic upload i/o operations in transactions
- Source: https://github.com/charmbracelet/soft-serve/pull/897
- Disposition: **Review candidate patch**. LFS upload transaction wrapping; related to #770. Needs careful review of transaction scope.

### Issue #896

- Title: Collab user with read-write can delete the whole repository
- Source: https://github.com/charmbracelet/soft-serve/issues/896
- Disposition: **Addressed locally**. Likely covered by bbd6f7f (deletion limited to owners and admins); verify a read-write collaborator is rejected, then close.

### PR #894

- Title: fix: reap zombie child processes in container environments
- Source: https://github.com/charmbracelet/soft-serve/pull/894
- Disposition: **Review candidate patch**. Reaps zombie children in-process (see #891, #797). Overlaps #920; pick one approach.

### Issue #891

- Title: soft serve doesn't reap child git processes, causing zombie accumulation
- Source: https://github.com/charmbracelet/soft-serve/issues/891
- Disposition: **Needs reproduction**. Zombie git processes; reproduce in a container. Candidate fixes #894 and #920.

### PR #886

- Title: add support for docker container to run as any uid/gid
- Source: https://github.com/charmbracelet/soft-serve/pull/886
- Disposition: **Review candidate patch**. Docker arbitrary uid/gid support; review for security and compatibility implications.

### Issue #882

- Title: Why is the Dockerfile not building the package first?
- Source: https://github.com/charmbracelet/soft-serve/issues/882
- Disposition: **No code action: support/question**. Dockerfile design question; answer or document, no code change implied.

### Issue #880

- Title: Regression: cannot create webhooks targeting internal services after SSRF fix
- Source: https://github.com/charmbracelet/soft-serve/issues/880
- Disposition: **Needs reproduction**. Internal webhook targets are blocked by the SSRF protection by design; needs a policy decision (see #945), not a bug fix.

### Issue #865

- Title: security: prioritized remediation plan - plaintext passwords, SQL injection, path traversal
- Source: https://github.com/charmbracelet/soft-serve/issues/865
- Disposition: **Unverified security claim**. Broad remediation plan that appears bulk-generated. Do not act without independently confirming each claimed flaw against current code.

### Issue #863

- Title: fix: comprehensive security and correctness fixes - round 59
- Source: https://github.com/charmbracelet/soft-serve/issues/863
- Disposition: **Unverified security claim**. Titled as a batch of unspecified fixes ('round 59'); not actionable as written. Verify any concrete claim independently.

### Issue #862

- Title: security(web): path traversal vulnerability and missing input validation
- Source: https://github.com/charmbracelet/soft-serve/issues/862
- Disposition: **Unverified security claim**. Path traversal claim in the web layer; verify independently against current code before acting.

### Issue #860

- Title: fix(web): path traversal, JWT token expiration handling, input validation - round 58
- Source: https://github.com/charmbracelet/soft-serve/issues/860
- Disposition: **Unverified security claim**. Path traversal and JWT claims ('round 58'); verify each independently against current code.

### Issue #858

- Title: fix: review round 57 - findings already fixed in previous rounds
- Source: https://github.com/charmbracelet/soft-serve/issues/858
- Disposition: **Unverified security claim**. Self-described as findings already fixed; likely close after a spot check.

### Issue #856

- Title: fix(web): JWT claims missing validation - round 56
- Source: https://github.com/charmbracelet/soft-serve/issues/856
- Disposition: **Unverified security claim**. JWT claim validation claim ('round 56'); verify independently against current code.

### Issue #805

- Title: fix(server): bind errors on privileged ports silently swallowed — server appears started but SSH/HTTP never bind
- Source: https://github.com/charmbracelet/soft-serve/issues/805
- Disposition: **Addressed locally**. Fixed by the listener startup error change (upstream #933).

### Issue #800

- Title: access tokens don't provide authentication
- Source: https://github.com/charmbracelet/soft-serve/issues/800
- Disposition: **Needs reproduction**. Authentication with access tokens; reproduce and identify the exact flow.

### PR #799

- Title: feat: implements \`git gc\` cron job & add \`cronjob\` ssh command
- Source: https://github.com/charmbracelet/soft-serve/pull/799
- Disposition: **Review candidate patch**. Large feature (git gc cron job plus a cronjob SSH command); needs design review and is not a narrow change.

### Issue #797

- Title: Orphaned Processes
- Source: https://github.com/charmbracelet/soft-serve/issues/797
- Disposition: **Needs reproduction**. Orphaned processes; likely the same root cause as #891.

### Issue #781

- Title: testscript fails on i686
- Source: https://github.com/charmbracelet/soft-serve/issues/781
- Disposition: **Needs reproduction**. Test failure on i686; platform-specific, low priority.

### Issue #779

- Title: SSH identity changes on restart when \`ssh.key\_path\` is set
- Source: https://github.com/charmbracelet/soft-serve/issues/779
- Disposition: **Needs reproduction**. SSH host identity changing on restart with ssh.key_path set; check key generation and path handling.

### Issue #777

- Title: Mirroring a large repo pegs cpu at 100% for a long time
- Source: https://github.com/charmbracelet/soft-serve/issues/777
- Disposition: **Needs reproduction**. Mirror CPU usage; needs a profile and repository details.

### Issue #773

- Title: \[Feature Request\] Provide a method to disable a cron job
- Source: https://github.com/charmbracelet/soft-serve/issues/773
- Disposition: **Defer: feature request**. Disable a cron job.

### Issue #770

- Title: LFS upload reports errors related to locking, either of the database (when using HTTP) or of the files itself
- Source: https://github.com/charmbracelet/soft-serve/issues/770
- Disposition: **Needs reproduction**. LFS locking errors; see #897.

### Issue #769

- Title: Copy clone command when viewing repository
- Source: https://github.com/charmbracelet/soft-serve/issues/769
- Disposition: **Defer: feature request**. TUI convenience feature.

### PR #761

- Title: feat: add commit and ref counters to repo tab view
- Source: https://github.com/charmbracelet/soft-serve/pull/761
- Disposition: **Review candidate patch**. TUI feature; review for scope and performance on large repositories.

### PR #757

- Title: fix(ssh): support split key in user create
- Source: https://github.com/charmbracelet/soft-serve/pull/757
- Disposition: **Review candidate patch**. Candidate fix for the user create key parsing bug (#750); review and test.

### Issue #756

- Title: Read-only, public-only web interface
- Source: https://github.com/charmbracelet/soft-serve/issues/756
- Disposition: **Defer: feature request**. Read-only public web interface; design discussion.

### Issue #753

- Title: Set gitweb.owner in the config on repo create
- Source: https://github.com/charmbracelet/soft-serve/issues/753
- Disposition: **Defer: feature request**. Set gitweb.owner on repo create.

### Issue #750

- Title: Bug: user create \<user\> -k
- Source: https://github.com/charmbracelet/soft-serve/issues/750
- Disposition: **Needs reproduction**. user create with -k; candidate fix is #757.

### Issue #740

- Title: \`Error: user not found\` when calling soft-serve from remote
- Source: https://github.com/charmbracelet/soft-serve/issues/740
- Disposition: **No code action: support/question**. Remote 'user not found' error; probably a client or key setup problem. Needs details.

### Issue #728

- Title: Keep soft serve and GitHub in sync ?
- Source: https://github.com/charmbracelet/soft-serve/issues/728
- Disposition: **No code action: support/question**. Question about syncing with GitHub; mirroring already exists.

### Issue #720

- Title: Error: unauthorized
- Source: https://github.com/charmbracelet/soft-serve/issues/720
- Disposition: **No code action: support/question**. 'Error: unauthorized'; setup question. Needs details.

### Issue #718

- Title: One line read-only serve of all git repositories in a folder
- Source: https://github.com/charmbracelet/soft-serve/issues/718
- Disposition: **Defer: feature request**. One-line read-only serve of a folder of repositories.

### Issue #717

- Title: Support GSSAPI/Kerberos authentication for SSO
- Source: https://github.com/charmbracelet/soft-serve/issues/717
- Disposition: **Defer: feature request**. GSSAPI/Kerberos authentication.

### Issue #709

- Title: Arch Linux package is outdated and incorrect conf example
- Source: https://github.com/charmbracelet/soft-serve/issues/709
- Disposition: **No code action: out of scope**. Downstream Arch Linux package; not this repository.

### Issue #690

- Title: Anonymous access to go-get for private repos
- Source: https://github.com/charmbracelet/soft-serve/issues/690
- Disposition: **Defer: feature request**. Anonymous go-get for private repositories; has security implications, needs a design decision.

### Issue #670

- Title: Support direct Tailscale listening via tsnet
- Source: https://github.com/charmbracelet/soft-serve/issues/670
- Disposition: **Defer: feature request**. Tailscale tsnet listening; likely out of scope.

### Issue #664

- Title: Allow mirror soft-serve projects/repos to gitlab/github
- Source: https://github.com/charmbracelet/soft-serve/issues/664
- Disposition: **Defer: feature request**. Mirror to GitLab/GitHub (push mirroring).

### Issue #653

- Title: Syscall filter on systemd prevents repo creation
- Source: https://github.com/charmbracelet/soft-serve/issues/653
- Disposition: **No code action: support/question**. systemd syscall filter in a deployment unit; likely a documentation or packaging matter.

### Issue #633

- Title: Mirrored Repos Broken
- Source: https://github.com/charmbracelet/soft-serve/issues/633
- Disposition: **Needs reproduction**. Mirrored repos broken; no detail, needs reproduction.

### Issue #625

- Title: Run server side hooks in own namespace
- Source: https://github.com/charmbracelet/soft-serve/issues/625
- Disposition: **Defer: feature request**. Per-hook namespaces; large design change.

### Issue #621

- Title: add merge requests
- Source: https://github.com/charmbracelet/soft-serve/issues/621
- Disposition: **Defer: feature request**. Merge requests; out of project scope.

### Issue #616

- Title: should return a error but return a nil value error
- Source: https://github.com/charmbracelet/soft-serve/issues/616
- Disposition: **Needs reproduction**. Returns a nil error where one is expected; locate the code path first.

### Issue #613

- Title: Per-repository storage of LFS objects (optional)
- Source: https://github.com/charmbracelet/soft-serve/issues/613
- Disposition: **Defer: feature request**. Per-repository LFS storage.

### Issue #596

- Title: Make the arguments more UNIX like.
- Source: https://github.com/charmbracelet/soft-serve/issues/596
- Disposition: **Defer: feature request**. CLI argument style; breaking change if adopted.

### Issue #589

- Title: Allow admin to create tokens for other users
- Source: https://github.com/charmbracelet/soft-serve/issues/589
- Disposition: **Defer: feature request**. Admin-created tokens for other users; security-sensitive, needs design.

### Issue #560

- Title: Spurious error message when Flux's Source controller checks to see if there are any updates
- Source: https://github.com/charmbracelet/soft-serve/issues/560
- Disposition: **Needs reproduction**. Spurious error logged on Flux source checks; reproduce and identify the request.

### Issue #556

- Title: Copy .patch when viewing a commit
- Source: https://github.com/charmbracelet/soft-serve/issues/556
- Disposition: **Defer: feature request**. TUI feature.

### Issue #542

- Title: Support for config repo or a config file like before v0.5.0
- Source: https://github.com/charmbracelet/soft-serve/issues/542
- Disposition: **Defer: feature request**. Config repo or config file as before v0.5.0.

### Issue #538

- Title: Allowing accessing server via an ssh proxy user
- Source: https://github.com/charmbracelet/soft-serve/issues/538
- Disposition: **Defer: feature request**. SSH proxy user access.

### Issue #536

- Title: Integrate with Gitea Server
- Source: https://github.com/charmbracelet/soft-serve/issues/536
- Disposition: **Defer: feature request**. Gitea integration; out of scope.

### Issue #535

- Title: Tailscale integration
- Source: https://github.com/charmbracelet/soft-serve/issues/535
- Disposition: **Defer: feature request**. Tailscale integration; see #670.

### Issue #533

- Title: Cannot create a repo on docker image(latest): Error: not a git repository.
- Source: https://github.com/charmbracelet/soft-serve/issues/533
- Disposition: **Needs reproduction**. Repo creation failing in the docker image; old report, retest on current image.

### Issue #530

- Title: Option to have bare repositories without .git in name
- Source: https://github.com/charmbracelet/soft-serve/issues/530
- Disposition: **Defer: feature request**. Bare repository names without .git.

### Issue #522

- Title: Readme view displays Loading spinner in empty repo
- Source: https://github.com/charmbracelet/soft-serve/issues/522
- Disposition: **Needs reproduction**. Readme spinner in an empty repository; small TUI bug.

### Issue #505

- Title: Unable to start soft serve
- Source: https://github.com/charmbracelet/soft-serve/issues/505
- Disposition: **No code action: support/question**. Startup failure with no detail; ask for logs. May relate to #805 if it was a bind error.

### Issue #504

- Title: Set repo metadata (e.g. project-name, description, ...) from repo
- Source: https://github.com/charmbracelet/soft-serve/issues/504
- Disposition: **Defer: feature request**. Set repo metadata from the repository itself.

### Issue #497

- Title: Unable to init my "admin" account
- Source: https://github.com/charmbracelet/soft-serve/issues/497
- Disposition: **No code action: support/question**. Admin account initialisation; long thread. Verify against the current init-key docs.

### Issue #485

- Title: align soft-serve ssh server \[config\] with ssh-audits recommendations
- Source: https://github.com/charmbracelet/soft-serve/issues/485
- Disposition: **Defer: feature request**. Align SSH config with ssh-audit recommendations; check current algorithm defaults.

### Issue #479

- Title: Support S3 storage
- Source: https://github.com/charmbracelet/soft-serve/issues/479
- Disposition: **Defer: feature request**. S3 storage; large architectural change.

### Issue #462

- Title: Cannot copy from soft-serve's out (via mouse in the terminal)
- Source: https://github.com/charmbracelet/soft-serve/issues/462
- Disposition: **Needs reproduction**. Cannot copy from the TUI with the mouse; terminal behaviour.

### Issue #456

- Title: Access raw file content from HTTP
- Source: https://github.com/charmbracelet/soft-serve/issues/456
- Disposition: **Defer: feature request**. Raw file content over HTTP; check whether the existing web routes already cover it.

### Issue #451

- Title: ssh login issues
- Source: https://github.com/charmbracelet/soft-serve/issues/451
- Disposition: **No code action: support/question**. SSH login problem; needs details.

### Issue #450

- Title: Dancing cursor
- Source: https://github.com/charmbracelet/soft-serve/issues/450
- Disposition: **Needs reproduction**. Cursor glitch in the TUI.

### Issue #421

- Title: Add support for sha256 repositories
- Source: https://github.com/charmbracelet/soft-serve/issues/421
- Disposition: **Defer: feature request**. SHA-256 repositories; depends on git and go-git support.

### Issue #393

- Title: Support ssh-certificates
- Source: https://github.com/charmbracelet/soft-serve/issues/393
- Disposition: **Defer: feature request**. SSH certificates.

### Issue #366

- Title: Package Registry Support
- Source: https://github.com/charmbracelet/soft-serve/issues/366
- Disposition: **Defer: feature request**. Package registry; out of scope.

### Issue #303

- Title: Unable to view private/hidden repositories without using key in config.yaml
- Source: https://github.com/charmbracelet/soft-serve/issues/303
- Disposition: **No code action: support/question**. Viewing private repositories without a configured key; likely expected behaviour, needs confirmation.

### Issue #266

- Title: Begginer problem: zsh:1: command not found: help
- Source: https://github.com/charmbracelet/soft-serve/issues/266
- Disposition: **No code action: support/question**. Beginner usage question.

### Issue #257

- Title: feat: support group and teams
- Source: https://github.com/charmbracelet/soft-serve/issues/257
- Disposition: **Defer: feature request**. Groups and teams.

### Issue #208

- Title: Feature Request: Branch Diffs
- Source: https://github.com/charmbracelet/soft-serve/issues/208
- Disposition: **Defer: feature request**. Branch diffs in the TUI.

### Issue #194

- Title: Feature Request: Search/Filter in Files tab
- Source: https://github.com/charmbracelet/soft-serve/issues/194
- Disposition: **Defer: feature request**. File search in the TUI.

### Issue #137

- Title: Feature Request: Search
- Source: https://github.com/charmbracelet/soft-serve/issues/137
- Disposition: **Defer: feature request**. Search.

### Issue #70

- Title: Kubernetes Helm chart
- Source: https://github.com/charmbracelet/soft-serve/issues/70
- Disposition: **Defer: feature request**. Kubernetes Helm chart; could live outside this repository.

### Issue #48

- Title: Unable to login with my rsa key
- Source: https://github.com/charmbracelet/soft-serve/issues/48
- Disposition: **Needs reproduction**. RSA key login failure from 2021; retest on current code, likely stale.

### Issue #43

- Title: Access Soft Serve On Remote Server
- Source: https://github.com/charmbracelet/soft-serve/issues/43
- Disposition: **No code action: support/question**. Remote access question; likely stale.

### Issue #22

- Title: Auto-update docker.io repo reademe on release
- Source: https://github.com/charmbracelet/soft-serve/issues/22
- Disposition: **No code action: out of scope**. Docker Hub readme automation; release tooling, not this codebase.
