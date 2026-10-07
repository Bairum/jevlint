# Primary Rust suite

## Scope and status

The first-class Rust suite lives in `packs/rust-*`, separate from the
`examples/packs/database-joins` tutorial. Start with the native baseline, then
use `rust-core` alone for strict, evidence-backed semantic checks. Add
`rust-core-advisory` for broader review and `rust-performance` only when the
workload question is in scope.
Run the opt-in `rust-readability` pack separately for Rust-specific
maintainability review; generic cross-language style rules are not part of
the semantic workflow.
Unsafe, async, and trait-law defects are not a Jevlint pack. See
[Rust tooling beyond Jevlint](#rust-tooling-beyond-jevlint).

The [coverage ledger](COVERAGE.md) maps every research candidate to implementation,
native-tool ownership, or an explicit outstanding item and promotion trigger.
Rule files and fixture compilation are not evidence of model precision.
The [2026-10-06 calibration](CALIBRATION.md) records fixture rates at the shipped
0.80 fail-probability threshold, the decision experiment, labelled real-code
precision, and held-out scans. No pack is a blocking gate.
Packs retired on 2026-10-07 are a dated note in [CALIBRATION.md](CALIBRATION.md#retired-packs).

| Layer | Deliverable | Purpose |
| --- | --- | --- |
| 1 | [Native baseline](TOOLING.md) and [check-rust.sh](../../scripts/check-rust.sh) | Rustfmt, compiler, tests/doctests, documentation and Clippy |
| 2, strict | [Rust core](../../packs/rust-core/) | Semantic checks requiring visible evidence and contracts where applicable |
| 3, advisory | [Core advisory](../../packs/rust-core-advisory/) | Broader review without treating inferred contracts as strict failures |
| 3, specialist | [Performance](../../packs/rust-performance/) | Workload-dependent materialization and I/O review |
| 3, separate review | [Readability](../../packs/rust-readability/) | Contextual domain values and Rustdoc accuracy |
| opt-in | [Design](../../packs/design/) | Design-taste rules, including the Rust abstraction-level rule |

Packs are independent, with fixture calibration measured in
[CALIBRATION.md](CALIBRATION.md). Strict describes the evidence required by
`rust-core`, not a claim of production accuracy or automatic blocking enforcement.
No Rust pack is automatically enabled in this repository's configuration. Select advisory and
specialist packs deliberately; their presence does not establish that a project
needs a particular performance policy.

## Rust tooling beyond Jevlint

Unsafe, async, and trait-law defects measured on 2026-10-07 were not caught by a function-scoped model rule. Use these tools. Several lints did not fire on the measured bugs. A clean run is not a pass.

- rustc `invalid_value`, and the `mem::uninitialized` deprecation. They fired on concrete uninit shapes (a `HashMap`, a concrete `u64`, a concrete header struct). They do not fire on generic `mem::uninitialized::<A>()`.
- Clippy `cast_ptr_alignment`. It fired on both minimized alignment casts. One upstream function already had `#[allow(clippy::cast_ptr_alignment)]`, so an allow-list hides it.
- Clippy `non_send_fields_in_send_ty`. It fired when a stored field was `!Send` without the bound. It did not fire on a `*mut T` field (`*mut T` is `Send`).
- Clippy `await_holding_lock`, `await_holding_refcell_ref`, and `let_underscore_future`. `await_holding_lock` fired on a control std `MutexGuard` held across `.await`, not on any of 19 historical async fixes. `await_holding_refcell_ref` had nothing to catch in that set. `let_underscore_future` fired only on a control `let _ = sleep(...)`.
- Clippy `derived_hash_with_manual_eq`, `derive_ord_xor_partial_ord`, and `non_canonical_partial_ord_impl`. They did not fire on 15 semantic trait-law breaks (exit 0). A clean Clippy run does not mean Eq, Hash, and Ord agree.
- Miri, or debug assertions / `cargo careful`, on tests that execute the unsafe path. Most FFI is unsupported. A pass is not soundness. `cargo careful` was not installed for that measurement; one openssl advisory already describes a debug assertion.
- loom for lock and waker protocols, when someone writes a loom test. It was not run on the 19 fixes.
- Small law or property tests, one check per contract: Hash call-sequence versus `eq`; `a == b` iff `partial_cmp == Equal`; sort transitivity; `ExactSizeIterator::len` equals `count`; `new()` field-equals `Default` when both exist; `Borrow` hash equals the owner's hash. On 15 reductions, 15/15 buggy checks failed and the executed fixed twins passed.
- `cargo audit` for known crate bugs. That is how 18 RustSec defects in the unsafe measurement were found.

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

# Compile and score the strict core the way the 2026-10-06 table was recorded.
# The runner's config default is 0.5; pass 0.8 to match shipped reporting.
# Requires the built jevlint binary and its configured provider credentials.
python3 scripts/check-rust-packs.py --eval --repeat 2 --min-fail-probability 0.8 \
  --pack rust-core --jevlint /path/to/jevlint

# Optionally choose the JSON summary destination.
python3 scripts/check-rust-packs.py --eval --repeat 2 --min-fail-probability 0.8 \
  --pack rust-core --jevlint /path/to/jevlint --summary /path/to/rust-core-summary.json
```

After the native baseline, read [CALIBRATION.md](CALIBRATION.md) before
treating a semantic pack as a gate. Packs ship no rule-level `minFailProbability`.
The shipped default is 0.80.

The native script accepts a project directory or `Cargo.toml`. It selects
`--workspace` by default; repeated `-p` options select packages instead.
Compiler checks, tests/doctests, documentation and Clippy retain `--locked`;
`--skip-fmt` skips rustfmt only.

The fixture runner uses a temporary Cargo project with no external crate
dependencies, and the real `jevlint eval --refresh-cache --format json --verbose`
command on every evaluation run. It never substitutes synthetic model answers.
It does not download crates. This is not a fully locked/MSRV/target matrix.

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
`null`. These corpus metrics do not establish production precision. See
[CALIBRATION.md](CALIBRATION.md). Compilation alone establishes neither precision nor recall.

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
