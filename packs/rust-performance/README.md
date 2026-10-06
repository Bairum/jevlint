# Rust performance pack

`Bairum/rust-performance` is an **opt-in advisory** pack with two warning rules. It is not a default Rust baseline, a benchmark, a correctness certificate, or a guarantee of fewer allocations/syscalls or better runtime speed. Missing evidence is not a violation.

## Research and sources

| Rule | Research mapping | Required evidence and primary sources |
| --- | --- | --- |
| `rust-perf-repeated-materialization` | [memory M10](../../docs/rust/research/memory.md#m10--avoidable-hot-path-copyingallocation-ap2), with [API26](../../docs/rust/research/api.md#api26--preserve-already-computed-useful-results-when-demonstrated-callers-need-them) supporting reuse of useful intermediate results | Established hot/large workload; invariant input; real preprocessing into owned std data; visible consumers that can reuse it. [Vec guarantees](https://doc.rust-lang.org/std/vec/struct.Vec.html#guarantees), [Clone contract](https://doc.rust-lang.org/std/clone/trait.Clone.html), [C-CALLER-CONTROL](https://rust-lang.github.io/api-guidelines/flexibility.html#c-caller-control), [C-INTERMEDIATE](https://rust-lang.github.io/api-guidelines/flexibility.html#c-intermediate). |
| `rust-perf-small-writes` | [concurrency C18](../../docs/rust/research/concurrency.md#c18--rustiobatch-small-operations-consider-bufferingbatching-repeated-small-external-io) | High-volume throughput contract; identifiable unbuffered external sink; tiny repeated writes; buffering and latency context. [std BufWriter](https://doc.rust-lang.org/std/io/struct.BufWriter.html). The original research also cites [Tokio BufWriter](https://docs.rs/tokio/latest/tokio/io/struct.BufWriter.html) and [Tokio file I/O tuning](https://docs.rs/tokio/latest/tokio/fs/index.html#tuning-your-file-io); this pack's fixtures use only std. |

API26 is supporting conditional reuse guidance here, not a blanket demand to expose every intermediate or a complete implementation of all public-API result-design cases. Repeated-materialization fixtures use invariant parsing and collect/sort preprocessing, not Clippy's simple `redundant_clone` pattern. Necessary snapshots, independent owned results, changing inputs and justified workload tradeoffs remain legitimate. Small-write exceptions include interactive round trips and an already-buffered sink. Flush completion is not file durability.

Both descriptions contain only rule-specific criteria. [`guidance.md`](guidance.md), referenced by `pack.json`, supplies shared primary-unit judging and evidence requirements. Neither rule enables type/callee context or sets `minFailProbability`: fixtures make the std operations, consumer requirements and workload contract visible in the primary function. Missing hidden implementation context is not evidence that work or buffering is absent. An absent workload/performance contract passes.

Each rule asks a 3-level score plus subject-gated violation checks. Fail probability is the minimum of those signals. `rust-perf-repeated-materialization` scores an owned rebuild inside a hot loop; the subject is a loop plus an owned-value call. `rust-perf-small-writes` scores repeated small writes to an unbuffered sink; the subject is a loop plus a write call.

## Applicability filters

| Rule | `sourceMatch` rationale |
| --- | --- |
| `rust-perf-repeated-materialization` | Require a repetition token (`for`, `while`, `loop`, or common iterator repetition methods) and a materialization/preprocessing operation such as `collect`, `parse`, `clone`, `to_vec`, or `sort`, in either order. Matching the operation before repetition preserves reused-result passing examples. |
| `rust-perf-small-writes` | Require a repetition token and `write`, `write_all`, `write_fmt`, or `writeln`, in either order. This covers trait-qualified calls, ordinary methods, aliases and write macros without assuming their sink is external. |

These are cheap RE2 subject filters, not name/type resolution or a diagnosis. They intentionally retain buffered, in-memory and contract-free near-neighbors for semantic judgment. Each fixture contains at least one non-test matching unit. Test units are skipped by default; no existing test-only performance fixture required removal.

## Public bug/fix provenance

| Rule and matched fixtures | Public upstream fix | Minimized behavior and limits |
| --- | --- | --- |
| `rust-perf-repeated-materialization`: `real-tsrun-template-{bug,fixed}.rs` | [DmitryBochkarev/tsrun, `b9736d1`: Avoid unnecessary clones when reading registers](https://github.com/DmitryBochkarev/tsrun/commit/b9736d129530ca696ccfed5b02a77f5b287c3387) | The `Op::TemplateConcat` fix borrows register values and moves construction/interning of the invariant `toString` property key outside the register loop. Fixtures retain the template loop, invariant key and borrowed consumer; custom GC values, interning and method invocation become std strings and `BTreeMap` rendering results. The significant workload is an explicit fixture contract, not an upstream measurement. |
| `rust-perf-small-writes`: `real-nydus-bootstrap-{bug,fixed}.rs` | [dragonflyoss/nydus, `66cb2e1`: Builder: increase dump performance using BufWriter](https://github.com/dragonflyoss/nydus/commit/66cb2e134c2e2025bd153afa3733a9232891e494) | The fix wraps bootstrap/blob `OpenOptions` files in `BufWriter::with_capacity(2 << 17, ...)` and checks completion flushes; its message specifically identifies many small writes. Fixtures merge writer construction and the bootstrap-node emission loop into one primary function, replacing image nodes/encoding traits with fixed-width preencoded entries. Their 100,000-entry, 128-byte workload sizes are synthetic contracts, not upstream timings. |

These are adapted, std-only matched reproductions, not copied upstream files or claims that the source itself proves a speedup. The tsrun key-hoisting shape is retained independently of simple redundant-clone advice. The Nydus pair preserves open flags, bounded buffer capacity, repeated external writes and checked completion; flush is not a crash-durability guarantee.

## Install and activate

Use an existing Jevlint installation and consuming `jevlint.json` with Rust enabled (`"languages": {"rust": {}}`, merged with your existing settings). Once this pack is present in a **committed revision** of the local repository, run these commands from that repository root:

```sh
jevlint plugin install './#packs/rust-performance'
jevlint plugin list
jevlint check .
```

`plugin install` accepts a local Git source plus `#subdirectory`, resolves the real commit, caches the pack and writes its pin to `jevlint.json`. It does not install uncommitted working-tree files. `check` automatically activates configured packs and merges their rules. For a different consuming project, use that project's config with `--config` and the real local repository path. No remote publication, branch or fabricated SHA is assumed.

The syntax/behavior above was checked against [`ParseSpec`](../../internal/packs/source.go), [`pluginInstall`](../../internal/cli/plugin.go), the [`loadProject` pack resolution path](../../internal/cli/cli.go), and the repository's [pack documentation](../../README.md#packs); the commands were not executed. Pack warnings are optional advice, not mandatory performance policy.

## Evaluation fixtures

`jevlint-evals.json` assigns unique names, rules and expected decisions to standalone Rust 2021, std-only library sources under `fixtures/<rule-id>/`. Each rule includes distinct failing shapes, passing near-neighbors and legitimate exceptions, a contract-free passing variant, a realistic multi-unit file with exactly one violating unit, a realistic clean counterpart, and a minimized public upstream bug/fix pair. Documentation states real contracts rather than narrating defects or verdicts. Compile each file separately; do not combine their items or execute faulty fixtures.

The original invariant-parse, invariant-sort, owned-consumer, changing-snapshot, byte-export, record-export, interactive and caller-buffered fixture shapes are retained. Their commentary is rewritten where necessary instead of discarding valid coverage. No existing fixture files were removed.

| Rule | Fail cases | Pass cases | Realistic multi-unit pair |
| --- | --- | --- | --- |
| `rust-perf-repeated-materialization` | 5: invariant parse, collect/sort, owned byte copying, service ingestion, upstream template rendering | 7: reused parse/sort, changing snapshots, independent owned consumers, no-contract variant, service exceptions, upstream fix | `fail_service.rs` / `pass_service.rs` (each at least 80 lines); only `scan_ingestion_batch` violates in the failing file |
| `rust-perf-small-writes` | 6: byte/record export, formatted append, bulk TCP, upstream bootstrap, archive export | 11: buffered byte/record export, caller buffering, interactive protocols, in-memory vectors, bounded qualified streaming, few/large writes, no-contract variant, upstream fix, clean archive | `fail_archive.rs` / `pass_archive.rs` (each at least 80 lines); only `export_pending` violates in the failing file |

The clean materialization service covers cheap `Arc`/`Rc` sharing, scratch capacity reuse, bounded cold setup/reservation and lock-release snapshot lifetime requirements. Small-write exceptions cover both passed-in and locally connected interactive streams. The no-contract fixtures preserve the corresponding failing code while removing its workload documentation, so missing contract evidence is a pass.

After installing a real committed pack and configuring a provider, the existing CLI can run its eval cases:

```sh
jevlint eval --packs --rule rust-perf-repeated-materialization
jevlint eval --packs --rule rust-perf-small-writes
```

## Verification status

Fixture rates at 0.80 are in [CALIBRATION.md](../../docs/rust/CALIBRATION.md). This pack sets no rule-level floor. The shipped default is 0.80. Fixture recall is not a runtime speedup.
