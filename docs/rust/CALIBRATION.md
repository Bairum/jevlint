# Rust pack calibration (baseline, leak-free)

Source: a local `--summary` JSON from the run below (1.4 MB, not committed; regenerate with the reproduction command).
Method: `python3 scripts/check-rust-packs.py --eval --repeat 3` at global minConfidence 0.8; evaluator saw opaque fixture names; 426 cases, 1278 case-runs.
Case-level majority (eval semantics, below-floor fail = inconclusive): recall 71/162, specificity 229/264; flip rate 0.054 (23 cases); abstain decisions 4; below-floor fail decisions 264.
Per-run decision metrics below treat below-floor failures as not reported (what `check` users see).
Recommended override = floor in 0.30..0.80 (step 0.05) maximizing per-run recall while keeping per-run specificity >= 0.90 (ties keep the higher floor; "—" = keep 0.8). Fitted on the same fixtures: a starting point, not a guarantee.

| Pack | Rule | Recall @0.8 | Specificity @0.8 | Recommended override | Recall @override | Specificity @override |
|---|---|---|---|---|---|---|
| rust-core | `rust-expected-failure-panics` | 0.80 | 0.89 | — | 0.80 | 0.89 |
| rust-core | `rust-error-cause-loss` | 0.53 | 0.89 | — | 0.53 | 0.89 |
| rust-core | `rust-conversion-contract` | 0.80 | 0.83 | — | 0.80 | 0.83 |
| rust-core | `rust-validation-bypass` | 0.00 | 1.00 | 0.40 | 0.87 | 0.95 |
| rust-core | `rust-partial-state-on-error` | 0.60 | 1.00 | 0.45 | 1.00 | 1.00 |
| rust-core | `rust-buffered-output-completion` | 0.83 | 0.91 | — | 0.83 | 0.91 |
| rust-core | `rust-fallible-drop` | 1.00 | 1.00 | — | 1.00 | 1.00 |
| rust-core | `rust-unbounded-input-buffering` | 0.80 | 0.88 | — | 0.80 | 0.88 |
| rust-core | `rust-exclusive-create-race` | 1.00 | 0.71 | — | 1.00 | 0.71 |
| rust-core | `rust-shared-clone-isolation` | 0.27 | 0.88 | — | 0.27 | 0.88 |
| rust-core-advisory | `rust-advisory-input-panics` | 0.80 | 1.00 | — | 0.80 | 1.00 |
| rust-core-advisory | `rust-advisory-error-kind-erasure` | 0.50 | 1.00 | 0.65 | 1.00 | 1.00 |
| rust-core-advisory | `rust-advisory-partial-mutation-before-error` | 0.73 | 1.00 | 0.70 | 0.80 | 1.00 |
| rust-core-advisory | `rust-advisory-unbounded-read` | 0.83 | 1.00 | 0.75 | 1.00 | 1.00 |
| rust-core-advisory | `rust-advisory-check-then-create` | 0.67 | 1.00 | 0.30 | 0.87 | 1.00 |
| rust-core-advisory | `rust-advisory-unflushed-bufwriter` | 0.80 | 1.00 | 0.65 | 1.00 | 1.00 |
| rust-tokio | `rust-tokio-blocking-boundary` | 0.33 | 1.00 | 0.40 | 0.80 | 1.00 |
| rust-tokio | `rust-tokio-cancellation-progress` | 0.00 | 1.00 | 0.50 | 0.17 | 1.00 |
| rust-tokio | `rust-tokio-required-task-ownership` | 0.00 | 1.00 | 0.35 | 0.07 | 1.00 |
| rust-tokio | `rust-tokio-admission-bound` | 0.40 | 1.00 | 0.60 | 0.60 | 1.00 |
| rust-tokio | `rust-tokio-shutdown-contract` | 0.20 | 1.00 | 0.60 | 0.60 | 1.00 |
| rust-tokio | `rust-tokio-lock-dependency` | 0.00 | 1.00 | 0.40 | 0.33 | 1.00 |
| rust-tokio | `rust-tokio-blocking-cancellation` | 0.00 | 1.00 | 0.45 | 1.00 | 0.95 |
| rust-api | `rust-api-default-consistency` | 0.00 | 1.00 | 0.30 | 0.67 | 1.00 |
| rust-api | `rust-api-trait-law-consistency` | 0.07 | 1.00 | 0.60 | 0.40 | 1.00 |
| rust-api | `rust-api-deref-contract` | 0.60 | 0.83 | — | 0.60 | 0.83 |
| rust-unsafe | `rust-unsafe-precondition-contract` | 0.00 | 1.00 | 0.35 | 0.20 | 1.00 |
| rust-unsafe | `rust-unsafe-unwind-state` | 0.00 | 1.00 | 0.65 | 0.07 | 1.00 |
| rust-unsafe | `rust-unsafe-ffi-pointer-contract` | 0.24 | 1.00 | 0.75 | 0.43 | 1.00 |
| rust-unsafe | `rust-unsafe-pyo3-reference-ownership` | 0.17 | 1.00 | 0.35 | 0.61 | 0.91 |
| rust-performance | `rust-perf-repeated-materialization` | 0.13 | 1.00 | 0.45 | 0.80 | 1.00 |
| rust-performance | `rust-perf-small-writes` | 1.00 | 1.00 | — | 1.00 | 1.00 |

The **rust-unsafe rows retain the original 62-case calibration baseline** under the restored original rules and guidance. Six additional multi-unit defect/pass cases are retained as limitation regressions and measured separately; they do not replace these baseline rows or the suite-wide totals above.

Three uncached targeted evaluations of the **six added cases under the restored original rules** returned pass for every case in every run: defect recall **0/3**, pass-twin specificity **3/3**, both per run and by majority. Each run had three matched and three mismatched cases, zero reported or below-floor failures, and exit status 1 for the missed defects. Across 18 case-runs there were nine mismatches, zero inconclusive outcomes, zero flips, and zero errors. Defect pass confidences were 0.48/0.52/0.38 (extent), 0.54/0.55/0.53 (C-string owner lifetime), and 0.85/0.84/0.85 (Python reference ownership); these are pass confidences, not below-floor findings. The [separate original-rule six-case table](../../packs/rust-unsafe/README.md#original-rule-measurements-six-limitation-cases) includes the pass twins. This targeted measurement is **not a full 68-case recalibration**.

The generalized wording evaluated on 2026-10-05 was **rejected**, not shipped as a recall improvement. Its 68-case, 204-case-run experiment (`--eval --repeat 3 --pack rust-unsafe`) measured per-run recall at 0.8 of 3/18 (0.17), 0/15 (0.00), 3/24 (0.13), and 6/21 (0.29) for preconditions, unwind, FFI, and PyO3 respectively; specificity was 1.00 for every rule. FFI recall regressed from the recorded 0.24 baseline to 0.13, so the no-recall-regression target failed. Experiment-only fitted floors/recall were 0.30/0.61, 0.60/0.07, 0.30/0.29, and 0.65/0.57 respectively; specificity at those fitted floors was 1.00. These are rejected experiment results, not current calibration rows or override recommendations.

The rejected experiment had six flips, 28 mismatches, 60 inconclusive case-runs, seven abstain decisions, 53 below-floor fail decisions, and no evaluation errors. Its larger extent defect remained below-floor (0.56/0.56/0.48), while the lifetime and Python-reference defects passed all three runs; every pass twin passed all three runs. The [miss diagnosis and ablations](../../packs/rust-unsafe/README.md#real-project-miss-diagnosis-2026-10-05) distinguish this from a seed-specific draft that reported three in-sample defects in three runs. In-sample success is not blind recall evidence, and the provider returned no reasoning that would establish why these misses occurred.

For eight independent blind defects in a held-out Rust workspace (43 files), three uncached full-workspace checks of the frozen generalized wording reported **0/8 each (majority 0/8)**; one original-rule baseline check also reported **0/8**. Raw failed decisions on intended rules fell from 3/8 in the original baseline to 2/8 in each generalized run: a foreign-output ordering defect regressed from below-floor failure to pass, and a retained-storage wrapper was not evaluated by the FFI source filter in either version. Every scan had zero reported or below-floor findings on non-seed units. The [class-only held-out table](../../packs/rust-unsafe/README.md#rejected-generalized-wording-blind-defects-2026-10-05) preserves exact statuses/confidences and extra cross-rule below-floor items. Three separate generalized clean-workspace checks also had **zero reported and zero below-floor findings each**. Neither clean results nor the blind comparison establishes improved unsafe recall.

**Do not rely on this pack for unsafe recall.** Retain manual contract/ownership review and the existing [project-owned tooling lanes](TOOLING.md#separate-optional-layers), with [tooling research limits](research/tooling.md#5-optional-established-tool-layers): targeted supported Miri tests (most FFI is unsupported; only exercised executions are checked), supported ASan/LSan lanes with coordinated native/FFI instrumentation, debug-assertion builds exercising raw-pointer extent/lifetime paths, and explicit Python reference-count assertions for leaks or double decrefs. These are recommendations, not checks executed here or additions to the native baseline script; none proves soundness.

## Core and advisory refresh (2026-10-05)

The table above is the original six-pack baseline. The current core/advisory
refresh used three uncached real-provider runs at the unchanged global floor
0.8, with shared pack guidance and opaque fixture names. Core has 140 cases;
Advisory has 68. Core's final error-cause row replaces its preliminary samples
with three targeted runs after clarifying direct error-category mapping; the
other core rows retain the full-pack runs. Advisory evaluated its entire final
pack three times. These are per-run reporting metrics, not majority-pass
specificity; below-floor failures remain unreported.

| Pack | Rule | Reported defects / fail decisions | Recall @0.8 | Unreported clean cases / pass decisions | Specificity @0.8 |
| --- | --- | --- | --- | --- | --- |
| rust-core | `rust-expected-failure-panics` | 12/15 | 0.80 | 24/27 | 0.89 |
| rust-core | `rust-error-cause-loss` | 9/18 | 0.50 | 27/30 | 0.90 |
| rust-core | `rust-conversion-contract` | 12/15 | 0.80 | 26/30 | 0.87 |
| rust-core | `rust-validation-bypass` | 0/15 | 0.00 | 21/21 | 1.00 |
| rust-core | `rust-partial-state-on-error` | 9/15 | 0.60 | 30/30 | 1.00 |
| rust-core | `rust-buffered-output-completion` | 15/18 | 0.83 | 30/33 | 0.91 |
| rust-core | `rust-fallible-drop` | 14/15 | 0.93 | 24/24 | 1.00 |
| rust-core | `rust-unbounded-input-buffering` | 12/15 | 0.80 | 21/24 | 0.88 |
| rust-core | `rust-exclusive-create-race` | 15/15 | 1.00 | 15/21 | 0.71 |
| rust-core | `rust-shared-clone-isolation` | 3/15 | 0.20 | 21/24 | 0.88 |
| rust-core-advisory | `rust-advisory-input-panics` | 12/15 | 0.80 | 18/18 | 1.00 |
| rust-core-advisory | `rust-advisory-error-kind-erasure` | 6/12 | 0.50 | 18/18 | 1.00 |
| rust-core-advisory | `rust-advisory-partial-mutation-before-error` | 15/18 | 0.83 | 27/27 | 1.00 |
| rust-core-advisory | `rust-advisory-unbounded-read` | 12/12 | 1.00 | 15/15 | 1.00 |
| rust-core-advisory | `rust-advisory-check-then-create` | 12/15 | 0.80 | 15/15 | 1.00 |
| rust-core-advisory | `rust-advisory-unflushed-bufwriter` | 14/15 | 0.93 | 24/24 | 1.00 |

**Not a no-regression result.** Advisory reporting recall/specificity did not
regress. Core cause-loss recall is 0.50 versus baseline 0.53 on the expanded
corpus; on the original five failing cases it is 6/15 (0.40). Its new collapsed
error mapper failed 3/3 at 0.86/0.89/0.89 and the faithful mapper passed 3/3.
Unchanged core fallible-drop and shared-clone recall measured 0.93 and 0.20
versus 1.00 and 0.27. Those rules were not modified to fit these samples.
Core recorded 37 mismatches, 70 inconclusive cases, 92 below-floor units, no
abstentions/provider errors and four flipped cases (2.86%); Advisory recorded
four mismatches, 18 inconclusive cases, 22 below-floor units, no
abstentions/provider errors and two flipped cases (2.94%). Calibration commands
therefore did not pass the runner's zero-mismatch criterion.

The baseline override recommendations above were not refitted. In particular,
the old partial-mutation 0.70 override is withdrawn after narrowing: keep 0.8.
Neither pack ships a rule-level floor.

### Held-out private workspace measurements

On a held-out private Rust workspace (43 files, 958 units), broadening
the error-cause applicability filter selects **357 → 381** non-test function
units (+24, +6.72%). A one-rule uncached scan costs **357 → 381 model requests**
and took 14.002 → 15.010 seconds; the CLI does not expose token or dollar cost.
This counts actual units passing the source filter, not occurrences of error
tokens. The final combined core/advisory scans each performed 2,526 rule-unit
evaluations batched into 1,278 uncached model requests.

| Clean workspace | Reported | Below floor, all core/advisory | Partial-mutation advisory below floor | Error-cause below floor |
| --- | --- | --- | --- | --- |
| Original recorded all-pack scan, core/advisory subset | 1 | 41 | 28 | 11 |
| Final run 1 | 0 | 22 | 7 | 14 |
| Final run 2 | 0 | 22 | 7 | 14 |
| Final run 3 | 0 | 23 | 6 | 15 |

No new reported finding appeared. The original reported mutation finding is
gone; residual below-floor review noise remains. Do not equate the fixture
specificity with production precision.

The seeded error-category mapper is now selected in the seeded workspace. Three **full-workspace** uncached
scans with error-cause and both mutation rules evaluated 384 error-cause units
per run, but yielded **0/3 seeded error-category mapper reports**: pass/pass/fail, with the failure at
**0.34**. `check` does not expose confidence for passes. These runs discovered
all 43 files (965 units) for bounded cross-file context and each used 847
rule-unit evaluations / 415 model requests. Earlier targeted-file scans judged
the seeded error-category mapper fail 3/3 at **0.60/0.62/0.64**; filter-only wording measured
0.30/0.30/0.39. A targeted file changes the discovered declaration/context set,
so it is not the final held-out result. Jevlint's provider protocol supplies
choice/confidence only; no textual confidence explanation is available.
Recognition improved on the isolated boundary but remains context-sensitive
and fails the majority reporting target. No confidence floor was lowered.

In the full seeded scans the seeded atomicity violation's core rule reports 3/3; the narrowed advisory only
co-fails below floor at **0.71/0.73/0.72**, adding no held-out seed detection.
Removal was rejected because a controlled comparison on the original 11
advisory mutation fixtures measured 11/15 reported defects for advisory versus
0/15 for strict, with no reported clean-case findings from either rule.

Reproduction (configs are written inside each detached project's
workspace; copy both packs' rule arrays and inherit each rule's
`guidance.md`, matching installation):

```sh
python3 scripts/check-rust-packs.py --eval --repeat 3 --pack rust-core \
  --jevlint <binary> --summary <summary>
# After the final direct-mapping wording change, repeat three times:
jevlint eval --config <config> --evals <evals> --refresh-cache --format json --verbose
# Repeat three times from the project root. Clean: both full packs.
# Seeded: error-cause and both mutation rules; omit paths to retain discovery.
jevlint check --config <config> --refresh-cache --format json --show-below-floor
# Earlier targeted seeded error-category mapper comparison, not the final full-workspace evidence:
jevlint check --config <config> --refresh-cache --format json --show-below-floor <targeted-file>
```

Advisory comparison/final eval commands and opaque-fixture copying are recorded
in its [README](../../packs/rust-core-advisory/README.md#partial-mutation-decision-and-provider-refresh-2026-10-05).

## Reproduce

From the repository checkout, with Cargo, Python 3.9+, a built Jevlint binary and its real provider credentials configured:

```sh
python3 scripts/check-rust-packs.py --eval --repeat 3 --jevlint <binary>
```

The runner compiles fixtures and performs three uncached real-provider evaluation runs. Faulty fixtures are never executed. See the [native-first workflow](README.md#run-the-checks) for dependencies, summary output and support-matrix limits. The raw per-run summary is written to the `--summary` path and is not committed.

## Apply a project override

Keep the global floor at **0.8**. No rule-level `minConfidence` is shipped in the packs. To opt into a recommended override, merge an entry by rule ID into the **consuming project's** `jevlint.json`, preserving its installed `packs`, languages, provider settings and other rules:

```json
{
  "minConfidence": 0.8,
  "rules": [
    {"id": "rust-validation-bypass", "minConfidence": 0.40},
    {"id": "rust-partial-state-on-error", "minConfidence": 0.45}
  ]
}
```

Project overlays replace the pack rule's `minConfidence`; a rule floor **overrides**, rather than being clamped by, the global floor. The snippet is an overlay example, not a complete standalone configuration. Recommendations are optional per-rule starting points for project review, not pack defaults. “—” means keep 0.8; it does not mean that the rule meets the 0.90 specificity target. Lowering floors can expose more findings and false positives; a confidence score is not a measured probability that a finding is correct.

## Caveats and adoption status

- These **fixtures double as the calibration set**. The overrides were fitted on them, with no held-out set; fixture recall and specificity do not establish real-project precision, which remains **unmeasured**.
- Only **3 runs** were recorded. The case flip rate was **5.4%** (23 cases; the summary records 0.054). Outcomes and useful floors can change with the model, provider, prompt or project context.
- Since this change, the evaluator sees **opaque fixture names**. The earlier **42/42** result was inflated by fixture comments and verdict-bearing file names that reached the model; it is not comparable evidence of leak-free accuracy.
- The majority metrics above use eval semantics, including inconclusive below-floor failures. The table uses per-run reporting semantics, so these are different aggregates. Specificity is the clean-fixture pass rate, not precision among reported findings.
- Compilation establishes syntax and dependency compatibility, not correctness or soundness. No pack is automatically enabled; retain native compiler/Clippy checks and human review before blocking enforcement.

The following rules are **not yet useful** at their recommended override because recall remains low (for “—”, the retained 0.8 floor is shown):

| Rule | Recall @override |
|---|---|
| `rust-tokio-cancellation-progress` | 0.17 |
| `rust-tokio-required-task-ownership` | 0.07 |
| `rust-tokio-lock-dependency` | 0.33 |
| `rust-unsafe-precondition-contract` | 0.20 |
| `rust-unsafe-unwind-state` | 0.07 |
| `rust-shared-clone-isolation` | 0.27 |

The following rules are **false-positive-prone** on these fixtures: specificity is below 0.90 at the retained global 0.8 floor. No recommended lower floor repairs that tradeoff in the measured search.

| Rule | Specificity @0.8 |
|---|---|
| `rust-exclusive-create-race` | 0.71 |
| `rust-conversion-contract` | 0.83 |
| `rust-unbounded-input-buffering` | 0.88 |
| `rust-expected-failure-panics` | 0.89 |
| `rust-error-cause-loss` | 0.89 |
| `rust-shared-clone-isolation` | 0.88 |
| `rust-api-deref-contract` | 0.83 |

## Rust readability

This is a separate measurement of the new `Bairum/rust-readability` pack,
not a replacement for the original six-pack baseline above. Three rules
ship: `rust-readability-magic-domain-values` (warning),
`rust-readability-mixed-levels-of-abstraction` (info), and
`rust-readability-accurate-doc-comments` (warning). All retain the global
**0.8** floor, with no rule-level overrides. The
[coverage ledger](COVERAGE.md#rust-readability-upstream-generic-rule-accounting)
accounts for all 26 upstream generic rules and the measured exclusions.

### Fixture calibration

Executed with a locally built binary, real provider credentials, opaque
evaluator fixture names, and uncached requests. To reproduce from this checkout,
put Jevlint and Cargo on `PATH` and configure the provider credentials:

```sh
python3 scripts/check-rust-packs.py --eval --repeat 3 --pack rust-readability \
  --jevlint jevlint --summary readability-summary.json
```

All **25 isolated Rust 2021 fixtures** compiled; **23 model cases** produced
**69 case-runs**. Two literal-free named/enum repair twins remain compile-only
because the numeric prefilter skips them. Majority recall was **9/10** and
specificity **13/13**, with **1 flipped case**, **0 mismatches**,
**2 inconclusive case-runs**, **0 abstain decisions**, **4 below-floor fail
decisions**, and **0 evaluation errors**. The byte-budget and numerical-policy
failures were reported at **0.99 in each run**. All four literal-policy failure
classes failed in every run; ordering confidence was **0.96 / 0.98 / 0.97**
and sentinel confidence **0.97 / 0.98 / 0.98**. The registry-persistence failure
remained majority inconclusive and flipped. The command exited **1** for these
honestly recorded calibration limitations, not compilation or provider errors.
No fixture ran.

| Rule | Majority recall @0.8 | Majority specificity @0.8 |
| --- | --- | --- |
| `rust-readability-magic-domain-values` | 4/4 (1.00) | 5/5 (1.00) |
| `rust-readability-mixed-levels-of-abstraction` | 1/1 (1.00) | 2/2 (1.00) |
| `rust-readability-accurate-doc-comments` | 4/5 (0.80) | 6/6 (1.00) |

The corpus includes general multi-unit solver, telemetry encoding and
registry twins; mathematical coefficients, declared storage layouts,
diagnostic encodings, 0/1 flags and conversions are hard negatives.
Function units keep attached multiline Rustdoc and implementation together.
Undocumented units and literal-free named repairs are skipped by prefilters;
the documented pass replacement has accurate basic contracts.
These small denominators do not establish production recall or precision.

Historically, an initial ten-rule calibration compiled **63** candidate
fixtures and ran **189** real case-runs: **3 flips**, **10 mismatches**,
**37 inconclusive case-runs**, **6 abstain decisions**, **104 below-floor
fail decisions**, and **0 evaluation errors**. Its candidate-specific
majority metrics and removal reasons are recorded in the coverage ledger.
Those discarded fixtures/rules are not shipped as unused scaffolding.

### Three-run tuning-set/in-sample project comparison

The audit used a held-out private Rust workspace (43 files, 958 units).
Its triage shaped candidate selection and rule exceptions, so these are
**tuning-set/in-sample measurements**, not an unbiased held-out evaluation.
Each shipped run scanned **43 files**, extracted **958 units**, performed
**1163 rule evaluations**, and had **0 cache hits / 947 misses**.
The configuration enabled only Rust, set `minConfidence: 0.8`, copied the
shipped three-rule array, and set each rule's `guidance` to the contents of
this pack's `guidance.md`, matching pack-install inheritance.

```sh
# From the audited project's root, with the configuration described above:
jevlint check --config jevlint.json --refresh-cache --format json --show-below-floor
# Run the identical command/configuration three times.
```

Raw reports, configuration snapshots, fixture summaries and independent
review outputs were retained locally rather than committed as large artifacts.
`check` exited **1** on each run because warnings were reported; no
provider/scan error was substituted for an empty report.

The supplied upstream baseline had **53 reports: 12 actionable, 6 nits,
35 false positives**, actionable share **22.6%**. Comparison matches the
same rule concern, path and enclosing symbol, normalizing Rustdoc region
names to their parent symbol rather than relying on line numbers.
Original items retain the supplied independent triage; every new shipped
item was independently reviewed read-only against the audited source or
matched to an already independently reviewed source finding.

| Shipped run | Reports | Actionable | Nits | False positives | Actionable share | Original actionable retained | Original false positives retained | Below floor |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | 17 | 10 | 0 | 7 | 58.8% | 6/12 | 1/35 | 69 |
| 2 | 17 | 10 | 0 | 7 | 58.8% | 6/12 | 1/35 | 70 |
| 3 | 17 | 10 | 0 | 7 | 58.8% | 6/12 | 1/35 | 73 |

All three runs reported the same **17 distinct findings**, so the union
and two-of-three majority both contain **10 actionable, 0 nits and
7 false positives** (58.8% actionable share). Across **51 reported
occurrences**, **30 were actionable**, **0 nits** and **21 false positives**.
These denominators are distinct even though their shares happen to agree.

- Original actionable items retained in every run: **U10, U14, U28,
  U38, U41, U49** (**6/12** by run, union and majority).
- Of the original **35 false positives**, only **U27** remained in every
  run (**1/35**). No original nits remained.

#### Every new shipped finding

The **10 new distinct findings** were **4 actionable and 6 false positives**,
all reported in every run. All concern `rust-readability-magic-domain-values`;
there were **0 new documentation or abstraction findings**. Anonymous classes
preserve the independently reviewed evidence without private identifiers.

| Finding class | Independent classification | Rationale |
| --- | --- | --- |
| Default numerical policy | Actionable | Unexplained numerical defaults encode tunable policy rather than mathematical identities. |
| Owner metadata allowance | Actionable | A metadata reservation allowance lacks a visible exact-layout derivation. |
| Snapshot per-slice metadata allowance | Actionable | Per-slice overhead is an unexplained allowance, not established storage width. |
| Publication metadata allowances | Actionable | Reservation allowances need policy names or an explanation of their values. |
| Scratch-buffer layout | False positive | The charged bytes follow concrete scratch storage, not discretionary policy. |
| Boundary-row input/output storage | False positive | Element widths and declared input/output columns establish the byte total. |
| Boundary-grid storage | False positive | Declared input/output buffers establish the per-cell layout. |
| Marker/outcome integer storage | False positive | Integer marker and outcome widths account for the reservation. |
| Calibration/output buffer planning | False positive | Explicit buffer element types and counts explain the layout multipliers. |
| Declared surface container layout | False positive | Container storage is layout accounting, not tunable numerical policy. |

A **historical, single seeded Rustdoc-only scan**, before the pure cost
prefilters below, reported **S11**, an invalid-input panic: documentation
promised invalid overrides were reported, while the implementation unwrapped
an external timestamp parse. It had **1 report** and **1 below-floor finding**
in **613 evaluations** over **43 files / 965 units**. It is neither a shipped
cost benchmark nor a three-run estimate of seed recall.

### Authorized cost prefilters

Rule descriptions, exceptions and guidance were frozen **before public
inspection**. After the first public scan, only authorized pure cost-control
`sourceMatch` prefilters were added; no criterion or confidence floor changed.
The three-run results here use that **shipped filtered configuration**.

| Corpus | Evaluations before | Evaluations shipped | Reduction | Magic / abstraction / Rustdoc shipped |
| --- | --- | --- | --- | --- |
| Private tuning set | 1821 (607 per rule) | 1163 | 658 (36.1%) | 269 / 589 / 305 |
| Public production scope | 3159 (1053 per rule) | 1489 | 1670 (52.9%) | 237 / 981 / 271 |

Literal policy selects digit-bearing candidates; literal-free repairs are
skipped. Rustdoc selects attached documentation; undocumented functions are
**skipped, not explicit model passes**. The abstraction filter broadly selects
structural construction/control flow, scalar/bit/array operations and codec
signals, deliberately over-including references and attributes. These filters
are not complexity detectors and do not establish that a selected unit fails.

### Genuinely held-out public comparison

The public corpus was [serde_json](https://github.com/serde-rs/json), pinned to
[`afdf6fc67247dd7fa4fcde1381e6ecc6bcc7a30e`](https://github.com/serde-rs/json/tree/afdf6fc67247dd7fa4fcde1381e6ecc6bcc7a30e).
The explicit production scope **`src` plus `build.rs`** contains **38 Rust
files**, within the declared 30–70-file production scope; the full repository
has **71 tracked Rust files**, including tests and fuzz code excluded from this
comparison. Both configurations scanned exactly the same production scope.
The upstream baseline enabled Rust only and all **26 generic rules**.

An initial corpus attempt used [nom](https://github.com/rust-bakery/nom) at
[`51c3c4e44fa78a8a09b413419372b97b2cc2a787`](https://github.com/rust-bakery/nom/tree/51c3c4e44fa78a8a09b413419372b97b2cc2a787)
(65 tracked Rust files), but failed before provider evaluation on a parser
syntax error in `tests/float.rs`. It yielded no accuracy measurement. Neither
rules nor source were patched to force that attempt through.

#### Public reproduction

From this Jevlint checkout, with a built `jevlint` on `PATH`, Python 3.9+ and
the same real provider/model credentials configured, the following creates
standalone Rust-only configurations. Copying guidance text into each pack
rule reproduces installation inheritance without requiring a committed local
pack. The baseline copies the [pinned upstream generic rules](https://github.com/codegirl-007/jevlint/blob/73a7a94ab08a4e314709d68c2863687a4646f29b/jevlint.json) unchanged, not the local root configuration's later exceptions.

```sh
export JEVLINT_REPO="$PWD"
git clone https://github.com/serde-rs/json.git serde-json-readability
git -C serde-json-readability checkout --detach afdf6fc67247dd7fa4fcde1381e6ecc6bcc7a30e
python3 - <<'PY'
import json
import os
from pathlib import Path
from urllib.request import urlopen

repo = Path(os.environ["JEVLINT_REPO"])
corpus = Path("serde-json-readability")
pack = repo / "packs/rust-readability"
rules = json.loads((pack / "rules.json").read_text())["rules"]
guidance = (pack / "guidance.md").read_text()
for rule in rules:
    rule["guidance"] = guidance
upstream_url = "https://raw.githubusercontent.com/codegirl-007/jevlint/73a7a94ab08a4e314709d68c2863687a4646f29b/jevlint.json"
with urlopen(upstream_url) as response:
    upstream = json.load(response)["rules"]
assert len(upstream) == 26
for name, selected in (("readability", rules), ("upstream", upstream)):
    config = {"languages": {"rust": {}}, "minConfidence": 0.8, "rules": selected}
    (corpus / (name + ".json")).write_text(json.dumps(config, indent=2) + "\n")
PY
cd serde-json-readability
for run in 1 2 3; do
  jevlint check --config readability.json --refresh-cache --format json \
    --show-below-floor src build.rs > "readability-run-$run.json"
  status=$?
  test "$status" -le 1 || exit "$status"
done
jevlint check --config upstream.json --refresh-cache --format json \
  --show-below-floor src build.rs > upstream-run.json
status=$?
test "$status" -le 1 || exit "$status"
```

The observed checks exited **1 for reported findings**, not a clean pass.
`--refresh-cache` makes every run uncached; below-floor decisions stay separate
from the reported-finding denominator. Keep generated configurations/reports
local. Provider variation means reproduction need not yield identical judgments.

| Configuration / run | Files | Units | Evaluations | Cache hits / misses | Reports | Actionable | Nits | False positives | Actionable share | Below floor |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Shipped pack 1 | 38 | 1363 | 1489 | 0 / 1206 | 2 | 0 | 0 | 2 | 0% | 9 |
| Shipped pack 2 | 38 | 1363 | 1489 | 0 / 1206 | 2 | 0 | 0 | 2 | 0% | 9 |
| Shipped pack 3 | 38 | 1363 | 1489 | 0 / 1206 | 2 | 0 | 0 | 2 | 0% | 7 |
| Upstream generic, once | 38 | 3961 | 26758 | 0 / 5197 | 41 | 1 | 18 | 22 | 2.4% | 567 |

The shipped runs reported the same **2 findings**: union and majority both
contain **0 actionable, 0 nits and 2 false positives**. All **6 occurrences**
were false positives. Upstream rule kinds differ, hence the different unit
counts despite identical files. Report overlap, matching the same concern and
source span, was **0**; actionable overlap was also **0**.

#### Independent public audit: all 43 reported items

Every distinct reported rule/span item (**P01–P02 and U01–U41**) was reviewed
independently, read-only against the pinned source. Different rules on one
span remain separate reported items. The grouped accounting below covers all
43 IDs; raw review evidence and reports are retained locally, not committed.
Public IDs are separate from the private baseline's U-numbering.

| Rule | IDs and independent classification | Evidence summary |
| --- | --- | --- |
| Shipped magic-domain-values | P01–P02: 2 false positives | `exponent_limit` in `src/lexical/num.rs:325–328` (f32 ±10) and `382–385` (f64 ±22) gives mathematical exact-representability bounds. Adjacent exact-power tables at 7–49 and trait documentation at 236–237 explain them; `src/lexical/algorithm.rs:18–41` uses them for fast-path conversion. |
| Upstream comment-quality | U26: 1 actionable | `src/lexical/math.rs:633` says “Iteratively add elements from y to x” inside the subtraction function at 629–650. Correcting or removing this factual mismatch is useful. |
| Upstream comment-quality | U03–U05, U07–U09, U13, U16–U18, U21–U25, U27–U28, U34: 18 nits | Accurate but optional phase/field labels in error parsing and lexical arithmetic/conversion; deleting or refining them is stylistic, not a substantive defect. |
| Upstream comment-quality | U01, U06, U10–U12, U14–U15, U38–U39: 9 false positives | Grammar constraints, numerical reasoning, binary representation and hex-validation explanations add information. Several reported fragments omit the rest of a meaningful multiline comment. |
| Upstream temporal-coupling | U02: 1 false positive | `MapKey` in `src/de.rs:2161–2165` is a private parser adapter; its construction sites already check the opening-quote precondition. |
| Upstream unnecessary-abstraction | U19, U29, U31, U33, U35–U37: 7 false positives | Required `Hi64`, `Float` and borrowed `Read` trait adapters preserve generic dispatch and source-specific behavior. |
| Upstream wrapper-without-value | U20, U30, U32: 3 false positives | The same numeric trait delegation is required conformance, not removable indirection. |
| Upstream accurate-doc-comments | U40–U41: 2 false positives | `src/ser.rs:1895–1915` explicitly permits either object-key/value hook to write the colon; the default value hook does so, consistent with both contracts. |

Thus upstream per-rule totals are **comment-quality 28 = 1 actionable /
18 nits / 9 false positives**, **accurate-doc-comments 2 false positives**,
**temporal-coupling 1 false positive**, **unnecessary-abstraction 7 false
positives**, and **wrapper-without-value 3 false positives**; the other
21 upstream rules reported nothing.

This is **not a held-out actionable-share improvement**: the pack measured
**0% versus upstream 2.4%**. It removed bulk reported noise but added two
mathematical false positives and missed the one useful upstream comment
finding because comment-quality was dropped. Public runtime/readability
seed recall remains unmeasured. No descriptions, exceptions or guidance were
tuned from held-out outcomes; the authorized cost-only prefilters above are
the sole post-first-scan change.

### Selection tradeoffs and native checks

Historically, the first conservative ten-rule project scan reported
**3 findings**: **2 original actionable items** and **1 new nit**, retaining
**0/35** upstream false positives. A broader tuned scan recovered **all 12**
original actionable items, but produced **55 reports**. Its independent
new-item triage found:

- Control flow: **27 new reports**, **3 actionable**, **1 nit**,
  **23 false positives**. Including the two original actionable reports,
  the rule's **29 reports** were only **5/29 actionable**. Focused numerical
  stages and FFI validation/lifecycle glue were the main noise classes,
  so this rule was dropped rather than shipping a mostly noisy port.
- Literal policy: **9 new reports**, **7 actionable**, **2 false positives**.
  Concrete storage-layout arithmetic was legitimate; the final rule adds
  bounded type context and an explicit declared-layout/encoding exception.
  It also prohibits diagnosing a caller solely for a callee's literal.
- Boolean parameter: its sole report was a **nit**, because the only
  Rust caller forwarded a clearly named external operation selector.
  It was dropped along with candidates lacking useful measured recall
  or reported project value.

A historical, unshipped stronger byte-allowance/layout-proof formulation
matched all **11 literal fixtures in one targeted real-provider run**
(not a three-run calibration), but its project scan reported **24 findings**
and **70 below floor**, retaining **8/12** original actionable items.
It reintroduced independently confirmed scratch-size and declared storage-layout
false positives. That broader trial was rejected rather than claiming fixture
recall alone established project value. A narrower explicit byte-allowance
clarification subsequently resolved the mandatory byte-budget class at 0.99
in all three shipped calibration runs without lowering the confidence floor.

The shipped pack's private actionable share is **58.8% versus 22.6%**,
retaining only **6/12** original actionable findings and **1/35** original
false positives. The broader trial's higher retention must not be attributed
to the shipped pack. Candidate selection and exceptions used private audit
triage, so that gain is tuning-set/in-sample evidence, not held-out accuracy.
The genuinely held-out public result above instead exposes mathematical
false positives and a missed useful comment. Three runs are a small sample;
model outcomes can change across prompts, providers and runs.

The following native checks actually passed before the final commit:

```sh
python3 scripts/check-rust-packs.py
python3 -m unittest discover -s scripts -p 'test_check_rust_packs.py'
go test ./internal/packs/
```

The full runner compiled **451 isolated fixtures across seven packs**
(the original 426 plus 25 readability targets), and **3 Python unit
tests** passed. `scripts/__pycache__` was removed. The targeted Go pack tests
passed in **0.720s** after the readability registry addition and prefilters.
An initial Go run failed because the source filters were empty; the filters
were fixed, **not** the cost-control `sourceMatch` assertion weakened. The Go
implementation was unchanged; its registry test was updated. No full Go test
suite is claimed. Native fixture compilation does not execute deliberately
incorrect or unsafe examples; the provider calibration's exit 1 remains
recorded separately above.
