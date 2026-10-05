# Rust tooling baseline research for Jevlint

Research/proposal only. Commands below are suggestions, not executed checks. Sources were read live on 2026-10-04. `stable`, `latest`, and GitHub branch URLs are moving sources: eventual pack metadata should record reviewed toolchain/lint versions, not promise all Rust releases share this inventory.

## 1. What is normative, what is policy

- **Language/compiler requirements:** successful compilation and Rust's safety requirements are actual requirements. A warning is not automatically a language prohibition. Rustc distinguishes allow/expect/warn/force-warn/deny/forbid; borrow checking is not a stylistic rule. [Rustc lint levels](https://doc.rust-lang.org/rustc/lints/levels.html).
- **Official ecosystem defaults:** rustfmt's default style follows the Rust Style Guide; Clippy defaults comprise correctness, suspicious, complexity, perf, and style. Correctness is deny-by-default, the other default groups warn. The style group is explicitly opinionated. [Style Guide](https://doc.rust-lang.org/style-guide/index.html), [Clippy groups](https://doc.rust-lang.org/clippy/lints.html).
- **Recommended project policy, not Rust law:** warning-clean CI, documented public APIs, target/feature matrices, tests and doctests. Cargo's CI guide explicitly recognizes tradeoffs and limited warning compatibility guarantees. [Cargo CI](https://doc.rust-lang.org/cargo/guide/continuous-integration.html#checking-for-warnings).
- **Conditional/API recommendations:** Rust API Guidelines are recommendations, primarily for reusable public APIs, not a mandate to impose all public-crate conventions on every private application helper. Their examples guideline explicitly says apply within reason and allows linking to an applicable example elsewhere. [API checklist](https://rust-lang.github.io/api-guidelines/checklist.html), [documentation](https://rust-lang.github.io/api-guidelines/documentation.html).
- **Source-age conflict matters:** API Guidelines still list `authors` as common metadata; the current Cargo manifest reference marks `authors` deprecated. Do not turn the old checklist into a new mandatory authors-field rule. `homepage` should not duplicate repository/docs URLs, and `documentation` is unnecessary for ordinary docs.rs hosting. [Current manifest](https://doc.rust-lang.org/cargo/reference/manifest.html#the-authors-field), [homepage](https://doc.rust-lang.org/cargo/reference/manifest.html#the-homepage-field).

## 2. Boring deterministic core

### Prerequisites and reproducibility

Use the project's declared edition/MSRV, existing formatting/lint policy, and supported platforms. Pin the blocking CI toolchain and external tool versions; update deliberately. Separate a latest-stable canary if desired. Pinning avoids surprise new lint failures, but must not become indefinite avoidance of compiler/security updates. Set `package.rust-version` to the real supported version; it is not evidence that the crate actually works on that version. [MSRV documentation](https://doc.rust-lang.org/cargo/reference/rust-version.html), [CI MSRV verification](https://doc.rust-lang.org/cargo/guide/continuous-integration.html#verifying-rust-version).

For locked CI, an intentional `Cargo.lock` must already exist. `--locked` fails if absent or resolution changes. It fixes dependency selection, not OS/compiler/test nondeterminism. Published-library consumers resolve dependencies separately, so a locked maintainer build is not proof that every supported dependency combination works. Latest-dependency scheduled jobs are separate from locked PR jobs. `--frozen` also disables network access; it needs populated caches and can hide fresh advisory information. [Cargo check options](https://doc.rust-lang.org/cargo/commands/cargo-check.html#manifest-options), [latest-dependency CI](https://doc.rust-lang.org/cargo/guide/continuous-integration.html#verifying-latest-dependencies).

### Suggested default-feature host lane

Run under the project's selected/pinned development toolchain, not necessarily the MSRV. These are alternatives/recipes, not instructions to run in this investigation:

```sh
cargo fmt --all -- --check
cargo check --workspace --all-targets --locked
cargo build --workspace --locked
cargo test --workspace --locked
cargo doc --workspace --no-deps --locked
cargo clippy --workspace --all-targets --locked
```

- `cargo fmt --all -- --check` checks workspace formatting without rewriting. Prefer rustfmt defaults unless the project intentionally configures otherwise. Rustfmt has macro/comment/fragment limitations and some configuration is nightly-only. Do not duplicate indentation, line wrapping, brace layout, or import formatting as Jev rules. When using standalone rustfmt and Cargo fmt together, explicitly align parsing edition and style edition if needed; Cargo fmt infers from manifests. [Rustfmt README](https://github.com/rust-lang/rustfmt#verifying-code-is-formatted), [style editions](https://github.com/rust-lang/rustfmt#style-editions).
- `check --all-targets` includes library, binary, tests, benches, examples; **not every target triple**, not every feature configuration, and not doctests. Cargo check skips code generation, so it cannot catch diagnostics emitted only there. A build/test lane remains necessary. Do not mechanically add a redundant build if the existing real test/build pipeline already exercises the artifacts that matter. [Cargo check](https://doc.rust-lang.org/cargo/commands/cargo-check.html#description).
- Plain `cargo test --workspace` runs default-selected unit/integration tests and library doctests, and normally compiles examples. Selection and manifest flags (`test = false`, `doctest = false`, required features) can exclude targets. **`cargo test --all-targets` is not equivalent:** it selects lib/bins/tests/benches/examples, not doctests. If using it for explicit all-target coverage, separately run `cargo test --workspace --doc --locked`. Cross-target test execution needs a runner or compatible machine; `--no-run` compiles but does not test behavior. [Cargo test](https://doc.rust-lang.org/cargo/commands/cargo-test.html#target-selection).
- `cargo doc --no-deps` checks own documentation; Clippy/rustc do not run most rustdoc-only lints. Default rustdoc warnings include broken/private intra-doc links and malformed code-block attributes. `missing_docs` is allow-by-default and available to rustc too. Opt in for a documented library, not automatically every private app item. [Rustdoc lints](https://doc.rust-lang.org/rustdoc/lints.html).
- Doctests keep examples compiling/running; assertions are needed to test claimed results. `ignore` skips tests, `no_run` compiles without execution, and `compile_fail` establishes rejection rather than runtime behavior. These annotations can be valid for OS/service-dependent examples or negative API examples. Do not label every ignored/no-run example a defect. [Rustdoc documentation tests](https://doc.rust-lang.org/rustdoc/write-documentation/documentation-tests.html).

### Warning enforcement: CI separately from crate policy

Current official Clippy/Cargo docs introduce a version-sensitive change: **Cargo 1.97+ recommends `build.warnings`** instead of injecting `-D warnings`, avoiding cache invalidation from changed compiler flags.

```sh
# Cargo >= 1.97, CI-only environment policy
CARGO_BUILD_WARNINGS=deny cargo clippy --workspace --all-targets --locked
CARGO_BUILD_WARNINGS=deny cargo check --workspace --all-targets --locked

# Older Cargo alternative, not a command to combine with the above
cargo clippy --workspace --all-targets --locked -- -D warnings

# Explicit documentation warning policy (works independently of Clippy)
RUSTDOCFLAGS="-D warnings" cargo doc --workspace --no-deps --locked
```

`build.warnings` affects configurable lint warnings for local packages, not all imaginable warnings (non-lint warnings and dependency warnings visible via verbosity remain distinct). Read versioned docs for the selected Cargo. These recipes use POSIX-shell environment assignment syntax; adapt for PowerShell/Windows runners. [Clippy current usage](https://doc.rust-lang.org/clippy/usage.html#elevating-warning-to-errors), [Cargo build.warnings](https://doc.rust-lang.org/cargo/reference/config.html#buildwarnings).

Do not bake `#![deny(warnings)]`/`#![forbid(warnings)]` into every library as an alleged ecosystem rule. Keep strict warning escalation in controlled CI; toolchain upgrades can introduce warnings. Cargo generally caps non-path dependency lints, while workspace/path dependencies can remain relevant. Ordinary warn-level crate policy permits deliberate narrow exceptions. `#[expect(..., reason = "...")]` can make a justified suppression self-checking where MSRV supports it; `allow` remains valid for persistent policy or old toolchains. [Cargo manifest lints](https://doc.rust-lang.org/cargo/reference/manifest.html#the-lints-section), [expect semantics](https://doc.rust-lang.org/rustc/lints/levels.html#expect).

### Minimal opt-in configuration suggestions

No config is needed to run the default Clippy groups. If recording explicit policy on Cargo >= 1.74:

```toml
[lints.rust]
unsafe_op_in_unsafe_fn = "warn"

[lints.clippy]
all = { level = "warn", priority = -1 }
correctness = { level = "deny", priority = 0 }
```

Do **not** lower `clippy::correctness` from its default deny merely to spell out `all`: either omit `all` entirely (simplest), or also set `correctness = { level = "deny", priority = 0 }` if explicit group levels are desired. Group priority must be lower than individual exceptions. For a workspace, use `[workspace.lints...]` and opt member packages in with `[lints] workspace = true` rather than assuming inheritance is automatic. Cargo's lints tables are respected from 1.74; older MSRV projects can use source attributes. [Manifest lint configuration](https://doc.rust-lang.org/cargo/reference/manifest.html#the-lints-section), [workspace lints](https://doc.rust-lang.org/cargo/reference/workspaces.html#the-lints-table).

Potential library/unsafe overlays, adopted individually rather than as universal mandates:

```toml
[lints.rust]
missing_docs = "warn"
unsafe_op_in_unsafe_fn = "warn"

[lints.clippy]
missing_errors_doc = "warn"
missing_panics_doc = "warn"
undocumented_unsafe_blocks = "warn"
```

`missing_safety_doc` already warns by default. `unsafe_op_in_unsafe_fn` warns by default in edition 2024 and can be enabled in earlier editions. It requires an explicit unsafe block, **not a proof that its operations are sound**. A blanket `unsafe_code = "forbid"` is an optional no-unsafe project policy, not a universal Rust best practice. [Edition guide](https://doc.rust-lang.org/edition-guide/rust-2024/unsafe-op-in-unsafe-fn.html).

Clippy MSRV defaults to `package.rust-version` for lints supporting that configuration; an explicit `clippy.toml` `msrv = "..."` is available when needed. Not every lint recognizes MSRV, and Clippy is not an actual old-compiler compatibility test. `avoid-breaking-exported-api` protects some suggestions, not all changes. [Clippy configuration](https://doc.rust-lang.org/clippy/configuration.html#specifying-the-minimum-supported-rust-version), [lint configuration table](https://doc.rust-lang.org/clippy/lint_configuration.html).

### Group policy

| Group | Baseline position | Authoritative caveat |
|---|---|---|
| correctness | Keep deny default | Designed to identify outright wrong/useless code; treat suppression as exceptional, not a general escape hatch |
| suspicious | Keep warn default | Intentional code can trigger; narrow justified exceptions are legitimate |
| complexity/perf | Keep warn defaults | Suggestions are not performance measurements; semantic/API constraints still matter |
| style | Keep ecosystem default, permit project exceptions | Official docs expressly identify this default group as most opinionated |
| pedantic | Opt-in/cherry-pick | Production-usable but intentionally opinionated/false-positive-prone; not a universal baseline |
| restriction | Never enable whole group as baseline | Officially discouraged; lints may contradict each other; choose by project/domain |
| nursery | Opt-in/cherry-pick | Official docs say buggy/needs more work, discourage whole-group enablement |
| cargo | Publishing/manifest overlay | Choose useful metadata/dependency policy lints, not every private project |

Source for all rows: [Clippy's Lints](https://doc.rust-lang.org/clippy/lints.html).

## 3. Feature, target, profile, and MSRV coverage

- Start with default features, then the supported no-default configuration if there is one, plus meaningful supported combinations. `--all-features` is useful only if all features can coexist on that toolchain/platform. Cargo advises additive features and avoiding mutually exclusive ones, but existing legitimate exclusive backends/nightly-only features make blindly requiring all-features builds incorrect. Enumerate valid combinations and test invalid combinations only when an explicit diagnostic is the contract. [Feature unification](https://doc.rust-lang.org/cargo/reference/features.html#feature-unification), [exclusive features](https://doc.rust-lang.org/cargo/reference/features.html#mutually-exclusive-features).
- All-features alone does not test disabled-feature code. Workspace feature unification can hide failures in isolated package use. Use explicit package/feature lanes when required; cargo-hack is an optional matrix aid linked by Cargo, not a replacement for deciding supported configurations. [Cargo CI/MSRV](https://doc.rust-lang.org/cargo/guide/continuous-integration.html#verifying-rust-version).
- `--all-targets` means Cargo artifact kinds, not Linux/Windows/macOS, 32/64-bit, wasm, or embedded. `--target <triple>` activates that target's cfg/dependency paths. Check declared supported targets, and execute on real runners/emulators where behavior matters. `no_std` may require `--lib`, selected features, and a non-host target; host-only development/tests can legitimately require std. [Target selection](https://doc.rust-lang.org/cargo/commands/cargo-check.html#target-selection), [compilation target](https://doc.rust-lang.org/cargo/commands/cargo-check.html#compilation-options).
- Run actual compilation on the declared MSRV. Prefer per-package lanes if workspace MSRVs differ; tooling and dev dependencies can need newer versions, but the claimed support boundary must be documented and tested rather than quietly skipped. [MSRV support expectations](https://doc.rust-lang.org/cargo/reference/rust-version.html#support-expectations).
- A release-profile test/build lane is conditional on relevant optimized/profile behavior (e.g. overflow/debug-assertion assumptions); neither always require it nor assume debug testing covers it. Profiles can change panic/overflow/debug settings. [Cargo profiles](https://doc.rust-lang.org/cargo/reference/profiles.html).
- Keep `unexpected_cfgs` meaningful; declare static custom cfg names, or emit `cargo::rustc-check-cfg` alongside build-script cfgs. Do not blanket-silence it just to enable `loom`/`fuzzing`/custom flags. Verify required cfg support against MSRV. [Cargo-specific cfg checking](https://doc.rust-lang.org/rustc/check-cfg/cargo-specifics.html).

## 4. Existing lint overlap: do not reimplement these as semantic Jev rules

The following links target the **stable Clippy catalogue** read live, with exact group/default level. Availability depends on chosen toolchain. They are deterministic implementation checks, not exhaustive semantic analysis. Jev should explain any residual contextual finding beyond the lint's actual trigger.

| Topic | Existing check and group/default | Boundary / remaining contextual work |
|---|---|---|
| Copy/clone | [`clone_on_copy`](https://rust-lang.github.io/rust-clippy/stable/index.html#clone_on_copy), complexity/warn; [`redundant_clone`](https://rust-lang.github.io/rust-clippy/stable/index.html#redundant_clone), nursery/allow | Do not ban clone. Redundant clone analysis is conservative; shared ownership/snapshot/task lifetimes may require clones |
| Rc/Arc clones | [`clone_on_ref_ptr`](https://rust-lang.github.io/rust-clippy/stable/index.html#clone_on_ref_ptr), restriction/allow | `Arc::clone(&x)` vs `x.clone()` is explicitness style, not less allocation or different semantics |
| Borrowed parameters | [`ptr_arg`](https://rust-lang.github.io/rust-clippy/stable/index.html#ptr_arg), style/warn; [`needless_pass_by_value`](https://rust-lang.github.io/rust-clippy/stable/index.html#needless_pass_by_value), pedantic/allow; [`needless_borrow`](https://rust-lang.github.io/rust-clippy/stable/index.html#needless_borrow), style/warn | Requires actual ownership/capacity/API/trait context; exported API changes may break callers; extra borrows can affect trait selection |
| Avoidable owned conversion | [`unnecessary_to_owned`](https://rust-lang.github.io/rust-clippy/stable/index.html#unnecessary_to_owned), perf/warn; [`needless_collect`](https://rust-lang.github.io/rust-clippy/stable/index.html#needless_collect), nursery/allow | Broader hot-path copying/repeated materialization needs dataflow and measured significance, not clone count |
| Indirection/layout | [`box_collection`](https://rust-lang.github.io/rust-clippy/stable/index.html#box_collection), perf/warn; [`vec_box`](https://rust-lang.github.io/rust-clippy/stable/index.html#vec_box), complexity/warn; [`large_enum_variant`](https://rust-lang.github.io/rust-clippy/stable/index.html#large_enum_variant), perf/warn | Stable addresses, ABI, pinning, large elements, API compatibility can justify boxing; boxing a large variant is not automatically a net performance win |
| Stack/future size | [`large_stack_arrays`](https://rust-lang.github.io/rust-clippy/stable/index.html#large_stack_arrays), pedantic/allow; [`large_stack_frames`](https://rust-lang.github.io/rust-clippy/stable/index.html#large_stack_frames), nursery/allow; [`large_futures`](https://rust-lang.github.io/rust-clippy/stable/index.html#large_futures), pedantic/allow | Thresholds are configurable heuristics, not universal stack budgets; platforms/executors matter |
| Unsafe boundary | [`not_unsafe_ptr_arg_deref`](https://rust-lang.github.io/rust-clippy/stable/index.html#not_unsafe_ptr_arg_deref), correctness/deny; [`mut_from_ref`](https://rust-lang.github.io/rust-clippy/stable/index.html#mut_from_ref), correctness/deny | Compiler and narrow lint patterns do not establish soundness of every safe wrapper |
| Initialization | [`uninit_vec`](https://rust-lang.github.io/rust-clippy/stable/index.html#uninit_vec), correctness/deny | Detects adjacent set_len/allocation patterns; cross-function initialization/cleanup invariants remain contextual |
| Unsafe documentation | [`missing_safety_doc`](https://rust-lang.github.io/rust-clippy/stable/index.html#missing_safety_doc), style/warn; [`undocumented_unsafe_blocks`](https://rust-lang.github.io/rust-clippy/stable/index.html#undocumented_unsafe_blocks), restriction/allow | Section/comment presence is not sufficient: actual alignment, validity, aliasing, lifetime, initialization obligations must be met/explained |
| Panic methods | [`unwrap_used`](https://rust-lang.github.io/rust-clippy/stable/index.html#unwrap_used), [`expect_used`](https://rust-lang.github.io/rust-clippy/stable/index.html#expect_used), [`panic`](https://rust-lang.github.io/rust-clippy/stable/index.html#panic), restriction/allow | No blanket ban. Tests/examples/invariants/startup policy can legitimately panic; fallible trust-boundary operations need contextual analysis |
| Guaranteed/unnecessary unwrap | [`panicking_unwrap`](https://rust-lang.github.io/rust-clippy/stable/index.html#panicking_unwrap), correctness/deny; [`unnecessary_unwrap`](https://rust-lang.github.io/rust-clippy/stable/index.html#unnecessary_unwrap), complexity/warn | Limited dataflow; neither proves all other unwraps safe or unsafe |
| Error information | [`result_unit_err`](https://rust-lang.github.io/rust-clippy/stable/index.html#result_unit_err), style/warn | Does not judge adequate domain information, source chaining, contextual diagnostics, or intentional control-flow enums |
| Error/panic docs | [`missing_errors_doc`](https://rust-lang.github.io/rust-clippy/stable/index.html#missing_errors_doc), [`missing_panics_doc`](https://rust-lang.github.io/rust-clippy/stable/index.html#missing_panics_doc), pedantic/allow | Checks presence/recognized patterns; docs may be inaccurate/incomplete. API Guidelines exempt excessive caller-provided panic documentation |
| From/TryFrom | [`from_over_into`](https://rust-lang.github.io/rust-clippy/stable/index.html#from_over_into), style/warn; [`fallible_impl_from`](https://rust-lang.github.io/rust-clippy/stable/index.html#fallible_impl_from), nursery/allow | Contextual lossless/infallible/value-preserving conversions go beyond syntactic panic/unwrap detection |
| Constructor/common APIs | [`new_without_default`](https://rust-lang.github.io/rust-clippy/stable/index.html#new_without_default), [`len_without_is_empty`](https://rust-lang.github.io/rust-clippy/stable/index.html#len_without_is_empty), [`should_implement_trait`](https://rust-lang.github.io/rust-clippy/stable/index.html#should_implement_trait), style/warn; [`iter_without_into_iter`](https://rust-lang.github.io/rust-clippy/stable/index.html#iter_without_into_iter), pedantic/allow | No universal requirement to derive every trait/create a builder; implement appropriate standard interfaces |
| Trait consistency | [`derive_ord_xor_partial_ord`](https://rust-lang.github.io/rust-clippy/stable/index.html#derive_ord_xor_partial_ord), [`derived_hash_with_manual_eq`](https://rust-lang.github.io/rust-clippy/stable/index.html#derived_hash_with_manual_eq), correctness/deny; [`non_canonical_clone_impl`](https://rust-lang.github.io/rust-clippy/stable/index.html#non_canonical_clone_impl), [`non_canonical_partial_ord_impl`](https://rust-lang.github.io/rust-clippy/stable/index.html#non_canonical_partial_ord_impl), suspicious/warn | Pattern checks, not proof of Eq/Hash/Ord laws over arbitrary inputs |
| Mutable map keys | [`mutable_key_type`](https://rust-lang.github.io/rust-clippy/stable/index.html#mutable_key_type), suspicious/warn | Interior mutable fields omitted from Hash/Ord can be legitimate; inspect actual key semantics |
| Async guard lifetime | [`await_holding_lock`](https://rust-lang.github.io/rust-clippy/stable/index.html#await_holding_lock), [`await_holding_refcell_ref`](https://rust-lang.github.io/rust-clippy/stable/index.html#await_holding_refcell_ref), suspicious/warn | Explicit-drop false positives are documented; scoped guards avoid them. Do not prohibit every sync mutex in async code |
| Domain guard types | [`await_holding_invalid_type`](https://rust-lang.github.io/rust-clippy/stable/index.html#await_holding_invalid_type), suspicious/warn | Only triggers on configured forbidden-across-await types; not universal resource lifecycle analysis |
| Discarded future | [`let_underscore_future`](https://rust-lang.github.io/rust-clippy/stable/index.html#let_underscore_future), suspicious/warn; rustc `unused_must_use` | Future work normally needs polling; dropped task handles can intentionally detach depending on runtime/type semantics |
| No await in async fn | [`unused_async`](https://rust-lang.github.io/rust-clippy/stable/index.html#unused_async), pedantic/allow | Interface/trait uniformity and lazy execution can justify async shape |
| Partial I/O | [`unused_io_amount`](https://rust-lang.github.io/rust-clippy/stable/index.html#unused_io_amount), correctness/deny | Read/write can be partial, including recognized Tokio/futures APIs; broader framing/truncation/cancellation still needs context |
| Arc/thread choice | [`arc_with_non_send_sync`](https://rust-lang.github.io/rust-clippy/stable/index.html#arc_with_non_send_sync), suspicious/warn; [`mutex_atomic`](https://rust-lang.github.io/rust-clippy/stable/index.html#mutex_atomic), restriction/allow | Shared ownership/invariants/locking semantics can justify choices. Neither is a mandate to introduce atomics |
| Manual copy | [`manual_memcpy`](https://rust-lang.github.io/rust-clippy/stable/index.html#manual_memcpy), perf/warn | Prefer applicable safe slice operations; wider algorithmic cost needs evidence |
| Function shape | [`too_many_arguments`](https://rust-lang.github.io/rust-clippy/stable/index.html#too_many_arguments), [`type_complexity`](https://rust-lang.github.io/rust-clippy/stable/index.html#type_complexity), complexity/warn; [`too_many_lines`](https://rust-lang.github.io/rust-clippy/stable/index.html#too_many_lines), pedantic/allow; [`unnecessary_wraps`](https://rust-lang.github.io/rust-clippy/stable/index.html#unnecessary_wraps), pedantic/allow | Configurable heuristics (arguments default 7; lines default 100) are **not universal Rust limits**. Semantic cohesion is different from size |
| Context-specific forbidden APIs | [`disallowed_methods`](https://rust-lang.github.io/rust-clippy/stable/index.html#disallowed_methods), style/warn but inactive without configured methods | Can enforce a project-specific API policy; cannot by itself decide whether a blocking operation is appropriate at a specific call site |
| MSRV | [`incompatible_msrv`](https://rust-lang.github.io/rust-clippy/stable/index.html#incompatible_msrv), suspicious/warn | Useful std API availability checks, not complete MSRV build verification |
| Basic linting | Rustc defaults (unused/dead code, naming, must-use, unexpected cfg); rustfmt; rustdoc defaults | Do not create Jev rules that imitate deterministic syntax/format/name checks |

Clippy's `all` does not mean every listed lint. Enable optional entries only when their intended policy applies. False positives and version availability must stay visible in the proposal.

## 5. Optional established tool layers

These are mostly non-core ecosystem tools, not Rust language requirements. Select for actual risk, use existing tooling if already present, and record what each result does and does not establish.

| Layer | Suggested command / applicability | Limitations and authoritative source |
|---|---|---|
| Known dependency advisories | `cargo audit`; lockfile-based PR/scheduled check for libraries/apps | RustSec database detects **reported** vulnerabilities, not unknown vulnerabilities/malicious code or reachability. With no lockfile it can generate one via Cargo; run only trusted projects. Review ignored advisories with concrete applicability rationale. [RustSec official README](https://github.com/rustsec/rustsec/tree/main/cargo-audit#readme) |
| Dependency policy | `cargo deny --workspace --locked check`; explicit reviewed `deny.toml` policy | Checks licenses, bans/duplicates, advisories, allowed sources. Restrictions are organizational/license policy, not universal Rust norms; duplicate crate versions can be legitimate. It overlaps audit's advisory layer—one can suffice. Choose feature/target graph coverage intentionally; offline mode prevents advisory refresh. Passing is not proof dependencies are safe or licenses legally approved. [Checks](https://embarkstudios.github.io/cargo-deny/checks/index.html), [CLI graph controls](https://embarkstudios.github.io/cargo-deny/cli/common.html), [advisories](https://embarkstudios.github.io/cargo-deny/checks/advisories/index.html) |
| Miri | `cargo +nightly miri test` on targeted supported tests; especially unsafe/FFI-adjacent pure Rust data structures, ownership/provenance, atomics, endian-sensitive logic | Nightly component, slower interpreter; most FFI/network/platform APIs unsupported. Finds UB in exercised executions, not proof of soundness for arbitrary safe callers; scheduling/weak-memory coverage incomplete. Unsupported operations are not automatically bugs; isolate supported tests without masking real defects. Experimental aliasing models must not be presented as final Rust law. Pin compatible nightly; consider seeds/other targets when relevant. [Official Miri README](https://github.com/rust-lang/miri#readme) |
| Sanitizers | Example supported ASan test lane: `RUSTFLAGS="-Zsanitizer=address" RUSTDOCFLAGS="-Zsanitizer=address" cargo +nightly test -Zbuild-std --target x86_64-unknown-linux-gnu` | Requires nightly, matching components including rust-src when rebuilding std, supported target and toolchain/linker setup. ASan memory errors; MSan uninitialized reads; TSan races; choose per risk. Native/FFI instrumentation needs coordination; uninstru­mented code weakens coverage. Different sanitizers have different targets/requirements; don't copy ASan flags to TSan. Runtime overhead and exercised-path limits remain. [Rust Unstable Book sanitizers](https://doc.rust-lang.org/nightly/unstable-book/compiler-flags/sanitizer.html) |
| Loom | After integrating loom replacement primitives/model tests: `RUSTFLAGS="--cfg loom" cargo test --test loom_my_struct --release` | Purpose-built concurrent data structures/atomic protocols. All relevant nondeterminism must be modeled; ordinary std/dependency primitives are invisible. Large models need bounds and are not exhaustive if bounded; Loom also cannot model every relaxed-memory reordering. Stable-capable ecosystem library, not automatically a nightly tool. Register custom cfg where required. [Official Loom docs](https://docs.rs/loom/latest/loom/) |
| Coverage-guided fuzzing | Existing harness: `cargo +nightly fuzz run parser` (replace parser with actual target); parsers/decoders/protocols/unsafe input-dependent paths | cargo-fuzz/libFuzzer needs supported platform, C++ toolchain and nightly sanitizer support. Requires meaningful harness/oracles/corpus; no crashes is not correctness/security proof. Feature flags affect fuzz-target crate and must be forwarded to tested crate. Persist/minimize regressions and replay; bounded CI/scheduled campaigns are policy choices. [Rust Fuzz setup](https://rust-fuzz.github.io/book/cargo-fuzz/setup.html), [guide](https://rust-fuzz.github.io/book/cargo-fuzz/guide.html) |
| Property-based tests | Existing proptest/QuickCheck tests via `cargo test`; useful for trait laws, round trips, parser invariants, state-machine properties | Generated tests complement fixed regression/edge-case tests; appropriate strategies/oracles needed, not an exhaustive mathematical proof. No new dependency solely to advertise coverage. [Proptest author docs](https://proptest-rs.github.io/proptest/intro.html) |
| Public API compatibility | `cargo semver-checks` against explicit released/revision baseline; libraries before releases | Third-party maintained tool, rustdoc JSON/version coupling; current default feature heuristics are not exhaustive and target-specific APIs need target lanes. Not every behavioral or type/generic break is detected; review Cargo SemVer contract too. [Author README](https://github.com/obi1kenobi/cargo-semver-checks#readme), [Cargo SemVer](https://doc.rust-lang.org/cargo/reference/semver.html) |
| Package readiness | `cargo package --locked` for published packages; optionally `cargo package --list` to inspect included files | Cargo extracts/builds package and checks source mutation; does not publish. Path-only deps/package missing files differ from working checkout. Do not use `--no-verify` then claim package build checked. Not needed for every internal app. [Cargo package](https://doc.rust-lang.org/cargo/commands/cargo-package.html) |

Supply-chain checks are intentionally responsive to changing advisory data: the same code/lockfile can acquire a new failure. For reproducible historical evidence record tool/config/lockfile/database revision, but do not freeze advisory data forever in a live security lane. None of fmt, Clippy, tests, audit/deny, Miri, sanitizers, Loom, or fuzzing proves general correctness or dependency safety.

## 6. Implications for Jev rule selection

1. **Do not build a second rustfmt/Clippy/rustc.** Keep deterministic command checks as baseline guidance or separate tool integration. Duplicate reports should be suppressed/merged by actual overlapping finding, not merely matching a broad topic. Optional Clippy lint being disabled does not automatically justify reimplementing it with an LLM.
2. **Separate tool failure from semantic defect.** Missing component, mutually exclusive features, unsupported Miri operation, or unavailable cross runner are coverage/configuration limitations. They are not proof the source violates Rust practice. Likewise, green tooling is evidence of the executed checks only.
3. **Use contextual residuals.** Examples: an existing SAFETY comment that omits an actual validity/aliasing obligation; public docs promising stronger panic/error behavior than implementation; trust-boundary panic despite domain-recoverable failure; Tokio blocking work in an executor context (runtime docs needed); ownership/copying causing demonstrated avoidable work; meaningful regression coverage for exposed behavior. Avoid universal limits and broad bans on clone/unwrap/unsafe/traits/dyn.
4. **Carry explicit scope.** Rule metadata should identify library/application, tests/doctests/generated code, runtime, feature cfg, target, MSRV, and evidence needed. Tests legitimately use panic/assert/unwrap; API docs are different from private-helper comments. Avoid requiring Loom/fuzzing/Miri for ordinary pure safe helpers.
5. **Capability gate (parent-provided scout finding):** current Jev extracts Rust functions/types and limited same-file related type/impl context, but does not provide compiler typing, name resolution, or macro expansion. Method calls and `Type::qualified` calls do not resolve into direct callee context. Therefore the contextual candidates below are **opt-in/manual-review or deferred unless all necessary evidence is explicitly available**. Do not ship global reachability, trait semantics, inferred runtime identity, or unsafe soundness judgments as reliable defaults. A visible local contradiction can support a narrowly scoped candidate, but unknown callees are not evidence of absence of checks. Parent is separately verifying attribute/documentation extraction and fixture support; these proposals do not assume that support.

### Contextual fixture concepts (not proposed deterministic duplicates)

| Candidate | Trigger and rationale | Exceptions/context required | Violating / passing fixture concept |
|---|---|---|---|
| Safety argument fails to establish used unsafe preconditions | Unsafe operation is reachable while required validity/lifetime/initialization/aliasing conditions are neither established by code nor delegated in correct unsafe API contract; a comment's existence is insufficient | Need complete wrapper/callers/types/ownership and exact operation docs. Raw pointers proven valid by external contract are legitimate; do not infer unsoundness merely from terse wording | Violation: safe slice wrapper checks length but constructs misaligned/uninitialized `&T` from arbitrary bytes with a generic SAFETY comment. Pass: wrapper validates layout/representation or stays with byte reads; unsafe API states caller obligations and code respects them |
| Public failure contract is inaccurate | Public docs state no panic or particular `Err` conditions but implementation has reachable contradictory behavior; consumer predictability matters | Need public contract, input domain, full callgraph and caller-supplied trait behavior. API Guidelines say not every conceivable caller-induced panic needs listing. No rule demanding universal `# Panics`/`# Errors` duplicates | Violation: docs promise malformed input returns Err, parser unwraps fallible parse. Pass: parse propagates mapped error, or explicitly documents/validates genuinely intended panic contract |
| Test claim skips the behavior it advertises | Existing regression/example claims an observable result but only compiles/constructs inputs and never exercises/asserts that behavior, or ignored doctest leaves a claimed portable example unverified without reason | Need actual test intent, existing adjacent tests, runtime/platform needs; smoke/compile-fail/no-run tests are legitimate when matching intent. Do not require an assertion in every test or minimum coverage percentage | Violation: regression named rejects_truncated_frame invokes parser and discards returned Result. Pass: asserts exact rejection, or compile-only fixture deliberately tests API type availability |

Sources for these contextual candidates: [API failure documentation](https://rust-lang.github.io/api-guidelines/documentation.html#c-failure), [doctest pass conditions/assertions](https://doc.rust-lang.org/rustdoc/write-documentation/documentation-tests.html#passing-or-failing-a-doctest), [Miri limits](https://github.com/rust-lang/miri#readme), and exact unsafe operation's std/Reference contract supplied in the memory/unsafe research slice.

During the original research phase, no checks were executed, packages installed, project code changed, providers called, or implementation pack written.