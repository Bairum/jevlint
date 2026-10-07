# Rust pack calibration (2026-10-06)

Shipped decision: fail probability is the minimum of a 3-level score and subject-gated violation checks. Report at **0.80** unless a project sets `minFailProbability`. Packs ship no per-rule floor. The below-floor audit band at that default is **0.40–0.80** (`--show-below-floor`); it does not change the exit code.

Fixture rates below are rounded to three decimals. A fitted threshold is the in-sample value that kept specificity at least 0.90. It is not a shipped override.

## Fixture table

Remaining Rust packs, threshold 0.80, specificity target 0.90, two runs recorded 2026-10-06. Case counts are that run's evaluated fixtures for these packs (260). The run-wide mismatch counts covered packs retired on 2026-10-07; see [Retired packs](#retired-packs).

| Suite | Fail | Pass | Recall @0.80 | Specificity @0.80 | AUC | Recall at fit | Fitted threshold | Flip rate |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| rust-core | 52 | 88 | 0.635 | 1.000 | 0.825 | 0.731 | 0.579 | 0.029 |
| rust-core-advisory | 29 | 39 | 0.621 | 1.000 | 0.967 | 0.966 | 0.363 | 0.000 |
| rust-performance | 11 | 18 | 0.909 | 1.000 | 0.929 | 0.909 | 0.803 | 0.000 |
| rust-readability | 10 | 13 | 0.900 | 1.000 | 0.992 | 1.000 | 0.790 | 0.000 |

The rust-readability row includes `rust-readability-mixed-levels-of-abstraction` (1 fail and 2 pass cases), which moved to the opt-in design pack on 2026-10-07.

Generic rules (`root`), same specificity target, two runs, recorded at threshold **0.50** (not the shipped 0.80): 337 fail and 337 pass, recall 0.961, specificity 1.000, AUC 0.985, recall at fit 0.979, fitted threshold 0.277, flip rate 0.004.

Do not treat fixture specificity as production precision.

## Architecture experiment

1,135 fixture cases, 1,679 scored units, report band probability >= 0.70, specificity gate 0.90. Compared: 3-level score (`s-expected`, `s-level`) with paraphrases; subject-gated violation checks (`checks-gate`); exception questions that zero the subject gate (`checks-gate-ex`); product, noisy-OR, and soft-policy combiners; hybrids; ensembles of score and checks (mean, max, min, agree); and the old holistic choice. States were minimal, escalation, and context.

Products and standalone exception questions failed the recall gate. On context holdout, `checks-product` recall was 0.078 at specificity 1.000 (0.073–0.078 across states). Noisy-OR was 0.251–0.262. Soft policy was 0.335–0.353. Hybrid was 0.521–0.535. Zeroing the subject gate when an exception was strong (`checks-gate-ex@0.7`, context) raised specificity to 0.980 but dropped AUC to 0.888 from `checks-gate`'s 0.934, and recall fell from 0.803 to 0.753.

`ensemble-min` (minimum of score and checks), context holdout: recall 0.733, specificity 0.986, AUC 0.946. `ensemble-max` missed the 0.90 specificity gate (0.884 / 0.885). Score alone had higher fixture recall (0.814) and lower specificity (0.948). On labelled real code, requiring both signals to agree doubled precision versus the old holistic choice at similar recall (0.57 vs 0.28 at 0.80). That is the shipped combiner.

Averaging wordings cut band flips by about a quarter. On 272 context holdout units, two paraphrases flipped 0.125; the mean of the original plus one paraphrase versus the held-out wording flipped 0.096 and 0.085. Averaging removes part of the wording disagreement, not most of it. A threshold fit on the tune split did not keep holdout specificity.

## Real-code precision

Blind-labelled set: 9 public repos (ripgrep 3fce3b5, mini-redis 3d93b42, hashbrown 7109e3a, libloading f5dc3b8, rust-numpy da6bf5b, gin 43fe48e, requests 611c616, zod 0b216ef, gson 845664b); 10,825 sampled rule-unit pairs, 2,921 labelled (all pairs any variant scored >= 0.30 plus 15% random of the rest), 360 double-labelled with 97.8% agreement. Shipped decision precision/recall(estimated): at 0.50 0.23/0.52; 0.70 0.39/0.33; 0.80 0.57/0.22; 0.85 0.61/0.15. At 0.80 the old holistic choice form had 0.28 precision at similar recall. Most remaining false positives at 0.80 come from comment-quality (15 correct / 10 false) and unnecessary-abstraction (0 / 4); excluding unnecessary-abstraction and wrapper-without-value, precision is 0.65.

unnecessary-abstraction was removed because it had 0 correct reports at any threshold on the blind-labelled real-code set (0 correct / 4 false at 0.80, 0 / 3 at 0.90). comment-quality now reports at 0.90 because precision rose from 15 correct / 10 false at 0.80 to 5 / 1 at 0.90, trading recall for precision.

### Rust correctness threshold round

Question: should the correctness packs (`rust-core`, `rust-core-advisory`) report at 0.70 instead of 0.80? The shipped CLI scanned 8 more public repos (walkdir 6fd031c, fd 14dcd92, rusqlite 91f876c, pydantic-core 383eb95, h2 5b17a0e, reqwest 2230ed3, axum 9f1226d, insta 5df39ce; production code only, 7,486 functions, 98M input tokens). Every result at or above 0.50 was labelled blind (129 items; 26 double-labelled, 23 agreed).

- Only `rust-core-advisory` produced material: 126 of 129 items. The other items in that round were not a reason to lower 0.80.
- `rust-advisory-error-kind-erasure` looked strong below 0.80 (16 correct / 1 false in 0.60–0.80), but every correct item was the same `map_err(|e| ... e.to_string())` pattern in one pydantic-core file. That is one defect repeated, not independent evidence.
- Setting that cluster aside, precision by band was 0.26 (0.50–0.60), 0.35 (0.60–0.70), 0.33 (0.70–0.80, 3 correct / 6 false) and 0.83 at 0.80 or above (5 / 1). `rust-advisory-partial-mutation-before-error` was 3 / 3 in 0.70–0.80, and `rust-advisory-input-panics` was 0 / 1.

Decision: keep 0.80 for every Rust correctness rule. Lowering to 0.70 would add about two false reports for each real one.

### Partial-mutation scope

`rust-advisory-partial-mutation-before-error` now uses a narrower definition. A violation is pre-existing caller-visible application state (a collection, setting, protocol window, persisted file, history list, mode, or buffer of data not yet delivered) changed before a later fallible step and left changed on Err, with no rollback and no stated partial-progress contract. Writer, formatter, serializer, and event-sink output, bookkeeping the function's own Result already reports, an already-closed handle, and a restored `mem::take` or `mem::replace` are out of scope.

128 previously labelled units were relabelled blind. A second labeller marked a 32-unit random subset; 31 agreed (96.9%). Unclear labels are excluded below. Tune is round-2 pydantic-core, insta, fd, and axum, plus earlier labelled units from mini-redis, ripgrep, hashbrown, rust-numpy, and libloading. Holdout is round-2 h2, rusqlite, and reqwest, scored once after the wording was frozen.

| Set | Wording | n | Violations | 0.80 TP/FP | 0.80 recall | 0.70 TP/FP | 0.70 recall |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Tune | previous | 105 | 18 | 2/0 | 0.11 | 8/2 | 0.44 |
| Tune | shipped | 105 | 18 | 6/0 | 0.33 | 8/0 | 0.44 |
| Holdout | previous | 19 | 13 | 3/1 | 0.23 | 3/1 | 0.23 |
| Holdout | shipped | 19 | 13 | 3/0 | 0.23 | 5/0 | 0.38 |

Holdout precision at 0.80 rose from 0.75 to 1.00 with the same recall. Two pass fixtures now sit below the report floor and are not reported: `documented-partial-append` at 0.43 and `caller-discarded-output-buffer` at 0.50. `take-before-validation` scores 0.46 and stays inconclusive. The report floor stays 0.80.


## Held-out

Two runs each, reporting threshold 0.80. A held-out private Rust workspace (43 files): 12 seeded defects, 3/12 reported (6 more between 0.40 and 0.71, visible with `--show-below-floor`). Clean workspace: 7 reports in each run, 8 distinct across the two runs, all `rust-readability`. Independent review: 2 actionable (`rust-readability-magic-domain-values` 1/1, an unnamed solver tolerance and parameter limits; `rust-readability-mixed-levels-of-abstraction` 1/7, an inline hex-digest decoder inside orchestration) and 6 false positives, all `mixed-levels-of-abstraction` on FFI adapter functions whose primary job is marshalling or publication. That rule is the current noise source on adapter-heavy Rust (precision 1/7 on this set). Projects with large FFI or adapter layers should exclude those paths or disable the rule. This scan was not used to change the rule. [serde_json `afdf6fc`](https://github.com/serde-rs/json/tree/afdf6fc67247dd7fa4fcde1381e6ecc6bcc7a30e) (38 production files): 0 reports.


## Retired packs

2026-10-07. These packs are deleted. Do not reinstall them or lower 0.80 to chase the misses.

- `rust-unsafe`: 0/18 RustSec defects caught at 0.80 (2/18 at 0.50).
- `rust-tokio`: 0/19 merged async fixes (best score 0.055).
- `rust-api`: 0/3 in-scope historical defects (0/15 across the broader trait-law benchmark, of which 12 were outside the rules' scope).
- All three: no result >= 0.50 on 17 public repos.
- Root cause shared by all three: a token `sourceMatch` gate, comment-gated subjects, and defects that span more than one function.

Use the native tools in [Rust tooling beyond Jevlint](README.md#rust-tooling-beyond-jevlint). The 2026-10-06 run of all seven packs (461 cases) recorded 62 mismatches, 135 inconclusive case-runs, 11 flipped cases, and no errors.

## History

Before 2026-10-06, calibration fitted per-rule choice-confidence floors on these fixtures and recommended project overlays. That metric and those overlays are retired; the shipped decision is min(score, checks) at 0.80. A 2026-10-05 unsafe wording experiment was rejected and is not this calibration.

## Doc-comment recall (2026-10-07)

Root `accurate-doc-comments` stays on `docComment` units and the current wording.

Ceiling on the 35-commit public benchmark, judged from the doc comment plus that function's own body and not a callee: 8/20 tune and 9/15 holdout are visible. The rest are not catchable by any single-unit rule.

Three `function`-kind variants, scored on tune and the round-1 labelled items only. With no checks, the score signal is the fail probability, as the README says. At 0.80, post-fix false alarms were 0/20 for all three.

| Variant | Tune recall @0.80 | Visible subset | Tune @0.70 | Labelled precision @0.80 |
| --- | --- | --- | --- | --- |
| Score only | 2/20 | 2/8 | 2/20 | 0/1 |
| Score plus one literal check, no subject gate | 1/20 | 1/8 | 2/20 | 0/1 |
| Score plus one polarity check, no subject gate | 2/20 | 2/8 | 2/20 | 0/1 |

The polarity variant was frozen before a fresh 18-commit holdout from repos outside that split. At 0.80 it caught 1/18, the same as the current rule, with 0 post-fix false alarms for both. On the round-1 repos it reported one unit at 0.80, already labelled compliant. No new report reached 0.80. Precision 0. Not shipped.

`rust-readability-accurate-doc-comments` stays. It judges the function, not the comment unit, and its checks name `# Errors`, `# Panics`, and `# Safety`. It is not a copy of the root rule.
