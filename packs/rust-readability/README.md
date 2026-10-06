# Rust readability review

`Bairum/rust-readability` is an opt-in, separate maintainability layer with two **warning** rules and one **info** rule. It ports only the upstream generic rules that retained useful measured behavior after a Rust workspace audit. It is not a replacement for rustc, rustfmt, Clippy, the strict core, or human review.

## Installation

Enable `"rust": {}` in the consuming project's `languages`, then install the committed snapshot:

```sh
jevlint plugin install --config jevlint.json "$PWD#packs/rust-readability"
jevlint plugin list --config jevlint.json
jevlint check --config jevlint.json --fail-on warning path/to/rust/project
```

Installation reads committed content and records its resolved commit SHA, not uncommitted files. `--fail-on warning` leaves the abstraction-level informational findings visible without making those alone fail CI; the default `--fail-on info` does fail on informational findings. Remove with `jevlint plugin remove --config jevlint.json Bairum/rust-readability`.

## Rules and evidence

| Rule | Severity | Scope |
| --- | --- | --- |
| `rust-readability-magic-domain-values` | warning | Unexplained tunable numerical thresholds, floors, bounds, byte allowances, numeric ordering and sentinel ranks in the primary function. Mathematical coefficients, adjacent declared wire/storage layouts or diagnostic encodings, 0/1 flags and unit conversions pass. |
| `rust-readability-mixed-levels-of-abstraction` | info | Substantial coherent codec/layout implementation interleaved with high-level orchestration, not incidental arithmetic or ordinary glue. |
| `rust-readability-accurate-doc-comments` | warning | Concrete attached Rustdoc behavior and `# Errors`/`# Panics`/`# Safety` contradictions. Missing documentation/headings alone pass. |

All three target Rust function units and retain cost-control `sourceMatch` filters. Literal policy selects digit-bearing literal candidates; literal-free enum/constant repairs are skipped, not model judgments. Rustdoc selects attached `///`, `//!`, block-doc and `#[doc]` forms; undocumented units are skipped. The abstraction filter conservatively selects control flow, local construction/mutation, scalar/bit/array operations and byte/character codec signals. It deliberately over-includes references and attributes rather than guessing that only one encoding idiom matters. A signal is not a violation. The parser attaches preceding Rustdoc to function source, keeping a multiline contract and body together. All rules use bounded callee context; literal policy and Rustdoc also request type context. Context is not another finding target. Tests are structurally skipped unless the project opts into `includeTests: true`.

Newtypes, trait implementations, enums, `?`, exhaustive matches, focused parsers/state machines and intentional performance layouts remain normal Rust. A readability failure is not proof of a runtime defect; a passing judgment is not a soundness certificate. No rule-level `minFailProbability` is shipped: retain the global floor of **0.8**. Those recorded floors were fitted on the old choice-confidence metric and pack calibration will refit them; the field name is `minFailProbability`.

## Fixtures and calibration

The **25 isolated Rust 2021 library fixtures** use general numerical fitting, ranking, storage/transport and telemetry/registry examples, not audited-project identifiers or code. **23 cases** receive model evaluation; the two literal-free enum/constant repairs remain compile-only pass twins because the literal prefilter skips them. Cohesive multi-unit solver, encoding and registry twins and mathematical/layout/diagnostic/0–1/conversion negatives remain. Rustdoc twins cover Errors, Panics, Safety and result claims; the formerly undocumented pass now states accurate basic contracts so the documentation prefilter has applicable units.

From this checkout, with Jevlint and Cargo on `PATH` and provider credentials configured:

```sh
python3 scripts/check-rust-packs.py --eval --repeat 3 --pack rust-readability \
  --jevlint jevlint --summary readability-summary.json
```

Three uncached real-provider runs at choice-confidence floor 0.8 (floors were not refit) measured majority recall **9/10** and specificity **13/13**, with **one flipped case**, **zero mismatches**, **two inconclusive case-runs**, **zero abstain decisions** and **four below-floor fail decisions**. The unexplained byte-budget failure was reported at **0.99 in all three runs**, alongside the numerical policy, numeric ordering and sentinel failures. The command still exits 1 because the registry-persistence case is majority inconclusive and flips; compilation succeeded and there were no evaluation errors. Do not infer perfect recall from compilable fixtures.

| Rule | Majority fixture recall | Majority fixture specificity |
| --- | --- | --- |
| `rust-readability-magic-domain-values` | 4/4 | 5/5 |
| `rust-readability-mixed-levels-of-abstraction` | 1/1 | 2/2 |
| `rust-readability-accurate-doc-comments` | 4/5 | 6/6 |

The [calibration record](../../docs/rust/CALIBRATION.md#rust-readability) reports three shipped runs on a held-out private Rust workspace (43 files, 958 units). Its triage shaped the rules, making this **tuning-set/in-sample** evidence: each run reported **17 findings (10 actionable, 0 nits, 7 false positives)**, an actionable share of **58.8% versus the upstream 22.6%**. All runs retained **6/12** original actionable findings and **1/35** original false positives; union and majority are identical.

The genuinely held-out public comparison pinned [serde_json at `afdf6fc67247dd7fa4fcde1381e6ecc6bcc7a30e`](https://github.com/serde-rs/json/tree/afdf6fc67247dd7fa4fcde1381e6ecc6bcc7a30e), scanning only **`src` plus `build.rs` (38 production Rust files)** in both configurations, not all 71 tracked Rust files. Three shipped runs each reported the same **2 mathematical false positives and no actionable findings (0%)**; the 26-rule upstream Rust-only baseline reported **41 findings: 1 actionable, 18 nits, 22 false positives (2.4%)**. There was no report/actionable overlap. The pack removed bulk noise but missed a useful incorrect subtraction comment; this is **not a held-out accuracy gain**.

Descriptions, exceptions and guidance froze before public inspection. Only authorized pure cost prefilters followed the first public scan, reducing evaluations **1821 → 1163 (36.1%)** privately and **3159 → 1489 (52.9%)** publicly; these are not complexity detectors. The calibration record includes portable public reproduction, independent review of all 43 public report items, the unsuccessful initial nom corpus attempt and native-check evidence. Its historical single seeded Rustdoc scan reported S11 before the cost prefilters; public seed recall remains unmeasured.

## Deliberate exclusions

The [upstream coverage ledger](../../docs/rust/COVERAGE.md#rust-readability-upstream-generic-rule-accounting) accounts for all 26 upstream rules. A broader control-flow port recovered the two known useful findings but mostly reported focused algorithms and FFI lifecycle glue (5 actionable, 1 nit, 23 false positives out of 29); it was removed, not left enabled with caveats. Six additional candidates were evaluated on fail/pass Rust fixtures and the workspace, then removed for poor recall or no demonstrated actionable project value. Rust newtypes and trait adapters were explicitly legitimate wrapper negatives, not penalized abstractions.

Stringly typed control-flow and comment-quality had no actionable reports in the private baseline; database joins is outside readability. The public comparison above nevertheless found one useful upstream comment-quality report that this pack misses. Clippy owns redundant Boolean branching and optional numerical line/Boolean/complexity heuristics; Rustdoc completeness lints do not establish prose accuracy. Consult the [native baseline](../../docs/rust/TOOLING.md) before model review.

Neither small fixture denominators nor the private tuning-set gain establish accuracy for new projects or justify automatic blocking adoption; retain human review.
