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
