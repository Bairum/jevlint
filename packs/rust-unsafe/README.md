# Bairum/rust-unsafe

Opt-in semantic **warnings**, not a Rust soundness checker. Four rules cover locally visible std raw-storage contracts, callback/unwind state, foreign-pointer contracts, and PyO3/CPython reference ownership. **Do not rely on this pack for unsafe-defect recall:** measured misses include extent, lifetime and Python reference-balance defects. Neither a pass nor an abstention guarantees soundness, exception safety, or absence of UB. Run rustc and Clippy separately; this pack does not replace runtime checks or human review.

The following are **recommendations, not checks exercised in this investigation**. Reuse the [project-owned optional tool lanes](../../docs/rust/TOOLING.md#separate-optional-layers) and [tooling research](../../docs/rust/research/tooling.md#5-optional-established-tool-layers):

- Run targeted supported Miri tests for unsafe ownership, extent and lifetime paths. Most FFI operations are unsupported; Miri covers exercised executions, not soundness.
- Use supported ASan/LSan lanes for memory errors and leaks, coordinating native/FFI instrumentation with Rust instrumentation. Toolchain/target support and exercised paths limit coverage.
- Exercise raw-pointer, extent and lifetime paths in debug-assertion builds with meaningful invariant assertions; passing assertions cover only the tested paths and stated invariants.
- Add explicit Python reference-count assertions around ownership transitions for reference leaks and double decrements in supported runtime tests. Compile-only PyO3 fixtures do not exercise Python reference management.

## Rules and applicability

Shared primary-item judging and evidence policy lives in [guidance.md](guidance.md), loaded once from the pack manifest rather than repeated in each description. Tests are skipped structurally by default; use a project rule overlay with `includeTests: true` only when review of test code is intentional.

| Rule | Scope | `sourceMatch` rationale |
| --- | --- | --- |
| `rust-unsafe-precondition-contract` | Allocation extent, initialized publication and allocator/layout/capacity/single-owner reconstruction. | `unsafe` includes both blocks and signatures without depending on std imports or aliases. |
| `rust-unsafe-unwind-state` | Unsafe owned/initialized state exposed by a safe callback, trait operation, unwind or forgotten restoration guard. | `unsafe` retains all raw-state shapes; callback semantics are judged, not inferred by a callback-name filter. |
| `rust-unsafe-ffi-pointer-contract` | Foreign returned pointers, pointer/length out-parameter structs and retained input pointers: nullness, NUL termination, extent, lifetime, alignment, out-parameter writability and matching free responsibility. | Requires `unsafe` plus pointer conversion/access, raw-pointer types or FFI syntax in either order. Bare and qualified API names, including `CStr`/`slice` type aliases, match without depending on a particular crate name; raw-pointer types retain wrappers using aliased calls. |
| `rust-unsafe-pyo3-reference-ownership` | CPython new/borrowed/stolen reference semantics, PyO3 owned/borrowed constructors, owner-release boundaries and unchecked object casts. | Requires `unsafe` plus raw-pointer constructors, unchecked casts or `Py*` APIs in either order, including qualified and unqualified spellings. |

For FFI, failure requires a **visible contradiction**, not merely an absent foreign guarantee. A null buffer with length zero must become an empty Rust slice without calling `from_raw_parts` on null. `CStr::from_ptr` additionally needs a live NUL-terminated allocation. Foreign-owned buffers stay borrowed or are copied before their documented matching release; Rust ownership reconstruction must not silently change the allocator.

For PyO3, `Bound::from_borrowed_ptr` acquires a strong reference while `Borrowed::from_ptr` does not. New references must be consumed once or explicitly balanced; borrowed references must not be consumed as owned without an ownership transfer/INCREF. A weak reference does not keep its referent alive. An unchecked cast requires an established compatible type predicate or construction invariant. Reference-balance mismatches are in scope even when their immediate result is a leak; the std unwind rule still excludes resource leakage alone.

These rules do not attempt exhaustive pinning, aliasing/provenance-model or whole-program destructor analysis. An unsafe signature does not excuse an internally contradicted obligation. Correctly delegated, uncontradicted unsafe caller contracts are legitimate exceptions.

## Research references

The original memory research is in [memory.md](../../docs/rust/research/memory.md). Research IDs and source URLs belong here, not in model-facing rule descriptions or exceptions.

| Rule | Research references and API contracts |
| --- | --- |
| `rust-unsafe-precondition-contract` | M12 safe-interface enforcement, M15 extent, M17 initialized publication, M18 deallocation layout; [MaybeUninit](https://doc.rust-lang.org/std/mem/union.MaybeUninit.html#initialization-invariant), [raw slices](https://doc.rust-lang.org/std/slice/fn.from_raw_parts.html#safety), [Vec ownership reconstruction](https://doc.rust-lang.org/std/vec/struct.Vec.html#method.from_raw_parts). |
| `rust-unsafe-unwind-state` | M13 invalid state across panic, M14 safe-callback trust/leak safety; [ptr::read ownership](https://doc.rust-lang.org/std/ptr/fn.read.html#ownership-of-the-returned-value), [exception safety](https://doc.rust-lang.org/nomicon/exception-safety.html), [leaking](https://doc.rust-lang.org/nomicon/leaking.html). |
| `rust-unsafe-ffi-pointer-contract` | Rust plan item 6; [CStr::from_ptr safety](https://doc.rust-lang.org/std/ffi/struct.CStr.html#method.from_ptr), [from_raw_parts safety](https://doc.rust-lang.org/std/slice/fn.from_raw_parts.html#safety), [Nomicon FFI](https://doc.rust-lang.org/nomicon/ffi.html). |
| `rust-unsafe-pyo3-reference-ownership` | Rust plan item 6; [Bound pointer constructors](https://docs.rs/pyo3/0.29.2/pyo3/struct.Bound.html), [Borrowed](https://docs.rs/pyo3/0.29.2/pyo3/struct.Borrowed.html), [CPython list reference semantics](https://docs.python.org/3/c-api/list.html), [CPython weakref ownership](https://docs.python.org/3/c-api/weakref.html), [PyO3 weakref issue #3134](https://github.com/PyO3/pyo3/issues/3134) and [owned-reference fix #4528](https://github.com/PyO3/pyo3/pull/4528). |

## Real-world fixture provenance

Each `real-*-bug.rs` / `real-*-fixed.rs` pair is a small derived reproduction, not a verbatim copy or dependency on the vulnerable crate. Fixes preserve the illustrated contract, not necessarily every upstream implementation detail.

| Rule | Project and public source | What was minimized |
| --- | --- | --- |
| `rust-unsafe-precondition-contract` | Fyrox `fyrox-core`: [RUSTSEC-2024-0435](https://rustsec.org/advisories/RUSTSEC-2024-0435.html), [issue #630](https://github.com/FyroxEngine/Fyrox/issues/630), [fix #662](https://github.com/FyroxEngine/Fyrox/pull/662). | Turning typed vector storage into byte ownership without satisfying initialized-representation and allocation-layout requirements. |
| `rust-unsafe-unwind-state` | `qwutils`: [RUSTSEC-2021-0018](https://rustsec.org/advisories/RUSTSEC-2021-0018.html), [upstream issue #3](https://github.com/qwertz19281/rust_utils/issues/3). | A raw element shift temporarily duplicates ownership while a user `Clone` can panic; valid published length must be restored safely or unpublished before that boundary. |
| `rust-unsafe-ffi-pointer-contract` | gtk-rs `glib`: [RUSTSEC-2024-0429](https://rustsec.org/advisories/RUSTSEC-2024-0429.html), [fix #1343](https://github.com/gtk-rs/gtk-rs-core/pull/1343). | A variadic foreign string getter receives an immutable rather than mutable out-pointer before `CStr::from_ptr`; the corrected out-argument is writable. |
| `rust-unsafe-pyo3-reference-ownership` | PyO3: [RUSTSEC-2024-0378 / CVE-2024-9979](https://rustsec.org/advisories/RUSTSEC-2024-0378.html), [mitigation #4590](https://github.com/PyO3/pyo3/pull/4590). | `real-pyo3-weakref-*` borrows a weak referent then releases a potentially last strong handle. The fixed adaptation promotes it to a `Bound` strong reference before release; it uses current raw APIs instead of the removed vulnerable convenience methods. |

## Installation and activation

Use an existing `jevlint` executable. Installation clones a **Git repository**, so the source checkout must already contain this pack in the commit being installed; an uncommitted working-tree directory is not installed. In the consuming project, first enable Rust in its existing `jevlint.json` (merge this field, preserving other configuration):

```json
{
  "languages": {
    "rust": {}
  }
}
```

From that consuming project, install from your committed local checkout (adjust the relative checkout path):

```sh
jevlint plugin install ../jevlint#packs/rust-unsafe
jevlint plugin list
jevlint check .
```

If the consuming project is the Jevlint repository itself, the concrete local source form is:

```sh
jevlint plugin install .#packs/rust-unsafe
```

`plugin install` activates the pack by writing its resolved local commit pin into `jevlint.json`; it does **not** enable the required Rust language. No remote revision or fabricated SHA is needed. These forms follow `internal/packs/source.go` and `internal/cli/plugin.go`; Rust activation is enforced by `internal/packs/packs.go`.

## Fixtures and evaluation

`jevlint-evals.json` is the fixture index. Every rule has at least three distinct failure shapes, at least three passing cases including legitimate exceptions, a realistic multi-unit failing file (one violating unit), a realistic clean file, and a cited public-bug/fix pair. The realistic files are at least 80 lines, rather than isolated numerical examples. Comments state actual API or caller contracts, not verdicts or defect narration.

The index contains 68 cases: 19 std precondition cases, 14 unwind cases, 16 FFI cases and 19 PyO3 cases. The real pairs are `real-fyrox-bytes-*`, `real-qwutils-insert-*`, `real-glib-variant-out-param-*` and `real-pyo3-weakref-*` under their respective rule directories. Six added cases form three larger multi-unit fail/pass twins: a byte/element extent mismatch, a copied C-string owner's lifetime mismatch, and a new-reference publication mismatch among legitimate unsafe operations. These are **known-miss/limitation fixtures**, not evidence of improved recall or a reason to ship the rejected wording.

The two external-contract rules additionally have no-contract variants with foreign contract documentation removed, expected to pass: missing external reference/validity guarantees alone do not establish a contradiction. Removing comments cannot erase known std or CPython API requirements, so the existing std rules do not use comment-removal variants with artificially passing verdicts.

All fixtures are independent Rust 2021 library crates. Std/FFI fixtures need no external dependency; PyO3 fixtures use `pyo3 = { version = "=0.29.2", default-features = false, features = ["macros", "abi3-py38", "extension-module"] }`. The compile-only runner sets `PYO3_NO_PYTHON=1`; it neither discovers an interpreter nor links or runs Python. **Never execute failing fixtures.**

After installation, existing CLI evaluation commands are:

```sh
jevlint eval --packs --rule rust-unsafe-precondition-contract
jevlint eval --packs --rule rust-unsafe-unwind-state
jevlint eval --packs --rule rust-unsafe-ffi-pointer-contract
jevlint eval --packs --rule rust-unsafe-pyo3-reference-ownership
```

These commands invoke the configured model provider. The shared Rust pack fixture runner checks each source with `cargo check` under the pinned dependencies; declarations of foreign symbols do not require linking.

## Verification status

The shipped `rules.json` and `guidance.md` are restored exactly to the original `f350461`/master versions. The generalized description, exception and guidance changes were **rejected**: FFI recall regressed and the blind eight-defect experiment showed no reporting-recall gain. The six larger fixture cases remain to document limitations; they do not change the original rules or confidence floor.

Recorded compile-only checks on 2026-10-05 covered all 68 isolated Rust 2021 targets in this pack (432 across the six-pack checkout); the three fixture-runner unit tests passed. Faulty fixtures were never executed. Compilation does not establish that the model detects their defects.

### Original-rule baseline: 62 cases

The recorded original 62-case baseline at the global 0.8 floor is:

| Rule | Recall @0.8 | Specificity @0.8 |
| --- | --- | --- |
| `rust-unsafe-precondition-contract` | 0.00 | 1.00 |
| `rust-unsafe-unwind-state` | 0.00 | 1.00 |
| `rust-unsafe-ffi-pointer-contract` | 0.24 | 1.00 |
| `rust-unsafe-pyo3-reference-ownership` | 0.17 | 1.00 |

### Rejected generalized-wording experiment: 68 cases

Three uncached real-provider calibration runs measured the frozen generalized wording on 68 cases. These results belong to the rejected experiment, **not the shipped rules or their current calibration**. Below-floor failures count as not reported:

| Rule | Recall @0.8 | Specificity @0.8 | In-sample fitted floor | Recall @fitted floor | Specificity @fitted floor |
| --- | --- | --- | --- | --- | --- |
| `rust-unsafe-precondition-contract` | 0.17 (3/18) | 1.00 (39/39) | 0.30 | 0.61 | 1.00 |
| `rust-unsafe-unwind-state` | 0.00 (0/15) | 1.00 (27/27) | 0.60 | 0.07 | 1.00 |
| `rust-unsafe-ffi-pointer-contract` | 0.13 (3/24) | 1.00 (24/24) | 0.30 | 0.29 | 1.00 |
| `rust-unsafe-pyo3-reference-ownership` | 0.29 (6/21) | 1.00 (36/36) | 0.65 | 0.57 | 1.00 |

Compared with the original 62-case recall baseline of 0.00/0.00/0.24/0.17, these 68-case results improve precondition/PyO3 recall, leave unwind recall unchanged, and regress FFI recall from 0.24 to 0.13. Reporting specificity stays 1.00 for all four rules. Restricting the rejected runs to the unchanged 62-case corpus gives recall 3/15, 0/15, 3/21 and 6/18 with specificity 1.00 throughout, so the FFI regression is not explained solely by adding the six cases. Lower-floor fitted FFI/PyO3 recall is 0.29/0.57 versus the old 0.43/0.61. These fitted floors are historical in-sample experiment results, not recommended current overrides or shipped defaults.

In the rejected experiment, case-majority eval outcomes (below-floor failures remain inconclusive) detect 1/6 precondition, 0/5 unwind, 1/8 FFI and 2/7 PyO3 failures; clean-case pass majorities are 13/13, 4/9, 7/8 and 11/12. There are six flipped cases (6/68), 28 mismatched case-runs, 60 inconclusive case-runs, seven abstain decisions, 53 below-floor fail decisions and zero evaluation errors. The calibration command exits 1 for the known imperfect corpus, not compilation/provider errors. The larger extent defect remains below floor (0.56, 0.56, 0.48), while the larger CString and Python defects are judged clean 3/3; all three pass twins pass 3/3. These scores measure rejected wording and do not demonstrate improved general large-function recall.

Global `minConfidence` stays `0.8`; the restored pack ships no rule-level floor. The fitted lower floors above cannot substitute for demonstrated unsafe recall, and none of these metrics certifies soundness. See [calibration method and caveats](../../docs/rust/CALIBRATION.md).

### Original-rule measurements: six limitation cases

The six added cases were evaluated separately in three uncached runs using the original master rules and their inherited guidance. This is **not a full 68-case recalibration** of the restored rules; the recorded 62-case baseline above remains unchanged. All six cases received raw **pass** decisions in all three runs, including all three violating primary units:

| Defect class / twin | Original-rule status | Confidence, runs 1–3 |
| --- | --- | --- |
| Byte/element extent, fail twin | pass, pass, pass | 0.48, 0.52, 0.38 |
| Byte/element extent, pass twin | pass, pass, pass | 0.96, 0.95, 0.94 |
| Copied C-string owner lifetime, fail twin | pass, pass, pass | 0.54, 0.55, 0.53 |
| Copied C-string owner lifetime, pass twin | pass, pass, pass | 0.71, 0.75, 0.68 |
| Python new-reference balance, fail twin | pass, pass, pass | 0.85, 0.84, 0.85 |
| Python new-reference balance, pass twin | pass, pass, pass | 0.86, 0.84, 0.86 |

Each run matched three passing expectations and mismatched all three failing expectations, with **0 reported findings, 0 below-floor findings and 0 inconclusive cases**. Recall at 0.8 and by case majority is **0/3**; specificity is **3/3**. Across the three runs there were nine mismatched case-runs, zero inconclusive case-runs, zero errors and zero flips. Each eval exited 1 because of the mismatches, not provider errors. The fail twins are known misses under the restored rules, not merely failures hidden by the reporting floor.

The existing CLI was run three times with a six-case-only evaluation index:

```sh
jevlint eval --config original.json --evals six-cases.json \
  --refresh-cache --format json --verbose
```

The index filename above is illustrative; this command documents the separate six-case lane rather than a full-pack recalibration.

Existing verdict-commentary was removed rather than preserved as evidence; no test-only fixtures are retained or relied on. No existing fixtures were deleted; all 12 original std cases retain their paths and verdicts.

## Real-project miss diagnosis (2026-10-05)

Three seeded defects in a held-out Rust workspace (43 files) were missed: byte count used as element count, a copied CString owner dropped before borrow, and a new Python reference wrapped as borrowed without balancing its original reference. The subsequent wording experiments are historical diagnosis only; neither the seed-focused draft nor the generalized revision is shipped. The retained larger fixtures express general Rust/API contracts without project-specific names or domain assumptions.

Three uncached original-pack scans limited to the two affected files each reported zero findings and zero below-floor findings. Isolating the original primary units while retaining their applicable rule batch exposed the following cached decisions:

| Defect | Original status/confidence, runs 1–3 |
| --- | --- |
| Byte count used as element count | pass 0.71, pass 0.74, pass 0.61 |
| Copied CString owner dropped before borrow | pass 0.47, pass 0.40, pass 0.42 |
| New Python reference wrapped as borrowed | pass 0.67, pass 0.66, pass 0.67 |

The controlled ablations below use `eval --refresh-cache --format json --verbose` on those same primary functions, selecting one rule at a time. All existing results and ablation controls are in-sample because they were used while drafting wording. `P`/`F` are raw pass/fail decisions, not reporting outcomes; failures below 0.8 remain unreported. Exception numbers refer to their original order. The original control has no `context.types`; the type-context control enables it. Each cell records three uncached calls.

| Control | Byte/element extent | CString lifetime | New-reference balance |
| --- | --- | --- | --- |
| Original | P .55, .68, .59 | P .48, .60, .52 | P .66, .68, .65 |
| No exceptions | P .51, .30, .39 | P .37, .44, .36 | P .66, .63, .68 |
| Remove exception 1 | P .46, .53, .60 | P .50, .44, .47 | P .73, .67, .67 |
| Remove exception 2 | P .67, .55, .63 | P .58, .52, .56 | P .65, .66, .70 |
| Remove exception 3 | P .53, .58, .60 | P .48, .46, .51 | P .66, .61, .68 |
| Remove exception 4 | P .64, .62, .57 | P .45, .47, .47 | P .65, .70, .68 |
| Remove exception 5 | — | P .43, .52, .62 | P .68, .66, .65 |
| Shortened description | F .63, .63, .65 | P .39, .38, .38 | P .48, .41, .45 |
| Add `context.types` | P .66, .73, .60 | P .57, .52, .53 | P .69, .67, .66 |
| Reduce unit to relevant operations | P .58, .49, .51 | P .48, .52, .53 | P .44, .38, .42 |
| Add explicit API-contract comment | P .76, .75, .62 | P .32, .32, .37 | P .62, .70, .65 |
| Normative operation requirements, original exceptions | F .91, .89, .86 | F .76, .72, .77 | F .84, .81, .84 |
| Normative requirements, no exceptions | F .96, .95, .94 | F .82, .73, .79 | F .88, .90, .87 |
| First operation-specific exception/guidance revision | F .90, .91, .93 | F .79, .78, .78 | F .91, .89, .89 |

No individual exception removal, extra type context, explicit contract comment or source reduction flipped a verdict. Shortening alone flipped the extent case, but only below the floor. Seed-specific normative operation requirements flipped all three raw verdicts in these in-sample controls. Prompt formulation was the observed lever; the provider returned no reasoning text, so its internal reasoning and the causes of the misses are unknown. Draft pointer-by-pointer checks, moved-versus-dropped owner distinctions and same-pointer reference-balancing requirements were experimental wording, not implemented guards or guarantees in the restored rules.

Seed-focused draft in-sample scans covered all 43 Rust files in the audited workspace. Each run made 14 uncached provider requests (41 rule decisions; zero cache hits). All three seeded defects were reported in all three runs. These are in-sample draft results used while drafting wording; they are NOT blind held-out recall evidence and measure neither the rejected generalized wording nor a change shipped in the restored pack:

| Seeded defect | In-sample draft fail confidence, runs 1–3 | Reported majority |
| --- | --- | --- |
| Byte count used as element count | 0.87, 0.88, 0.89 | 3/3 |
| Copied CString owner dropped before borrow | 0.82, 0.84, 0.84 | 3/3 |
| New Python reference wrapped as borrowed | 0.88, 0.86, 0.89 | 3/3 |

The seed-focused draft's clean workspace scans reported **0, 0, 0 findings** and **0, 0, 0 below-floor findings**. There are no new clean-project below-floor items to list. These results cover this audited workspace, not whole-program soundness or production accuracy on other projects; they are NOT blind held-out recall evidence and do not measure generalized wording. Only descriptions, exception wording and guidance changed in that draft; applicability filters and confidence floors did not. Those wording changes are not shipped.

## Rejected generalized-wording clean check (2026-10-05)

The rejected generalized descriptions specified the length argument's unit, every later use of pointers derived from owned storage, and the called Python C API's new/borrowed/stolen reference contract without naming the in-sample seeded APIs or their exact construction patterns.

Three uncached checks of a held-out Rust workspace (43 files) with that frozen experimental wording reported **0, 0, 0 findings** and **0, 0, 0 below-floor findings**. Each run made 14 provider requests and 41 rule decisions with zero cache hits. No clean-workspace false positives or new below-floor items were observed. These are rejected-experiment clean results, not evidence of current recall; the earlier in-sample defect results belong to the seed-focused draft.

## Rejected generalized-wording blind defects (2026-10-05)

The experimental generalized rules and guidance were committed before eight independent defects were published for the same held-out Rust workspace (43 files). No rule, guidance, filter, exception or confidence floor changed after publication. The frozen experimental pack received three uncached full-workspace checks; the original pack from `f350461` received one baseline check. Each scan made 14 provider requests with zero cache hits. The generalized revision was subsequently rejected and the original rules/guidance restored.

`belowFloor` means a failed decision below the global 0.8 reporting floor. `missed/pass` records the model's pass decision and confidence; it is not a finding. Scores below are from the originally cached full-workspace responses, recovered with cache-only lookups that blocked network connections, not additional model evaluations.

| Seed | Defect class | Original baseline | Rejected frozen run 1 | Rejected frozen run 2 | Rejected frozen run 3 |
| --- | --- | --- | --- | --- | --- |
| H01 | Shifted raw pointer with unchanged extent | belowFloor 0.38 | belowFloor 0.64 | belowFloor 0.68 | belowFloor 0.64 |
| H02 | Second owning reconstruction of live owned storage | missed/pass 0.55 | missed/pass 0.29 | missed/pass 0.31 | missed/pass 0.31 |
| H03 | Partially written array assumed fully initialized | missed/pass 0.89 | missed/pass 0.89 | missed/pass 0.92 | missed/pass 0.91 |
| H04 | Foreign static storage adopted by the wrong allocator | belowFloor 0.76 | belowFloor 0.51 | belowFloor 0.49 | belowFloor 0.56 |
| H05 | Foreign output converted before its success/null checks | belowFloor 0.54 | missed/pass 0.46 | missed/pass 0.53 | missed/pass 0.47 |
| H06 | Foreign view retains storage whose local owner drops | missed/not evaluated | missed/not evaluated | missed/not evaluated | missed/not evaluated |
| H07 | Borrowed Python reference consumed as owned | missed/pass 0.63 | missed/pass 0.76 | missed/pass 0.81 | missed/pass 0.77 |
| H08 | Borrowed Python reference passed to a stealing API | missed/pass 0.50 | missed/pass 0.52 | missed/pass 0.49 | missed/pass 0.49 |

At the unchanged 0.8 floor, reported recall was **0/8 in every rejected frozen run (majority 0/8)** and **0/8 in the original baseline**. Raw failed decisions on the intended rules fell from 3/8 in the single baseline to 2/8 in each frozen run; H05 regressed to pass. H06 failed the FFI applicability gate in both versions: its wrapper has an unsafe callee call but no pointer-operation token matched by that rule's source filter. The rejected wording therefore demonstrates no blind held-out recall improvement.

There were **zero reported or below-floor findings on non-seed units** in all four seeded scans. Additional cross-rule below-floor findings occurred on seed units only: H04 precondition 0.24 in frozen run 1 and 0.60 in the baseline; H04 unwind 0.48/0.44/0.51 in frozen runs and 0.54 in the baseline; H02 unwind 0.27 in frozen run 1. These do not change reported recall.

All four historical checks used the full workspace and the corresponding installed pack's merged guidance:

```sh
jevlint check --config frozen.json --refresh-cache --format json --show-below-floor .
jevlint check --config original.json --refresh-cache --format json --show-below-floor .
```

## Extraction boundaries

Critical foreign contracts and ownership/state transitions are in the primary functions wherever they are needed to decide the case. These rules do not request extended type comparison because their fixtures carry concrete std/API semantics locally. Unknown foreign semantics are not inferred from names; context-only defects are not findings against the primary item. Source filters are cheap applicability gates, not proofs of a defect.
