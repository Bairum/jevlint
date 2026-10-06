# Rust pack calibration (2026-10-06)

Shipped decision: fail probability is the minimum of a 3-level score and subject-gated violation checks. Report at **0.80** unless a project sets `minFailProbability`. Packs ship no per-rule floor. The below-floor audit band at that default is **0.40–0.80** (`--show-below-floor`); it does not change the exit code.

Fixture rates below are rounded to three decimals. A fitted threshold is the in-sample value that kept specificity at least 0.90. It is not a shipped override.

## Fixture table

Rust packs, threshold 0.80, specificity target 0.90, two runs. Case counts are the evaluated fixtures (461). The run recorded 62 mismatches, 135 inconclusive case-runs, 11 flipped cases, and no errors.

| Suite | Fail | Pass | Recall @0.80 | Specificity @0.80 | AUC | Recall at fit | Fitted threshold | Flip rate |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| rust-api | 15 | 20 | 0.467 | 1.000 | 1.000 | 1.000 | 0.234 | 0.029 |
| rust-core | 52 | 88 | 0.635 | 1.000 | 0.825 | 0.731 | 0.579 | 0.029 |
| rust-core-advisory | 29 | 39 | 0.621 | 1.000 | 0.967 | 0.966 | 0.363 | 0.000 |
| rust-performance | 11 | 18 | 0.909 | 1.000 | 0.929 | 0.909 | 0.803 | 0.000 |
| rust-readability | 10 | 13 | 0.900 | 1.000 | 0.992 | 1.000 | 0.790 | 0.000 |
| rust-tokio | 34 | 64 | 0.353 | 0.984 | 0.897 | 0.853 | 0.470 | 0.000 |
| rust-unsafe | 26 | 42 | 0.308 | 1.000 | 0.936 | 0.808 | 0.382 | 0.000 |

Generic rules (`root`), same specificity target, two runs, recorded at threshold **0.50** (not the shipped 0.80): 337 fail and 337 pass, recall 0.961, specificity 1.000, AUC 0.985, recall at fit 0.979, fitted threshold 0.277, flip rate 0.004.

Tokio and unsafe fixture recall at 0.80 is low. Do not treat fixture specificity as production precision.

## Architecture experiment

1,135 fixture cases, 1,679 scored units, report band probability >= 0.70, specificity gate 0.90. Compared: 3-level score (`s-expected`, `s-level`) with paraphrases; subject-gated violation checks (`checks-gate`); exception questions that zero the subject gate (`checks-gate-ex`); product, noisy-OR, and soft-policy combiners; hybrids; ensembles of score and checks (mean, max, min, agree); and the old holistic choice. States were minimal, escalation, and context.

Products and standalone exception questions failed the recall gate. On context holdout, `checks-product` recall was 0.078 at specificity 1.000 (0.073–0.078 across states). Noisy-OR was 0.251–0.262. Soft policy was 0.335–0.353. Hybrid was 0.521–0.535. Zeroing the subject gate when an exception was strong (`checks-gate-ex@0.7`, context) raised specificity to 0.980 but dropped AUC to 0.888 from `checks-gate`'s 0.934, and recall fell from 0.803 to 0.753.

`ensemble-min` (minimum of score and checks), context holdout: recall 0.733, specificity 0.986, AUC 0.946. `ensemble-max` missed the 0.90 specificity gate (0.884 / 0.885). Score alone had higher fixture recall (0.814) and lower specificity (0.948). On labelled real code, requiring both signals to agree doubled precision versus the old holistic choice at similar recall (0.57 vs 0.28 at 0.80). That is the shipped combiner.

Averaging wordings cut band flips by about a quarter. On 272 context holdout units, two paraphrases flipped 0.125; the mean of the original plus one paraphrase versus the held-out wording flipped 0.096 and 0.085. Averaging removes part of the wording disagreement, not most of it. A threshold fit on the tune split did not keep holdout specificity.

## Real-code precision

Blind-labelled set: 9 public repos (ripgrep 3fce3b5, mini-redis 3d93b42, hashbrown 7109e3a, libloading f5dc3b8, rust-numpy da6bf5b, gin 43fe48e, requests 611c616, zod 0b216ef, gson 845664b); 10,825 sampled rule-unit pairs, 2,921 labelled (all pairs any variant scored >= 0.30 plus 15% random of the rest), 360 double-labelled with 97.8% agreement. Shipped decision precision/recall(estimated): at 0.50 0.23/0.52; 0.70 0.39/0.33; 0.80 0.57/0.22; 0.85 0.61/0.15. At 0.80 the old holistic choice form had 0.28 precision at similar recall. Most remaining false positives at 0.80 come from comment-quality (15 correct / 10 false) and unnecessary-abstraction (0 / 4); excluding unnecessary-abstraction and wrapper-without-value, precision is 0.65.

unnecessary-abstraction was removed because it had 0 correct reports at any threshold on the blind-labelled real-code set (0 correct / 4 false at 0.80, 0 / 3 at 0.90). comment-quality now reports at 0.90 because precision rose from 15 correct / 10 false at 0.80 to 5 / 1 at 0.90, trading recall for precision.

## Held-out

Two runs each, reporting threshold 0.80. A held-out private Rust workspace (43 files): 12 seeded defects, 3/12 reported (6 more between 0.40 and 0.71, visible with `--show-below-floor`); 8 blind unsafe defects, 0/8. Do not rely on `rust-unsafe` for unsafe-defect recall. Clean workspace: 7 reports in each run, 8 distinct across the two runs, all `rust-readability`. Independent review: 2 actionable (`rust-readability-magic-domain-values` 1/1, an unnamed solver tolerance and parameter limits; `rust-readability-mixed-levels-of-abstraction` 1/7, an inline hex-digest decoder inside orchestration) and 6 false positives, all `mixed-levels-of-abstraction` on FFI adapter functions whose primary job is marshalling or publication. That rule is the current noise source on adapter-heavy Rust (precision 1/7 on this set). Projects with large FFI or adapter layers should exclude those paths or disable the rule. This scan was not used to change the rule. [serde_json `afdf6fc`](https://github.com/serde-rs/json/tree/afdf6fc67247dd7fa4fcde1381e6ecc6bcc7a30e) (38 production files): 0 reports.

## History

Before 2026-10-06, calibration fitted per-rule choice-confidence floors on these fixtures and recommended project overlays. That metric and those overlays are retired; the shipped decision is min(score, checks) at 0.80. A 2026-10-05 unsafe wording experiment was rejected and is not this calibration.
