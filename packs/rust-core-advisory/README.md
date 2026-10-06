# Rust core advisory tier

`Bairum/rust-core-advisory` contains six opt-in **info** rules. It infers review-worthy contracts from strong Rust conventions rather than requiring recovery, atomicity, resource-limit or output-completion documentation. Findings are questions for human review, not proof of a documented promise being broken.

## Strict versus advisory

Install `Bairum/rust-core` for evidence-gated warnings about visible contract contradictions. Install this pack when reviewing undocumented Rust APIs, external-input boundaries or legacy I/O code and you want additional recall at the cost of more policy-sensitive findings. Explicit fatal startup, partial progress, redaction, overwrite and best-effort policies remain legitimate. The same undocumented code can pass the strict rule and fail the corresponding advisory rule; paired `no-contract.rs` fixtures demonstrate that distinction. Both packs can be installed, but the advisory tier is not enabled merely by installing the strict tier.

Tests are skipped structurally unless a project deliberately sets `includeTests: true`. This is a new pack: no old fixtures were removed. Examples are ordinary Rust 2021 library files, not test-only units. Source filters identify possible subjects, not violations, and all cases retain a matching subject even when demonstrating an exception.

## Installation

With `"rust": {}` in your project's `languages`, install the committed Git snapshot containing this pack:

```sh
jevlint plugin install --config jevlint.json "$PWD#packs/rust-core-advisory"
jevlint plugin list --config jevlint.json
jevlint check --config jevlint.json --fail-on warning path/to/rust/project
```

Installation explicitly opts in and records the resolved commit SHA. The CLI installs committed snapshots, not this directory's uncommitted edits. No invented release pin is supplied. `--fail-on warning` leaves informational findings visible without failing CI for this tier alone; the default `--fail-on info` does fail on informational findings. Remove with `jevlint plugin remove --config jevlint.json Bairum/rust-core-advisory`.

For real-provider calibration after installing the pack:

```sh
jevlint eval --config jevlint.json --evals packs/rust-core-advisory/jevlint-evals.json --rule rust-advisory-input-panics --verbose
```

Repeat for each rule and inspect false positives and abstentions before using this tier as a CI gate. Measured fixture calibration and optional consuming-project overrides are documented below.

## Rules and source filters

| Rule | Conventional concern | `sourceMatch` rationale |
| --- | --- | --- |
| `rust-advisory-input-panics` | A public `Result` parser should report malformed external input rather than panic. | `pub` keeps qualified/aliased Result and parser APIs eligible; signature and actual parsing are judged semantically. |
| `rust-advisory-error-kind-erasure` | Public I/O errors should retain a useful kind, source or context. | `map_err(` catches qualified and method mapping without guessing the replacement error's name. |
| `rust-advisory-partial-mutation-before-error` | Pre-existing application state remains partly replaced, cleared or persisted after Err; disposable scratch/output and temporary leases are exempt. | `&mut`, including explicit lifetimes, catches aliases of state and Result types; `context.types` supplies bounded type context, not a presumption of atomicity. |
| `rust-advisory-unbounded-read` | External streams should not allocate without an effective bound. | `read_to_end`, `read_to_string` and `lines` cover method and fully qualified trait calls; each method may be safe with visible bounds. |
| `rust-advisory-check-then-create` | Checking absence does not provide exclusive creation. | Existence/metadata checks identify qualified and aliased creation APIs; `create_new` retains atomic-creation pass cases. |
| `rust-advisory-unflushed-bufwriter` | Success from buffered File/TcpStream I/O should check completion errors. | `BufWriter` plus common write/completion methods also retain imported writer aliases. |

The filters deliberately over-include near-neighbors rather than pretending to type-check Rust. Function units omit standalone imports; opaque aliases, foreign methods and unavailable helpers may require abstention. Shared guidance is in `guidance.md`; rule-specific legitimate policies are in `rules.json`. No rule demands a library, ubiquitous transactions, a universal memory cap or fsync. A byte cap alone satisfies the unbounded-read advisory; protocol completeness requires a separate established contract. Source/callee/type context is not a separate finding target.

## Research references

Research IDs map to the [preserved inventory](../../RUST-PACK-RESEARCH.md) and [API](../../docs/rust/research/api.md), [memory](../../docs/rust/research/memory.md), and [concurrency](../../docs/rust/research/concurrency.md) research.

| Rule | Research | Primary convention/API reference |
| --- | --- | --- |
| `rust-advisory-input-panics` | API01/02 | [Rust Book: panic policy](https://doc.rust-lang.org/book/ch09-03-to-panic-or-not-to-panic.html) |
| `rust-advisory-error-kind-erasure` | API03 | [Error::source](https://doc.rust-lang.org/std/error/trait.Error.html#method.source), [io::Error::kind](https://doc.rust-lang.org/std/io/struct.Error.html#method.kind) |
| `rust-advisory-partial-mutation-before-error` | M08 | [Exception safety](https://doc.rust-lang.org/nomicon/exception-safety.html), [Read::read_to_end](https://doc.rust-lang.org/std/io/trait.Read.html#method.read_to_end) |
| `rust-advisory-unbounded-read` | M09/C06 | [Read::take](https://doc.rust-lang.org/std/io/trait.Read.html#method.take), [BufRead::lines](https://doc.rust-lang.org/std/io/trait.BufRead.html#method.lines) |
| `rust-advisory-check-then-create` | C09 | [OpenOptions::create_new](https://doc.rust-lang.org/std/fs/struct.OpenOptions.html#method.create_new) |
| `rust-advisory-unflushed-bufwriter` | M04/C07/API27 | [BufWriter](https://doc.rust-lang.org/std/io/struct.BufWriter.html), [C-DTOR-FAIL](https://rust-lang.github.io/api-guidelines/dependability.html#c-dtor-fail) |

## Strict/advisory fixture pairs

Each advisory `fixtures/<rule>/no-contract.rs` is the byte-identical undocumented body of its strict counterpart, not an approximation. The strict oracle is pass; the advisory oracle is fail. Paths below are relative to `packs/`.

| Advisory rule | Strict fixture directory |
| --- | --- |
| `rust-advisory-input-panics` | `rust-core/fixtures/rust-expected-failure-panics` |
| `rust-advisory-error-kind-erasure` | `rust-core/fixtures/rust-error-cause-loss` |
| `rust-advisory-partial-mutation-before-error` | `rust-core/fixtures/rust-partial-state-on-error` |
| `rust-advisory-unbounded-read` | `rust-core/fixtures/rust-unbounded-input-buffering` |
| `rust-advisory-check-then-create` | `rust-core/fixtures/rust-exclusive-create-race` |
| `rust-advisory-unflushed-bufwriter` | `rust-core/fixtures/rust-buffered-output-completion` |

## Real-world provenance and realistic cases

Each pair below is `fixtures/<rule>/real-<slug>-{bug,fixed}.rs`. Both files contain at least 80 lines of cohesive Rust with several units; the bug file has one violating primary unit. These are derived reproductions, not large verbatim copies. The other fixtures cover distinct failure shapes and legitimate policy exceptions.

| Rule / slug | Public source and project | What was minimized |
| --- | --- | --- |
| `rust-advisory-input-panics` / `findutils-width` | uutils/findutils [issue 732](https://github.com/uutils/findutils/issues/732), [fix PR 734](https://github.com/uutils/findutils/pull/734) | External `-printf` width consisting only of digits still overflows `usize` and reaches `parse().unwrap()`. The stdlib format planner returns a format error in the fixed parser. Both retain a rendering-width bound to avoid a separate resource-limit issue. |
| `rust-advisory-error-kind-erasure` / `burn-config-read` | Burn [public fork report 114](https://github.com/Mikyx-1/burn/issues/114), [upstream implementation](https://raw.githubusercontent.com/tracel-ai/burn/main/crates/burn-core/src/config.rs) | `Config::load` maps every `read_to_string` error to `FileNotFound`, including permissions, directory reads and invalid UTF-8. The reproduction replaces serde with a key/value parser; its derived repair retains `io::Error`, path and `Error::source`. The report is on a public fork, not an upstream acknowledgement or a merged upstream fix. |
| `rust-advisory-partial-mutation-before-error` / `miden-execute` | 0xMiden/rust-sdk [issue 2221](https://github.com/0xMiden/rust-sdk/issues/2221), [fix PR 2222](https://github.com/0xMiden/rust-sdk/pull/2222) | Input notes and scripts persisted before fallible execution become caller-visible mutable maps. The fixed executor borrows a local request-script overlay and commits store updates only after success. Async database and proof-generation details are omitted. |
| `rust-advisory-unbounded-read` / `vector-gelf` | vectordotdev/vector [issue 25577](https://github.com/vectordotdev/vector/issues/25577) | Network-originated decoder output grows through unlimited `read_to_end`; compressed-message bounds do not bound expansion. The stdlib reproduction accepts the already-decoded stream, omits the codec/framing, and the derived repair uses `take(max + 1)` before validation. It is not claimed to be an upstream patch. |
| `rust-advisory-check-then-create` / `cargo-liner-import` | PaulDance/cargo-liner [fix commit 5140ee7](https://github.com/PaulDance/cargo-liner/commit/5140ee7efbb2cfa044f652ebe6e3de8d3a7652b1), [old existence guard](https://raw.githubusercontent.com/PaulDance/cargo-liner/d1ff3a479f9518a874dd2716c298be2013a9823c/src/main.rs) | Configuration import checks absence then saves with `fs::write`; the fix uses `create_new` unless force-overwrite is requested. Guard and save are combined in one primary unit while preserving force policy and package/version configuration. |
| `rust-advisory-unflushed-bufwriter` / `broken-pipe` | rust-lang/rust [issue 37045](https://github.com/rust-lang/rust/issues/37045) | Public reproducer demonstrates `BufWriter` returning success although Drop suppresses BrokenPipe. The telemetry pair explicitly substitutes `TcpStream` for the original `ChildStdin`, within this rule's sink scope, and checks `flush()?` in the fixed sender. This is not an upstream application patch. |

## Verification status

All **68 isolated Rust 2021 fixture targets** were compiled without warnings in the final six-pack fixture compilation (432 targets total), including four retained/disposable-state regressions added below. Faulty fixtures were never executed. Compilation establishes syntax, not judgment correctness. Derived public-bug reproductions are minimized examples, not claims that every upstream incident had this exact public API.

The initial three real provider-backed runs (`python3 scripts/check-rust-packs.py --eval --repeat 3`) measured fixture recall and specificity at global choice-confidence floor 0.8; the floors were not refit. Opaque fixture names were used. This historical table reports per-run decision metrics: below-floor failures are not reported, matching `check` behavior. The partial-mutation row predates the narrowing below; its fitted 0.70 override is withdrawn.

| Rule | Recall @0.8 | Specificity @0.8 | Original fitted override | Recall @override | Specificity @override |
| --- | --- | --- | --- | --- | --- |
| `rust-advisory-input-panics` | 0.80 | 1.00 | — | 0.80 | 1.00 |
| `rust-advisory-error-kind-erasure` | 0.50 | 1.00 | 0.65 | 1.00 | 1.00 |
| `rust-advisory-partial-mutation-before-error` (before narrowing) | 0.73 | 1.00 | 0.70 (withdrawn) | 0.80 | 1.00 |
| `rust-advisory-unbounded-read` | 0.83 | 1.00 | 0.75 | 1.00 | 1.00 |
| `rust-advisory-check-then-create` | 0.67 | 1.00 | 0.30 | 0.87 | 1.00 |
| `rust-advisory-unflushed-bufwriter` | 0.80 | 1.00 | 0.65 | 1.00 | 1.00 |

All six Rust packs retain the global floor of **0.8** and contain no rule-level `minFailProbability`; no pack-level floor changes were made. Only consuming projects may opt into the recommended overrides through project rule overlays such as `{"id": "rust-advisory-error-kind-erasure", "minFailProbability": 0.65}`. An overlay replaces the rule floor, which overrides the global floor; `—` means keep 0.8. Those recorded floors were fitted on the old choice-confidence metric and pack calibration will refit them; the field name is `minFailProbability`. Overrides were fitted on these same fixtures and are starting points, not guarantees. The measured fixture specificity of 1.00 does not establish real-project precision, which remains unmeasured. See [calibration method, caveats and override configuration](../../docs/rust/CALIBRATION.md).

### Partial-mutation decision and provider refresh (2026-10-05)

**Retain and narrow, rather than remove.** A mutable reference and an earlier append do not imply a transaction. The rule now distinguishes pre-existing caller application state from scratch/output buffers or temporary leases whose failed contents are discarded, reset or never published. A finding must identify retained-state observation; it must not invent a caller consuming failed output. Explicit atomicity documentation is still unnecessary for clearing or replacing pre-existing application state.

Before editing, three uncached provider runs compared the old advisory and strict `rust-partial-state-on-error` on the same 11 advisory fixtures, with each pack's shared guidance, opaque paths and global floor 0.8. Both rules used the advisory defect labels to measure added recall, not to assert that the strict rule should diagnose undocumented contracts.

| Rule on original advisory partial-mutation fixtures | Reported defects / fail decisions | Recall @0.8 | Reported false positives / pass decisions | Specificity @0.8 |
| --- | --- | --- | --- | --- |
| Old advisory | 11/15 | 0.7333 | 0/18 | 1.00 |
| Strict core | 0/15 | 0.00 | 0/18 | 1.00 |

The undocumented `no-contract.rs` was reported in 2/3 advisory runs and 0/3 strict runs. Endpoint replacement, payload replacement and the retained-store reproduction were each reported in 3/3 advisory runs and 0/3 strict runs. Thus removing the advisory would lose measured additional signal. The price-map case was missed by both rules in all three runs. Old advisory had one below-floor unit; strict had 12. No provider errors or abstentions occurred.

The final full-pack refresh ran all **68 cases three times**, uncached, using the same guidance and global floor. Per-run `check` reporting metrics are aggregated over all three runs:

| Rule | Reported defects / fail decisions | Recall @0.8 | Reported false positives / pass decisions | Specificity @0.8 |
| --- | --- | --- | --- | --- |
| `rust-advisory-input-panics` | 12/15 | 0.80 | 0/18 | 1.00 |
| `rust-advisory-error-kind-erasure` | 6/12 | 0.50 | 0/18 | 1.00 |
| `rust-advisory-partial-mutation-before-error` | 15/18 | 0.8333 | 0/27 | 1.00 |
| `rust-advisory-unbounded-read` | 12/12 | 1.00 | 0/15 | 1.00 |
| `rust-advisory-check-then-create` | 12/15 | 0.80 | 0/15 | 1.00 |
| `rust-advisory-unflushed-bufwriter` | 14/15 | 0.9333 | 0/24 | 1.00 |

The three new clean scratch/output/lease cases passed every run, with **zero failing units, including below-floor units**. The new caller-visible reuse case failed every run at 0.96; `no-contract.rs` also failed every run at 0.87/0.88/0.81. Partial-mutation majority recall/specificity was **5/6 and 9/9**, with no flips or abstentions; the price-map case remained below floor in all three runs. Its six below-floor units comprise those three misses and three additional decisions on the genuine reuse caller.

For the full pack, runs returned matched/mismatched/inconclusive counts of **60/1/7**, **61/1/6** and **61/2/5**. Across 204 case decisions there were four mismatches, 18 inconclusive cases, zero abstentions, 22 below-floor units and zero provider errors; two of 68 cases flipped (2.94%). All commands exited 1 because the corpus still has mismatches/inconclusive results, not because provider execution failed. Reporting specificity above means no reported clean-case findings; unlike majority-pass specificity, it does not require a conclusive pass from every unit.

Reproduction uses an external config containing `languages.rust`, floor 0.8 and the pack's rules with `guidance.md` injected, plus external copies of fixtures named `case-NNN.rs`. Repeat each command three times; the comparison config freezes the old advisory and strict rule and the comparison evals pair each original fixture under both IDs:

```sh
jevlint eval --config <config> --evals <evals> --refresh-cache --format json --verbose
```

Keep the partial-mutation floor at 0.8; the old 0.70 recommendation is not recalibrated for this narrower rule. Those recorded floors were fitted on the old choice-confidence metric and pack calibration will refit them; the field name is `minFailProbability`.

On a held-out private Rust workspace (43 files, 958 units), three uncached combined core/advisory scans reported
**zero findings in every run**. Partial-mutation advisory noise fell from the
recorded **1 reported + 28 below-floor** findings to **0 reported + 7/7/6
below-floor** findings. Total core/advisory below-floor counts fell from 41 to
22/22/23. This reduces, but does not eliminate, real-project noise.

On the seeded atomicity violation, full-workspace scans show core still reports the atomicity violation 3/3.
The narrowed advisory co-fails below floor at 0.71/0.73/0.72, so it adds no seed
recall. Its retention is supported by the additional undocumented-state fixture
signal above, not that seed. Full commands and scan/cost evidence are recorded
in [CALIBRATION.md](../../docs/rust/CALIBRATION.md#core-and-advisory-refresh-2026-10-05).
