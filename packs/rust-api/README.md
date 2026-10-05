# Bairum/rust-api

Opt-in API-contract warnings, not blanket public-API policy, soundness certification or performance guarantees. Common evidence/abstention guidance lives in `guidance.md` and is loaded through the pack manifest; descriptions contain only rule-specific policy.

## Rules and sourceMatch rationale

| Rule | Primary subject and filter | Contract |
| --- | --- | --- |
| `rust-api-default-consistency` | `kinds: ["type"]`, `context.types: true`; sourceMatch selects `struct`, `enum`, `union` or `type` declarations. | A material observable disagreement between zero-argument `new()` and `Default` when both claim the same baseline. No shared construction contract means pass. |
| `rust-api-trait-law-consistency` | `kinds: ["type"]`, `context.types: true`; the same declaration filter. | A concrete manual Eq equivalence-law counterexample or equal values feeding different Hash inputs. Unequal values may collide. |
| `rust-api-deref-contract` | `kinds: ["function"]`, `context.types: true`; sourceMatch selects `fn deref(`, including raw identifiers and generic signature spelling. | Implicit dereference work, state changes or failure contradicting documented transparent behavior. No transparency contract means pass. |

Constructor and trait consistency belong to the owning type, not each helper method. The primary declaration plus its supplied matching impls establishes that type's behavior. Their filters intentionally do not require `Default`, `Hash` or `new` text in the declaration: those spellings normally appear only in impl context, which sourceMatch does not inspect. This conservative gate handles qualified traits and imported aliases without falsely excluding legitimate types; unrelated declarations pass. Deref stays function-primary because the effect is in one dereference body, so findings stay on that operation rather than on a struct or an otherwise-correct coercion caller. The stable method name does not depend on how `Deref` was imported.

`context.types` supplies bounded same-file/cross-file declarations and matching impls, subject to exclusions. This is source context, not compiler type checking or macro expansion. Missing or ambiguous necessary evidence calls for abstention, never an invented contradiction. Context belonging only to another type is not a finding against the primary subject.

## Research references

| Rule | Research mapping | Primary sources |
| --- | --- | --- |
| `rust-api-default-consistency` | [API16](../../docs/rust/research/api.md#api16--default-and-empty-new-agree-when-both-mean-default-construction) | [C-CTOR](https://rust-lang.github.io/api-guidelines/predictability.html#c-ctor), [C-COMMON-TRAITS](https://rust-lang.github.io/api-guidelines/interoperability.html#c-common-traits). |
| `rust-api-trait-law-consistency` | [API15](../../docs/rust/research/api.md#api15--common-traits-should-fit-semantics-and-law-related-traits-must-agree) | [Hash and Eq](https://doc.rust-lang.org/std/hash/trait.Hash.html#hash-and-eq), [Eq laws](https://doc.rust-lang.org/std/cmp/trait.Eq.html), [PartialEq](https://doc.rust-lang.org/std/cmp/trait.PartialEq.html). |
| `rust-api-deref-contract` | [API14](../../docs/rust/research/api.md#api14--deref-is-transparent-pointer-like-behavior-not-inheritance) | [Deref advice](https://doc.rust-lang.org/std/ops/trait.Deref.html#when-to-implement-deref-or-derefmut), [fallibility](https://doc.rust-lang.org/std/ops/trait.Deref.html#fallibility), [C-DEREF](https://rust-lang.github.io/api-guidelines/predictability.html#c-deref). |

## Fixtures and real-world provenance

Each source is a standalone Rust 2021, std-only library fixture. Faulty fixtures are compile-only analysis inputs, never executed. `jevlint-evals.json` registers 35 cases: 13 constructor, 11 trait-law and 11 dereference cases. All contain a non-test primary unit selected by their rule's sourceMatch. Tests are structurally skipped by default; none of the prior fixtures was test-only, and no cases were removed. Existing dereference fixtures now put the relevant contract directly on the primary method.

- Constructor failures cover scalar baseline state, optional output and enum variants. Passes cover delegation, explicit input constructors, distinguished policies, equivalent allocation capacity and a domain-input `bind` without meaningful Default. `no_contract.rs` removes the same-baseline documentation from the original counterexample and expects pass.
- Trait-law failures cover case normalization, Eq reflexivity/transitivity, identity versus metadata, and representation-dependent hashing. Passes include coherent normalization, permitted collisions, Hash without Eq and floating PartialEq without Eq. There is no no-contract exception: implementing Eq/Hash itself establishes the unconditional trait laws, even without application documentation.
- Dereference failures cover observable state mutation, repeated linear work contrary to a constant-time contract, and failure on valid empty content. Passes cover cheap transparent access, programmer-error panic only after explicit invalidation, and nontransparent work in a named accessor. `no-contract.rs` removes all transparency documentation from the stateful counterexample and expects pass.

Every rule also has a realistic multi-unit fail file and clean neighbor, each at least 80 lines. The fail file has exactly one violating primary subject: `SchedulerOptions`, `ResourceKey`, or `RequestDocument::deref`, respectively. Comments document real contracts and never narrate expected verdicts.

| Rule / pair under its fixture directory | Public project and bug/fix | What was minimized or adapted |
| --- | --- | --- |
| default-consistency: `real-substrate-inherents-{bug,fixed}.rs` | Substrate, [PR #1561](https://github.com/paritytech/substrate/pull/1561), [patch](https://github.com/paritytech/substrate/pull/1561.diff). | Retains `CheckInherentsResult`'s `okay` baseline mismatch and the fix making `new` delegate to a successful Default. Replaces codec/InherentData with std storage and expands the faulty derived Default into its equivalent manual body. Shared-baseline documentation makes the strict review contract explicit; it is adapted, not an upstream quotation. |
| trait-law-consistency: `real-vecdeque-slices-{bug,fixed}.rs` | Rust std VecDeque, [issue #80303](https://github.com/rust-lang/rust/issues/80303), [PR #81170](https://github.com/rust-lang/rust/pull/81170), [fix commit](https://github.com/rust-lang/rust/commit/ae3a5153377f2271ba7dfe686a9b5bca1632c32b). | Replaces a ring buffer with two concrete byte segments. Equality compares logical sequence; faulty slice-by-slice hashing exposes representation boundaries. The fixed version hashes logical length and elements independently of segmentation. No ahash dependency or copied large upstream implementation. |
| deref-contract: `real-tokio-uring-slice-{bug,fixed}.rs` | tokio-uring, [issue #45](https://github.com/tokio-rs/tokio-uring/issues/45), [PR #52](https://github.com/tokio-rs/tokio-uring/pull/52). | Retains capacity-based slice end versus initialized Vec length, the valid empty-buffer dereference panic, and the correction clamping to initialized length. Uses safe std Vec storage instead of io-uring, generic IoBuf traits, pointers or DerefMut. Initialized-byte transparency documentation is adapted from the clarified issue/fix semantics. |

## Install and activate

Installation reads committed content. From a consuming project initialized with `jevlint init`, install a real committed local repository revision:

```sh
jevlint plugin install /absolute/path/to/jevlint#packs/rust-api
jevlint plugin list
```

The CLI records the resolved real commit pin in `jevlint.json`. Enable Rust explicitly with `"languages": {"rust": {}}`, preserving other settings and the generated packs entry. Project overlays may refine these optional warnings.

```sh
jevlint check .
jevlint eval --packs --rule rust-api-default-consistency
jevlint eval --packs --rule rust-api-trait-law-consistency
jevlint eval --packs --rule rust-api-deref-contract
jevlint plugin remove Bairum/rust-api
```

## Verification status

`python3 scripts/check-rust-packs.py --pack rust-api` compiled all 35 isolated Rust 2021 fixture targets without errors or warnings. Faulty fixtures were never executed. Three real provider runs (`python3 scripts/check-rust-packs.py --eval --repeat 3`) measured fixture recall and specificity; the per-run results below treat below-floor failures as not reported.

| Rule | Recall @0.8 | Specificity @0.8 | Recommended override | Recall @override | Specificity @override |
| --- | --- | --- | --- | --- | --- |
| `rust-api-default-consistency` | 0.00 | 1.00 | 0.30 | 0.67 | 1.00 |
| `rust-api-trait-law-consistency` | 0.07 | 1.00 | 0.60 | 0.40 | 1.00 |
| `rust-api-deref-contract` | 0.60 | 0.83 | — | 0.60 | 0.83 |

`rust-api-deref-contract` is false-positive-prone at `0.8` (specificity `0.83`). Constructor and trait-law recall remain limited even at their recommended overrides. These measurements are on fixtures, not real-project precision, which remains unmeasured.

Global `minConfidence` stays `0.8`; the pack has no rule-level `minConfidence`. Only consuming projects may opt into recommended rule overlays, such as `{"id": "rust-api-default-consistency", "minConfidence": 0.30}`. An overlay replaces the rule floor, which overrides the global floor; `—` means keep `0.8`. Recommendations were fitted on the same fixtures and are starting points, not guarantees. See [calibration method, caveats and override configuration](../../docs/rust/CALIBRATION.md).
