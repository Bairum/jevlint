# Jevlint

Jevlint checks code against plain-language rules. It uses Tree-sitter to extract
code units, then asks Jev whether each one passes.

## Install

Download a prebuilt binary from
[Releases](https://github.com/codegirl-007/jevlint/releases) — `linux` and
`macos` on `amd64`/`arm64`, and `windows` on `amd64` — unpack it, and put
`jevlint` on your `PATH`.

Or install with Go:

```sh
go install github.com/codegirl-007/jevlint/cmd/jevlint@latest
```

The release workflow now generates GitHub artifact provenance for archives and
`checksums.txt`. For a release produced by this workflow, set `ASSET` to the
downloaded archive's path and verify it against the repository that published it:

```sh
gh attestation verify "$ASSET" --repo Bairum/jevlint
```

Use `--repo codegirl-007/jevlint` for upstream releases once upstream adopts this
workflow. Older releases may have checksums without attestations; the local
workflow change does not retroactively attest them.

## Getting started

```sh
cd your-project
jevlint init                          # write a starter jevlint.json
export TYPESAFE_API_KEY=apikey_...    # from console.typesafe.ai
jevlint doctor                        # verify the config and credentials
jevlint check .
```

`init` detects the languages in the project and writes a starter
`jevlint.json`. Add rules to `jevlint.json` (or install a pack) before the
first check.

## Requirements

Building from source (or installing with `go install`) needs:

- Go 1.26 or newer
- CGO enabled
- A C compiler

Running checks needs a TypeSafe API key from
<https://console.typesafe.ai/settings/keys>.

## Commands

| Command | Description |
| --- | --- |
| `jevlint init` | Write a starter `jevlint.json` for the project. |
| `jevlint doctor` | Check that the config loads and the credentials work. |
| `jevlint check [paths...]` | Check code against the rules. |
| `jevlint eval` | Score rules against fixtures in `jevlint-evals.json`. |
| `jevlint plugin ...` | Manage rule packs. |
| `jevlint version` | Print the version. |

`init` and `doctor` accept `--json` for machine-readable output.

## Run

Set `TYPESAFE_API_KEY` in your environment (see
[Configuration](#configuration)), then run from the project root:

```sh
go run ./cmd/jevlint init
go run ./cmd/jevlint doctor
go run ./cmd/jevlint check .
go run ./cmd/jevlint check --format json .
go run ./cmd/jevlint check --concurrency 8 . 
go run ./cmd/jevlint check --refresh-cache .
go run ./cmd/jevlint eval
go run ./cmd/jevlint eval --rule database-joins --format json
```

| Flag | Description |
| --- | --- |
| `--changed` | Check only git-modified files (staged, unstaged, and untracked). |
| `--clear-cache` | Clear this project's cached evaluations before checking. New results are cached. |
| `--color auto\|always\|never` | Control colored text output. Defaults to `auto`. |
| `--config path` | Use a different rule file. Its directory becomes the project root. |
| `--concurrency number` | Set the maximum number of concurrent Jev requests. Defaults to `4`. |
| `--fail-on info\|warning\|error` | Exit with `1` for reported findings at or above this severity. Defaults to `info`. |
| `--format text\|json` | Select human-readable or machine-readable output. Defaults to `text`. |
| `--refresh-cache` | Reevaluate code and replace matching cached results. |
| `--show-below-floor` | Show results below the reporting threshold. These never affect the exit code. |

Source reads, including callee context, stay inside the project root. Explicit
paths outside that root are rejected; `--changed` skips escaping symlinks.
Relative symlinks must stay inside the root. Absolute symlink targets are not
followed, even when they point inside the root.

Directory scans in Git worktrees honor Git's ignore rules, including nested
`.gitignore` files, `.git/info/exclude`, and global excludes. Tracked files remain
eligible, matching Git semantics. Initialized submodules use their own ignore
rules; an ignored nested repository stays excluded by its parent. Confined
directory symlinks use the target directory's Git rules. Ignored files are
removed before callee indexing. Git must be available and working when a Git
worktree is detected; otherwise the scan fails rather than silently ignoring
the privacy filter.
Non-Git directories retain filesystem discovery.

Explicit file arguments override Git ignores, including when combined with a
directory argument. Eval fixtures are also explicit selections. Treat selecting
an ignored file as permission to send its applicable code to the provider.

### Check output

Text output groups findings by code unit, showing its location and code frame
once, followed by each rule. A rule's description appears only on its first
appearance. Totals include abstained and below-floor decisions. Progress appears
on stderr only when stderr is a terminal and is cleared on completion.

`check --format json` writes a report with these fields:

| Field | Description |
| --- | --- |
| `scannedFiles` | Number of scanned files. |
| `codeUnits` | Number of extracted code units. |
| `evaluations` | Number of rule evaluations, including localization requests. |
| `cache` | Cache statistics, when available. |
| `models` | Sorted distinct answering model ids, including cached answers. |
| `usage` | Provider calls in this run: `requests`, `inputTokens`, `outputTokens`. Cache hits add nothing. |
| `rules` | Object keyed by rule ID, containing rules that received at least one evaluation. Each entry has `description`, `severity`, and `decisions`. |
| `findings` | Reported results whose fail probability meets the reporting threshold. |
| `belowFloor` | Notable results below the threshold, included only with `--show-below-floor` and when nonempty. Never affect the exit code. |
| `oversized` | Units skipped because they still exceeded the model input budget. Does not affect the exit code. |
| `contextDropped` | Units whose callee or type context was dropped so the request fit the budget. |

Each rule's `decisions` contains integer counts named `pass`, `fail`, `skip`,
`abstain`, `reported`, and `belowFloor`. `pass`/`fail`/`skip`/`abstain` count
the status. `reported` counts units whose fail probability is at least the
threshold. `belowFloor` counts units at least half the threshold and below it.

Findings contain `ruleId`, `severity`, `status`, `path`, `language`, `kind`,
`name`, `startLine`, `endLine`, `startColumn`, `endColumn`, `snippet`,
`failProbability`, and optional `locations`. They have no `description`; look
it up in `rules[ruleId]`.

Text output prints one line with the answering models and token usage, and
warns when units were skipped as oversized.

## Configuration

Jevlint reads `jevlint.json` for rules. Credentials and provider settings come
from environment variables.

```sh
export TYPESAFE_API_KEY=apikey_...
```

`init` writes only `jevlint.json`; it does not create or modify any other file.

- **Keep it secret:** export the key from your shell profile or a secret
  manager. Jevlint never writes credentials into the project.
- **Debugging:** set `JEVLINT_DEBUG=1` for metadata-only diagnostics: provider,
  credential kind/length, numeric response status, and byte counts. Raw endpoint
  URLs, model values, request bodies, and response bodies are not logged.
  Service error messages remain visible with terminal controls escaped;
  diagnostics are not a general-purpose secret scrubber.

### Providers

Jevlint selects a provider from `JEVLINT_PROVIDER`: `typesafe`, `jev`,
`cloudflare`, `clef`, or `openrouter` (default: `typesafe`). The default talks
to TypeSafe's Jev at `https://api.typesafe.ai/v1/systemone`.
`cloudflare`/`clef` use Cloudflare Workers AI, and `openrouter` uses Jev
through OpenRouter (see below).

| Variable | Meaning |
| --- | --- |
| `JEVLINT_PROVIDER` | Provider: `typesafe`, `jev`, `cloudflare`, `clef`, or `openrouter` (default: `typesafe`). |
| `TYPESAFE_API_KEY` | API key. |
| `TYPESAFE_BASE_URL` | Service base URL. Defaults to `https://api.typesafe.ai`. |
| `TYPESAFE_DEFAULT_MODEL` | Model name. Defaults to the pinned release `jev-1.13.0`. |
| `TYPESAFE_ENDPOINT` | Full request URL, bypassing `TYPESAFE_BASE_URL`. |

The default model is the pinned release `jev-1.13.0`, not the moving alias
`jev-latest`. Each provider response records the answering model and token
usage. Check and eval reports list the distinct models and the usage of
provider calls made in that run. Cache hits contribute their stored model and
add no tokens.

Remote endpoints must use HTTPS. HTTP is allowed only for literal loopback IPs
or exact `localhost` for local development; endpoint credentials and fragments
are rejected. Redirects must preserve scheme, hostname, and effective port:
cross-origin redirects and HTTPS downgrades are rejected before contacting the
destination. Custom HTTP clients retain their TLS settings and any stricter
redirect policy.

#### Cloudflare Workers AI (Clef)

Jevlint can use the [Clef decision models on Cloudflare Workers
AI](https://developers.cloudflare.com/workers-ai/models/clef) instead of Jev.
Set `JEVLINT_PROVIDER=cloudflare`:

```sh
export JEVLINT_PROVIDER=cloudflare
export CLOUDFLARE_ACCOUNT_ID=<account id>
export CLOUDFLARE_AUTH_TOKEN=<cloudflare api token>
export CLEF_MODEL=clef        # or clef-flash
```

| Variable | Meaning |
| --- | --- |
| `CLOUDFLARE_ACCOUNT_ID` | Cloudflare account id. |
| `CLOUDFLARE_AUTH_TOKEN` | Cloudflare API token. `CLOUDFLARE_API_TOKEN` is accepted as an alias; if both are set, they must match. |
| `CLEF_MODEL` | `clef` (default) or `clef-flash`. |

Requests go to
`https://api.cloudflare.com/client/v4/accounts/<account id>/ai/run/@cf/cloudflare/<model>`.

Create a **Workers AI API Token** with `Workers AI: Read` and
`Workers AI: Edit` permissions:

1. Open the [Workers AI page](https://dash.cloudflare.com/?to=/:account/ai/workers-ai)
   and select **Use REST API**.
2. Select **Create a Workers AI API Token**, then copy it — it is shown only
   once.

You can also create one from the
[API Tokens page](https://dash.cloudflare.com/profile/api-tokens) using the
**Workers AI** template. The account id is on the Workers AI page, or in the
dashboard URL. A Global API Key, or a token without the Workers AI permission,
is rejected. See the
[Workers AI REST API guide](https://developers.cloudflare.com/workers-ai/get-started/rest-api/)
for details.

Clef has the same request and response shape as Jev, so rules, findings, and
caching work unchanged.

#### OpenRouter (Jev)

[OpenRouter](https://openrouter.ai) serves Jev through a System One endpoint, so
you can run Jevlint with an OpenRouter key and OpenRouter billing instead of a
TypeSafe account. Set `JEVLINT_PROVIDER=openrouter`:

```sh
export JEVLINT_PROVIDER=openrouter
export OPENROUTER_API_KEY=sk-or-...
export OPENROUTER_MODEL=typesafe/jev-1.13   # or ~typesafe/jev-latest
```

| Variable | Meaning |
| --- | --- |
| `OPENROUTER_API_KEY` | OpenRouter API key. |
| `OPENROUTER_MODEL` | Model id. Defaults to `typesafe/jev-1.13`. |
| `OPENROUTER_BASE_URL` | Service base URL. Defaults to `https://openrouter.ai/api`. |
| `OPENROUTER_SITE_URL` | Optional; sent as the `HTTP-Referer` attribution header. |

Requests go to `https://openrouter.ai/api/v1/systemone`, and the
`X-Title: jevlint` attribution header is sent. Create a key at
<https://openrouter.ai/settings/keys>. Jev is a decision model with the same
request and response shape as TypeSafe's Jev, so rules, findings, and caching
work unchanged.

## Rules

Jevlint reads `jevlint.json` by default.

```json
{
  "languages": {
    "go": {},
    "typescript": {},
    "tsx": {}
  },
  "rules": [
    {
      "id": "database-joins",
      "description": "Join related database records in the database.",
      "severity": "error",
      "kinds": ["function"],
      "include": ["src/**/*.go"],
      "exclude": ["**/*_test.go"],
      "exceptions": ["The records come from different databases."],
      "localize": ["statement"]
    }
  ]
}
```

No languages are enabled by default, and a config must enable at least one
language and define at least one rule (or list a pack). Available presets are
`c`, `cpp`, `csharp`, `go`, `java`, `javascript`, `kotlin`, `php`, `python`,
`ruby`, `rust`, `tsx`, and `typescript`.

Each preset includes common extensions, extraction queries, and localization
regions. Override only what your project needs:

```json
{
  "languages": {
    "cpp": {
      "extensions": [".cpp", ".hpp"]
    }
  }
}
```

Presets use native Tree-sitter grammars compiled into Jevlint. Config can
customize a preset, but it cannot load an arbitrary external grammar.

- `severity`: `info`, `warning`, or `error`
- `include` and `exclude`: doublestar file patterns
- `sourceMatch`: optional RE2 regular expressions matched against the primary
  unit's source. At least one must match before Jevlint sends a request; omitted
  or `[]` means no source filter. Empty or invalid patterns are rejected.
- `guidance`: shared instruction text, included in each question that uses it.
  It is not sent as state.
- `includeTests`: evaluate structurally identified Rust test units when `true`;
  they are skipped by default.
- `kinds`: `comment`, `docComment`, `field`, `function`, `statement`, or `type`
- `exceptions`: optional cases that should pass. They are sent only with the
  default score, for a rule that sets neither `score` nor `checks`, and with
  localization questions. Explicit score and check questions do not repeat them.
- `localize`: `comment`, `docComment`, `field`, or `statement`. Omit the key
  or use `[]` to skip localization. On a reported failure, matching regions
  (up to 24) are judged together in one request. Each region question is built
  from the description and exceptions. A region is a location when its
  yes-probability reaches the reporting threshold.
- `score`: optional 3-level question. `question` plus exactly three `levels`
  (compliant or out of scope, borderline, clear violation) and up to two
  `paraphrases` of the same shape. Omit it and Jevlint sends the default
  question "How clearly does `source` violate `rule`?" with levels derived
  from the description. The score signal is the mean of `score / 2` across
  those wordings.
- `checks`: optional object. `subject` questions are a hard gate: every
  yes-probability must be at least `0.5`. `violation` is one or more ways to
  break the rule; each may have up to two paraphrases, which are averaged,
  and the violations are combined with max. The checks signal is that product
  of the gate and the strongest violation. A rule with no checks uses the
  score signal for both. `allowSkip`, `allowAbstain`, and the old `checks`
  array with `failWhen` are rejected.

### Reporting threshold

- `minFailProbability`: optional `0`–`1`. A unit is reported when its fail
  probability is at least the threshold. The threshold is the rule value, or
  the global value, or `0.80` when both are omitted. A rule value overrides the
  global value. Results at least half the threshold and below it are
  below-floor: notable, not reported, and shown only with `--show-below-floor`.
  `minConfidence` is rejected.
- The fail probability is the minimum of the score and checks signals, so a
  unit is reported only when both agree. Status is `fail` at `0.5` or above
  and `pass` otherwise. `eval --verbose` includes `score`, `checks`,
  `subjectGate`, and the per-question answers on each unit.
- `context.callees`: include confidently resolved direct project-local callees
  as extra state. Depth is 1 and bounded (12 callees, about 16 KiB of source).
  Ambiguous and external calls are ignored. Useful when the target function
  alone does not contain enough evidence. When mixed with ordinary rules on the
  same unit, Jevlint sends a second request.
  A rule's `exclude` patterns also remove matching callee paths before applying
  context limits, including during localization. `include` still selects only
  directly evaluated units. Rules with different allowed callee context use
  separate requests, so one rule cannot expose excluded context to another.
- `context.types`: include bounded additional related declarations and impls,
  including confidently resolved project-local types from other files. A rule's
  `exclude` removes candidate paths before the 12-declaration / 16 KiB limits.
  Ordinary same-file related types remain available without this option.

Rules sharing identical callee and type context are batched. Jev judges only
the primary source; defects present only in context must not fail that unit.

For `jev-1.13*` and `typesafe/jev-1.13*`, Jevlint counts each request's tokens
offline before sending (an exact Jev 1.13 counter ported from
[oh-my-pi](https://github.com/can1357/oh-my-pi)) and keeps it within the
documented 64,000-token request limit and 32,000-token state-plus-longest-question
limit. Other models are not
preflighted. Over budget, questions are split across requests (Cloudflare is
also split at 64 questions). If that is not enough, callees are dropped, then
types, and the report says so. A unit that still does not fit is skipped:
`oversized` lists it, the text summary warns, and the run continues. Oversized
units alone do not change the exit code.

```json
{
  "id": "database-joins",
  "description": "Fail when related database records are joined in application code instead of in the query.",
  "kinds": ["function"],
  "context": {
    "callees": true
  }
}
```

## How Jevlint uses Jev

One fan-out request per code unit: every applicable rule that shares context is a set of questions in that request. Budget splits are the exception, not a second question style.

Each rule asks a 3-level score (compliant or out of scope, borderline, clear violation) plus up to two paraphrases, and optional subject-gated violation checks. Subject checks are a hard gate: every yes-probability must be at least 0.5. Violation paraphrases are averaged; violations combine with max. The score signal is the mean of `score / 2` across wordings. `failProbability` is the minimum of the two signals, so a unit is reported only when both agree. With no checks, the score is used for both.

The default reporting threshold is `0.80` (rule `minFailProbability`, else the global value, else `0.80`). Audits show the below-floor band `0.40`–`0.80` with `--show-below-floor`. Those results do not change the exit code.

The default model is the pinned release `jev-1.13.0`, not `jev-latest`. Check and eval reports list answering models and the token usage of provider calls in that run. Cache hits contribute their stored model and add no tokens.

For `jev-1.13*` and `typesafe/jev-1.13*`, an offline counter ported from [oh-my-pi](https://github.com/can1357/oh-my-pi) preflights each request. Live measurement put the hard caps at 32,743 tokens for state plus the longest question and 65,533 tokens total. Jevlint stays 743 and 1,533 tokens under those caps (32,000 and 64,000). The estimator reproduces `usage.input_tokens` exactly, so no further margin is applied. Other models are not preflighted.

## How it works

- Rules can check functions, types, comments, fields, or statements.
- Comments, fields, and statements include their nearest function or type as context.
- Code units are checked in parallel, four at a time by default.
- Applicable rules with the same context requirements are batched into one
  request per code unit.
- Failed function and type rules can set `localize` for a focused second pass.
  That pass is extra Jev evaluations and is off unless the rule lists
  categories.
- Results include syntax-highlighted snippets and pointers when available.
- Network failures and retryable API responses are retried up to twice.

Jevlint sends extracted source code and file metadata to TypeSafe.

### Rust tests and type context

Rust functions with `#[test]` or a namespaced `#[...::test]` attribute, and
declarations inside modules gated by a positive `cfg(test)` predicate, are test
units. Inner `#![cfg(test)]` attributes and external `#[cfg(test)] mod name;`
files (including `#[path = "..."]`) are recognized. Comments between attributes
and declarations do not change detection; `cfg(not(test))` is not a test gate.
These units, including their comment/field/statement regions, are skipped unless
the rule sets `includeTests: true`.

With `context.types: true`, a type unit receives its same-file impls, and a
method using only `Self` receives its owner's sibling declarations and impls.
Cross-file context is added only for names with exactly one discovered type
definition; ambiguous names retain same-file context. This is conservative
name matching, not Rust type or module resolution. Only discovered files are
indexed. Rule-excluded candidate paths and duplicates of ordinary same-file
context are removed before the additional 12-declaration / 16 KiB source budget.

## Cache

Jevlint caches validated results in the operating system's user cache directory.
Entries are isolated by project and contain results only, never source code or
API keys.

The cache key includes the API endpoint, model, API credential fingerprint, and
the exact request sent to Jev. Code, context, rules, prompts, or batch changes
create a new entry. Severity changes reuse the result because severity only
affects reporting.

Entries do not expire automatically. The cache key version is v2 and stores
the parsed per-question answers plus the answering model. Use `--refresh-cache`
to reevaluate and replace cached results. Use `--clear-cache` to clear this
project's cache before a run.

## Eval

`jevlint eval` scores your rules against fixtures you list in
`jevlint-evals.json` (next to `--config`, or `--evals`). Check never loads that
file. Each case names a rule, a fixture relative to the eval file, and
`expect: pass` or `expect: fail`. Pack evals stay with the pack unless you
pass `--packs`.

Fixtures must be regular source files inside the eval document's directory
or its subdirectories. Absolute paths, `..` escapes, and escaping symlinks
are rejected. Pack evals use their own eval document directory as this
boundary, not the project's directory.

Eval evaluates only that rule. It clears the rule's include and exclude so
fixtures still run, and keeps kinds, exceptions, localize, checks, and minFailProbability.
A case must evaluate at least one applicable code unit or eval exits `2`.

The repository ships its own cases in `jevlint-evals.json` covering the
fixtures under `examples/rules`. Folder names such as `good` and `bad` are
organizational only; the case's `expect` value decides the outcome. Eval sends
the evaluator an opaque file name with the fixture's extension, so names like
`bad/` or `fail.go` never reveal the expected verdict. A rule can
have many cases, including several for the same language.

```json
{
  "version": 1,
  "cases": [
    {
      "rule": "database-joins",
      "file": "examples/rules/database-joins/bad/example.go",
      "expect": "fail"
    }
  ]
}
```

| Flag | Description |
| --- | --- |
| `--clear-cache` | Clear this project's cached evaluations before evaluating. |
| `--color auto\|always\|never` | Control colored text output. Defaults to `auto`. |
| `--config path` | Use a different rule file. Its directory becomes the project root. |
| `--concurrency number` | Set the maximum number of concurrent Jev requests. Defaults to `4`. |
| `--evals path` | Use a different eval file. Fixtures stay relative to that file. |
| `--format text\|json` | Select human-readable or machine-readable output. Defaults to `text`. |
| `--packs` | Also run evals from packs listed in `jevlint.json`. |
| `--refresh-cache` | Reevaluate code and replace matching cached results. |
| `--rule id` | Evaluate only this rule's cases. |
| `--verbose` | Include the per-unit decisions in JSON output. |

Each case evaluates the rule against every code unit in the fixture. Each
evaluation returns one raw decision. Status is `fail` when the fail
probability is at least `0.5`, otherwise `pass`. The
tool reports a violation only when the fail probability reaches the reporting
threshold (the rule's `minFailProbability`, the global one, or `0.80`); each
reported violation becomes a finding.

The case outcome follows from those decisions:

- `fail`: at least one violation was reported.
- `inconclusive`: no violation was reported, but a unit was below the
  reporting threshold, or no unit returned an explicit pass.
- `pass`: otherwise, meaning at least one explicit pass and no failure.

A case matches when its outcome equals `expect`, and `inconclusive` never
matches. This keeps an expected pass from succeeding just because evaluations
hid a failure under the reporting threshold, and it makes an expected fail
require a reportable violation.

Text output streams as Jev answers come back. It starts with a legend, prints
each code unit's answer and fail probability as it arrives, then prints the
case's expected and actual outcome. The run ends with
`N/M eval cases matched expectations, K inconclusive` and a models/usage line.
Cases stay in file order; units within a case print in completion order.

JSON is written once after the run finishes. It includes the per-case
`failProbability` (the maximum across that case's units), the raw decision
counts (`pass`, `fail`, `skip`, `abstain`, `reported`, `belowFloor`), the
reporting threshold (`floor`), `models`, `usage`, and the suite totals
(`reportedFailures`, `belowFloorFailures`). `--verbose` adds a `units` array
with each code unit's kind, name, lines, status, fail probability, `score`,
`checks`, `subjectGate`, per-question `answers`, and whether it was reported.

Exit codes:

- `0`: every case matched
- `1`: at least one case was mismatched or inconclusive
- `2`: configuration, parsing, or provider error

## Packs

A pack is a shared directory with a `pack.json` manifest, rules, optional evals,
and fixtures. Pins live in `jevlint.json`; fetched files live in the user cache,
not the project tree.

Each `sha` must be a full 40- or 64-character hexadecimal commit ID, not a
branch, tag, or abbreviated hash. Fetches verify the checked-out commit
before caching it; plugin installation can still select a branch or tag.

Verified packs use the `jevlint/packs/v2` directory under the user cache.
Older pack caches are left untouched but ignored, so the first run after
upgrading requires access to each pack's source to fetch it again.

```text
pack.json
rules.json
guidance.md
jevlint-evals.json
fixtures/
```

```json
{
  "version": 1,
  "id": "codegirl-007/database-joins",
  "languages": ["go"],
  "rules": "rules.json",
  "evals": "jevlint-evals.json",
  "guidance": "guidance.md"
}
```

`rules` and `evals` are optional and default to `rules.json` and
`jevlint-evals.json`. Optional `guidance` names a shared text file; its content
is applied to pack rules without their own `guidance`, before project overlays.
All three paths must be relative and stay inside the pack. Any languages a pack
declares must be enabled in your config. Pack ids are `owner/name`, where each
part is a simple identifier (letters, digits, `.`, `_`, `-`), and packs
containing symbolic links are rejected.

```json
{
  "languages": { "go": {} },
  "minFailProbability": 0.5,
  "packs": [
    {
      "id": "codegirl-007/database-joins",
      "source": "https://github.com/codegirl-007/jevlint.git",
      "path": "examples/packs/database-joins",
      "sha": "<commit-sha>"
    }
  ],
  "rules": [
    {
      "id": "database-joins",
      "minFailProbability": 0.5,
      "include": ["src/**/*.go"]
    }
  ]
}
```

Project rule overlays replace `sourceMatch` when present (`[]` clears the
filter), and replace `guidance` when non-empty. `includeTests: true` and
`context.types: true` enable those options; an overlay's `false` cannot disable
an inherited `true`.

Create a new pack with `plugin init`:

```sh
jevlint plugin init codegirl-007/database-joins
jevlint plugin init my-org/my-pack --languages go,typescript --dir ./my-pack
```

It writes `pack.json`, `rules.json`, and a README, plus example evals and Go
fixtures when `go` is one of the languages. Commit the result to a git
repository to install it.

`plugin install` writes the pin. A project can use only pack rules. Project
`rules` entries overlay a pack rule by id (confidence, include/exclude,
severity) or add a local rule. Check never opens pack evals.

The repository ships a sample pack at `examples/packs/database-joins`. Install
it with a GitHub tree URL, or install your own pack from a tree URL or a local
git repository:

```sh
go run ./cmd/jevlint plugin install https://github.com/codegirl-007/jevlint/tree/master/examples/packs/database-joins
go run ./cmd/jevlint plugin install /path/to/packs#database-joins
go run ./cmd/jevlint plugin list
go run ./cmd/jevlint plugin update
go run ./cmd/jevlint plugin remove codegirl-007/database-joins
```

Tree URLs may name branches or tags that contain slashes, such as
`.../tree/feat/new-rules/packs/mine`: the longest leading part of the path that
names a branch, tag or commit in the repository is the ref, and the rest is the
pack path.

### Primary Rust suite

The first-class Rust suite lives in [`packs/`](packs/), separate from the
`examples/packs/database-joins` tutorial. Start with the native Cargo/rustfmt/
Clippy baseline, then run [`rust-core`](packs/rust-core/) alone for strict,
evidence-backed semantic checks. Add
[`rust-core-advisory`](packs/rust-core-advisory/) for broader review and only
code-relevant specialists: [`rust-unsafe`](packs/rust-unsafe/) for unsafe/FFI,
[`rust-tokio`](packs/rust-tokio/) when Tokio is used,
[`rust-api`](packs/rust-api/) for API contracts, and
[`rust-performance`](packs/rust-performance/) for workload-sensitive review.
Use [`rust-readability`](packs/rust-readability/) separately for calibrated
Rust-specific maintainability review; generic cross-language style rules are
not part of the primary semantic workflow.

No pack is automatically enabled. Strict describes the core's evidence
requirements, not automatic blocking enforcement. The 2026-10-06 fixture,
real-code, and held-out measurements are in
[CALIBRATION.md](docs/rust/CALIBRATION.md). Packs ship no per-rule floor.
Unsafe fixtures are compile-only, and passing a rule is not a soundness proof.

Commit pack changes, documentation and tests together before pinning a local
Git pack: installation reads committed content, not working files. From the
consuming Rust project, enable Rust in **that project's `jevlint.json`**, not
Jevlint's own configuration, then install with
`jevlint plugin install /path/to/jevlint#packs/rust-core`.
Select advisory and specialist packs deliberately.

The native script accepts a project directory or `Cargo.toml`, selects the
workspace by default, accepts repeated `-p` package selections instead, and
supports `--skip-fmt`; other native checks retain `--locked`. The fixture
runner requires Python 3.9+. To reproduce the recorded core table, pass the
shipped threshold (the runner default is 0.5):
`python3 scripts/check-rust-packs.py --eval --repeat 2 --min-fail-probability 0.8 --pack rust-core`
with `--jevlint /path/to/jevlint` if needed and optional `--summary /path/to/summary.json`.
Default timestamped JSON summaries go to the system temporary directory
and record per-run cases, strict-majority outcomes (otherwise inconclusive),
flip rate, per-run mismatch/inconclusive counts, majority-based fixture
recall/specificity, and below-floor counts.
See [Rust suite usage](docs/rust/README.md) for commands and summary definitions,
and the [research-to-implementation coverage ledger](docs/rust/COVERAGE.md).

## Supported languages

| Preset | Extensions |
| --- | --- |
| `c` | `.c` |
| `cpp` | `.cc`, `.cpp`, `.cxx`, `.h`, `.hpp`, `.hxx` |
| `csharp` | `.cs` |
| `go` | `.go` |
| `java` | `.java` |
| `javascript` | `.js`, `.jsx`, `.mjs`, `.cjs` |
| `kotlin` | `.kt`, `.kts` |
| `php` | `.php`, `.phtml` |
| `python` | `.py` |
| `ruby` | `.rb`, `.rake`, `.gemspec` |
| `rust` | `.rs` |
| `tsx` | `.tsx` |
| `typescript` | `.ts`, `.mts`, `.cts` |

## Limits

- Evaluation is scoped to the configured code-unit kinds.
- Comments, fields, and statements receive only their nearest declaration as parent context.
- Jevlint can optionally include bounded depth-1 project-local callee context,
  but it does not perform recursive call-graph or cross-function data-flow
  analysis. Imports are not followed.
- Same-file related types come with the unit. `context.types` adds a bounded
  set of declarations and impls, including a cross-file type only when its name
  has one project-wide definition.
- Jev returns a 3-level score and yes/no checks, not a free-form explanation.

## Exit codes

- `0`: no reported failures at or above the `--fail-on` severity
- `1`: at least one reported failure at or above the `--fail-on` severity
- `2`: usage, configuration, or runtime error

`--fail-on` defaults to `info`, so any reported finding exits with `1`.
Results below the reporting threshold never cause exit `1`, even with
`--show-below-floor`.
