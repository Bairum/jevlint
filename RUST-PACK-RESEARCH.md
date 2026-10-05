# Rust rule-pack research

Researched 2026-10-04. This document preserves the original source-backed inventory and pack proposal, not a completed Rust audit. No Rust checks, example fixtures, provider evaluations, installations, or performance measurements were executed during that original research phase. Candidate names and priorities are proposed Jevlint policy, not official Rust lint names.

**Current implementation:** [Rust packs and usage](docs/rust/README.md).
**Complete original findings:** [four domain reports](docs/rust/README.md#preserved-research).
**Traceability and outstanding scope:** [coverage ledger](docs/rust/COVERAGE.md).
The ledger distinguishes delivered rules, native-tool guidance, verification,
partial coverage and deferred work; this proposal is not a completion checklist.

## 1. Recommended shape

Use three complementary layers:

1. **Deterministic Rust baseline:** rustc, rustfmt, Clippy, tests/doctests, and rustdoc. They own typing, borrowing, formatting, known idioms, and mechanical diagnostics. Do not spend model evaluations recreating them.
2. **A focused semantic Rust pack:** contextual errors in ownership intent, failure handling, resource completion, validation, and API contracts. Require a concrete counterexample, not a style preference.
3. **Opt-in specialist coverage:** Tokio lifecycle/cancellation, public-library API review, unsafe/FFI review, and workload-dependent performance. Some topics require evidence the current Jevlint extractor does not supply; retain them as review guidance rather than claiming reliable automated coverage.

The inventory deliberately covers more than the initial automated pack should enable. An ecosystem practice can be important without being a useful, sufficiently reliable model lint.

### Source authority and interpretation

- The **Rust Reference and concrete std API contracts** govern language/unsafe requirements. Prefer current, version-appropriate contracts over historical examples.
- **The Rust Book and Rust API Guidelines** establish idiomatic recommendations. API Guidelines primarily address reusable APIs, not every private helper.
- **Tokio documentation** governs Tokio behavior, not every async runtime or every type named `Mutex`, `File`, or `JoinHandle`.
- **Clippy documentation** distinguishes defaults from opinionated and experimental policies. A lint being available does not make it universally appropriate.
- Security-oriented guidance such as **ANSSI's Rust guide** is a conditional security profile, not a substitute for language/library contracts or ecosystem consensus.
- Cargo features, MSRV, supported targets, panic policy, ownership contracts, latency/memory budgets, and compatibility promises are project inputs. Do not invent them.

Two source conflicts illustrate why sources need interpretation: ANSSI's error chapter says generic error wrappers cannot provide context, but `anyhow::Context` explicitly does; the API Guidelines metadata checklist still mentions `authors`, while current Cargo marks that field deprecated. Neither stale statement should become a new rule. [ANSSI errors](https://github.com/ANSSI-FR/rust-guide/blob/master/src/en/errors.md), [anyhow Context](https://docs.rs/anyhow/1.0.98/anyhow/trait.Context.html), [Cargo authors](https://doc.rust-lang.org/cargo/reference/manifest.html#the-authors-field).

## 2. Basic linting and build baseline

These are suggested commands for a Rust workspace with an existing, intentionally maintained lockfile. They are **not commands run against this Go repository**:

```sh
cargo fmt --all -- --check
cargo check --workspace --all-targets --locked
cargo test --workspace --locked
cargo doc --workspace --no-deps --locked
cargo clippy --workspace --all-targets --locked
```

Use an actual build lane where the test pipeline does not already build the relevant artifacts: `cargo check` does not perform all code generation/linking. Keep the existing project's build conventions rather than add redundant jobs. [Cargo check][cargo-check], [Cargo test][cargo-test].

For warning-clean controlled CI, current Clippy documentation recommends this on **Cargo 1.97 or newer**:

```sh
CARGO_BUILD_WARNINGS=deny cargo clippy --workspace --all-targets --locked
```

For older Cargo, the established alternative is:

```sh
cargo clippy --workspace --all-targets --locked -- -D warnings
```

Do not combine these recipes or impose crate-wide `forbid(warnings)` as a Rust convention. Pin the CI toolchain and update it deliberately; new compilers can add warnings. Documentation warning policy is separate, for example `RUSTDOCFLAGS="-D warnings" cargo doc --workspace --no-deps --locked`. [Clippy usage][clippy-usage], [Cargo CI][cargo-ci].

### Scope and configuration caveats

- `--all-targets` means Cargo artifact kinds: libraries, binaries, tests, examples, benches. It does **not** mean every OS/architecture or every feature configuration.
- `cargo test --all-targets` does **not** cover doctests. Keep a normal library-doctest lane or add `cargo test --workspace --doc --locked` when using explicit all-target selection.
- Test default features, supported no-default configurations, and meaningful supported feature combinations. Use `--all-features` only when the features can coexist on that platform/toolchain. It does not test disabled-feature branches.
- Test the declared MSRV with that compiler. Clippy MSRV awareness is useful but not proof of compatibility. Workspace feature unification can hide failures in isolated package use.
- `--locked` needs a lockfile and stable dependency resolution; it does not make runtime behavior deterministic. Published library consumers resolve their own graph.
- `no_std` does not mean no allocation: `alloc` may be intentionally supported. Host-only tests can legitimately use `std` while the library supports an embedded target.
- Run target/profile-specific checks when they matter: pointer width, ABI, endian, atomics, overflow policy, panic strategy, and optimized behavior can change assumptions.
- Cargo workspace lint settings require member opt-in; do not assume a root lint table automatically applies everywhere.

Sources: [Cargo test][cargo-test], [features][cargo-features], [MSRV][cargo-msrv], [profiles][cargo-profiles], [workspace lints][workspace-lints], [no_std/preludes][preludes].

### Clippy policy

| Group | Appropriate baseline |
| --- | --- |
| `correctness` | Keep its deny-by-default behavior. |
| `suspicious`, `complexity`, `perf`, `style` | Use defaults; review concrete false positives and project-specific exceptions. Style is explicitly opinionated. |
| `pedantic` | Select deliberately; useful but intentionally more opinionated/false-positive-prone. |
| `restriction` | Never blanket-enable as a general baseline; some lints conflict. Select a justified project policy. |
| `nursery` | Select deliberately; not a universal production quality bar. |
| `cargo` | Publishing/dependency metadata policy where appropriate, not every private application. |

Clippy's `all` does not mean every lint. The simplest setup is to retain its defaults. If explicitly configuring group levels, do not accidentally lower correctness diagnostics from deny to warn. Where MSRV supports it, `#[expect(lint, reason = "...")]` can record a specific expected diagnostic; these Rust/Clippy controls do **not** currently suppress Jevlint findings. [Clippy groups][clippy-groups], [lint levels][lint-levels].

### Mechanical checks to reuse, not reimplement with Jev

| Area | Examples of existing checks | Context left for semantic review |
| --- | --- | --- |
| Ownership/borrowing | rustc borrow checker, moves/lifetimes, Send/Sync bounds | Resource lifecycle, ownership intent, safe-code logic errors. |
| Cloning/arguments | `clone_on_copy`, `ptr_arg`, `needless_borrow`; optional `redundant_clone`, `needless_pass_by_value` | Whether ownership transfer, shared state, snapshots, or real workload costs justify the API. |
| Allocation/container idioms | `unnecessary_to_owned`, `box_collection`, `vec_box`, `large_enum_variant`; optional `needless_collect` | Actual allocation frequency, stable addresses, pinning, ABI, latency/throughput tradeoffs. |
| Error/panic syntax | `unused_must_use`, `panicking_unwrap`, `unnecessary_unwrap`, `result_unit_err`; optional `unwrap_used`, `expect_used`, `panic` | Expected failure versus programmer error; information loss and false-success behavior. |
| Unsafe syntax/docs | `unsafe_op_in_unsafe_fn`, `not_unsafe_ptr_arg_deref`, `mut_from_ref`, `uninit_vec`, `missing_safety_doc`; optional `undocumented_unsafe_blocks` | Whether the actual safety obligation is satisfied, not merely documented. |
| Async | `await_holding_lock`, `await_holding_refcell_ref`, configured `await_holding_invalid_type`, `let_underscore_future` | Cancellation, task ownership, runtime-specific guard scope and shutdown. |
| I/O | `unused_io_amount`, `suspicious_open_options` | Framing, completion, durability promises, exclusive creation and cancellation. |
| API/trait idioms | `from_over_into`, `new_without_default`, `len_without_is_empty`, `should_implement_trait`, inconsistent Eq/Hash/Ord derive checks | Semantic conversions, invariant preservation and trait laws not captured mechanically. |
| Function shape | `too_many_arguments`, `type_complexity`; optional `too_many_lines` | Cohesion and testability, not another arbitrary numeric threshold. |
| Documentation/MSRV | rustdoc links/doctests, optional missing-doc checks, `incompatible_msrv` | Accuracy of contracts and actual build compatibility. |

Check names, availability, default levels and MSRV behavior against the chosen toolchain's [Clippy catalogue][clippy-catalogue]. An optional lint being disabled is not a reason to rebuild the same detector probabilistically.

## 3. Ecosystem practice inventory

Disposition labels: **Tool** = existing deterministic checks first; **Core candidate** = potentially useful local semantic judgment after fixture validation; **Opt-in** = project/runtime/caller evidence needed; **Specialist** = manual or deferred under current extraction. These are proposed rollout choices, not declarations that the model can prove correctness.

### A. Ownership, memory management and resources

| Practice from the ecosystem | Important exceptions / evidence | Disposition and sources |
| --- | --- | --- |
| Ownership and borrowing express access and lifecycle; use borrows for inspection and transfer ownership when retention/consumption is intended. Let callers control unnecessary copying. | `String`/`Vec` ownership can be necessary for storage or task lifetimes. Cheap Copy values, snapshots, trait signatures, stable public APIs and buffer reuse are legitimate. | Tool first; opt-in API review. [Borrowing][book-borrow], [caller control][api-flexibility]. |
| Moves, clones and shared ownership have different semantics. `Arc::clone`/`Rc::clone` share ownership, not independent mutable state. | Shared observers are intentional; copy-on-write and immutable sharing are valid. Only flag a contradiction with a demonstrated snapshot/isolation promise. | Core candidate with visible types/contract. [Clone][std-clone], [Arc][std-arc]. |
| RAII scopes determine resource release; the end of a borrow is not automatically an early destructor call. | Retaining a lock may preserve a transaction; capacity/handles may be deliberately reused. A short lexical lock scope is not an unconditional rule. | Opt-in lifecycle/dependency review. [Drop scopes][drop-scopes], [Mutex][std-mutex]. |
| Strong `Rc`/`Arc` cycles can retain resources; use non-owning backedges/other ownership designs where reclamation is required. | Intentional process-lifetime graphs, arenas and explicitly broken cycles are valid. `Weak` does not immediately free every allocation bookkeeping byte. | Opt-in; needs graph/teardown evidence. [Arc][std-arc], [leaks][nomicon-leaks]. |
| Interior mutability must respect runtime borrow/reentrancy and synchronization contracts. `UnsafeCell` alone does not make shared mutation thread-safe. | Valid RefCell/Mutex/RwLock usage is idiomatic. Need actual alias identity and callback/guard lifetimes; do not infer a conflict from a method name. | Tool first; opt-in reentrancy review; specialist for unsafe mutation. [RefCell][std-refcell], [UnsafeCell][unsafe-cell]. |
| Required output completion must observe flush/finish errors instead of relying on destruction. | Returning the writer to another owner or explicitly best-effort output is valid. Flushing is not crash durability. | Core candidate. [BufWriter][std-bufwriter], [destructor guidance][api-dependability]. |
| Destructors should generally avoid panic and provide an explicit fallible completion path when needed. | Deliberate fail-fast invariants exist. Do not add custom Drop for ordinary owned fields or ban every destructor `expect` mechanically. | Core candidate when a failure path is demonstrated. [Drop panics][drop-panics], [destructor guidance][api-dependability]. |
| Validate external size/resource claims before uncontrolled allocation or buffering; choose arithmetic semantics appropriate to the domain. | Use actual protocol/application limits, not invented capacities. `try_reserve` does not guarantee global OOM recovery; wrapping/saturating arithmetic can be intentional. | Core candidate with visible trust boundary/requirements. [Vec][std-vec], [Read][std-read], [ANSSI integer policy][anssi-integer], [profiles][cargo-profiles]. |
| Preserve required state consistency on errors/panics; do not publish an invalid intermediate state to subsequent safe observers. | Partial progress is explicitly allowed by many I/O APIs. Not every function is a transaction, and a safe-code logic error is not automatically UB. | Core candidate for a visible contradiction; unsafe variant is specialist. [Exception safety][exception-safety], [Read][std-read]. |
| `clear`, `drop`, and ordinary writes do not establish secure erasure guarantees. | Only relevant when there is an erasure/retention threat requirement. Even an erasure facility has limits concerning copies, registers, swap and foreign storage. | Security opt-in, not a rule to zero every buffer. [Vec security guarantees][std-vec]. |

### B. Function structure, types and public APIs

| Practice from the ecosystem | Important exceptions / evidence | Disposition and sources |
| --- | --- | --- |
| Keep responsibilities coherent; isolate independent parsing, computation, I/O and presentation when entanglement prevents testing/reasoning. | Straight-line orchestration is itself a responsibility. No universal line count, parameter count, nesting limit, module depth or mandatory main/lib split. | Reuse an existing contextual rule; advisory unless concrete harm. [Book separation of concerns][book-structure]. |
| Make signatures communicate ownership, fallibility and mutation. Prefer new return values over pure out-parameters. | In-place algorithms, caller-owned buffers, streaming I/O and FFI justify mutable parameters. Returning a tuple/struct does not require a heap allocation. | Tool first; opt-in API review. [Predictability][api-predictability], [flexibility][api-flexibility]. |
| Use the least restrictive practical input interface and caller-controlled ownership. | Do not make every function generic. Concrete domain types, required contiguity, code size, simple private helpers and semver can justify specificity. | Tool first; caller-assisted review. [Flexibility][api-flexibility]. |
| Standard conversions have semantic contracts: `From` should be infallible, lossless, value-preserving and obvious; use fallible/named conversions when appropriate. | Discarding irrelevant capacity is not semantic data loss. Allocation can fail in otherwise legitimate standard conversions. Do not confuse Into bounds with implementing Into directly. | Tool first; core candidate for uncovered semantic contradictions. [From][std-from], [interoperability][api-interop]. |
| Keep validated invariants enforceable across constructors and mutation paths. | Passive DTOs can have public fields. Unknown serializers/constructors cannot be assumed safe or unsafe; macro-generated bypasses need real expansion evidence. | Core candidate only when relevant type and paths are visible. [Book validation][book-panic], [dependability][api-dependability]. |
| Use enums/newtypes/custom option types where meanings or states are genuinely distinct. | Natural booleans/options, simple private tuples and FFI representations are fine. Do not wrap every primitive or mandate typestate/builders. | Opt-in API/domain review. [Type safety][api-types]. |
| Prefer appropriate standard trait interoperability; when Eq/Hash/Ord/Clone laws apply, implementations must agree. | Resource owners may not be Clone/Copy/Default; NaN does not support Eq; Debug may need redaction. Require actual type/impl evidence, not all common derives. | Compiler/Clippy and property tests first; specialist semantic residual. [Interoperability][api-interop], [Hash/Eq][std-hash]. |
| Trait boundaries can provide real domain contracts, std interoperability or extension points even with one current implementation. | Neither traits everywhere nor a blanket one-implementation-trait ban is Rust practice. | API review; avoid duplicate generic abstraction warnings. [Interoperability][api-interop], [flexibility][api-flexibility]. |
| Choose generics, enums and dynamic dispatch according to actual extensibility, heterogeneity, API and performance needs. | `&dyn Trait` does not inherently allocate. Not every trait must be dyn-compatible; definition-wide generic bounds may be necessary for associated types or Drop. | Compiler first; opt-in/API-baseline review. [Flexibility][api-flexibility], [future-proofing][api-future]. |
| Deref should offer unsurprising transparent pointer-like behavior, not accidental inheritance or invariant-bypassing mutable access. | Standard transparent wrappers are legitimate; use the current std contract, not an absolute syntactic ban. Need target methods, coercion semantics and intended API. | Opt-in; validated-invariant violations share one root finding. [Deref][std-deref], [predictability][api-predictability]. |
| Constructors, Default, methods, builders and collection APIs should behave predictably. | Builders are for actual construction complexity. No fake Default for a type without meaningful default; free functions can be symmetric or operate on foreign types. | Mechanical existence/naming checks first; API review for behavioral disagreement. [Predictability][api-predictability], [type safety][api-types]. |
| Keep intentional public boundaries and compatibility promises explicit. | Public records, open extension traits and exhaustive enums may be deliberate. Sealing traits, hiding fields or adding non_exhaustive later can itself break users. | API baseline/tools/manual review. [Future-proofing][api-future], [SemVer][cargo-semver]. |
| Public docs should accurately describe relevant errors, panics, safety obligations and partial-state behavior, with useful examples. | Do not demand every hypothetical caller-induced panic, duplicated examples for trivial items, or execution of network side effects in doctests. `no_run`/`compile_fail` can be correct. | rustdoc/Clippy first; contextual accuracy only when docs are supplied intact. [Documentation][api-docs], [doctests][doctests]. |

### C. Error handling, I/O and trust boundaries

| Practice from the ecosystem | Important exceptions / evidence | Disposition and sources |
| --- | --- | --- |
| Preserve caller choice for expected malformed-input/I/O failures with Result or appropriate domain handling. | Tests, examples, proved invariants, programmer-error preconditions and deliberate fatal startup policy can legitimately panic. | Core candidate, not blanket no-unwrap. [Book panic policy][book-panic]. |
| Errors should preserve useful identity/cause and context at meaningful abstraction boundaries. | Sanitized public errors can deliberately hide internals; lower-level errors may already have sufficient context. Do not expose secrets or mandate anyhow/thiserror/another crate. | Core candidate for proven information loss; reuse compatible generic error rules. [Error sources][std-error], [interoperability][api-interop]. |
| Error trait/bounds should support actual consumers and environment. | `Send + Sync + 'static` is not mandatory for every internal/thread-local/no_std error; older MSRV matters. | Tool/API review, not blanket representation policy. [Interoperability][api-interop], [core Error][core-error]. |
| Read/write operations may be partial; successful I/O calls do not imply a complete protocol message. | Correct loops, framing and read_exact/write_all can provide completion, but cancellation changes the latter's guarantee. | `unused_io_amount` first; semantic framing/cancellation residual. [Read][std-read], [Tokio select][tokio-select]. |
| Bound untrusted lines, frames and declared lengths before allocating/growing without limit. | A valid upstream bound or codec default may already suffice. `.lines()` alone is not a bound; `.take(limit)` alone can silently truncate rather than validate a complete frame. Time and byte limits solve different problems. | Core candidate when trust and contract are visible. [BufRead][std-bufread], [LinesCodec][lines-codec]. |
| Use atomic exclusive creation where the contract is not to overwrite an existing file. | Ordinary overwrite or an advisory existence check is fine. Atomic final-component creation is not a complete parent-directory/symlink confinement solution. | Core candidate for visible TOCTOU contradiction. [OpenOptions::create_new][create-new]. |
| Separate buffered completion, write completion, durability and atomic replacement. std BufWriter Drop ignores flush errors; Tokio BufWriter Drop discards pending buffered data. | No universal fsync requirement for logs/temp files. Filesystem and parent-directory durability protocols depend on the platform and contract. | Core completion candidate; durability/atomicity opt-in. [std BufWriter][std-bufwriter], [Tokio BufWriter][tokio-bufwriter], [File sync_all][sync-all], [Tokio fs][tokio-fs]. |
| Preserve required progress before irreversible mutation/consumption. | `Read::read_to_end` may append bytes before returning an error; `Read::read` guarantees no bytes were read on Err. Do not demand rollback where the specific method's documented contract permits partial completion. | Contextual state/error rule. [Read allocation/consumption guidance][std-read], [exception safety][exception-safety]. |

### D. Concurrency, async and Tokio

Tokio items are opt-in; runtime/library version and concrete API identity matter.

| Practice from the ecosystem | Important exceptions / evidence | Disposition and sources |
| --- | --- | --- |
| Do not block or monopolize an executor task that must allow other work to progress. | Small bounded computation and blocking work on a dedicated worker are fine. `block_in_place` has runtime/same-task restrictions; it is not a universal fix. | Async opt-in; real executor/call-path evidence. [Future][std-future], [Tokio tasks][tokio-task], [spawn_blocking][spawn-blocking], [block_in_place][block-in-place]. |
| Choose guard scope and mutex type for the protected operation. | Tokio recommends ordinary mutexes for short non-awaiting data access in many cases. Async guards can legitimately span await. | Clippy first; opt-in dependency/critical-section review. [Tokio Mutex][tokio-mutex]. |
| Cancellation is dropping a future, not rolling back completed side effects. Preserve required progress when restarting on a live resource. | Closing/discarding a connection on timeout, losing best-effort messages, or retaining the same future across selection can be correct. Queue-position loss is distinct from data loss. | High-value async candidate. [select cancellation][tokio-select], [Sender send][tokio-send], [timeout][tokio-timeout]. |
| Required spawned work needs a lifecycle owner and a route for task and operation errors. | Dropping std/Tokio JoinHandle detaches; intentional detachment or ownership by a supervisor can be correct. JoinSet drop behavior is different. | Async/concurrency opt-in. [Tokio JoinHandle][tokio-joinhandle], [std JoinHandle][std-joinhandle], [JoinSet][tokio-joinset]. |
| Bound the resource actually at risk: active work, waiting tasks, queued messages, bytes, handles or connections. Keep permits alive for that resource's lifetime. | Acquiring a semaphore inside a spawned task can bound active work without bounding task count; this is not automatically wrong. Upstream admission or a finite workload may suffice. | Async/resource opt-in; never invent a universal concurrency number. [Semaphore][tokio-semaphore], [mpsc][tokio-mpsc], [spawn_blocking][spawn-blocking]. |
| Graceful shutdown means stop/notify/wait for the cleanup the application promises. | Hard abort can be intentional. Abort does not itself await teardown; retained senders can prevent EOF; channel capacities do not guarantee draining. | Application-lifecycle opt-in. [Shutdown][tokio-shutdown], [mpsc][tokio-mpsc], [task cancellation][tokio-task]. |
| Timeout/abort of async waiting does not necessarily stop the underlying work. Started spawn_blocking work is not aborted by Tokio. | Finite background completion may be acceptable; actual blocking API deadlines/cooperative stop protocols can work. shutdown_timeout stops waiting, not the running closure. | Async specialist; merge with task/shutdown root cause. [spawn_blocking][spawn-blocking], [timeout][tokio-timeout]. |
| Lock/wait protocols require actual progress: preserve lock order, recheck Condvar predicates and use the real lock's fairness policy. | Different lock identities and outer retry loops can make a local shape safe. Tokio RwLock policy is not std RwLock policy. | Deterministic patterns first; contextual/manual residual. [Condvar][std-condvar], [Tokio RwLock][tokio-rwlock]. |
| Notify is a coalescing notification primitive, not a counting event queue; multi-consumer registration order matters. | Hint-and-recheck/drain protocols and documented single-consumer patterns can be correct. | Tokio protocol specialist. [Notify][tokio-notify]. |
| Fairness/yield are not universal scheduling guarantees. Biased selection makes progress ordering the caller's responsibility. | Intentional priority and bounded work are valid. Do not require a yield every N iterations or assume every await suspends. | Async specialist/advisory. [select][tokio-select], [yield_now][tokio-yield], [Future][std-future]. |
| Atomic orderings must establish the needed publication relationship, not merely atomicity of a flag. | Relaxed counters are idiomatic; existing locks/channels may establish ordering. SeqCst does not fix arbitrary broken algorithms. | Compiler ordering checks, Loom and specialist review. [Atomic Ordering][atomic-ordering], [atomic model][atomic-model]. |
| Tokio filesystem APIs have ordinary-file/blocking-worker semantics, not arbitrary descriptor cancellation semantics. | Known pipes/special descriptors may need their dedicated APIs. Do not infer file kind from a filename. | Runtime/platform specialist. [Tokio fs][tokio-fs]. |

### E. Unsafe code, raw storage, pinning and FFI

Safe clients must not be able to cause UB through a safe abstraction. A `SAFETY:` comment or successful compilation is not a proof. These topics remain specialist/manual or deferred when the complete contracts are unavailable.

| Required practice / obligation | Caveats and evidence | Sources |
| --- | --- | --- |
| Safe wrappers enforce every unsafe operation's preconditions, or the API is genuinely unsafe with appropriate caller obligations. | Need all safe constructors/mutators and the actual unsafe API contract; absence of a check in one function is not proof no invariant exists. | [Reference UB][reference-ub], [safe/unsafe interaction][safe-unsafe]. |
| Validity and ownership survive panic/error cleanup, including arbitrary safe callbacks and safe trait implementations. | Unsafe code cannot turn a safe client's violated Ord law or panic into UB. Poisoning is not a soundness proof. | [Exception safety][exception-safety], [Mutex][std-mutex], [safe/unsafe interaction][safe-unsafe]. |
| Memory safety must not depend on Drop always running. | Safe clients can forget/leak values. Resource leaks can be intentional and memory-safe; do not confuse them with UB. | [Leaks][nomicon-leaks]. |
| Raw references/slices respect allocation extent, alignment, validity, lifetime and aliasing. | Empty raw slices still have API preconditions. Adjacent allocations are not necessarily one allocation; lifetime annotations do not extend storage lifetime. Pointer/integer operations need their specific provenance contract, not a blanket ban. | [from_raw_parts][raw-slice], [ptr][std-ptr]. |
| Publication follows actual initialization; capacity is not initialized length. | MaybeUninit can hold uninitialized storage; ManuallyDrop cannot be used as a substitute for validity. FFI output may be initialized only on success. Padding has different rules from initialized fields. | [MaybeUninit][maybe-uninit], [Vec set_len][vec-set-len]. |
| Allocation/reclamation preserves allocator, layout, capacity and exactly-once ownership. | Borrowing foreign memory differs from claiming Rust ownership. CString::from_raw is not a general C-pointer adoption API. | [Vec raw ownership][vec-from-raw], [CString][cstring]. |
| Manual drop/state transitions never expose already-dropped contents through safe access or derived operations. | Need all derived/custom trait implementations and lifecycle paths. Historical ManuallyDrop restrictions have changed; use the supported version's docs. | [ManuallyDrop][manually-drop]. |
| Shared mutation and manual Send/Sync bounds match the actual alias/thread model. | UnsafeCell relaxes shared immutability, not uniqueness of &mut or race rules. Correct Send/Sync bounds depend on access and ownership; blanket T:Send+Sync is not a proof. | [UnsafeCell][unsafe-cell], [Send/Sync][send-sync], [atomic model][atomic-model]. |
| Pinning APIs preserve address-sensitive invariants and destruction guarantees. | Moving a Pin<Box<T>> handle does not move T; Unpin types and deliberately nonstructural fields differ. Box alone does not enforce pinning. | [Pin][std-pin]. |
| ABI/layout/encoding assumptions match explicit guarantees, including nested representations and validity. | repr(C) is not recursive and does not make arbitrary Vec/String/trait-object fields FFI-safe; equal sizes do not imply compatible valid values or ABI. | [Layout][type-layout], [Vec][std-vec]. |
| Foreign pointer retention and callback registration keep targets alive, stable, synchronized and quiescent before destruction. | A non-null pointer or deregistration call alone is insufficient; actual foreign API and in-flight callback contracts are required. | [Nomicon FFI][ffi]. |
| Unwind/abort behavior matches the actual ABI, panic profile and both runtimes. | Rust panic at a non-unwind boundary safely aborts; do not call that UB. catch_unwind does not catch panic=abort or guarantee general foreign-exception recovery. Do not blanket-change all interfaces to C-unwind. | [Reference panic/FFI][panic-ffi], [Nomicon FFI][ffi]. |

Prefer current Reference/std over stale tutorial wording. The Reference does not fully settle every aliasing question; Miri's Stacked/Tree Borrows models are not a permanent language specification.

### F. Performance, testing, builds and dependencies

- **Measure relevant workloads before prescriptive optimization.** Allocation/clone counts, generic versus dyn, Arc versus Rc, stack versus heap and iterator versus loop are tradeoffs. Clippy can identify patterns; it does not measure throughput, latency, memory ceilings or code size. [Clone][std-clone], [Vec][std-vec], [Arc][std-arc], [API flexibility][api-flexibility].
- **Avoid demonstrated repeated work.** Caller-controlled ownership, reused capacity, existing intermediate results and buffered small I/O can help when actual consumers benefit. Preserve latency and memory bounds; do not buffer an untrusted whole file just to reduce calls. [API flexibility][api-flexibility], [BufWriter][std-bufwriter], [Tokio fs][tokio-fs].
- **Tests should exercise and check the observable promise.** A parser rejection test should check rejection, not merely construct a parser. No-panic, compile-only, compile-fail, doctest and Result-returning tests can all be legitimate. No universal coverage percentage, framework or one-test-per-function requirement. [Book testing][book-tests], [doctests][doctests].
- **Test domain laws and boundaries where they matter.** Equality/hash consistency, serialization round trips, parsing boundaries, resource lifecycle and state transitions are appropriate property/regression targets. Do not snapshot unspecified std hash outputs. Property testing and fuzzing need meaningful oracles and do not prove all-input correctness. [Hash][std-hash], [Proptest][proptest], [Rust Fuzz][rust-fuzz].
- **Feature/MSRV/target support is an exercised contract.** Prefer additive library features; test the configurations actually promised. no_std, alloc, std, test-only dependencies, optional backends and exclusive features need explicit scope. [Features][cargo-features], [MSRV][cargo-msrv], [Cargo CI][cargo-ci].
- **Public compatibility needs a baseline.** Signature/trait/enum/cfg changes and behavioral promises cannot be judged as regressions from one isolated snapshot. Use Cargo's SemVer guidance plus an API comparison tool; intentional breaking releases are legitimate. [SemVer][cargo-semver], [cargo-semver-checks][semver-checks].
- **Check dependencies and packaging with dedicated tools.** cargo-audit checks known RustSec advisories; cargo-deny adds declared source/license/ban policies. These policies are organizational, and neither tool proves dependency safety. A packaging build catches missing packaged inputs that a checkout build can miss. [RustSec][rustsec], [cargo-deny][cargo-deny], [cargo package][cargo-package].
- **Use runtime safety tools selectively.** Miri is valuable for exercised unsafe Rust and some atomic behavior; native sanitizers complement FFI/runtime coverage; Loom explores deliberately modeled concurrency. They have target/API/model limits and are not whole-library soundness certificates. [Miri][miri], [sanitizers][sanitizers], [Loom][loom].

## 4. Proposed semantic pack contents

### Initial calibration queue: ten narrowly specified candidates

These are candidates to validate, **not ten rules declared ready to gate CI**. Start with warning/advisory rollout; promote only after per-rule evidence. Each needs a violating example, a near-neighbor fix, a legitimate-exception example, and inspection of the actual extracted context. Merge overlap with existing generic rules rather than run duplicate checks on the same root cause.

| Proposed rule | Fail only when | Passing / exception fixture concept | Evidence and source |
| --- | --- | --- | --- |
| `rust-expected-failure-panics` | Ordinary bad external input or recoverable operation is demonstrably made into an unintended panic. | Parser returns Err; parse of a proven-valid literal; test setup unwrap; explicit fatal startup policy. | Input/error contract and local control flow; [Book panic policy][book-panic]. |
| `rust-error-cause-loss` | An abstraction boundary demonstrably erases actionable cause/context and gives a misleading failure. | Typed/source-preserving error; adequate existing error; intentional sanitized public error with correct internal handling. | Error type/body and caller contract; [Error][std-error]. Reuse generic error coverage where equivalent. |
| `rust-conversion-contract` | A From conversion knowingly panics on accepted input or destroys semantically significant information. | TryFrom validates range; explicit truncation API; irrelevant capacity omitted. | Actual standard trait/impl and input domain; [From][std-from]. Clippy owns simple duplicate patterns. |
| `rust-validation-bypass` | Visible constructors claim an invariant but another visible safe field/mutator path breaks it and consumers rely on it. | Private validated representation; standard constrained type; passive DTO without invariant promise. | Full relevant local type/impl and consumer assumption; [Book validation][book-panic], [API dependability][api-dependability]. |
| `rust-partial-state-on-error` | A reachable error/panic leaves state inconsistent with an explicit guarantee and execution can continue observing it. | Validate/prepare before commit; correct rollback; documented partial progress. | Mutation/failure/observer path, not assumption every function is atomic; [exception safety][exception-safety]. |
| `rust-buffered-output-completion` | Success claims required output completion while unwritten data or completion errors are lost. | Observe flush/finish errors; return ownership to caller; intentionally disposable output. | Concrete writer stack and successful return path; [BufWriter][std-bufwriter], [Tokio fs][tokio-fs]. Not a durability proof. |
| `rust-fallible-drop` | Resource cleanup can panic on an ordinary reachable failure and violates the intended failure/cleanup policy. | Explicit fallible close/finish plus nonpanicking fallback; proved infallible operation; deliberate invariant abort. | Actual Drop impl and failure/unwind contract; [Drop][drop-panics], [API dependability][api-dependability]. |
| `rust-unbounded-input-buffering` | Known untrusted framing/size data expands storage without the required effective bound, or limit handling accepts invalid truncation. | Checked protocol cap, bounded upstream codec, streaming bounded state, trusted fixed input. | Input source, cap, arithmetic and growth path; [BufRead][std-bufread], [LinesCodec][lines-codec], [Vec][std-vec]. |
| `rust-exclusive-create-race` | A no-overwrite/exclusive-create promise is implemented with a racy existence check followed by ordinary creation. | Atomic create_new with AlreadyExists handling; intentional overwrite; independent exclusivity guarantee. | Explicit exclusivity contract and competing access; [create_new][create-new]. |
| `rust-shared-clone-isolation` | Code promises an independent mutable snapshot but cloning preserves sharing and a visible mutation violates that promise. | Intentionally shared Arc/Rc handle; independent copied inner value; appropriate copy-on-write. | Clone/type and observable isolation contract; [Clone][std-clone], [Arc][std-arc]. |

Arithmetic policy for resource sizes belongs in the resource-bound analysis or a separately validated narrow rule if it produces a distinct diagnostic. Do not add a generic ban on `+`, `*`, `as`, wrapping arithmetic or infallible allocations.

### Optional groups after core calibration

- **Tokio:** cancellation-progress loss; lost ownership/results of required tasks; effective admission/permit lifetime; concrete lock/wait dependencies; blocking boundaries; promised shutdown; started blocking-work cancellation assumptions. Notify/fairness/custom Future protocols need especially strong context. Not a blanket list of prohibited async method calls.
- **Public library APIs:** ownership/caller flexibility, meaningful domain types, conversion naming/contracts, trait-law consistency, accurate failure/safety docs and intended extension boundaries. Compatibility, MSRV, exported reachability and feature behavior require tooling/baselines or additional context.
- **Unsafe/FFI:** exact unsafe precondition violations, panic/leak safety, initialization and ownership, raw references/provenance, manual thread-safety, pinning, layout and foreign lifetime/unwind contracts. Human-reviewed specialist findings, never an automated "soundness passed" badge.
- **Performance:** demonstrated redundant ownership/materialization, retained large resources, repeated small I/O and repeated computation. Suggestions without a real workload/requirement, not correctness errors.
- **Function/test quality:** reuse applicable existing generic rules such as contextual control-flow, swallowed-error, misleading-name/doc and test-intent review; add Rust positive/exception fixtures instead of another Rust-prefixed copy of the same question.

One core pack plus separately selected specialist packs is a natural eventual distribution shape. The present schema has no first-class named profile switch; do not pretend that these conceptual groups are an implemented configuration feature. No pack IDs, published repositories, or final split are committed by this research.

## 5. Policies not to encode as universal Rust best practices

- No clone, allocation, Box, Rc/Arc, RefCell, unsafe, raw pointer or explicit lifetime anywhere.
- No unwrap/expect/assert/panic anywhere, including tests or proved programmer-error invariants.
- Every function below an arbitrary line/parameter/nesting limit; every large function must be split.
- Every trait needs multiple current implementations; every dependency needs a trait; always generics or always dyn.
- Every primitive needs a newtype; every constructor needs a builder; every type needs Copy/Clone/Default/Eq/Ord.
- Every mutable out-parameter is bad; every API should be fully generic; changing ownership is automatically API-compatible.
- Every async lock must be a Tokio mutex; no guard may ever cross await; every await or yield guarantees another task runs.
- Every spawn must be immediately joined; every shutdown must drain indefinitely; every queue should use the same numeric limit.
- All atomics must be SeqCst; Relaxed is always wrong; atomics are inherently preferable to mutexes.
- Every file write needs fsync; every byte buffer needs zeroization; every Drop must implement custom cleanup.
- A prescribed error-handling crate, mandatory no_std, blanket all-features builds, or one mandatory application architecture.
- A safety comment, green compiler/Clippy/Miri, or a passing model judgment proves a safe wrapper sound.

These bans would recreate the adapter false-positive problem: legitimate Rust mechanisms would fail because a heuristic was mistaken for a universal rule.

## 6. Current Jevlint feasibility and packaging constraints

Repository inspection, not a new runtime verification:

- Rust `.rs` parsing is already present. Primary units are function items and struct/enum/union/trait/type-alias items. `impl` blocks supply limited type context rather than being independent primary type units. [Rust preset](internal/parsing/presets.go), [queries](internal/parsing/parsing.go).
- Rust related-type context is same-file and syntactic. The impl query handles a simple type identifier, not general generic impl resolution. It does not provide complete constructor/mutator/caller coverage.
- Direct callee context is not compiler resolution: method calls and `Type::qualified` calls are not resolved into Rust callees by the current name-based implementation. Enabling `context.callees` does not fix that. [Callee resolver](internal/parsing/callees.go).
- There is no compiler typing/borrow analysis, macro expansion, target/feature evaluation, complete public-export graph, API-history comparison, or injected foreign header contract.
- Leading doc comments can be captured, but intervening Rust attributes stop the backward comment scan. Do not infer missing `# Safety`/`# Errors`, absent derives, or absent cfg(test) from incomplete context. [Source/comment capture](internal/parsing/parsing.go).
- Rules can allow skip/abstain. However, pack eval cases currently expect pass/fail; skip/abstain without findings is inconclusive, not a passing expected-pass case. Do not invent `expect: "abstain"`. Unknown-context behavior requires a separately observed check or a separately scoped engine change. [Eval execution](internal/evals/run.go).
- Each eval fixture needs an extracted unit applicable to the selected rule kinds. Avoid snippet-only statements with no containing declaration. Prefer one intended evaluated unit; otherwise a fixture can fail for the wrong helper/type. Check the reported location, not only a case-level fail.
- Existing pack format is sufficient for a data-only Rust pack: `pack.json` with `version`, `id`, `languages: ["rust"]`, `rules`, and optional `evals`; rule definitions; `jevlint-evals.json`; relative `.rs` fixtures. Sources: [manifest contract](internal/packs/packs.go), [example manifest](examples/packs/database-joins/pack.json), [eval schema](internal/evals/evals.go).
- Existing generic rules already include Rust examples; reuse the correct concepts but do not assume every generic abstraction rule is Rust-idiomatic. Rule IDs shared across packs/configuration overlay one another; duplication/overrides need deliberate review. [Project rules](jevlint.json), [example rules](examples/rules/), [pack documentation](README.md#packs).

A syntactic source fragment is often enough to propose a useful local counterexample. It is not enough to claim that checks are absent elsewhere, that an arbitrary method is a particular std/Tokio API, or that every safe caller is sound. Missing context should mean abstention/deferred review, not fabricated certainty.

## 7. Validation and promotion plan

For each candidate retained for implementation:

1. Record the ecosystem source, exact intended contract, applicability, exceptions and deterministic-tool overlap. Keep one underlying defect per rule.
2. Supply a violation, the nearest correct variant, and hard negatives: intentional sharing/detachment, tests/invariant unwraps, upstream limits, trait adapters, async guards with legitimate scope, best-effort cleanup, relevant no_std/MSRV cases.
3. Inspect the exact Rust unit/type/doc context Jevlint sends. Do not write a fixture whose required evidence is available to a human but absent from the request.
4. Compile positive/negative Rust fixtures under their declared environment when they are meant to be compilable. Parsing with Tree-sitter is not compilation. Keep unsafe counterexamples clearly designated; do not execute UB as normal tests. Tokio fixtures need their actual runtime/dependency context.
5. Run appropriate compiler/Clippy checks first. If they already identify the same defect, retain a deterministic baseline recommendation instead of another model rule unless the semantic residual is explicit.
6. Run pack evals and inspect which unit triggered. A per-file fail caused by an unrelated helper is not proof the rule works. Include realistic corpus review beyond small fixture pairs, with appropriate code-sharing approval.
7. Record model/version, configuration, observed false positives/negatives and inconclusive cases. Do not call the provider's confidence score measured precision or choose a universal confidence threshold without calibration.
8. Introduce rules as advisory/warnings; only make a proven, context-supported rule blocking after deliberate review. Avoid blanket enforcement of the entire inventory.
9. Retain source/version caveats and legitimate-exception fixtures as the toolchain/runtime evolves.

Current follow-ups [#7: deterministic finding suppression](https://github.com/Bairum/jevlint/issues/7) and [#8: provider usage accounting](https://github.com/Bairum/jevlint/issues/8) matter for rollout: the existing engine cannot yet persist a narrow accepted Jev finding or report measured token totals. They are separate implementation work, not features of this proposed pack.

## Source index

Primary references for the inventory. Moving stable/latest documentation must be reconciled with the eventual supported toolchain/runtime versions.

[book-borrow]: https://doc.rust-lang.org/book/ch04-02-references-and-borrowing.html
[book-panic]: https://doc.rust-lang.org/book/ch09-03-to-panic-or-not-to-panic.html
[book-structure]: https://doc.rust-lang.org/book/ch12-03-improving-error-handling-and-modularity.html#separating-concerns-in-binary-projects
[book-tests]: https://doc.rust-lang.org/book/ch11-01-writing-tests.html
[api-flexibility]: https://rust-lang.github.io/api-guidelines/flexibility.html
[api-dependability]: https://rust-lang.github.io/api-guidelines/dependability.html
[api-predictability]: https://rust-lang.github.io/api-guidelines/predictability.html
[api-types]: https://rust-lang.github.io/api-guidelines/type-safety.html
[api-interop]: https://rust-lang.github.io/api-guidelines/interoperability.html
[api-future]: https://rust-lang.github.io/api-guidelines/future-proofing.html
[api-docs]: https://rust-lang.github.io/api-guidelines/documentation.html
[clippy-usage]: https://doc.rust-lang.org/clippy/usage.html
[clippy-groups]: https://doc.rust-lang.org/clippy/lints.html
[clippy-catalogue]: https://rust-lang.github.io/rust-clippy/stable/index.html
[lint-levels]: https://doc.rust-lang.org/rustc/lints/levels.html
[cargo-ci]: https://doc.rust-lang.org/cargo/guide/continuous-integration.html
[cargo-check]: https://doc.rust-lang.org/cargo/commands/cargo-check.html
[cargo-test]: https://doc.rust-lang.org/cargo/commands/cargo-test.html
[cargo-features]: https://doc.rust-lang.org/cargo/reference/features.html
[cargo-msrv]: https://doc.rust-lang.org/cargo/reference/rust-version.html
[cargo-profiles]: https://doc.rust-lang.org/cargo/reference/profiles.html
[workspace-lints]: https://doc.rust-lang.org/cargo/reference/workspaces.html#the-lints-table
[cargo-semver]: https://doc.rust-lang.org/cargo/reference/semver.html
[preludes]: https://doc.rust-lang.org/reference/names/preludes.html
[doctests]: https://doc.rust-lang.org/rustdoc/write-documentation/documentation-tests.html
[std-clone]: https://doc.rust-lang.org/std/clone/trait.Clone.html
[std-arc]: https://doc.rust-lang.org/std/sync/struct.Arc.html
[std-mutex]: https://doc.rust-lang.org/std/sync/struct.Mutex.html
[std-refcell]: https://doc.rust-lang.org/std/cell/struct.RefCell.html
[drop-scopes]: https://doc.rust-lang.org/reference/destructors.html#drop-scopes
[drop-panics]: https://doc.rust-lang.org/std/ops/trait.Drop.html#panics
[std-bufwriter]: https://doc.rust-lang.org/std/io/struct.BufWriter.html
[std-vec]: https://doc.rust-lang.org/std/vec/struct.Vec.html
[std-read]: https://doc.rust-lang.org/std/io/trait.Read.html
[std-bufread]: https://doc.rust-lang.org/std/io/trait.BufRead.html
[std-from]: https://doc.rust-lang.org/std/convert/trait.From.html#when-to-implement-from
[std-error]: https://doc.rust-lang.org/std/error/trait.Error.html#error-source
[core-error]: https://doc.rust-lang.org/core/error/trait.Error.html
[std-hash]: https://doc.rust-lang.org/std/hash/trait.Hash.html#hash-and-eq
[std-deref]: https://doc.rust-lang.org/std/ops/trait.Deref.html
[std-future]: https://doc.rust-lang.org/std/future/trait.Future.html#runtime-characteristics
[std-joinhandle]: https://doc.rust-lang.org/std/thread/struct.JoinHandle.html
[std-condvar]: https://doc.rust-lang.org/std/sync/struct.Condvar.html
[atomic-ordering]: https://doc.rust-lang.org/std/sync/atomic/enum.Ordering.html
[atomic-model]: https://doc.rust-lang.org/std/sync/atomic/index.html#memory-model-for-atomic-accesses
[create-new]: https://doc.rust-lang.org/std/fs/struct.OpenOptions.html#method.create_new
[sync-all]: https://doc.rust-lang.org/std/fs/struct.File.html#method.sync_all
[anssi-integer]: https://github.com/ANSSI-FR/rust-guide/blob/master/src/en/integer.md
[nomicon-leaks]: https://doc.rust-lang.org/nomicon/leaking.html
[exception-safety]: https://doc.rust-lang.org/nomicon/exception-safety.html
[reference-ub]: https://doc.rust-lang.org/reference/behavior-considered-undefined.html
[safe-unsafe]: https://doc.rust-lang.org/nomicon/safe-unsafe-meaning.html
[raw-slice]: https://doc.rust-lang.org/std/slice/fn.from_raw_parts.html
[std-ptr]: https://doc.rust-lang.org/std/ptr/index.html
[maybe-uninit]: https://doc.rust-lang.org/std/mem/union.MaybeUninit.html
[vec-set-len]: https://doc.rust-lang.org/std/vec/struct.Vec.html#method.set_len
[vec-from-raw]: https://doc.rust-lang.org/std/vec/struct.Vec.html#method.from_raw_parts
[cstring]: https://doc.rust-lang.org/std/ffi/struct.CString.html
[manually-drop]: https://doc.rust-lang.org/std/mem/struct.ManuallyDrop.html
[unsafe-cell]: https://doc.rust-lang.org/std/cell/struct.UnsafeCell.html
[send-sync]: https://doc.rust-lang.org/nomicon/send-and-sync.html
[std-pin]: https://doc.rust-lang.org/std/pin/index.html
[type-layout]: https://doc.rust-lang.org/reference/type-layout.html
[ffi]: https://doc.rust-lang.org/nomicon/ffi.html
[panic-ffi]: https://doc.rust-lang.org/reference/panic.html#unwinding-across-ffi-boundaries
[tokio-task]: https://docs.rs/tokio/latest/tokio/task/index.html
[spawn-blocking]: https://docs.rs/tokio/latest/tokio/task/fn.spawn_blocking.html
[block-in-place]: https://docs.rs/tokio/latest/tokio/task/fn.block_in_place.html
[tokio-mutex]: https://docs.rs/tokio/latest/tokio/sync/struct.Mutex.html
[tokio-rwlock]: https://docs.rs/tokio/latest/tokio/sync/struct.RwLock.html
[tokio-select]: https://docs.rs/tokio/latest/tokio/macro.select.html#cancellation-safety
[tokio-send]: https://docs.rs/tokio/latest/tokio/sync/mpsc/struct.Sender.html#method.send
[tokio-timeout]: https://docs.rs/tokio/latest/tokio/time/fn.timeout.html
[tokio-joinhandle]: https://docs.rs/tokio/latest/tokio/task/struct.JoinHandle.html
[tokio-joinset]: https://docs.rs/tokio/latest/tokio/task/struct.JoinSet.html
[tokio-semaphore]: https://docs.rs/tokio/latest/tokio/sync/struct.Semaphore.html
[tokio-mpsc]: https://docs.rs/tokio/latest/tokio/sync/mpsc/index.html
[tokio-notify]: https://docs.rs/tokio/latest/tokio/sync/struct.Notify.html
[tokio-yield]: https://docs.rs/tokio/latest/tokio/task/fn.yield_now.html#non-guarantees
[tokio-shutdown]: https://tokio.rs/tokio/topics/shutdown
[tokio-fs]: https://docs.rs/tokio/latest/tokio/fs/index.html
[tokio-bufwriter]: https://docs.rs/tokio/latest/tokio/io/struct.BufWriter.html
[lines-codec]: https://docs.rs/tokio-util/latest/tokio_util/codec/struct.LinesCodec.html#method.new_with_max_length
[rustsec]: https://github.com/rustsec/rustsec/tree/main/cargo-audit
[cargo-deny]: https://embarkstudios.github.io/cargo-deny/checks/index.html
[cargo-package]: https://doc.rust-lang.org/cargo/commands/cargo-package.html
[miri]: https://github.com/rust-lang/miri
[sanitizers]: https://doc.rust-lang.org/nightly/unstable-book/compiler-flags/sanitizer.html
[loom]: https://docs.rs/loom/latest/loom/
[rust-fuzz]: https://rust-fuzz.github.io/book/cargo-fuzz/guide.html
[proptest]: https://proptest-rs.github.io/proptest/intro.html
[semver-checks]: https://github.com/obi1kenobi/cargo-semver-checks
