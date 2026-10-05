# Rust ecosystem rule research: concurrency, async, I/O and resource boundaries

Original research phase (before implementation): research/proposal only. No repository edits, installations, builds, tests, formatter runs or provider calls were performed during that phase. Sources below were read live; Tokio rustdoc currently reports 1.53.2 and tokio-util 0.7.19. `latest`/`master` URLs are moving references: implementation must match the project's actual dependency/compiler versions.

## Conclusions and policy boundaries

- Start with observable correctness failures: lost partial I/O on cancellation, unintentionally detached required work, admission limits that do not cover actual resources, untrusted unbounded framing, lost buffered output, and incomplete shutdown. These are more defensible Jev judgments than preferred synchronization styles.
- Tokio rules are optional runtime-specific rules, not general Rust norms. Activate only with resolved Tokio APIs or an explicit project runtime contract. An `async fn` name alone does not establish its execution environment; synchronous functions and destructors called from a task still run on that task's thread.
- std mutexes are legitimate and often preferred in async code for short, low-contention, non-awaiting critical sections. Tokio mutex guards are explicitly designed to cross awaits. Neither “always use async locks” nor “never hold any lock across await” is ecosystem-grounded.
- Cancellation is dropping a future, not rollback of completed side effects. Some unsafe-to-restart operations are perfectly legitimate when the connection is discarded or shutdown permits data loss. Thread/task detachment and hard abort are also legitimate when intentional and compatible with lifecycle contracts.
- A channel limit bounds queued message count, not necessarily total memory, message byte size, producer-side allocations or spawned tasks. A semaphore inside an unbounded spawn loop bounds active resource use, not task count. Do not misreport either as no bound; identify exactly the remaining unbounded resource.
- No universal numeric limit follows from these sources: concurrency, queue capacities, byte limits, deadlines and critical-section duration are workload/protocol decisions. Examples' numbers are not standards.
- Do not claim static proof of arbitrary deadlock freedom, data-race freedom, starvation freedom, cancellation safety or atomic correctness. Require concrete execution/dataflow evidence and state uncertainty when outside visible context.

**Classification:** “Official contract” means documented API semantics; “official conditional guidance” means a Rust/Tokio recommendation under stated conditions; “proposal” means the suggested lint policy derived from those facts, not an official named Rust lint. P1 is high-value initial semantic coverage; P2 is selective/context-heavy; P3 is advisory performance coverage. These priorities are this research's prioritization, not an ecosystem standard.

## P1 candidates

### C01 — `rust.async.blocking-boundary`: keep blocking and lengthy non-yielding work off task execution paths

- **Scope/status:** General async principle from std Future docs; Tokio-specific remedies. Official conditional guidance.
- **Trigger:** A known executor task directly or through visible callees performs blocking I/O, `std::thread::sleep`, blocking channel/condition-variable waits, an appreciable CPU loop, or a blocking destructor, without an appropriate boundary. Also a `block_in_place` branch whose same-task `join!`/`select!` sibling must progress while it blocks; or `block_in_place` inside an established current-thread runtime.
- **Why:** Task scheduling is cooperative. Blocking/non-yielding execution can stop unrelated tasks on that worker; same-task multiplexed branches are concurrent, not parallel. `block_in_place` cannot run within the current-thread runtime and suspends same-task siblings even on multithread runtimes.
- **Legitimate exceptions:** Code actually runs on a dedicated thread/blocking worker; small bounded computation; low-contention std lock with short synchronous operations; established multithread `block_in_place` with no needed same-task progress; a synchronous helper never called from task execution.
- **Necessary context:** Runtime flavor, actual call path, operation semantics, work bound, same-task versus spawned concurrency, lock contention expectations. Do not call every `std::fs` mention a violation or mandate `spawn_blocking` for tiny pure operations.
- **Violating fixture:** Current-thread Tokio handler calls blocking `recv` or `sleep`; `join!` combines `block_in_place(|| wait_for_signal())` and the async branch responsible for sending that signal.
- **Passing fixture:** `tokio::time::sleep`; finite blocking read inside `spawn_blocking` whose handle is awaited; CPU-bound work in an existing bounded executor; short std-lock mutation released before async network I/O.
- **Tool overlap:** rustc does not establish executor starvation. Guard-across-await patterns belong first to Clippy C02. Simple known blocking-call matching can be deterministic; Jev's value is hidden synchronous callees, runtime compatibility and dependency between same-task branches.
- **Sources:** https://doc.rust-lang.org/std/future/trait.Future.html#runtime-characteristics ; https://docs.rs/tokio/latest/tokio/task/index.html#blocking-and-yielding ; https://docs.rs/tokio/latest/tokio/task/fn.spawn_blocking.html ; https://docs.rs/tokio/latest/tokio/task/fn.block_in_place.html ; https://docs.rs/tokio/latest/tokio/macro.join.html#runtime-characteristics

### C02 — `rust.async.guard-scope`: release non-async synchronization guards before suspension; review unnecessary async critical sections

- **Scope/status:** Async-specific. Official conditional guidance, with deterministic overlap.
- **Trigger:** A non-async-aware lock guard remains held across an await in executable async code; or an async lock protects ordinary data but remains held through unrelated slow I/O or a wait whose completion visibly requires that protected resource.
- **Why:** Blocking lock acquisition can prevent the suspended lock owner from ever running again. Tokio locks permit awaits but still serialize access; unnecessarily retaining a guard expands contention/dependency risk. `RefCell` borrow guards across await can cause runtime borrow panics, not cross-thread data races.
- **Legitimate exceptions:** Async mutex intentionally protects an I/O resource across the operation; preserving an invariant requires one async critical section; no shared access can happen during suspension, established by ownership/context; a guard is genuinely released before await. Do not punish a compiler-supported explicit `drop` merely because old tutorial wording differs.
- **Necessary context:** Resolved guard type, precise lifetime, whether an await can suspend, shared alias/call graph, operation/invariant requiring the lock, contention. Guards from unrelated third-party APIs need their own contract, not name matching.
- **Violating fixture:** A Tokio mutex protecting a statistics map stays live while awaiting an unrelated external request; a local/current-thread task holds a std mutex while awaiting a worker that acquires the same mutex.
- **Passing fixture:** Extract/copy the needed state in a lexical block, release guard, then await; a Tokio mutex intentionally serializes the entire database connection operation.
- **Tool overlap:** Clippy `await_holding_lock`, `await_holding_refcell_ref`, configured `await_holding_invalid_type`, plus rustc Send errors already cover common cases. Do not duplicate common findings. Semantic candidate should focus on async-aware guards unnecessarily retaining resources or proven dependency cycles. Clippy documents false positives for explicitly dropped guards.
- **Sources:** https://docs.rs/tokio/latest/tokio/sync/struct.Mutex.html#which-kind-of-mutex-should-you-use ; https://tokio.rs/tokio/tutorial/shared-state ; https://rust-lang.github.io/rust-clippy/master/index.html#await_holding_lock ; https://rust-lang.github.io/rust-clippy/master/index.html#await_holding_refcell_ref
- **Source caution:** The shared-state tutorial includes dated compiler text saying explicit `drop` is insufficient for Send analysis. Use the current compiler/API contract and Clippy's documented limitations, not that historical statement as a permanent rule.

### C03 — `rust.async.cancellation-progress`: preserve required progress when cancelling and restarting operations

- **Scope/status:** Tokio-specific contracts plus contextual application correctness.
- **Trigger:** Repeated `select!`, timeout or task cancellation drops a partially-progressed operation and restarts it on the same live resource, with evidence that partial data, owned messages or an application invariant must survive. Named examples: `read_exact`, `read_to_end`, `read_to_string`, `write_all`; a `Sender::send(message)` losing `select!` branch; or custom state mutations separated by an await.
- **Why:** Those I/O methods are not cancellation safe and may lose progress. Dropping a losing send future drops its owned message, although the message was not sent. A lock dropped by cancellation is not transactional rollback of state already changed.
- **Legitimate exceptions:** Losing branch discards/closes the connection; best-effort message loss is the intended contract; shutdown permits incomplete operation loss; an existing transaction/RAII protocol restores invariants; a future is pinned and retained across loop iterations rather than recreated; a higher-level API documents cancellation safety.
- **Necessary context:** Exact operation API/version, whether it is actually dropped or retained by reference, post-cancel resource reuse, partial-progress bookkeeping, ownership of message, required invariants/transaction boundaries. Queue-position loss is separate from data loss: cancelling `Mutex::lock`, `RwLock` acquisition or `Semaphore::acquire` can lose fairness position, not an already-owned message.
- **Violating fixture:** `select!` loop recreates `read_exact` on the same stream after each tick, even after partial input; timed-out `write_all` retries the entire payload on the same stream; retry-required message moved into `tx.send` inside a cancellation race.
- **Passing fixture:** Persist a read/write future across selection; use cancel-safe `read`/`write` with progress stored outside the cancellable operation; discard the connection on timeout; keep message owned outside `select!`, reserve channel capacity and then send through the permit.
- **Tool overlap:** API call matching is mechanical, but whether lost progress is harmful requires context. Do not produce a blanket “no read_exact/write_all in select” rule. Clippy `unused_io_amount` handles ignored byte counts, not cancellation/restart protocols.
- **Sources:** https://docs.rs/tokio/latest/tokio/macro.select.html#cancellation-safety ; https://docs.rs/tokio/latest/tokio/time/fn.timeout.html ; https://docs.rs/tokio/latest/tokio/sync/mpsc/struct.Sender.html#method.send ; read method documentation via official source: https://raw.githubusercontent.com/tokio-rs/tokio/master/tokio/src/sync/mpsc/bounded.rs (send's Cancel safety section).

### C04 — `rust.concurrency.work-ownership`: do not lose ownership or failure results of required background work

- **Scope/status:** General thread ownership plus Tokio semantics. Official contracts; conditional lint proposal.
- **Trigger:** Work required to complete a request, commit data, release resources, or satisfy shutdown is spawned and its only join handle is dropped without a supervisor/error route. Also timeout consumes the only Tokio `JoinHandle` and code assumes timeout cancelled the child; or only the outer join result is checked while an important inner `Result` is intentionally discarded.
- **Why:** Dropping std/Tokio join handles detaches work. Tokio's task continues and its return value is lost. Awaiting `JoinHandle<Result<T,E>>` has two failure layers: task/join failure and operation failure. `timeout(duration, handle)` cancels the handle-wait future by dropping it, not the spawned task.
- **Legitimate exceptions:** Intentional detached telemetry/best-effort work with acceptable loss; lifecycle/error ownership elsewhere via a supervisor, channel or existing task group; operation result is intentionally nonessential; explicit cancellation policy. Detached task work starts independently; do not describe dropping a spawned handle as a future that never ran.
- **Necessary context:** Importance of result, ownership transfer and stored handles, supervisor behavior, timeout ownership/reference, panic/cancellation versus business errors, expected parent lifetime. `JoinSet` is different: dropping it aborts its async tasks; `TaskTracker` is not automatically the same policy.
- **Violating fixture:** Handler spawns a durable write, drops handle and responds success; timeout owns a task handle and then closes resources as if the task stopped; `handle.await?` returns an inner error ignored before reporting success.
- **Passing fixture:** Await/check both layers (possibly `await??` where types allow); transfer task to existing supervisor and route errors; timeout polls `&mut handle` and then explicitly selects cancel-and-join, continue-and-track, or detach policy. An intentional detached notification remains passing.
- **Tool overlap:** rustc unused-result/future warnings and Clippy `let_underscore_future` catch some syntax; they do not prove task ownership contracts. Avoid duplicating generic error-handling rules from the API slice. JoinSet/task-group type-specific behavior is mandatory context.
- **Sources:** https://docs.rs/tokio/latest/tokio/task/struct.JoinHandle.html ; https://doc.rust-lang.org/std/thread/struct.JoinHandle.html ; https://docs.rs/tokio/latest/tokio/time/fn.timeout.html ; https://docs.rs/tokio/latest/tokio/task/struct.JoinSet.html

### C05 — `rust.concurrency.admission-bound`: bound externally expandable resource usage at the boundary that actually owns it

- **Scope/status:** General resource-safety principle; official Tokio conditional guidance/examples for implementation.
- **Trigger:** An untrusted/unbounded request source causes unrestricted spawning, growing queues or resource acquisition without an effective bound; many CPU-heavy `spawn_blocking` jobs are submitted without limiting executed computations; a semaphore permit is dropped before the protected resource/work ends; or limiter placement is claimed to bound task count but only limits work inside already-spawned tasks.
- **Why:** Tokio's default blocking thread cap is large and excess work queues. Unbounded request concurrency can exhaust network/file handles and enable denial of service. A permit released early no longer bounds the still-live resource. Bounded channels provide backpressure at the sender, not universal whole-pipeline byte bounds.
- **Legitimate exceptions:** Small finite workload; bounded upstream admission; external executor/pool already establishes the relevant limit; fire-and-forget overload loss is explicitly appropriate; a semaphore intentionally bounds requests but not a separately bounded finite task set; token bucket permits intentionally consumed with `forget` rather than released.
- **Necessary context:** Source cardinality/trust, queue/message sizes, spawned waiting-task count, permit acquisition/release scope, actual lifetime of file/socket/computation, existing upstream quotas and pool limits. Acquire before spawning if task count is the resource needing a bound; acquiring inside the task may be correct for another resource.
- **Violating fixture:** Unbounded network accept loop spawns tasks that acquire permits only after spawning, with no bound on waiting tasks; CPU job per external message submitted to blocking pool with no admission control; `let permit = acquire().await; drop(permit); open_file().await`.
- **Passing fixture:** Admission acquires an owned permit before accepting/spawning and moves it through task completion and resource drop; bounded channel rejects/backpressures producers before allocations accumulate; CPU jobs go through an already installed bounded executor or semaphore with permit moved into the blocking closure.
- **Tool overlap:** No compiler check establishes resource bounds. rustc/Clippy `let_underscore_lock` covers some accidentally dropped lock patterns, not all semaphore ownership mistakes. Do not split the same unlimited pipeline into separate spawn/channel/semaphore complaints unless independent resources remain unbounded.
- **Sources:** https://docs.rs/tokio/latest/tokio/task/fn.spawn_blocking.html ; https://docs.rs/tokio/latest/tokio/sync/struct.Semaphore.html#limit-the-number-of-incoming-requests-being-handled-at-the-same-time ; https://docs.rs/tokio/latest/tokio/sync/mpsc/index.html ; https://doc.rust-lang.org/std/sync/mpsc/fn.sync_channel.html

### C06 — `rust.io.untrusted-frame-bound`: bound untrusted frames before unrestricted buffering or allocation

- **Scope/status:** General Rust I/O; concrete Tokio codec alternative. Official explicit security guidance.
- **Trigger:** Untrusted line/delimiter/length-prefixed input is read into an expanding buffer without a protocol-appropriate cap, or peer-supplied length drives an allocation before its admissible size is checked. Include `LinesCodec::new` only when no trusted upstream size bound applies.
- **Why:** std `BufRead::read_until`/`read_line` can wait for a delimiter never supplied by an attacker. Tokio-util specifically warns that unbounded line buffering permits unbounded memory consumption and highly recommends limits for untrusted input.
- **Legitimate exceptions:** Already-bounded in-memory input or a proven upstream envelope; trusted locally generated finite data; streaming rather than whole-frame buffering with a real resource/time policy; default codec has a sufficient documented maximum. In particular do not claim every length-delimited codec is unbounded just because `.max_frame_length` is not explicitly called.
- **Necessary context:** Trust boundary, protocol limits, parser allocation path, upstream codec defaults/version, bytes versus character limits, deadline semantics. Byte caps and timeouts address different threats: a byte cap does not stop a slow peer; a timeout alone does not cap rapid memory growth. A cap must reject incomplete/oversized framing rather than silently treat truncation as a valid frame.
- **Violating fixture:** Remote stream repeatedly passed to `read_line` into `String`, or default `LinesCodec` on untrusted TCP input; a network length header immediately feeds `vec![0; n]` without size validation.
- **Passing fixture:** Existing decoder enforces maximum line/frame bytes before unbounded growth and rejects oversize input; `LinesCodec::new_with_max_length(protocol_limit)`; bounded length-delimited codec; streaming parser with bounded working set and documented framing/error handling.
- **Tool overlap:** A literal API check can flag risky primitives but cannot know trust/upstream caps. Deduplicate generic input-validation/memory-bound rules. Clippy `read_zero_byte_vec` is a different deterministic buffer-length mistake, not this security rule.
- **Sources:** https://doc.rust-lang.org/std/io/trait.BufRead.html#method.read_until ; https://doc.rust-lang.org/std/io/trait.BufRead.html#method.read_line ; https://docs.rs/tokio-util/latest/tokio_util/codec/struct.LinesCodec.html#method.new_with_max_length ; exact official warning read at https://raw.githubusercontent.com/tokio-rs/tokio/master/tokio-util/src/codec/lines_codec.rs ; https://docs.rs/tokio-util/latest/tokio_util/codec/length_delimited/struct.Builder.html#method.max_frame_length

### C07 — `rust.io.completion-contract`: satisfy explicit output-completion and durability contracts

- **Scope/status:** General std buffering; Tokio-specific file/buffer semantics. Official API contracts.
- **Trigger:** Code reports a required buffered write complete, drops/converts a writer or exits a task without completing/error-checking the required flush; or it claims durable persistence but only performs a userspace flush/ordinary write. Target explicit data-integrity contracts rather than mandate syncing every file.
- **Why:** std BufWriter Drop attempts flushing but ignores errors. Tokio BufWriter Drop discards its buffer. Tokio File writes can return before underlying blocking write completes; its `flush` waits for completion. std `sync_all`/`sync_data` have a distinct filesystem synchronization role. `flush` is not synonymous with crash/power-loss durability.
- **Legitimate exceptions:** Buffer explicitly abandoned on an error/cancel path; contents already flushed or empty; higher-level owner performs flush and handles errors; disposable output; no durability promise. No blanket filesystem sync on transient logs or temporary data. Filesystem/directory durability protocols remain platform/context specific.
- **Necessary context:** Concrete writer stack, Drop/into_inner semantics, pending bytes, ownership, propagation of flush errors, externally promised completion versus durable commit, failure/cancellation paths. Do not confuse std File, Tokio File, std BufWriter and Tokio BufWriter behavior.
- **Violating fixture:** Tokio BufWriter writes a small result, goes out of scope, then returns success; std BufWriter relies on Drop while promising all output errors are reported; durable checkpoint function reports committed after only `flush`.
- **Passing fixture:** Await/check buffer/file flush before success; std `into_inner` whose error is properly handled when it flushes; optional `sync_data`/`sync_all` according to an established durability contract, without claiming that this alone implements atomic replacement or parent-directory durability.
- **Tool overlap:** `unused_io_amount` covers dropped partial counts, not buffering completion/durability. Generic “handle Result” rules overlap only when a flush call exists and its result is discarded. Consolidate with C03/C08 when the same lost output is caused by cancellation/shutdown.
- **Sources:** https://doc.rust-lang.org/std/io/struct.BufWriter.html ; https://docs.rs/tokio/latest/tokio/io/struct.BufWriter.html ; https://docs.rs/tokio/latest/tokio/fs/index.html#using-file ; https://doc.rust-lang.org/std/fs/struct.File.html#method.sync_all ; https://doc.rust-lang.org/std/fs/struct.File.html#method.sync_data

### C08 — `rust.concurrency.shutdown-contract`: stop admission, communicate shutdown and await required cleanup

- **Scope/status:** General lifecycle judgment with official Tokio shutdown guidance.
- **Trigger:** A service shutdown path whose contract requires draining/flushing simply returns from main, drops a receiver/task set or aborts workers without giving cleanup a chance and observing its completion. Also a worker expects channel EOF while reachable sender clones owned by the shutdown coordinator remain live.
- **Why:** Tokio runtime shutdown cancels async tasks. Tokio's graceful-shutdown guide separates deciding to stop, notifying all parts, and waiting. Receiver drop discards unread messages; Tokio mpsc documents closing receiver then draining as usual clean shutdown. EOF occurs only after all senders disappear and buffered messages are received. `abort` signals cancellation but does not wait for destructors to finish; await the handle to observe completion.
- **Legitimate exceptions:** Immediate hard shutdown explicitly permits loss; disposable/read-only work; application intentionally cancels tasks rather than drains; lifecycle owned by an existing supervisor; a documented deadline escalates from graceful cancellation to abort. Do not impose graceful shutdown on every tiny CLI/test.
- **Necessary context:** Application lifecycle and cleanup promise, who admits requests, outstanding channel permits/producer handles, task ownership, worker cancellation response, flush/commit steps, shutdown deadline/escalation behavior.
- **Violating fixture:** Shutdown signal received, coordinator returns from Tokio main while required write tasks remain; receiver dropped with queued durable jobs; coordinator awaits `recv == None` while retaining a sender; calls `handle.abort()` then immediately reuses resources requiring completed teardown.
- **Passing fixture:** Existing owner stops admission, closes/drains appropriate channel, requests cooperative shutdown, flushes required work and awaits tracked tasks; explicitly approved hard-abort mode joins aborted task handles. TaskTracker/JoinSet are options, not mandatory new dependencies or abstractions.
- **Tool overlap:** Deterministic unused handles cover a subset, not lifecycle correctness. Group C04 task ownership, C07 completion and C08 into one finding when one shutdown defect causes all three observations.
- **Sources:** https://tokio.rs/tokio/topics/shutdown ; https://docs.rs/tokio/latest/tokio/sync/mpsc/index.html#clean-shutdown ; https://docs.rs/tokio/latest/tokio/sync/mpsc/index.html#disconnection ; https://docs.rs/tokio/latest/tokio/task/index.html#cancellation

### C09 — `rust.fs.atomic-create`: avoid check-then-create where exclusive creation is the correctness/security requirement

- **Scope/status:** General Rust filesystem concurrency/security. Official API contract.
- **Trigger:** Code checks path absence then uses `File::create`/ordinary create to establish exclusivity, with evidence that overwriting/racing another process would violate the contract.
- **Why:** `create_new(true)` atomically fails if the target already exists, including a dangling symlink. A separate existence check can race creation by another process (TOCTOU), as explicitly documented by std.
- **Legitimate exceptions:** Overwriting an existing file is intended; existence check is UI/advisory and correctness is enforced separately; external exclusivity proven in the deployment model; an existing library/OS primitive provides atomic exclusive creation.
- **Necessary context:** Exclusive-create versus overwrite semantics, threat model, path ownership, competing processes, target symlink behavior. This rule does not claim to solve every parent-path/symlink attack, safe temporary-file protocol or atomic file replacement.
- **Violating fixture:** “Do not overwrite an existing export” code uses `if !path.exists() { File::create(path)? }`.
- **Passing fixture:** `File::create_new(path)` or `OpenOptions::new().write(true).create_new(true).open(path)` handles `AlreadyExists`; or intentionally overwrites a regular output without claiming exclusivity.
- **Tool overlap:** Clippy `suspicious_open_options` covers underspecified truncation, not this exclusivity protocol. This candidate is contextual TOCTOU, not a general ban on `exists`.
- **Sources:** https://doc.rust-lang.org/std/fs/struct.OpenOptions.html#method.create_new ; https://doc.rust-lang.org/std/fs/struct.File.html#method.create_new

## P2 selective candidates

### C10 — `rust.async.blocking-cancellation-contract`: do not assume an async deadline/abort stops started blocking work

- **Scope/status:** Tokio-specific official contracts.
- **Trigger:** Started `spawn_blocking` or `block_in_place` work must stop/rollback by a deadline, but the only mechanism is async abort/timeout or runtime `shutdown_timeout`. Also a persistent blocking service loop occupies the general blocking pool without an intentional finite lifetime/capacity plan.
- **Why:** Started blocking closures cannot be aborted by Tokio; runtime shutdown can wait indefinitely for them. `shutdown_timeout` stops waiting, not the closures. Tokio recommends dedicated threads for long-lived/persistent workloads. Async timeout also cannot preempt a future that never yields.
- **Legitimate exceptions:** Finite closure is safe to finish in background; actual blocking API has a deadline/cancel path; work cooperatively checks a stop signal between bounded operations; deliberate bounded long-lived blocking workers with understood capacity; task has not started yet and cancellation-before-start is not relied upon as a guarantee.
- **Necessary context:** Started/queued state, underlying blocking API, cooperative stop points and their maximum latency, persistent worker lifetime, pool sharing, runtime teardown requirements.
- **Violating fixture:** Timeout/abort of a blocking job followed by deleting its output file while it may still write; infinite `spawn_blocking` loop with no termination path, while application requires prompt runtime shutdown.
- **Passing fixture:** Blocking operation itself supports timeout or cooperative termination with completion observed; dedicated thread for persistent work and owned termination/join protocol; finite best-effort background completion is explicit.
- **Tool overlap:** API matching is mechanical; deadline correctness is contextual. Merge with C01/C04/C08 if all describe the same root cause rather than emitting three warnings.
- **Sources:** https://docs.rs/tokio/latest/tokio/task/fn.spawn_blocking.html ; https://docs.rs/tokio/latest/tokio/task/fn.block_in_place.html ; https://docs.rs/tokio/latest/tokio/time/fn.timeout.html

### C11 — `rust.concurrency.lock-dependency`: flag concrete lock/wait dependency cycles, not hypothetical global deadlocks

- **Scope/status:** Tokio API-specific lock behavior; contextual judgment.
- **Trigger:** Visible path holds a Tokio RwLock read guard while awaiting a write on the same lock; or holds a read guard, another write has been queued, then awaits another read whose write-preferring policy blocks behind that writer; or a held mutex awaits a child operation that demonstrably needs the same mutex.
- **Why:** Tokio RwLock is fair/write-preferring: later readers wait for earlier writers. These particular dependencies can prevent a guard holder from making progress toward releasing the guard.
- **Legitimate exceptions:** Different lock instance; guard released before the wait; operations do not overlap; nonblocking try-lock result handled; deliberate timeout breaks the cycle and does not claim guaranteed acquisition. A speculative alias or unproven schedule is insufficient for a categorical deadlock finding.
- **Necessary context:** Lock identity/aliases, acquisition order, live guards, actual await dependencies, FIFO policy and cancellation handling. Do not infer std RwLock uses Tokio's policy; std policy is OS-dependent.
- **Violating fixture:** `let r = lock.read().await; let w = lock.write().await;` with `r` required afterward; or documented read→queued writer→read scenario.
- **Passing fixture:** Release read guard then acquire write and revalidate state; lock-free ownership transfer; acquire the single required guard before work without nested acquisition.
- **Tool overlap:** Compiler may accept these patterns; general Clippy guard lints intentionally exclude async-aware locks. Deterministic same-object reentry catches simple cases; broader alias/dependency judgments must cite their trace.
- **Sources:** https://docs.rs/tokio/latest/tokio/sync/struct.RwLock.html#method.read ; method warning read at https://raw.githubusercontent.com/tokio-rs/tokio/master/tokio/src/sync/rwlock.rs ; https://docs.rs/tokio/latest/tokio/sync/struct.Mutex.html

### C12 — `rust.async.notify-protocol`: respect Notify's single-permit/non-counting semantics and waiter registration

- **Scope/status:** Tokio-specific official contract.
- **Trigger:** Application uses repeated `notify_one` as a queue of independently counted events; or a multi-consumer hand-built queue checks for absence and then waits without registering each pending Notified future before the check, despite concurrent producers/consumers matching the documented lost-wakeup scenario.
- **Why:** Notify stores at most one permit; multiple notifications before a wait coalesce. Tokio's official MPMC example explains the check-before-wait lost-wakeup schedule and uses `Notified::enable` before checking shared queue state.
- **Legitimate exceptions:** One-consumer documented queue pattern; notification is merely a hint and queue is drained/rechecked; coalescing is intentional; an existing channel/semaphore expresses counts directly; registration/synchronization elsewhere proves correctness.
- **Necessary context:** Number of simultaneous consumers, producer ordering, retained futures, queue predicate, whether notifications are counted or hints. A Notify mention alone is not a violation.
- **Violating fixture:** Producer calls notify twice for two required events, consumer expects two awaits to finish; two concurrent receivers both see empty, two sends happen, notification coalesces and one receiver sleeps with a message queued.
- **Passing fixture:** Reuse an existing mpsc channel/semaphore for actual events; official MPMC registration/check/await/reset sequence; documented single-consumer queue drains shared data.
- **Tool overlap:** No compiler proof of protocol intent. A deterministic notify-call count is not enough. Avoid recommending a new bespoke synchronization abstraction.
- **Source:** https://docs.rs/tokio/latest/tokio/sync/struct.Notify.html

### C13 — `rust.async.biased-progress`: ensure control/shutdown branches can progress in biased selection

- **Scope/status:** Tokio-specific official conditional guidance.
- **Trigger:** `select! { biased; ... }` has an always/nearly-always-ready high-volume branch above a required shutdown/control branch in a repeating loop; analogous `join!` branch consumes substantial per-poll work before control processing.
- **Why:** Biased selection polls top to bottom and makes fairness the caller's responsibility. Tokio explicitly advises putting shutdown earlier when a stream may be constantly ready.
- **Legitimate exceptions:** Priority is deliberate and acceptable; earlier branches provably become pending; control is serviced elsewhere; normal unbiased selection; deterministic ordering is needed and source throughput is bounded.
- **Necessary context:** Readiness under load, loop behavior, per-poll work, shutdown latency contract, separate supervisors. Ordering alone does not prove starvation.
- **Violating fixture:** Busy event source first and cancellation last in biased select loop, with test concept keeping first branch continuously ready.
- **Passing fixture:** Place required shutdown/control first, remove unnecessary biased polling, or establish budget/other owner making control progress.
- **Tool overlap:** Syntax check finds biased order; semantic rule requires evidence of readiness/work characteristics. Overlap C08 only when shutdown progress is the same defect.
- **Sources:** https://docs.rs/tokio/latest/tokio/macro.select.html#fairness ; https://docs.rs/tokio/latest/tokio/macro.join.html#fairness

### C14 — `rust.concurrency.atomic-publication`: do not use Relaxed as the only publication ordering for other state

- **Scope/status:** General Rust concurrency. Official ordering contracts; high-context judgment, not universal ordering preference.
- **Trigger:** One thread writes state, sets a ready/published atomic flag, another observes that flag and reads the associated state, with only Relaxed operations and no other happens-before relationship establishing publication. Also a read-modify-write ordering is incorrectly assumed to strengthen both halves when docs specify one half remains Relaxed.
- **Why:** Relaxed guarantees atomicity of that operation but does not establish ordering for unrelated memory. Acquire/Release publication depends on the appropriate observation/read-from relationship; using Acquire for a read-modify-write makes its store half Relaxed, and Release makes its load half Relaxed.
- **Legitimate exceptions:** Relaxed counters/metrics not synchronizing other data; Mutex/channel/semaphore/join or other established synchronization already covers the state; self-contained independent atomic value; correct lock-free algorithm with documented proof. SeqCst is an acceptable conservative choice but cannot repair every incorrect algorithm/lifetime protocol.
- **Necessary context:** Both communicating sides, atomic identity, read-from and re-use/generation protocol, non-atomic accesses, any fences/other synchronization, lifetime/unsafe invariants. Do not infer correctness solely from x86 testing, or recommend blanket SeqCst/Acquire/Release rewrites.
- **Violating fixture:** Producer stores a payload atomic then stores `ready=true` Relaxed, consumer sees ready Relaxed and assumes payload publication; no UB is necessary for this logical ordering fixture. A separate unsafe non-atomic payload version overlaps memory-safety research.
- **Passing fixture:** Version-appropriate correct Release publication plus Acquire observation with valid state/lifetime protocol; independent Relaxed `fetch_add` metrics; data transferred via existing channel or protected by mutex.
- **Tool overlap:** rustc `invalid_atomic_ordering` catches unsupported ordering arguments (e.g. load Release), not an admissible but insufficient Relaxed protocol. Unsafe conflicting atomic/non-atomic or mixed-size accesses belong in the memory-safety slice; avoid duplicate rules. Do not claim Jev proves lock-free algorithms.
- **Sources:** https://doc.rust-lang.org/std/sync/atomic/enum.Ordering.html ; https://doc.rust-lang.org/std/sync/atomic/index.html#memory-model-for-atomic-accesses ; https://doc.rust-lang.org/nomicon/atomics.html#relaxed ; https://doc.rust-lang.org/nomicon/atomics.html#acquire-release ; https://doc.rust-lang.org/rustc/lints/listing/deny-by-default.html#invalid-atomic-ordering

### C15 — `rust.concurrency.condvar-predicate`: recheck conditions after wakes under the protecting mutex

- **Scope/status:** General std synchronization. Official API contract.
- **Trigger:** `Condvar::wait` returns and code proceeds as if an application condition became true without checking it again; predicate is checked outside its protecting mutex; or the same Condvar is demonstrably paired with different mutexes across calls.
- **Why:** Condvar waits can wake spuriously, and docs require the predicate to be checked each time wait returns. A Condvar may panic when used with more than one mutex.
- **Legitimate exceptions:** `wait_while` already rechecks; outer loop performs predicate check; a wake is intentionally just a hint and subsequent code independently checks readiness; different independent Condvar instances.
- **Necessary context:** Predicate, protecting lock, loop/caller retry paths, lock identity, timeout handling. Do not mistake “woke” or “timeout did not fire” for “predicate true.”
- **Violating fixture:** `if !ready { guard = cv.wait(guard)?; } use_ready_data()`; same Condvar with two unrelated Mutex instances.
- **Passing fixture:** `while !ready { guard = cv.wait(guard)?; }`; `wait_while` with predicate checked while locked.
- **Tool overlap:** Simple missing-loop patterns may be deterministic checks; Jev adds helper/outer-loop predicate reasoning. No general deadlock/race guarantee follows from compiler acceptance.
- **Source:** https://doc.rust-lang.org/std/sync/struct.Condvar.html#method.wait ; https://doc.rust-lang.org/std/sync/struct.Condvar.html#method.wait_while

### C16 — `rust.async.progress-without-scheduler-assumptions`: preserve progress without treating await/yield as a scheduling guarantee

- **Scope/status:** General Future contract and Tokio cooperative-scheduling contract.
- **Trigger:** Custom Future repeatedly self-polls/spins without an appropriate waker/progress protocol; async loop awaits always-ready non-cooperative operations for unbounded work; code correctness assumes `yield_now` necessarily runs another task before continuing; or `unconstrained` disables the only cooperative budget on an unbounded hot loop.
- **Why:** Future poll should return quickly and must not tight-loop pending polls; always-ready awaits may never return control to the runtime. Tokio documents that yield_now may repoll the current task immediately and need not propagate through combinators.
- **Legitimate exceptions:** Bounded cheap loops; operations already integrate Tokio cooperative budget; intentional unconstrained finite work with evidence; correctness independent of interleaving; event/waker-driven custom future.
- **Necessary context:** Future implementation, always-ready behavior, existing budget/yield propagation, bound on work, exact scheduling dependency. Do not prescribe yield every N iterations or flag every async loop; Tokio APIs already participate in cooperative scheduling where documented.
- **Violating fixture:** Hand-written poll loop continuously retries Pending and assumes progress; spawned task handshake relies on yield_now making child run; unbounded custom ready stream drain with no cooperation.
- **Passing fixture:** Await a actual condition/channel for synchronization; correct register-waker-return-Pending implementation; bounded chunked CPU work with appropriate budget or offload, correctness independent of poll ordering.
- **Tool overlap:** Some custom Future mistakes can be deterministic; scheduling/progress dependencies require context. Merge with C01 for a CPU monopolization root cause.
- **Sources:** https://doc.rust-lang.org/std/future/trait.Future.html#runtime-characteristics ; https://docs.rs/tokio/latest/tokio/task/coop/index.html ; https://docs.rs/tokio/latest/tokio/task/fn.yield_now.html#non-guarantees

### C17 — `rust.async.file-kind`: use Tokio filesystem APIs only for supported ordinary-file semantics

- **Scope/status:** Tokio-specific official guidance.
- **Trigger:** Established named pipe/special descriptor is opened via `tokio::fs` as though it had cancellable socket-like async semantics, particularly while shutdown depends on stopping its blocked read.
- **Why:** Tokio documents that filesystem APIs run blocking operations behind the scenes and should only be used for ordinary files; named pipes can hang runtime shutdown. Dedicated pipe or AsyncFd APIs are appropriate for supported special descriptors.
- **Legitimate exceptions:** Ordinary file; a specialized API or dedicated thread handles the descriptor; intentionally bounded blocking operation with understood teardown; actual path type unknown (do not report definitively based on filename).
- **Necessary context:** Actual object type/platform, open/read semantics, operation lifecycle and shutdown contract. Use version/platform-supported dedicated APIs, not portable-name heuristics.
- **Violating fixture:** Known Linux FIFO read through Tokio File blocks forever while app expects cancellable shutdown.
- **Passing fixture:** Existing `tokio::net::unix::pipe`/AsyncFd-backed handling appropriate to descriptor; ordinary file through tokio::fs.
- **Tool overlap:** Usually no compiler failure. Merge C10/C08 if shutdown hangs are the same finding.
- **Source:** https://docs.rs/tokio/latest/tokio/fs/index.html

## P3 advisory performance candidate

### C18 — `rust.io.batch-small-operations`: consider buffering/batching repeated small external I/O

- **Scope/status:** General std and official Tokio conditional performance guidance, not a correctness requirement.
- **Trigger:** A clearly hot loop issues many tiny writes/reads to external sinks, or Tokio file operations repeatedly cross blocking-pool boundaries unnecessarily, and no buffering/batching occurs at another layer.
- **Why:** std/Tokio BufWriter docs describe benefits for small repeated writes, not large/single/in-memory writes. Tokio fs docs recommend batching work into few blocking calls. Advice must account for latency and memory, not assert a universal throughput win.
- **Legitimate exceptions:** Low-latency interactive protocol needs immediate writes; sink already buffers; few/large writes; in-memory Vec target; working-set constraints require streaming; measured performance establishes current design is appropriate.
- **Necessary context:** Workload/hot path, syscall/pool boundaries, sink buffering, latency requirements, memory ceiling, flush semantics. Do not propose reading an untrusted/very large file entirely merely to reduce calls.
- **Violating fixture concept:** Throughput-oriented file export writes individual bytes repeatedly directly to an unbuffered file with no streaming/latency rationale.
- **Passing fixture concept:** Existing buffer coalesces writes and explicitly flushes at required completion boundaries; already batched writes; deliberately immediate interactive output.
- **Tool overlap:** Mechanical detection of small calls cannot prove performance impact. Label as suggestion absent workload/benchmark evidence; deduplicate allocation/performance judgments from memory slice.
- **Sources:** https://doc.rust-lang.org/std/io/struct.BufWriter.html ; https://docs.rs/tokio/latest/tokio/io/struct.BufWriter.html ; https://docs.rs/tokio/latest/tokio/fs/index.html#tuning-your-file-io

## Deterministic tool checks to reuse rather than reinvent

| Existing check | What it establishes | Remaining semantic work |
|---|---|---|
| rustc ownership/Send/Sync bounds | Illegal moves/borrows and required type bounds | Does not prove lifecycle, lock-order, starvation or resource budgets |
| rustc `invalid_atomic_ordering` | Unsupported order argument on detected atomic operation | Admissible but insufficient ordering/publication |
| rustc `let_underscore_lock` | std lock immediately dropped via wildcard binding | Other guard/permit lifetime or resource coverage |
| Clippy `await_holding_lock` | Common non-async-aware lock across await | Async-lock scope, hidden dependency, documented false-positive handling |
| Clippy `await_holding_refcell_ref` | Common runtime-borrow guard across await | Custom guard protocols and actual cross-await need |
| Clippy `await_holding_invalid_type` | Configured invalid-across-await guard types | Contextual library contract and legitimate lifetime exceptions |
| Clippy `let_underscore_future`; rustc unused future/result checks | Some accidentally discarded future/result expressions | Required detached-task work and both layers of task Result; detached JoinHandle is not inert work |
| Clippy `unused_io_amount` | Common ignored partial read/write amount, including async common patterns | Wrong framing/progress handling even when count is syntactically used; cancellation-resume protocols |
| Clippy `read_zero_byte_vec` | Common capacity-versus-length zero-byte read bug | Untrusted length/frame allocation bounds |
| Clippy `suspicious_open_options` | Underspecified create/truncate options | Atomic exclusive creation and broader TOCTOU contract |

Exact primary references: https://doc.rust-lang.org/rustc/lints/listing/deny-by-default.html#invalid-atomic-ordering ; https://doc.rust-lang.org/rustc/lints/listing/deny-by-default.html#let-underscore-lock ; https://rust-lang.github.io/rust-clippy/master/index.html#await_holding_lock ; https://rust-lang.github.io/rust-clippy/master/index.html#await_holding_refcell_ref ; https://rust-lang.github.io/rust-clippy/master/index.html#await_holding_invalid_type ; https://rust-lang.github.io/rust-clippy/master/index.html#let_underscore_future ; https://rust-lang.github.io/rust-clippy/master/index.html#unused_io_amount ; https://rust-lang.github.io/rust-clippy/master/index.html#read_zero_byte_vec ; https://rust-lang.github.io/rust-clippy/master/index.html#suspicious_open_options . The unused-I/O documentation explicitly says it detects only common patterns: https://raw.githubusercontent.com/rust-lang/rust-clippy/master/clippy_lints/src/unused_io_amount.rs .

## Reject or defer as universal pack rules

- No “always Tokio mutex,” “never std mutex in async,” “never await with any guard,” “always message passing,” “always use RwLock for read-heavy code.” Official Tokio guidance is conditional; performance/contention and resource invariants determine the choice.
- No universal `Arc`/clone ban. Official Tokio shared-state/shutdown examples clone handles/tokens as required ownership operations; cheap sharing is not a payload copy.
- No prohibition on detached tasks, unsafe, atomics, threads or `unwrap`. Report the concrete violated correctness/safety/error contract, not the construct alone.
- No “always use SeqCst” or “SeqCst fixes races”; no “Relaxed is unsafe” blanket. Independent counters explicitly suit Relaxed; ordering validation and protocol proof are different concerns.
- No universal timeout around every async call. Timeouts are cooperative and introduce cancellation semantics; arbitrary durations are not standards. Evaluate external-resource holding only where the operation/lifecycle contract requires a bound.
- No “yield_now makes another task run” guarantee, no universal yield cadence, no fixed maximum critical-section duration or task/channel count.
- No general static race/deadlock/cancellation proof claim. Restrict findings to documented API contracts and visible dependency/ownership traces; defer uncertain custom synchronization rather than diagnose from naming.
- No mandatory TaskTracker/CancellationToken/JoinSet migration. Reuse existing lifecycle ownership if adequate; each has distinct drop/cancellation/result behavior.
- No blanket “flush after every write” or “fsync every file.” Flush on the appropriate output-completion boundary; synchronize according to an actual durability requirement.

## Cross-slice deduplication

Memory slice should own unsafe data-race/mixed-size access validity and allocation/clone recommendations. API slice should own general error policy; C04 adds task-specific join/operation error layers. Tooling slice should own compiler/Clippy configuration and version compatibility. One root-cause finding should normally cover overlapping guard/dependency, task/cancellation/shutdown, admission/frame-byte limits, or cancellation/flush failures rather than stack redundant warnings. Source-based first priorities are C03, C04, C05, C06, C07, C08, C09, with C01/C02 added only beyond existing deterministic coverage; C10–C17 require stronger context and C18 stays advisory. This is a research priority order, not a claim that the current Jevlint context makes these reliable default rules.

## Jevlint capability gate: default eligibility is not established by source strength

Parent's capability scout reports Rust function/type extraction and limited same-file related type/impl context, but no compiler typing, name resolution or macro expansion. Method calls and `Type::qualified` calls do not resolve into direct callee context. Therefore:

- Required source/context information listed above is a **precondition**, not a promise the current linter supplies it. Do not infer concrete Tokio/std API identity from an arbitrary variable or method name, infer hidden callee behavior, or infer runtime flavor from `async`.
- Current candidate rules should use only directly visible local evidence and explicitly supplied/recovered context. Otherwise abstain or classify as opt-in/manual review/deferred; do not present absent global information as evidence of absent supervision/bounds/cleanup.
- No contextual semantic candidate in this handoff is claimed reliable-by-default without fixture evidence that its required context is actually present. Deterministic compiler/Clippy checks remain tooling-owned and should not be recreated by model judgments.
- Macro-heavy rules require intact macro text and relevant local declarations to be available in the analyzed item; without macro expansion, do not assume the shape/name of a custom macro establishes Tokio semantics.

| Candidate | Narrow local review concept | Capability/context requiring opt-in/manual review or deferral |
|---|---|---|
| C01 blocking boundary | Explicit blocking call in a visibly established Tokio task/runtime; known same-task branch dependency | Hidden helper/destructor blocking, runtime config outside item, CPU-cost assumptions |
| C02 guard scope | Explicit known lock creation/guard lifetime and an unrelated await in the same item | Compiler types, custom guards, aliases, contention and invariant reasoning outside item; common cases already Clippy-owned |
| C03 cancellation progress | Visible loop recreates an identifiable operation and reuses the same local resource; owned message is moved into a visibly known send race | API identity via unresolved methods, custom future safety, cancellation rollback/invariants across methods |
| C04 work ownership | Task creation, handle consumption/drop/timeout and success claim all directly visible, with an explicit required-work contract | Missing global supervisor/caller lifecycle cannot be inferred; async method error types need actual type evidence |
| C05 admission bound | Visible producer→spawn→permit ordering and permit scope, with directly established source/resource semantics | Global/upstream bounds and byte accounting; existence of a local semaphore never proves whole-pipeline safety |
| C06 untrusted frame bound | Visible protocol input origin, identifiable buffering constructor/read and size-validation order | Trust/upstream cap/default codec version from outside item; absent local limit is insufficient by itself |
| C07 completion contract | Writer construction, writes, ownership-ending path and explicit completion claim in same item | Actual writer type and wrappers, caller-owned flush, durability contract or platform persistence protocol |
| C08 shutdown contract | Coordinator's explicit shutdown contract and all relevant local task/channel ownership visible | Whole-application admission, producer clones, supervisors and cleanup paths are generally global; opt-in/manual by default |
| C09 atomic create | Explicit local check-then-create and an exclusive-create contract | Path threat model and intentional overwrite policy unavailable locally; qualified source callee bodies are not resolved |
| C10 blocking cancellation | Blocking task setup/timeout and asserted stop-dependent action visible together | Closure's hidden blocking calls/cooperative stop points and runtime shutdown ownership |
| C11 lock dependency | Same locally created lock with live read guard followed by write acquisition | Alias/global lock-order proofs and queued-writer schedules; specialist review rather than generic deadlock detector |
| C12 Notify protocol | Notify creation and count expectations directly visible; official two-consumer queue scenario supplied in full | Cross-method producer/consumer protocol and registration guarantees normally unavailable; specialist opt-in |
| C13 biased progress | Intact known Tokio macro plus explicitly always-ready branch/control dependency visible | Readiness/per-poll load from unresolved stream methods; generic branch order alone is insufficient |
| C14 atomic publication | Both publication sides and absence of another ordering route established in supplied context | Read-from/global synchronization/unsafe lifetime proof; specialist opt-in/manual, not reliable default |
| C15 Condvar predicate | Direct known Condvar wait and predicate/loop in same item | Helper/outer caller recheck, lock identity across methods; conditional local rule only with evidence |
| C16 scheduler assumptions | Visible custom poll tight-loop or explicit yield-based synchronization assumption | Waker/progress protocol across types, ready-stream semantics and cooperative budgeting outside item |
| C17 file kind | Explicit known special-file setup and Tokio file use visible or supplied | Object kind/platform inferred from a path name; method type/lifecycle not resolved |
| C18 batching | Visible repeated tiny writes to an identifiable unbuffered external sink | Hot-path and performance measurement, buffering at other layers; advisory only |

These gates preserve full research coverage while preventing unsupported default enforcement. Passing fixture concepts must include negative controls with imported aliases, intentionally detached work, upstream bounds/owners, valid async guards and externally handled cleanup so missing context does not turn into a false positive.
