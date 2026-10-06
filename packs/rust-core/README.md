# Rust core semantic candidates

`Bairum/rust-core` contains ten evidence-gated **warning** rules and 140 explicit fixture oracles. These are semantic candidates with measured fixture calibration, not validated CI gates. Shared judgment requirements live in [`guidance.md`](guidance.md), loaded through the manifest; descriptions contain only rule-specific requirements. Missing evidence is not a defect or a safety certificate.

## Rules and research references

Research IDs refer to [the master inventory](../../RUST-PACK-RESEARCH.md) and preserved [API](../../docs/rust/research/api.md), [memory](../../docs/rust/research/memory.md), and [concurrency](../../docs/rust/research/concurrency.md) findings. They and public URLs are documentation, not repeated prompt text.

| Rule | Research | Required contradiction and primary contracts |
| --- | --- | --- |
| `rust-expected-failure-panics` | API01/02 | Apply to parsing/operations on explicitly external input whose contract permits caller recovery. Fail when ordinary malformed input or recoverable I/O reaches panic, unwrap, expect, or an equivalent assertion instead of the promised error handling. Identify the failing input/path, not merely an unwrap token. [Book panic policy](https://doc.rust-lang.org/book/ch09-03-to-panic-or-not-to-panic.html) |
| `rust-error-cause-loss` | API03 | Apply at a visible error abstraction boundary where callers need actionable cause/context. Fail when a reachable lower error is erased or misclassified so recovery/diagnosis promised by the boundary is impossible; show the lost identity/resource and the misleading outcome. Retaining a useful error directly or via source is sufficient. [Error sources](https://doc.rust-lang.org/std/error/trait.Error.html#error-source) |
| `rust-conversion-contract` | API05 | Apply only to an explicitly standard std::convert::From implementation with visible source/target domain meaning. Fail for accepted source values that panic due to domain conversion or lose semantically significant information, including a plausible-looking successful conversion with changed meaning. Standard From is expected to be infallible, lossless, value-preserving and obvious; use TryFrom or a named operation when appropriate. [From semantics](https://doc.rust-lang.org/std/convert/trait.From.html#when-to-implement-from) |
| `rust-validation-bypass` | API06 | Require an explicit visible validated-type invariant, a reachable safe path that contradicts it, and a visible consumer relying on that invariant. Fail the primary unit that exposes or implements the unchecked construction, field, mutation, accessor, or conversion; give a concrete safe value and consumer consequence. A checked constructor alone does not establish a whole-type contract. Do not report a primary unit solely for defects in attached type or implementation context. [Validated types](https://doc.rust-lang.org/book/ch09-03-to-panic-or-not-to-panic.html#custom-types-for-validation), [C-VALIDATE](https://rust-lang.github.io/api-guidelines/dependability.html#c-validate) |
| `rust-partial-state-on-error` | M08 | Apply when a visible operation promises atomicity or another explicit state invariant on error/panic and later safe observation/reuse is possible. Fail when mutation preceding a reachable fallible operation leaves observable state contradicting that promise. Show the mutation, failure and observer path; do not assume every function is transactional. [Exception safety](https://doc.rust-lang.org/nomicon/exception-safety.html), [Read](https://doc.rust-lang.org/std/io/trait.Read.html) |
| `rust-buffered-output-completion` | M04/C07/API27 | Apply to a concrete buffered writer stack and explicit required output/error-completion promise. Fail when success is reported although pending output or completion errors are lost by Drop, unchecked completion, or discarded ownership. Establish that bytes can remain buffered and trace every relevant success path. std::io::BufWriter Drop attempts writing its buffer but ignores errors; write_all into it alone does not ensure downstream completion. [BufWriter](https://doc.rust-lang.org/std/io/struct.BufWriter.html), [C-DTOR-FAIL](https://rust-lang.github.io/api-guidelines/dependability.html#c-dtor-fail) |
| `rust-fallible-drop` | M05/API27 | Apply to an actual Drop implementation and visible ordinary cleanup failure or unwind path. Fail when cleanup panics on that reachable failure contrary to the promised cleanup/error policy, including a possible second panic while unwinding. Show the real operation/failure and ownership path; panic in Drop is not automatically forbidden. [Drop panics](https://doc.rust-lang.org/std/ops/trait.Drop.html#panics), [C-DTOR-FAIL](https://rust-lang.github.io/api-guidelines/dependability.html#c-dtor-fail) |
| `rust-unbounded-input-buffering` | M09/C06 | Apply to a visible untrusted input boundary with an explicit protocol/resource limit and a storage-growth path. Fail if external length/framing expands storage beyond the required effective bound, checking occurs only after unbounded growth, or a cap silently accepts an incomplete/truncated frame as complete. Identify the actual byte/resource limit and missing/rejected framing validation. [BufRead growth](https://doc.rust-lang.org/std/io/trait.BufRead.html#method.read_until), [Read::take](https://doc.rust-lang.org/std/io/trait.Read.html#method.take) |
| `rust-exclusive-create-race` | C09 | Apply when a visible filesystem operation promises never to overwrite an existing target and competing creation is possible. Fail when a separate absence check followed by std File::create or ordinary create is relied upon for exclusivity: another creator between the operations can be overwritten. Require the actual create API and contract. [OpenOptions::create_new](https://doc.rust-lang.org/std/fs/struct.OpenOptions.html#method.create_new) |
| `rust-shared-clone-isolation` | M02 | Apply to an explicit independent mutable snapshot/workspace promise with visible clone semantics and subsequent mutation/observation. Fail when cloning std Arc/Rc or a visibly shared wrapper preserves mutable aliasing so the mutation changes the original contrary to the promise. Show shared allocation identity and the observable isolation violation, not merely clone syntax. [Clone](https://doc.rust-lang.org/std/clone/trait.Clone.html#derivable), [Arc cloning](https://doc.rust-lang.org/std/sync/struct.Arc.html#cloning-references) |

## Source applicability and context

`sourceMatch` is an OR of RE2 patterns over **the primary unit source**, not its imports, parent impl or attached context. Patterns identify operations, never contract wording; qualified calls, method calls and common `use` aliases remain candidates. Matches are a cheap applicability prefilter, not semantic evidence. Tests are structurally skipped by default (`includeTests` remains false).

| Rule | Primary kinds / extended types | `sourceMatch` rationale |
| --- | --- | --- |
| `rust-expected-failure-panics` | function / no | Panic/assert/unreachable macros, qualified/unqualified/method panic accessors, Result-returning APIs and error propagation. Operation matching preserves imported/aliased Result types and UFCS accessors; it does not infer type identity from names. |
| `rust-error-cause-loss` | function / yes | Fallible boundary signatures (including qualified Result and common I/O aliases), causal Error::source implementations, and lower-error mapping. Error/Err-suffixed type references select direct error-to-error signatures and variant-producing matches even without Result, ?, map_err or Err(). `fn from` selects conversion methods whose primary source omits the enclosing impl header; type context supplies the actual conversion identity. These spellings select subjects, not proof of error identity or cause loss. |
| `rust-conversion-contract` | function / yes | Method units omit their enclosing impl header: from/try_from method names select those units regardless of qualified, prelude or aliased trait spelling, while type context supplies explicit std::convert::From identity and domain contracts. Named truncation/rounding operations and conversion bounds are selected so legitimate exception fixtures remain applicable. None of the verdicts infer the standard trait from the method name alone. |
| `rust-validation-bypass` | type, function / yes | Type declarations own exposed representations; function units own safe constructors, setters, mutable accessors, conversions, and concrete construction expressions. Validated-domain identity can reside entirely in attached type documentation, so narrowing to new/validate/unchecked spellings would miss ordinary named entry points. These RE2 declaration patterns match actual non-test primary units in every fixture without relying on answer-bearing comments. context.types supplies declaration contracts and impl blocks containing constructors and consumers; only the primary source owns the reported defect. |
| `rust-partial-state-on-error` | function / yes | Fallibility/error/unwind operations select transactional candidates without depending on mutable parameters or contract wording, so interior-mutability and owned-state operations are retained. Visible mutation and atomicity/reuse remain judgment requirements. |
| `rust-buffered-output-completion` | function / no | All fixture primary functions explicitly name std::io::BufWriter. Completion spellings also admit common imported/aliased writer operations as candidates without treating method spelling as semantic evidence. Write operations preserve aliased buffered writers whose functions do not spell BufWriter or complete the output. |
| `rust-fallible-drop` | function / yes | Match destructor function primary sources rather than an impl Drop parent. Explicit std/core mem::drop additionally admits the ordinary-owned-fields exception for a subject-absent pass. |
| `rust-unbounded-input-buffering` | function / yes | Match actual read, length-allocation and accumulator operations in function units, including qualified std APIs, trait method syntax, no-contract counterparts, fixed trusted input, bounded upstream cursors and constant-space streaming exceptions. |
| `rust-exclusive-create-race` | function / no | Creation/opening operations, including qualified File, aliased File::create, OpenOptions builder methods and fs::write. Broad method matching avoids dropping imported aliases; judging still requires concrete filesystem identity and exclusivity. |
| `rust-shared-clone-isolation` | function / yes | Qualified/unqualified/method clone and copy-on-write make_mut cover Rc/Arc handles, aliases and container clones. Match operations rather than concrete pointer names to retain imported wrappers; extended type context supplies actual shared fields. |

Validation deliberately has a low-selectivity structural filter: a validated domain can have arbitrary type, constructor and accessor names, with its identity solely in attached declarations. Narrowing to `new`/`validate` would discard legitimate bypasses. Type primaries own exposed fields; function primaries own setters, conversions and construction expressions. A private declaration or correct constructor does not fail for an unchecked setter found only in context.

Extended type context supplies visible declarations and impls, including cross-file candidates subject to exclusions and caps; it is not compiler typing, macro expansion or complete caller discovery. Judge only the primary source. Concrete buffered writer identities and panic contracts are local in their fixtures. Completion means checked delivery/error handling, not fsync or crash durability.

## Fixture corpus

Each `fixtures/<rule-id>/` suite includes at least three distinct failing root shapes and at least three passing cases covering legitimate exceptions. Every strict rule has a no-contract counterpart expected to pass: documentation is removed rather than replacing the implementation. Contract documentation is ordinary API documentation; comments never explain a defect or announce a verdict. Each suite also contains a realistic-size failing/clean pair (at least 80 lines and several primary units), with exactly one violating primary unit in the failing file. For validation these are `fail.rs` and `pass.rs`; other suites use `realistic-fail.rs` / `realistic-pass.rs`.

The eval document is the authoritative list of file paths and expected outcomes. All fixtures are intended as Rust 2021 std-only library crates. Faulty fixtures, including the minimized unsafe validated-reader example, are compiled only and must never be executed. Real-world pairs below are minimized/adapted reproductions, not large upstream source copies and not assertions that every upstream project used the exact fixture API.

| Rule | Fail / pass oracles | Real-world pair and provenance |
| --- | --- | --- |
| `rust-expected-failure-panics` | 5 / 9 | **multiformats/rust-multihash** [source 1](https://rustsec.org/advisories/RUSTSEC-2020-0068.html), [source 2](https://github.com/multiformats/rust-multihash/pull/72), [source 3](https://github.com/multiformats/rust-multihash/issues/70); `real-multihash-bug.rs`, `real-multihash-fixed.rs`. Std-only Rust 2021 adaptation, not verbatim upstream source. Keep the code/length/digest wire structure, a supported algorithm table, structural input validation, unsupported-code panic, and fallible repair. Restrict unsigned varints to their single-octet subset. Fuse hash construction and algorithm accessor into decode_multihash so the local primary function contains the full input-to-panic path. Omit generic code tables, hashing, arbitrary-length varint machinery, and unrelated APIs. |
| `rust-error-cause-loss` | 6 / 10 | **rust-lang/rust (std::io::Error source chain)** [source 1](https://github.com/rust-lang/rust/issues/101817), [proposed correction](https://github.com/rust-lang/rust/pull/101818); `real-io-source-bug.rs`, `real-io-source-fixed.rs`. Adapted the reported immediate-cause skipping shape: source delegates to the inner leaf's source instead of returning the leaf. Removed io::Error storage/platform machinery and TLS dependencies, using a private TransportFailure wrapper and a std-only UnknownIssuer leaf. Explicit source-chain recovery documentation derives from the report's AWS credential-provider trust-store diagnosis requirement. The wrapper's Display is high-level; the contract intentionally requires typed source-chain access. `fail-error-mapper.rs` / `pass-error-mapper.rs` additionally cover collapsed versus faithful direct error-category mapping without a Result boundary. |
| `rust-conversion-contract` | 5 / 10 | **vectordotdev/vector (VRL)** [source 1](https://github.com/vectordotdev/vector/issues/9182), [repair](https://github.com/vectordotdev/vector/commit/4b880840b6359a33c411ce2094c048b2645f5bc8); `real-vector-ip-aton-bug.rs`, `real-vector-ip-aton-fixed.rs`. Retained u32::from(Ipv4Addr).to_be() versus u32::from(Ipv4Addr). Removed VRL expression, parsing and Value scaffolding; wrapped the same numeric conversion in an explicit std::convert::From implementation on a std-only IpNumber type. Public numeric contract is derived from the issue's 1.2.3.4 = 16909060 requirement. The erroneous byte swap is observable on little-endian hosts; the contract covers all architectures. This is an adapted domain conversion reproduction, not a verbatim upstream trait implementation. |
| `rust-validation-bypass` | 5 / 7 | **capnproto-rust / capnp** [source 1](https://github.com/capnproto/capnproto-rust/issues/605); `real-capnp-constant-reader-bug.rs`, `real-capnp-constant-reader-fixed.rs`. Retains the public encoded-word representation, arbitrary safe literal construction, and invariant-relying unchecked reader. Replaces Cap'n Proto pointer arithmetic and generated Owned/Reader machinery with a two-word u64 message whose first word is its root index. Adds a checked constructor to expose the domain contract directly; this constructor is fixture scaffolding for evidence, not a claim that upstream had that exact API. Fixed adaptation restricts words and provides an unsafe constructor with explicit obligations, matching the advisory's encapsulation/unsafe-construction repair. No external crates or verbatim upstream implementation. |
| `rust-partial-state-on-error` | 5 / 10 | **0xMiden/rust-sdk** [source 1](https://github.com/0xMiden/rust-sdk/issues/2221); `real-miden-store-bug.rs`, `real-miden-store-fixed.rs`. Reduce persistent expected input-note and output-script stores to BTreeMaps and executor to a fallible closure. Preserve both pre-execution writes, script lookup during execution, and caller-visible store reuse. Fixed version stages the script view in memory, executes against it, and persists notes/scripts only after success, following the issue proposal. Public atomicity docs derive from the issue acceptance criteria. |
| `rust-buffered-output-completion` | 6 / 11 | **rust-lang/rust** [source 1](https://github.com/rust-lang/rust/issues/37045); `real-child-pipe-bug.rs`, `real-child-pipe-fixed.rs`. Adapt reported f2/f3 ChildStdin reproductions to library functions accepting the pipe rather than spawning a process or sleeping. Keep short buffered payload, possible closed child read end, ignored drop errors versus checked flush. Public contract restates the reported local writing-error requirement, not child acknowledgment or durability. |
| `rust-fallible-drop` | 5 / 8 | **oxidecomputer/dropshot** [source 1](https://github.com/oxidecomputer/dropshot/issues/709); `real-dropshot-bug.rs`, `real-dropshot-fixed.rs`. Retain CloseHandle optional sender, take/send/expect destructor and disconnected receiver ownership path. Replace Tokio oneshot with std mpsc so the paired libraries are std-only; omit runtime/HTTP machinery. Add actual best-effort shutdown docs derived from the issue's requested nonpanicking teardown policy. Fixed version ignores disconnected send, as the issue explicitly permits; explicit close reports errors. |
| `rust-unbounded-input-buffering` | 5 / 8 | **websockets-rs/rust-websocket** [source 1](https://github.com/websockets-rs/rust-websocket/security/advisories/GHSA-qrjv-rf5q-qpxc); `real-websocket-bug.rs`, `real-websocket-fixed.rs`. Retain untrusted network-order extended length, Vec::with_capacity(length), exact complete payload and application size limit. Omit opcode/masking/fragmentation/HTTP handling; document their upstream validation as a precondition. Move the size-limit rejection before allocation in fixed library. Explicit frame_limit contract reflects the upstream patch's tunable limits, without inventing a universal numeric cap. |
| `rust-exclusive-create-race` | 5 / 7 | Ledgerful [PR #459](https://github.com/Ryan-AI-Studios/Ledgerful/pull/459), commit `cbc39bc4e4660e99ca46b0d453e7f78e2e353fe8`: diagnose output’s existence check plus File::create/write_all reduced to a std-only helper; fix uses create_new and preserves advisory check. CLI/manifest/SQLite machinery omitted. |
| `rust-shared-clone-isolation` | 5 / 8 | octocrab [issue #835](https://github.com/XAMPPRocky/octocrab/issues/835), fixed by [PR #842](https://github.com/XAMPPRocky/octocrab/pull/842): Arc<RwLock<BoxBody>> shallow cloning consumed the retry body. Std-only pair replaces the streaming body with Option<Vec<u8>> consumption and retains buffered bytes for a fresh retry owner; HTTP/tower machinery omitted. [Investigation](https://kobzol.github.io/rust/2025/12/30/investigating-and-fixing-a-nasty-clone-bug.html). |

The error-source correction in Rust PR #101818 is a proposed correction, not a claimed merged std change. The Miden state-store pair implements the issue’s proposed staging/commit repair. Their `fixed` fixture names denote the corpus’s corrected implementations; the citations retain upstream status. Conversion’s Vector pair wraps the reported IPv4 numeric conversion in an explicit standard From implementation, and validation’s capnp pair adds a checked constructor solely to make the domain contract visible. These adaptations and omitted machinery are disclosed in the table.

## Removed fixtures

`fixtures/rust-expected-failure-panics/exception-test.rs` was removed because its only subject was a `#[test]` function. Test exemption is now structural, not a model-wording exception or an eval fixture that would produce zero applicable units. No other existing core fixtures were removed. Existing minimal fixtures were retained or rebuilt, and all eval paths were migrated.

## Installation and calibration

Installation uses committed Git snapshots, not arbitrary dirty directories. After committing the pack in the repository containing it, with Rust enabled in the project configuration:

```sh
jevlint plugin install --config jevlint.json "$PWD#packs/rust-core"
jevlint plugin list --config jevlint.json
jevlint check --config jevlint.json path/to/rust/project
```

The manifest language list does not itself scope checked files; every rule includes `**/*.rs`. Installation records the resolved Git SHA rather than a fabricated release pin. Remove with `jevlint plugin remove --config jevlint.json Bairum/rust-core`; after committing an update, refresh with `jevlint plugin update --config jevlint.json Bairum/rust-core`.

With a real provider configured, inspect per-rule eval decisions and actual primary/context sources:

```sh
jevlint eval --config jevlint.json --evals packs/rust-core/jevlint-evals.json --rule rust-shared-clone-isolation --verbose
```

## Verification status

All **140 isolated Rust 2021 fixture targets** were compiled with zero errors or warnings in the final six-pack fixture compilation (432 targets total). The earlier `go test ./internal/packs -run 'RustPack'` covered the original cases' non-test primary-kind/sourceMatch applicability; it was not rerun for this refresh. Faulty fixtures were never executed. Compilation and applicability establish syntax and subject selection, not judgment correctness; compiler/Clippy checks remain the first layer.

The original three real provider-backed runs (`python3 scripts/check-rust-packs.py --eval --repeat 3`) measured fixture recall and specificity at global choice-confidence floor 0.8; the floors were not refit. Opaque fixture names were used. This historical table reports per-run decision metrics: below-floor failures are not reported, matching `check` behavior. Current core/advisory refresh rows and held-out measurements are in [CALIBRATION.md](../../docs/rust/CALIBRATION.md#core-and-advisory-refresh-2026-10-05).

| Rule | Recall @0.8 | Specificity @0.8 | Recommended override | Recall @override | Specificity @override |
| --- | --- | --- | --- | --- | --- |
| `rust-expected-failure-panics` | 0.80 | 0.89 | — | 0.80 | 0.89 |
| `rust-error-cause-loss` | 0.53 | 0.89 | — | 0.53 | 0.89 |
| `rust-conversion-contract` | 0.80 | 0.83 | — | 0.80 | 0.83 |
| `rust-validation-bypass` | 0.00 | 1.00 | 0.40 | 0.87 | 0.95 |
| `rust-partial-state-on-error` | 0.60 | 1.00 | 0.45 | 1.00 | 1.00 |
| `rust-buffered-output-completion` | 0.83 | 0.91 | — | 0.83 | 0.91 |
| `rust-fallible-drop` | 1.00 | 1.00 | — | 1.00 | 1.00 |
| `rust-unbounded-input-buffering` | 0.80 | 0.88 | — | 0.80 | 0.88 |
| `rust-exclusive-create-race` | 1.00 | 0.71 | — | 1.00 | 0.71 |
| `rust-shared-clone-isolation` | 0.27 | 0.88 | — | 0.27 | 0.88 |

In the baseline, `rust-shared-clone-isolation` recall was only 0.27. At 0.8, `rust-expected-failure-panics`, `rust-error-cause-loss`, `rust-conversion-contract`, `rust-unbounded-input-buffering`, `rust-exclusive-create-race` and `rust-shared-clone-isolation` were false-positive-prone (specificity below 0.90). These rules are not blanket bans on unwrap, From, public fields, buffering, Drop, clone or overwrite.

All six Rust packs retain the global floor of **0.8** and contain no rule-level `minFailProbability`; no pack-level floor changes were made. Only consuming projects may opt into the recommended overrides through project rule overlays such as `{"id": "rust-validation-bypass", "minFailProbability": 0.40}`. An overlay replaces the rule floor, which overrides the global floor; `—` means keep 0.8. Those recorded floors were fitted on the old choice-confidence metric and pack calibration will refit them; the field name is `minFailProbability`. Overrides were fitted on these same fixtures and are starting points, not guarantees. Real-project precision remains unmeasured. See [calibration method, caveats and override configuration](../../docs/rust/CALIBRATION.md).

### Error-mapping refresh (2026-10-05)

Direct error-to-error mappers no longer require a Result/?/map_err/Err() token
to be evaluated. Error/Err type references and From method units are selected;
an explicit distinction promised by the boundary still supplies the contract.
On a held-out private Rust workspace (43 files, 958 units), error-cause candidates increased **357 → 381**
(+24, 6.72%); single-rule uncached requests increased by the same amount
(14.002 → 15.010 seconds). Token/dollar costs are not exposed by the CLI.

After three full-core fixture runs and a three-run targeted refresh of the final
error-cause wording, its reporting recall/specificity is **9/18 (0.50)** and
**27/30 (0.90)** versus baseline 0.53/0.89. The new collapsed-category fixture
reports 3/3 at 0.86/0.89/0.89; its faithful counterpart passes 3/3.
This is not an overall no-regression calibration: the original cause-loss
failures report 6/15 (0.40), and unchanged fallible-drop/shared-clone recall
measured 0.93/0.20 versus 1.00/0.27. No choice-confidence floor was lowered; those floors were not refit.

The seeded error-category mapper is now selected, but three full-workspace scans
produce **pass/pass/fail at 0.34** (0/3 reports); confidence for passes is not
emitted by `check`. Earlier targeted-file scans judged it fail 3/3 at
0.60/0.62/0.64, but their discovered declaration/context set differs from the
full workspace. Jevlint's provider protocol returns choice/confidence, not a
textual confidence explanation. On the clean
workspace, combined core/advisory runs report **0/0/0**, with **22/22/23**
below-floor findings versus the original recorded 1 reported and 41 below floor.
See the linked calibration section for exact commands, cost, and mutation-rule
overlap; these measurements do not certify production
precision.
