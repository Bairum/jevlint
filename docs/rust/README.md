# Primary Rust suite

## Scope and status

The first-class Rust suite lives in `packs/rust-*`, separate from the
`examples/packs/database-joins` tutorial. Start with the native baseline, then
use `rust-core` alone for strict, evidence-backed semantic checks. Add
`rust-core-advisory` for broader review and only the specialists relevant to the
code: unsafe/FFI, Tokio when used, public API contracts, or performance.
Run the opt-in `rust-readability` pack separately for Rust-specific
maintainability review; generic cross-language style rules are not part of
the semantic workflow.

The [coverage ledger](COVERAGE.md) maps every research candidate to implementation,
native-tool ownership, or an explicit outstanding item and promotion trigger.
Rule files and fixture compilation are not evidence of model precision.
[Fixture calibration](CALIBRATION.md) now records three real-provider runs,
measured recall/specificity, tradeoffs and optional per-rule project overrides.
Readability separately records a tuning-set/in-sample audit of a held-out private
Rust workspace (43 files, 958 units); its triage shaped the rules, so this is not
held-out accuracy. Real-project precision for the semantic packs remains
unmeasured before blocking adoption.

| Layer | Deliverable | Purpose |
| --- | --- | --- |
| 1 | [Native baseline](TOOLING.md) and [check-rust.sh](../../scripts/check-rust.sh) | Rustfmt, compiler, tests/doctests, documentation and Clippy |
| 2, strict | [Rust core](../../packs/rust-core/) | Semantic checks requiring visible evidence and contracts where applicable |
| 3, advisory | [Core advisory](../../packs/rust-core-advisory/) | Broader review without treating inferred contracts as strict failures |
| 3, specialist | [Unsafe/FFI contracts](../../packs/rust-unsafe/) | Locally visible unsafe and FFI contracts; not soundness certification |
| 3, specialist | [Tokio](../../packs/rust-tokio/) | Runtime, cancellation, admission and lifecycle checks when Tokio is used |
| 3, specialist | [API contracts](../../packs/rust-api/) | Public constructor, trait and Deref contracts |
| 3, specialist | [Performance](../../packs/rust-performance/) | Workload-dependent materialization and I/O review |
| 3, separate review | [Readability](../../packs/rust-readability/) | Contextual domain values, abstraction levels and Rustdoc accuracy |

Packs are independent, with fixture calibration measured in
[CALIBRATION.md](CALIBRATION.md). Strict describes the evidence required by
`rust-core`, not a claim of production accuracy or automatic blocking enforcement.
No Rust pack is automatically enabled in this repository's configuration. Select advisory and
specialist packs deliberately; their presence does not establish that a project
uses Tokio or needs a particular performance policy.

## Run the checks

From this checkout, with Cargo, rustfmt, Clippy and Python 3.9+ available
(the Python runner uses `Path.is_relative_to`):

```sh
# Layer 1: a real Rust project with an intentional Cargo.lock; workspace by default.
sh scripts/check-rust.sh /path/to/rust-project
sh scripts/check-rust.sh /path/to/rust-project/Cargo.toml

# Select packages instead of the workspace; omit only the formatting step.
sh scripts/check-rust.sh /path/to/rust-project -p server -p shared --skip-fmt

# Compile all pack fixtures; deliberately bad/unsafe examples are never executed.
python3 scripts/check-rust-packs.py

# Compile and calibrate only the strict core, with three uncached provider runs.
# Requires the built jevlint binary and its configured provider credentials.
python3 scripts/check-rust-packs.py --eval --repeat 3 --pack rust-core \
  --jevlint /path/to/jevlint

# Optionally choose the JSON summary destination.
python3 scripts/check-rust-packs.py --eval --repeat 3 --pack rust-core \
  --jevlint /path/to/jevlint --summary /path/to/rust-core-summary.json
```

After the native baseline, consult [CALIBRATION.md](CALIBRATION.md) before
selecting semantic rules or changing a consuming project's confidence floors.
It includes the original six-pack baseline and the separately measured
readability fixture calibration and tuning-set/in-sample audit, alongside the
low-recall/false-positive caveats and optional overlay recommendations. Keep the
global floor at 0.8; the packs ship no rule-level `minConfidence`.

The native script accepts a project directory or `Cargo.toml`. It selects
`--workspace` by default; repeated `-p` options select packages instead.
Compiler checks, tests/doctests, documentation and Clippy retain `--locked`;
`--skip-fmt` skips rustfmt only.

The fixture runner uses a temporary Cargo project, real Tokio and PyO3
dependencies, and the real `jevlint eval --refresh-cache --format json --verbose`
command on every evaluation run. It never substitutes synthetic model answers.
It may download uncached dependencies. Direct dependency versions are pinned by
the runner; transitive dependencies are resolved into a temporary lockfile on
each invocation, so this is not a fully locked/MSRV/target matrix.

Evaluation summaries are timestamped JSON files in the system temporary
directory by default; `--summary` selects a destination.
They retain per-run fixture cases and aggregate outcomes. A case has a majority
outcome only when more than half the runs agree; otherwise it is inconclusive.
The flip rate is the proportion of cases whose outcomes differ across runs;
a missing result differs from an observed result. Mismatch and inconclusive
counts are per-run case counts: mismatches compare each run's result with the
fixture expectation, while inconclusive counts include missing results from
errors. Errors are also recorded separately. Per-rule fixture recall and
specificity use majority outcomes; a metric with no fixture denominator is
`null`. These corpus metrics, alongside abstain and below-confidence-floor
counts, do not establish production accuracy. The [recorded calibration](CALIBRATION.md)
also reports per-run recall/specificity under `check` reporting semantics; its
recommended overrides are fitted on the same fixtures, not held-out project
code. Compilation alone establishes neither precision nor recall.

### Use a pack in a Rust project

Commit the pack changes, their documentation and tests together before pinning
a local Git pack. Installation reads committed content, not working files:

```sh
# Run in the consuming Rust project, with rust enabled in its jevlint.json.
jevlint plugin install /path/to/jevlint#packs/rust-core
```

Use the corresponding directory for advisory or specialist packs. Configure
the consuming project's `jevlint.json`, not Jevlint's own configuration, and
preserve that project's existing settings. A GitHub tree URL is an alternative
after publication; no unpublished commit is advertised here as installable.
See [pack installation](../../README.md#packs) and each pack's README.

For an uncommitted local pack, the fixture runner can evaluate its cases directly.
To check other Rust source before committing, enable
`"languages": {"rust": {}}` in the consuming project's normal `jevlint.json` and
copy the selected pack's `rules` array into it. Do not use `rules.json` alone as a
configuration: it does not enable a language. Select only intended rules.

## Preserved research

The reports below preserve the complete original research snapshots, including candidate IDs, primary-source URLs, exceptions, fixture concepts, tool overlap, and extraction limitations.

| Report | Preserved scope |
| --- | --- |
| [Memory and resources](research/memory.md) | M01–M26: 26 candidates, unsafe/FFI/pinning contracts, and review/tool limits |
| [APIs, functions, and errors](research/api.md) | API01–API27: 27 candidates, documentation, compatibility, and excluded universal preferences |
| [Concurrency, async, and I/O](research/concurrency.md) | C01–C18: 18 candidates, runtime/resource contracts, and capability gates |
| [Native tooling baseline](research/tooling.md) | Complete compiler, Clippy, rustfmt, rustdoc, CI coverage, and optional tool findings |

See the [master research proposal](../../RUST-PACK-RESEARCH.md) for the condensed
synthesis and [coverage ledger](COVERAGE.md) for current scope and verification.

The reports preserve the original research phase, not current implementation
status. Historical extraction limitations are not the current capability
contract: `context.types` opts into bounded same-file type/impl context,
including type units and Self-only methods, plus cross-file context when a type
name has one project-wide definition. Ambiguous names stay same-file. This uses
discovered source files, not Rust name, module or compiler type resolution;
rule-excluded declaration paths are filtered before the extra-context caps of
12 declarations and 16 KiB. Consult the current pack READMEs for each rule's
evidence requirements and calibration status.
