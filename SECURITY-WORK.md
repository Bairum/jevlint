# Security work record

Local maintainer handoff and potential PR-description material, **not a security
policy**. This records completed work and observed checks, not a blanket security
certification.

## Publication status

On 2026-10-05 the owner approved publication: `fix/security-boundaries` was
pushed to `Bairum/jevlint` and merged into the fork's `master` via
[PR #9](https://github.com/Bairum/jevlint/pull/9), closing issues #1–#4 and #6.
Issue #5 stays open until a real GitHub release run verifies OIDC artifact
provenance. The original author has not been contacted and no advisory exists;
the snapshot sections below describe the state before publication.

## Original snapshot and disclosure status

As of **2026-10-04T03:04:35Z** (before adding this record):

- Public fork / `origin`: <https://github.com/Bairum/jevlint.git>.
- Original repository / `upstream`: <https://github.com/codegirl-007/jevlint.git>.
- Current local branch: `fix/security-boundaries`, with no upstream tracking.
- Local `master` tracks `origin/master`.
- Both local branch HEADs / baseline:
  `73a7a94ab08a4e314709d68c2863687a4646f29b`
  (`Merge pull request #42 from codegirl-007/feat/openrouter`).
- Fixes, README updates, and new regressions are **uncommitted working-tree
  changes**. There is no security-fix commit hash yet; this document is also new.
- No security branch push, issue, PR, or advisory has been published in this work.

**Coordinate privately with the original author before pushing.** Both the fork
and a fork PR are public; pushing the branch or this document discloses the
technical details. No contact address, approval, disclosure date, or advisory
identifier has been established here. Do not treat a local fix as permission to
publish it.

## Hardening follow-up status

On 2026-10-04, the user authorized six public hardening issues in
`Bairum/jevlint`; fork issue tracking was enabled to create them. Those issues
cover only the follow-ups below, not the three unpublished boundary findings.
At completion of hardening verification, all code and documentation changes were
local and uncommitted on `fix/security-boundaries`; `master` was unchanged.
No branch, PR, or advisory has been published. Keep the issues open until the
corresponding changes are published and reviewed.
Each issue now has a progress comment recording its local implementation and
observed verification; the workflow issue explicitly retains the unexecuted
GitHub release/OIDC checks. No issue was closed.

## Completed fixes

### 1. Pack pins resolve to the exact commit

**Prerequisite / impact:** an attacker controls a pack repository and makes its
default branch's name exactly match the configured full commit pin, while that
branch points to a different commit. On a cache miss, the old checkout accepted
the branch and cached substituted rules under the trusted pin. This is a
rule/evaluation-integrity failure, not remote code execution.

- Configured pins must be full 40- or 64-character hexadecimal commit IDs.
- Resolution peels to a commit, compares it with the requested pin, checks out
  that commit detached, and verifies `HEAD` before copying pack content.
- Explicit plugin installation still supports selecting branches and tags;
  installation records the resolved commit rather than a moving ref.
- Verified fetches use `jevlint/packs/v2`. Unverified legacy caches are ignored,
  not deleted. A verified warm cache remains usable offline.

Sources: [pin validation](internal/config/config.go),
[resolution / checkout](internal/packs/resolve.go),
[pack loading / cache namespace](internal/packs/packs.go),
[path validation](internal/packs/paths.go).
Regressions: [config pins](internal/config/pin_regression_test.go),
[pin substitution](internal/packs/pin_regression_test.go),
[install refs](internal/packs/install_ref_regression_test.go),
[cache cutover](internal/packs/cache_cutover_regression_test.go).

### 2. Source reads stay inside the project root

**Prerequisite / impact:** attacker-controlled source paths or symlinks are
checked, or an outside path is explicitly selected. Previously, supported source
outside the project could be read and included in a provider request. The target
must be readable, parse as a supported language, and yield code units selected
for evaluation; provider submission requires working credentials. The observed
impact is unintended source disclosure to the provider, not arbitrary-byte
exfiltration to the repository author.

- One `os.Root` per runner evaluation confines discovery and disk reads,
  including callee context. Existing source-overlay behavior is retained.
- Explicit `../` escapes and outside absolute paths are rejected, including
  explicit selections used with `--changed`.
- `--changed` skips escaping symlinks. Relative in-root links remain supported;
  absolute-target symlinks are rejected even if their target is inside the root.
- Review uncovered a tracked-directory replacement edge case: worktree-deleted
  porcelain records are now skipped after consuming rename/copy records, so an
  unsafe replacement symlink does not abort checks of legitimate files. A
  staged-deleted file recreated in the worktree remains eligible.

Sources: [runner reads / discovery](internal/runner/runner.go),
[changed-path selection](internal/changed/changed.go).
Regressions: [runner boundaries](internal/runner/source_boundary_test.go),
[changed paths / deletion handling](internal/changed/source_boundary_test.go).

### 3. Eval fixtures stay inside the eval document directory

**Prerequisite / impact:** an attacker influences an eval document or fixture
symlink and the user runs evals (`--packs` is required for pack evals). Previously,
traversal or symlink escapes could select readable, valid supported source
outside the fixture directory for provider submission. The source/parsing and
credential prerequisites above also apply.

- Fixture paths must be relative and confined to their eval document directory.
  Rooted validation requires a regular file in a supported source language.
- Each case passes its own fixture root to the runner. Project evals and pack
  evals stored in an external user cache both continue to work.
- Eval-specific clearing of rule include/exclude filters is retained.

Sources: [fixture loading / validation](internal/evals/evals.go),
[case execution](internal/evals/run.go).
Regressions: [fixture boundaries](internal/evals/fixture_boundary_test.go),
[CLI project / cached-pack evals](internal/cli/eval_boundary_test.go).

### Compatibility and scope

[README.md](README.md) already documents the source boundary, fixture relativity,
and cache refetch behavior. These changes deliberately reject formerly accepted
outside paths, escaping fixtures, absolute-target source symlinks, and non-full
pins. First access after cache cutover needs the pack source, even if a legacy
cache exists; legacy files are preserved. Branch/tag selection during install
and verified warm-cache offline use remain supported. This is not a general
sandbox for configuration, provider traffic, or every file the program opens.

## Initial boundary-fix verification

The attack regressions were exercised against a pristine detached baseline at
the commit above and demonstrated failures before the fixes; the permanent
suite passed after. Positive compatibility cases were also retained. The later
directory-replacement regression first failed on the initial fixed snapshot
before its correction. The temporary baseline worktree and Python CLI smoke
harness were removed after verification; they are not retained repository tools.

Final observed checks passed with **Go go1.26.8 linux/amd64**, `CGO_ENABLED=1`:

- `go test -race -p 2 -count=1 ./...` — all nine internal packages passed.
- `go vet -p 2 ./...` — passed.
- `go build -p 2 -o jevlint-fixed ./cmd/jevlint`
  — passed.

An independent reviewer identified the directory-replacement regression above;
it was fixed and verified. No other findings were reported in the scoped changes.
That review is not a certification of the entire repository.

### CLI attack and positive-path smoke checks

These exercised real CLI binaries, synthetic source and local Git repositories,
a **loopback HTTP provider**, and dummy keys only. They were not live production
provider integration tests. The baseline leaked outside source or substituted
pack content in the nine attack scenarios; fixed runs exposed zero unintended
data. Exit `2` denotes rejection/runtime error; exit `0` below denotes safe skip
or successful checking, not acceptance of the malicious source.

| Attack scenario | Fixed exit | Fixed provider requests |
| --- | --- | --- |
| Changed source symlink escaping root | 0 (skip) | 0 |
| Direct source symlink escaping root | 2 | 0 |
| Absolute outside source path | 2 | 0 |
| `--changed` with explicit `../` escape | 2 | 0 |
| `--changed` with explicit outside absolute path | 2 | 0 |
| Tracked directory replaced by escaping symlink | 0 | 1 legitimate request only |
| Eval fixture `../` escape | 2 | 0 |
| Eval fixture symlink escape | 2 | 0 |
| Hash-named default branch substituting a pack pin | 2 | 0 |

Positive paths all exited `0`: nested-source check (one request), nested-fixture
eval (one request), and combined project plus externally cached pack evals (two
requests). These checks demonstrate the exercised boundaries, not production
provider availability or behavior.

### Reproducing local Go checks

From the repository root, use the Go version required by `go.mod`, enable CGO,
and provide a working C compiler for Tree-sitter. The observed run used an
existing Zig C compiler and a
scratch-only Go toolchain; no Docker, Podman, or system packages were installed,
and the global `PATH` was not changed. The official go.dev archive checksum was
verified for that toolchain.

Configure a compatible Go/C environment, then run:

```sh
CGO_ENABLED=1 go test -race -p 2 -count=1 ./...
CGO_ENABLED=1 go vet -p 2 ./...
CGO_ENABLED=1 go build -p 2 ./...
```

The scratch path is machine-local, not a repository dependency. The permanent
regressions are included in `go test`; the removed CLI harness is not recreated
by these commands. No additional checks were run merely to write this record.

## Hardening follow-ups tracked in GitHub

These were privacy limitations and conditional hardening risks, not claims of
additional proven remote exploits. They are now tracked in the public fork:

| Issue | Local implementation |
| --- | --- |
| [#1: Git ignore rules](https://github.com/Bairum/jevlint/issues/1) | Native Git ignore filtering before source reads/callee indexing; explicit files remain opt-in; known Git failures do not silently disable filtering. |
| [#2: Debug privacy](https://github.com/Bairum/jevlint/issues/2) | Metadata-only initialization/request/response logs; endpoint, model, source, response-body, and URL-secret logging removed. |
| [#3: Secure transport](https://github.com/Bairum/jevlint/issues/3) | HTTPS except exact localhost/literal loopback HTTP; same-origin redirects, including after caller policy callbacks; supplied client/TLS policy preserved. |
| [#4: Callee exclusions](https://github.com/Bairum/jevlint/issues/4) | Per-rule exclusions applied before context quotas and during localization; incompatible contexts use separate request batches. |
| [#5: Workflow hardening](https://github.com/Bairum/jevlint/issues/5) | Verified upstream commit pins, read-only CI/build tokens, publishing-only write permissions, and official GitHub artifact provenance over archives/checksums. |
| [#6: Terminal diagnostics](https://github.com/Bairum/jevlint/issues/6) | Shared provider error formatting escapes control/format characters and invalid UTF-8 while retaining ordinary Unicode text. |

Implementation: [Git filtering](internal/runner/gitignore.go),
[runner/context assembly](internal/runner/runner.go),
[shared exclusions](internal/scoping/scoping.go),
[client transport/logging](internal/evaluation/client.go),
[provider diagnostics](internal/evaluation/provider.go),
[CI](.github/workflows/ci.yml), [release](.github/workflows/release.yml).
Regression coverage: [privacy](internal/runner/privacy_test.go),
[Git layouts](internal/runner/gitignore_layout_test.go),
[client boundaries](internal/evaluation/client_hardening_test.go),
[provider diagnostics](internal/evaluation/provider_security_test.go),
[doctor output](internal/cli/doctor_security_test.go).

Compatibility: tracked files follow Git semantics; explicitly selecting an
ignored file (including an eval fixture) permits its submission. Non-Git
directories retain filesystem discovery. `include` selects evaluated units,
while `exclude` also limits that rule's callee context; evals still deliberately
clear those rule filters. HTTP to arbitrary remote hosts is no longer accepted.
Ordinary provider error messages remain visible, terminal-escaped: metadata-only
debugging is not a general-purpose secret scrubber.

Workflow pins were resolved from live upstream tags and checked against the
corresponding repository's commit API; version comments remain for Dependabot.
`actions/attest` is the official action recommended by the current
`attest-build-provenance` documentation for new implementations. Local checks
parsed both YAML workflows, checked action pin/permission/provenance structure,
and syntax-checked Bash steps. The actual checksum step and attestation subject
globs were exercised with a built Linux CLI archive and a synthetic ZIP fixture;
both archive checksums verified. This did **not** build Windows/macOS artifacts
or execute GitHub OIDC, attestation upload, or release publication.

### Hardening verification — 2026-10-04

Final integrated checks passed using the same Go 1.26.8/CGO environment:

- `go test -race -p 2 -count=1 ./...` — all nine internal packages passed.
- `go vet -p 2 ./...` — passed.
- `go build -p 2 -o jevlint-hardening-fixed ./cmd/jevlint`
  — passed.

New regression tests demonstrated the original privacy, transport, and terminal
failures on a pre-hardening filesystem snapshot. Review also found valid Git
submodule/directory-alias failures and a caller redirect-policy rewrite bypass
in the initial implementation; all were reproduced before correction. The
final suite covers these cases, parent-ignore protection for nested repositories,
custom TLS and redirect policies, normalized origins, and useful redacted
transport diagnostics.

Two older source-boundary tests used timing-sensitive `context.Err` mutation
hooks; they now mutate the filesystem between real discovery and evaluation
planning stages. Their confinement assertions remain. A discovery fixture's
fake `.git` directory was changed into a real repository rather than relaxing
the new fail-closed handling of corrupt Git metadata.

A freshly rebuilt CLI passed **14 end-to-end scenarios**, using only synthetic
source, local Git repositories, loopback providers, and dummy credentials:

| Scenario | Observed fixed behavior |
| --- | --- |
| Broad Git scan | One legitimate request; ignored source absent. |
| Explicit ignored file | One successful request: intentional opt-in. |
| Confined directory alias | One legitimate request; target's ignored source absent. |
| Initialized submodule | Two legitimate requests; submodule's ignored source absent. |
| Ignored nested repository | One outer-repository request; nested source absent. |
| Per-rule callee exclusion | Two separate batches; excluded context absent from the restricted rule and present for the permitted rule. |
| Debug logging | Source, response, and query secret markers absent. |
| Cross-origin redirect | Exit 2; zero requests to the redirect destination. |
| Same-origin redirect | Exit 0; initial and redirected requests succeed. |
| `check`/`doctor`, JSON/plain provider errors (four scenarios) | Exit 2; terminal control/format sequences escaped; ordinary Unicode retained. |
| Existing outside-root source symlink | Exit 2; zero provider requests. |

The pre-hardening CLI exposed the synthetic ignored/context/log/control markers
and followed the cross-origin redirect; positive explicit-file/same-origin
behavior and the previously fixed source confinement were retained. Temporary
baseline files, the baseline executable, and the CLI harness were removed after
verification. The installed toolchain and fixed executable remain in scratch;
permanent Go regressions remain in the repository.

An earlier Git-option-injection RCE suspicion was disproved for the actual Git
invocation. **No RCE was established by this audit.** Assess the observations
separately rather than escalating them to exploit claims without evidence.

### Pre-commit Jevlint review

The user ran the rebuilt CLI with `check --changed`: 31 files, 921 code units,
7,842 evaluations, and one `unnecessary-abstraction` warning at
`internal/evaluation/client_hardening_test.go:176-178`.
Review classified this as a false positive: `hardeningTransport.RoundTrip`
adapts a function to the standard `http.RoundTripper` interface and is reused
by four tests. The rule explicitly permits adapters. No suppression or code
change was made to silence it.

The user authorized a local commit, not a push. The resulting commit identity
belongs in Git history (`git log -1 --format=%H -- SECURITY-WORK.md`), avoiding
a self-referential commit hash in this document.

## Publication checklist (written before publication; superseded by the status above)

1. Privately coordinate findings, timing, and the intended public fork PR with
   the original author. Review this record for disclosure-sensitive details.
2. Review the local commit and any subsequent changes before publication. Keep
   unrelated files out of any follow-up commits.
3. Rerun the Go checks above against the final commit before publication. Record
   their actual results and any newly exercised smoke checks; do not imply that
   historical working-tree results verify later changes automatically.
4. Only after private coordination, publish explicitly to the fork:
   `git push -u origin fix/security-boundaries`.
5. Create the fork PR with explicit routing; the `gh` default may be upstream:
   `gh pr create --repo Bairum/jevlint --base master --head fix/security-boundaries --title "Fix pack pin and source fixture security boundaries" --body-file SECURITY-WORK.md`.
   This targets the fork's `master`, not the original author's repository.
6. Update this record with the real commit and fork-PR links, publication date,
   and any advisory or upstream handoff links only when they exist. No advisory,
   CVE, PR number, author approval, or future commit hash is asserted here.
