# Rust readability review

`Bairum/rust-readability` is an opt-in, separate maintainability layer with two **warning** rules. It ports only the upstream generic rules that retained useful measured behavior after a Rust workspace audit. It is not a replacement for rustc, rustfmt, Clippy, the strict core, or human review.

## Installation

Enable `"rust": {}` in the consuming project's `languages`, then install the committed snapshot:

```sh
jevlint plugin install --config jevlint.json "$PWD#packs/rust-readability"
jevlint plugin list --config jevlint.json
jevlint check --config jevlint.json --fail-on warning path/to/rust/project
```

Installation reads committed content and records its resolved commit SHA, not uncommitted files. Remove with `jevlint plugin remove --config jevlint.json Bairum/rust-readability`.

## Rules and evidence

| Rule | Severity | Scope |
| --- | --- | --- |
| `rust-readability-magic-domain-values` | warning | Unexplained tunable numerical thresholds, floors, bounds, byte allowances, numeric ordering and sentinel ranks in the primary function. Mathematical coefficients, adjacent declared wire/storage layouts or diagnostic encodings, 0/1 flags and unit conversions pass. |
| `rust-readability-accurate-doc-comments` | warning | Concrete attached Rustdoc behavior and `# Errors`/`# Panics`/`# Safety` contradictions. Missing documentation/headings alone pass. |

Each rule asks a 3-level score plus subject-gated violation checks. Fail probability is the minimum of those signals. `rust-readability-magic-domain-values` scores an unnamed tunable literal; the subject is a numeric literal. `rust-readability-accurate-doc-comments` scores a Rustdoc contradiction; the subject is attached Rustdoc.

Both target Rust function units and retain cost-control `sourceMatch` filters. Literal policy selects digit-bearing literal candidates; literal-free enum/constant repairs are skipped, not model judgments. Rustdoc selects attached `///`, `//!`, block-doc and `#[doc]` forms; undocumented units are skipped. A signal is not a violation. The parser attaches preceding Rustdoc to function source, keeping a multiline contract and body together. Both rules use bounded callee and type context. Context is not another finding target. Tests are structurally skipped unless the project opts into `includeTests: true`.

Newtypes, trait implementations, enums, `?`, exhaustive matches, focused parsers/state machines and intentional performance layouts remain normal Rust. A readability failure is not proof of a runtime defect; a passing judgment is not a soundness certificate. This pack sets no rule-level floor. The shipped default is 0.80.

## Fixtures and calibration

The **22 isolated Rust 2021 library fixtures** use general numerical fitting, ranking, storage and registry examples, not audited-project identifiers or code. **20 cases** receive model evaluation; the two literal-free enum/constant repairs remain compile-only pass twins because the literal prefilter skips them. Cohesive multi-unit solver and registry twins and mathematical/layout/diagnostic/0–1/conversion negatives remain. Rustdoc twins cover Errors, Panics, Safety and result claims; the formerly undocumented pass now states accurate basic contracts so the documentation prefilter has applicable units.

From this checkout, with Jevlint and Cargo on `PATH` and provider credentials configured:

```sh
python3 scripts/check-rust-packs.py --eval --repeat 3 --pack rust-readability \
  --jevlint jevlint --summary readability-summary.json
```

Fixture rates at 0.80 are in [CALIBRATION.md](../../docs/rust/CALIBRATION.md). Do not infer production precision from these fixtures.

On a held-out private Rust workspace (43 files), two runs at 0.80 produced 7 reports each, 8 distinct. `rust-readability-magic-domain-values` had 1 report, judged actionable (an unnamed solver tolerance and parameter limits). The other 7 came from `rust-readability-mixed-levels-of-abstraction`, then part of this pack: 1 actionable and 6 false positives on FFI adapter functions. That rule moved to the opt-in [design pack](../design/) on 2026-10-07. This scan was not used to change either rule.

[serde_json `afdf6fc`](https://github.com/serde-rs/json/tree/afdf6fc67247dd7fa4fcde1381e6ecc6bcc7a30e) (38 production files): 0 reports.

## Deliberate exclusions

The [upstream coverage ledger](../../docs/rust/COVERAGE.md#rust-readability-upstream-generic-rule-accounting) accounts for all 26 upstream rules. A broader control-flow port recovered the two known useful findings but mostly reported focused algorithms and FFI lifecycle glue (5 actionable, 1 nit, 23 false positives out of 29); it was removed, not left enabled with caveats. Six additional candidates were evaluated on fail/pass Rust fixtures and the workspace, then removed for poor recall or no demonstrated actionable project value. Rust newtypes and trait adapters were explicitly legitimate wrapper negatives, not penalized abstractions.

Stringly typed control-flow and comment-quality had no actionable reports in the rule-selection audit; database joins is outside readability. Dropping comment-quality can miss a useful comment finding. Clippy owns redundant Boolean branching and optional numerical line/Boolean/complexity heuristics; Rustdoc completeness lints do not establish prose accuracy. Consult the [native baseline](../../docs/rust/TOOLING.md) before model review.

Neither small fixture denominators nor the held-out scan establish accuracy for new projects or justify automatic blocking adoption; retain human review.
