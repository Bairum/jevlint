# Bairum/rust-unsafe

Opt-in semantic **warnings**, not a Rust soundness checker. Four rules cover locally visible std raw-storage contracts, callback/unwind state, foreign-pointer contracts, and PyO3/CPython reference ownership. Neither a pass nor an abstention guarantees soundness, exception safety, or absence of UB. Run rustc and Clippy separately; this pack does not replace Miri, sanitizers or human review.

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

From that consuming project, install from your committed local checkout:

```sh
jevlint plugin install /absolute/path/to/jevlint#packs/rust-unsafe
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

The index contains 62 cases: 17 std precondition cases, 14 unwind cases, 14 FFI cases and 17 PyO3 cases. The real pairs are `real-fyrox-bytes-*`, `real-qwutils-insert-*`, `real-glib-variant-out-param-*` and `real-pyo3-weakref-*` under their respective rule directories.

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

The compile-only runner successfully checked all 62 isolated Rust 2021 fixture targets on 2026-10-04, with no diagnostics emitted. Faulty fixtures were never executed. Three real provider runs (`python3 scripts/check-rust-packs.py --eval --repeat 3`) measured fixture recall and specificity; the per-run results below treat below-floor failures as not reported.

| Rule | Recall @0.8 | Specificity @0.8 | Recommended override | Recall @override | Specificity @override |
| --- | --- | --- | --- | --- | --- |
| `rust-unsafe-precondition-contract` | 0.00 | 1.00 | 0.35 | 0.20 | 1.00 |
| `rust-unsafe-unwind-state` | 0.00 | 1.00 | 0.65 | 0.07 | 1.00 |
| `rust-unsafe-ffi-pointer-contract` | 0.24 | 1.00 | 0.75 | 0.43 | 1.00 |
| `rust-unsafe-pyo3-reference-ownership` | 0.17 | 1.00 | 0.35 | 0.61 | 0.91 |

Precondition-contract recall `0.20` and unwind-state recall `0.07` are not yet useful even at the recommended overrides. FFI and PyO3 recall also remain incomplete; this calibration does not establish soundness. Recall is measured on fixtures; real-project precision remains unmeasured.

Global `minConfidence` stays `0.8`; the pack has no rule-level `minConfidence`. Only consuming projects may opt into recommended rule overlays, such as `{"id": "rust-unsafe-precondition-contract", "minConfidence": 0.35}`. An overlay replaces the rule floor, which overrides the global floor. Recommendations were fitted on the same fixtures and are starting points, not guarantees. See [calibration method, caveats and override configuration](../../docs/rust/CALIBRATION.md).

Existing verdict-commentary was removed rather than preserved as evidence; no test-only fixtures are retained or relied on. No existing fixtures were deleted; all 12 original std cases retain their paths and verdicts.

## Extraction boundaries

Critical foreign contracts and ownership/state transitions are in the primary functions wherever they are needed to decide the case. These rules do not request extended type comparison because their fixtures carry concrete std/API semantics locally. Unknown foreign semantics are not inferred from names; context-only defects are not findings against the primary item. Source filters are cheap applicability gates, not proofs of a defect.
