# Optional Tokio semantic pack

`Bairum/rust-tokio` is an opt-in, data-only pack for concrete Tokio lifecycle, cancellation, admission and lock-dependency contradictions. It is not a general async style pack or a proof of concurrency correctness. All seven rules use warning severity, Rust-only file matching, function units and abstention when an identified obligation cannot be decided from available evidence. Common evidence and primary-unit judging instructions live in [`guidance.md`](guidance.md), loaded once through the manifest. No rule-specific confidence thresholds are set.

## Source and runtime applicability

The original [concurrency research](../../docs/rust/research/concurrency.md) read Tokio **1.53.2** documentation (tokio-util 0.7.19 was researched but is not a fixture dependency). The rules apply only when the analyzed source identifies the relevant actual Tokio API and execution/ownership contract. For other Tokio versions, check the application's resolved version against the linked contracts before activation; do not transfer Tokio behavior to arbitrary async runtimes or similarly named types.

Jevlint extracts functions without compiler name resolution or macro expansion. Module-level imports and runtime attributes are not necessarily present in a function request. Fixtures therefore keep API identity in qualified calls, signatures or genuine runtime/application documentation. Intact `tokio::select!`, `tokio::pin!` and task bodies supply local evidence; no expanded macro output is assumed. Optional direct-callee context on the blocking-boundary and lock-dependency rules supplies local effects, not global supervisors, upstream bounds or shutdown ownership. Only the primary function is judged.

| Rule | Research coverage | Contract sources and boundary |
| --- | --- | --- |
| `rust-tokio-blocking-boundary` | C01 | [Task execution/blocking](https://docs.rs/tokio/1.53.2/tokio/task/index.html#blocking-and-yielding), [block_in_place](https://docs.rs/tokio/1.53.2/tokio/task/fn.block_in_place.html), [same-task join](https://docs.rs/tokio/1.53.2/tokio/macro.join.html#runtime-characteristics). Require actual Tokio execution and blocked progress; short std mutex sections and compatible multithread blocking sections pass. |
| `rust-tokio-cancellation-progress` | C03 | [select cancellation safety](https://docs.rs/tokio/1.53.2/tokio/macro.select.html#cancellation-safety), [Sender::send](https://docs.rs/tokio/1.53.2/tokio/sync/mpsc/struct.Sender.html#method.send). Require harmful dropped progress/data and live-resource continuation; retained futures and discarded connections pass. |
| `rust-tokio-required-task-ownership` | C04 | [JoinHandle](https://docs.rs/tokio/1.53.2/tokio/task/struct.JoinHandle.html), [timeout](https://docs.rs/tokio/1.53.2/tokio/time/fn.timeout.html), [JoinSet](https://docs.rs/tokio/1.53.2/tokio/task/struct.JoinSet.html). Require explicit required completion/result and a visibly lost owner; intentional best-effort detachment passes. |
| `rust-tokio-admission-bound` | C05 | [Semaphore](https://docs.rs/tokio/1.53.2/tokio/sync/struct.Semaphore.html), [spawn_blocking](https://docs.rs/tokio/1.53.2/tokio/task/fn.spawn_blocking.html), [mpsc](https://docs.rs/tokio/1.53.2/tokio/sync/mpsc/index.html). Name the actual unbounded resource, not a universal capacity; finite/upstream-bounded task populations and valid work-only limits pass. |
| `rust-tokio-shutdown-contract` | C08 | [Graceful shutdown](https://tokio.rs/tokio/topics/shutdown), [mpsc clean shutdown/disconnection](https://docs.rs/tokio/1.53.2/tokio/sync/mpsc/index.html#clean-shutdown), [task cancellation](https://docs.rs/tokio/1.53.2/tokio/task/index.html#cancellation). Only enforce promised cleanup; approved hard shutdown/loss passes. |
| `rust-tokio-lock-dependency` | C11; limited C02/M03 | [RwLock read warning](https://docs.rs/tokio/1.53.2/tokio/sync/struct.RwLock.html#method.read), [Mutex](https://docs.rs/tokio/1.53.2/tokio/sync/struct.Mutex.html). Only concrete same-lock/dependent-child waits; valid async guards across await pass. C02 guard syntax remains Clippy-owned, and M03 is covered only when guard retention causes this visible dependency. |
| `rust-tokio-blocking-cancellation` | C10 | [spawn_blocking cancellation](https://docs.rs/tokio/1.53.2/tokio/task/fn.spawn_blocking.html), [block_in_place](https://docs.rs/tokio/1.53.2/tokio/task/fn.block_in_place.html), [timeout](https://docs.rs/tokio/1.53.2/tokio/time/fn.timeout.html). Started blocking work does not stop merely because an async handle was aborted/timed out; cooperative joined termination and explicitly permitted finite background completion pass. |

### Source prefilters

`sourceMatch` runs on each function's source before evaluation, not the entire file. Patterns deliberately accept unqualified API names as well as `tokio`/`Tokio` identity in the unit, so conventional imports and locally identified aliases are not tied to one path spelling. The filter is cheap subject selection, not proof that an API belongs to Tokio; semantic judgment still requires its identity. Arbitrarily renamed APIs with no locally visible Tokio identity cannot be resolved by the parser.

| Rule suffix | Selected subject spellings |
| --- | --- |
| `blocking-boundary` | Tokio identity, `spawn`, `spawn_blocking`, `block_in_place`, `JoinHandle`, `Runtime`, `join!`, `select!` |
| `cancellation-progress` | Tokio identity, read/write extension APIs, `read_exact`, `read_to_end`, `read_to_string`, `write_all`, `timeout`, reservation, `select!` |
| `required-task-ownership` | Tokio identity, spawn operations, `JoinHandle`, `JoinSet`, `join_next`, `timeout` |
| `admission-bound` | Tokio identity, spawn operations, `Semaphore`, acquisition, `unbounded_channel` |
| `shutdown-contract` | Tokio identity, task groups/handles, abort/join, receiver close/receive, shutdown |
| `lock-dependency` | Tokio identity, `Mutex`, `RwLock`, read/write/lock acquisitions including owned and try forms |
| `blocking-cancellation` | Tokio identity, `spawn_blocking`, `block_in_place`, `shutdown_timeout` |

Test units are excluded by the engine by default, including `#[test]`/`#[tokio::test]` functions and structurally detected test modules. Fixtures use ordinary non-test library functions so their semantic cases remain applicable. Contract-requiring rules include no-contract variants expected to pass: a missing application obligation is not invented. Missing evidence for an already identified obligation may instead require abstention.

### Deferred protocol topics

C12 Notify permit/registration protocols, C13 biased-selection progress, C14 atomic publication, C15 Condvar predicate protocols, C16 scheduler/yield assumptions and C17 special-file kinds remain beyond this selected coverage. They require additional protocol, scheduling, platform or global ownership evidence; no rule here claims to enforce them. Likewise C06 framing, C07 output completion and C09 exclusive file creation are separate obligations, not hidden extra Tokio rules. The original findings, exceptions and source cautions remain in the research document.

## Local activation

`plugin install` accepts GitHub sources and committed local Git repositories, for example `jevlint plugin install --config jevlint.json /path/to/repo#packs/rust-tokio`. It installs the committed snapshot and records a real pin; it does not install a dirty plain pack directory. For direct local use of uncommitted pack edits without inventing a pin:

1. Enable the built-in Rust parser in your existing project configuration: set `"languages": {"rust": {}}` (merge with existing languages).
2. Copy the seven objects from this pack's `rules.json` `rules` array into the project's `rules` array, and set each copied rule's `guidance` to the **text** of `guidance.md` (not its filename). Keep their `include: ["**/*.rs"]` and `sourceMatch` filters. The manifest's language list is not a per-rule file filter. Avoid duplicate IDs if a pack is already installed.
3. From the repository root, use the real check/eval interfaces with that configuration:

```sh
jevlint check --config jevlint.json path/to/rust/source
jevlint eval --config jevlint.json --evals packs/rust-tokio/jevlint-evals.json --verbose
```

Provider-backed check/eval needs the project's actual Jev provider setup. Installed/pinned pack configurations can use `jevlint eval --config jevlint.json --packs`; ordinary local copying does not install a pack or fabricate a SHA. See the canonical [verification and status](../../docs/rust/COVERAGE.md#verification-and-status) for exercised checks and calibration limits.

## Public-bug provenance

These are minimized, derived Tokio-user reproductions, not large verbatim copies or claims that every repair was merged upstream. Both files in each `real-…-{bug,fixed}.rs` pair are registered in the eval corpus. The issue-derived repairs retain the reported resource/progress contract; substitutions remove unavailable databases, network stacks and large external inputs.

| Rule suffix | Project and public evidence | Fixture pair and minimization |
| --- | --- | --- |
| `blocking-boundary` | Heddle: [issue #62](https://github.com/HeddleCo/heddle/issues/62), [PR #66](https://github.com/HeddleCo/heddle/pull/66), [fix commit](https://github.com/HeddleCo/heddle/commit/b87adf06ce40a8fb3d95b0f172754bc5e9824bb4) | `real-heddle-runtime-bridge-{bug,fixed}.rs`: synchronous PostgreSQL adapters called `block_in_place` + `Handle::block_on` on current-thread callers. Replaces database work with an owned reference string; the repair isolates a private runtime on a blocking worker, adapting the upstream dedicated-runtime bridge. |
| `cancellation-progress` | pgwire-replication: [issue #5](https://github.com/vnvo/pgwire-replication/issues/5), [v0.3.2 fix changelog](https://github.com/vnvo/pgwire-replication/blob/v0.3.2/CHANGELOG.md) | `real-pgwire-idle-read-{bug,fixed}.rs`: idle timeout cancelled header/payload `read_exact` on a persistent PostgreSQL connection. Uses `DuplexStream` and `Vec` instead of networking, TLS and `BytesMut`; repaired single-shot reads retain header/payload counters outside cancelled futures. Maintainer confirms the v0.3.2 release in the issue. |
| `required-task-ownership` | Duroxide / pg_durable: [issue #44](https://github.com/microsoft/duroxide/issues/44), [open repair PR #45](https://github.com/microsoft/duroxide/pull/45), [proposed repair commit](https://github.com/microsoft/duroxide/commit/46c51fae93ba30b73a74e0b43b0c75b933f8327f) | `real-duroxide-children-{bug,fixed}.rs`: cancellation lost a supervisor-local collection of child `JoinHandle`s. A pending provider poller and `Arc` resource replace SQLx and dispatch loops; a oneshot establishes child creation. Repair owns, cancels and joins the child. PR #45 is open, not a merged-fix claim. |
| `admission-bound` | Quinn: [user bug issue #1131](https://github.com/quinn-rs/quinn/issues/1131) | `real-quinn-queue-{bug,fixed}.rs`: unbounded connection-to-endpoint datagram queues between Tokio tasks. Fixed-size datagrams and Tokio mpsc replace QUIC processing and futures mpsc. The repair implements the issue's proposed shared semaphore reservation through endpoint transmission; it is an issue-derived remedy, not a verified merged patch. |
| `shutdown-contract` | Duroxide / pg_durable: [issue #44](https://github.com/microsoft/duroxide/issues/44), [open PR #45](https://github.com/microsoft/duroxide/pull/45), [proposed commit](https://github.com/microsoft/duroxide/commit/46c51fae93ba30b73a74e0b43b0c75b933f8327f) | `real-duroxide-shutdown-{bug,fixed}.rs`: the same incident's shutdown-before-pool-close obligation. `Arc` models a checked-out provider resource and pending work models blocked I/O. The bug joins only the supervisor; the repair observes child cancellation through an owned `JoinSet`. |
| `lock-dependency` | Tokio application report: [issue #2849](https://github.com/tokio-rs/tokio/issues/2849), [documentation response PR #3389](https://github.com/tokio-rs/tokio/pull/3389) | `real-recursive-reader-{bug,fixed}.rs`: recursive reading and a queued writer cause an application wait cycle. A versioned cache replaces the small reproducer; explicitly polling and retaining a pending writer replaces a scheduling sleep. Repair releases the first reader, completes the writer and requests another snapshot. Upstream documented the hazard; no lock-semantics fix is claimed. |
| `blocking-cancellation` | VM0 (renamed Okou): [issue #13705](https://github.com/vm0-ai/vm0/issues/13705), [merged PR #13719](https://github.com/okou-ai/okou/pull/13719), [fix commit](https://github.com/okou-ai/okou/commit/462dda5e4af14a62569a2b84add4e899aa879c94) | `real-vm0-prefetch-{bug,fixed}.rs`: fire-and-forget blocking memory prefetch continued after Stopped status. Faithfully drops its only handle without draining, rather than inventing an upstream abort call. Uses one `Cursor` worker instead of profile discovery and multi-GB files; channels establish started/post-Stopped ordering. Repair cooperatively cancels between reads and joins before publishing Stopped. |

## Fixture inventory and retained exceptions

The corpus contains **98 intended cases**. Counts below are declarations, not observed model results. The named realistic pairs contain several ordinary application units and at least 80 lines per file; a real-world pair also supplies that size coverage where listed.

| Rule suffix | Fail / pass | Realistic failing / clean pair | Pass coverage |
| --- | --- | --- | --- |
| `blocking-boundary` | 5 / 8 | `realistic-{fail,pass}.rs` | Async sleep; awaited blocking worker; dedicated thread; bounded CPU work; short private std mutex; compatible multithread blocking section. |
| `cancellation-progress` | 6 / 13 | `real-pgwire-idle-read-{bug,fixed}.rs` | Pinned future; external read/read_buf/write progress; reservation; rollback; connection discard; best-effort/approved shutdown loss; semaphore/mutex queue-position loss; no preservation contract. |
| `required-task-ownership` | 5 / 8 | `large-service-{fail,pass}.rs` | Both result layers observed; supervisor transfer; cancel-and-join and continue-and-track timeouts; best-effort detachment; no completion contract. |
| `admission-bound` | 5 / 10 | `large-worker-{fail,pass}.rs` | Finite/work-only population; bounded upstream/executor; lifetime-covering permits; overload loss; intentional token-bucket forget; no resource-bound contract. |
| `shutdown-contract` | 5 / 8 | `large-journal-{fail,pass}.rs` | Drain/join; close/drain; owned supervisor; approved hard loss with teardown observed; documented deadline escalation; no shutdown contract. |
| `lock-dependency` | 3 / 10 | `real-recursive-reader-{bug,fixed}.rs` | Separate locks; released guards/revalidation; nonoverlapping child; handled try acquisition; cycle-breaking timeout; legitimate async I/O guard; recursive read without writer; short std mutex. |
| `blocking-cancellation` | 5 / 7 | `real-vm0-prefetch-{bug,fixed}.rs` | Permitted finite background completion; real underlying operation timeout; cooperative stop/join; bounded pool worker and owned dedicated thread; no stop contract. |

No original fixture paths were removed. Existing commentary was rewritten as genuine application/runtime documentation rather than defect/verdict narration, and cases were expanded to cover distinct root shapes. No test-only fixture remains as a semantic exception: test applicability is structural in the engine.

## Fixture dependency and calibration

Each `.rs` fixture is an independent Rust **edition 2021 library**. Ownership fixtures use the integration compiler's `AtomicUsize::try_update` API; edition 2021 does not imply an older compiler MSRV. The shared compile harness must use the real dependency:

```toml
[dependencies]
tokio = { version = "=1.53.2", default-features = false, features = ["rt", "rt-multi-thread", "sync", "time", "io-util", "macros"] }
```

No tokio-util, filesystem/network feature or mock runtime is required. Capacities, frame sizes and delays express individual fixture contracts, not universal thresholds. Each rule has at least three distinct fail shapes, at least three pass cases covering legitimate exceptions, a several-unit realistic-size failing/clean pair, and a public-bug-derived pair. The failing realistic file contains exactly one violating primary unit; surrounding units provide ordinary application behavior. Fixture comments describe application requirements only, not expected verdicts.

## Verification status

All **98 isolated Rust 2021 fixture targets** passed the shared compile harness without compiler warnings. The focused `TestRustPackEvalCasesHaveApplicableUnits/rust-tokio` check also passed, confirming every case has a non-test function matching its rule's kinds and `sourceMatch`. Faulty fixtures were never executed, because some deliberately block or deadlock. Compilation and applicability establish syntax and subject selection, not judgment correctness.

Three real provider-backed runs (`python3 scripts/check-rust-packs.py --eval --repeat 3`) measured fixture recall and specificity at global `minConfidence: 0.8`, with opaque fixture names. The table reports per-run decision metrics: below-floor failures are not reported, matching `check` behavior.

| Rule | Recall @0.8 | Specificity @0.8 | Recommended override | Recall @override | Specificity @override |
| --- | --- | --- | --- | --- | --- |
| `rust-tokio-blocking-boundary` | 0.33 | 1.00 | 0.40 | 0.80 | 1.00 |
| `rust-tokio-cancellation-progress` | 0.00 | 1.00 | 0.50 | 0.17 | 1.00 |
| `rust-tokio-required-task-ownership` | 0.00 | 1.00 | 0.35 | 0.07 | 1.00 |
| `rust-tokio-admission-bound` | 0.40 | 1.00 | 0.60 | 0.60 | 1.00 |
| `rust-tokio-shutdown-contract` | 0.20 | 1.00 | 0.60 | 0.60 | 1.00 |
| `rust-tokio-lock-dependency` | 0.00 | 1.00 | 0.40 | 0.33 | 1.00 |
| `rust-tokio-blocking-cancellation` | 0.00 | 1.00 | 0.45 | 1.00 | 0.95 |

`rust-tokio-cancellation-progress`, `rust-tokio-required-task-ownership` and `rust-tokio-lock-dependency` are not yet useful even at the recommended overrides: fixture recall is only 0.17, 0.07 and 0.33 respectively. High fixture specificity does not compensate for these missed failures or establish real-project precision.

All six Rust packs retain the global floor of **0.8** and contain no rule-level `minConfidence`; no pack-level floor changes were made. Only consuming projects may opt into the recommended overrides through project rule overlays such as `{"id": "rust-tokio-blocking-boundary", "minConfidence": 0.40}`. An overlay replaces the rule floor, which overrides the global floor; `—` means keep 0.8. Overrides were fitted on these same fixtures and are starting points, not guarantees. Real-project precision remains unmeasured. `jevlint-evals.json` records intended outcomes, not observed provider decisions. See [calibration method, caveats and override configuration](../../docs/rust/CALIBRATION.md).
