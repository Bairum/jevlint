# Rust native tooling baseline

Jevlint is a Go repository; the native Rust baseline runs against a **caller-selected real Rust project**, not this checkout. These commands are usage instructions, not a report of executed checks. See the [tooling research](research/tooling.md) for findings and caveats.

## Invocation and prerequisites

From the Jevlint repository root:

```sh
sh scripts/check-rust.sh /absolute/path/to/rust-project
```

From anywhere, use the absolute script path and an explicit project directory:

```sh
sh /absolute/path/to/jevlint/scripts/check-rust.sh /absolute/path/to/rust-project
```

The project-directory argument is optional. When omitted, the script uses the caller's current directory, so run this form **from the Rust project**:

```sh
sh /absolute/path/to/jevlint/scripts/check-rust.sh
```

The script enters the selected directory before invoking Cargo. It preserves `PATH`, rustup toolchain selection (including `RUSTUP_TOOLCHAIN`, directory overrides and project toolchain files), Cargo configuration, and caller compiler/documentation flags. It does not choose a toolchain, guess an MSRV, detect versions, accept arbitrary Cargo flags, or generate a coverage matrix. Directory-dependent configuration and toolchain selection therefore apply to the Rust project. See [rustup overrides](https://rust-lang.github.io/rustup/overrides.html) and [Cargo configuration](https://doc.rust-lang.org/cargo/reference/config.html).

Prerequisites:

- POSIX `sh`; working Cargo, rustc and rustdoc for the project's selected toolchain.
- rustfmt and Clippy installed **for that toolchain**, not just another installed toolchain. Their availability and version compatibility are the caller's responsibility.
- An intentional `Cargo.lock` at the **workspace root**. The script locates the root manifest with `cargo locate-project --workspace --message-format plain`, which also works when invoked from a workspace member, and requires the adjacent lockfile before running the checks. It does not create or update that lockfile. See [locate-project](https://doc.rust-lang.org/cargo/commands/cargo-locate-project.html) and [locked resolution](https://doc.rust-lang.org/cargo/commands/cargo-check.html#manifest-options).
- Any project-specific build tools, native libraries, services or test setup required by the existing project.

The script does not install components or dependencies. Invoking a rustup proxy can independently trigger toolchain/component downloads through rustup's toolchain-file behavior; provision the selected toolchain beforehand when that is undesirable. Ordinary Cargo commands can fetch **locked** dependencies: `--locked` is not offline mode. Cargo may also execute the project's build scripts and tests; use trusted projects. `--frozen`/offline operation requires separately prepared caches and is not this baseline's policy.

## Exact baseline

After lockfile discovery, the script logs and runs these five commands in order, stopping at the first failure and retaining its exit status:

```sh
cargo fmt --all -- --check
cargo check --workspace --all-targets --locked
cargo test --workspace --locked
cargo doc --workspace --no-deps --locked
cargo clippy --workspace --all-targets --locked -- -D warnings
```

This is a **default-feature host lane**, subject to inherited Cargo configuration and environment (for example, an existing default compilation target). It neither overrides those settings nor proves coverage of other configurations.

- **Formatting:** checks without rewriting files. Cargo fmt uses manifest editions and existing rustfmt configuration; macro/fragment limitations and nightly-only settings still apply. [rustfmt](https://github.com/rust-lang/rustfmt#verifying-code-is-formatted).
- **Compilation:** `--all-targets` covers Cargo artifact kinds—libraries, binaries, tests, benches and examples—not all target triples, feature combinations or doctests. `cargo check` skips code generation. [Cargo check](https://doc.rust-lang.org/cargo/commands/cargo-check.html).
- **Tests:** plain `cargo test --workspace` includes default-selected unit/integration tests and library doctests, and normally compiles examples. Manifest exclusions such as `test = false`, `doctest = false` and required features affect selection. `cargo test --all-targets` is not a substitute for this doctest coverage. `ignore` and `no_run` examples have narrower coverage than executed doctests. No extra `cargo build` is added: the test lane already builds its selected artifacts; add a separate build only for artifacts it does not exercise. [Cargo test selection](https://doc.rust-lang.org/cargo/commands/cargo-test.html#target-selection), [doctests](https://doc.rust-lang.org/rustdoc/write-documentation/documentation-tests.html).
- **Documentation:** builds workspace documentation without dependency documentation. Rustdoc-only lints are distinct from rustc/Clippy; warnings follow the project's existing policy and are not automatically errors. Missing documentation is not universally forbidden. [Rustdoc lints](https://doc.rust-lang.org/rustdoc/lints.html).
- **Clippy:** uses default groups—correctness, suspicious, complexity, perf and style—plus any existing project configuration. Correctness is deny-by-default; the other default groups warn. `-D warnings` is this lane's **CI policy**, not a language requirement; check/test/doc retain the project's existing warning policy. The script does not blanket-enable pedantic, restriction, nursery or cargo groups. Narrow, justified `allow` attributes remain legitimate; `expect` can make exceptions self-checking where the MSRV supports it. Avoid broad suppression or weakening correctness just to make CI green. [Clippy groups](https://doc.rust-lang.org/clippy/lints.html), [lint levels](https://doc.rust-lang.org/rustc/lints/levels.html).

Locked resolution fixes dependency selection, not compiler/OS behavior or test nondeterminism. A locked maintainer build also does not prove all dependency combinations available to library consumers work.

## Optional CI warning policy

For **Cargo 1.97+**, official Clippy guidance offers `build.warnings` instead of injecting `-D warnings`, avoiding compiler-flag cache invalidation. In a project-owned lane, **replace the final Clippy command** with:

```sh
CARGO_BUILD_WARNINGS=deny cargo clippy --workspace --all-targets --locked
```

Do not merely prefix `check-rust.sh` with that variable: its existing `-- -D warnings` would remain. The script deliberately has no version detection or alternative mode.

On a compatible Cargo, projects can also independently choose `CARGO_BUILD_WARNINGS=deny` for their check/test/doc commands. This controls configurable lint warnings for local packages; it is not a promise to escalate every non-lint or dependency warning. Consult the selected version's [Cargo build.warnings documentation](https://doc.rust-lang.org/cargo/reference/config.html#buildwarnings) and [Clippy warning guidance](https://doc.rust-lang.org/clippy/usage.html#elevating-warning-to-errors).

Rustdoc warning policy is separate. To opt into documentation warning errors without discarding existing flags, a project-owned POSIX lane can use the following **when `CARGO_ENCODED_RUSTDOCFLAGS` is unset**:

```sh
RUSTDOCFLAGS="${RUSTDOCFLAGS:+$RUSTDOCFLAGS }-D warnings" cargo doc --workspace --no-deps --locked
```

If encoded rustdoc flags are set, they take precedence: append the new argument using the project's existing encoded-flag mechanism instead, preserving its arguments. Do not replace caller `RUSTFLAGS`, `RUSTDOCFLAGS` or their encoded forms with unrelated policy. See [Cargo flag environment variables](https://doc.rust-lang.org/cargo/reference/environment-variables.html).

## Project-owned coverage additions

Declare these explicitly where the project supports them; the script does not infer them:

- **Edition and MSRV:** use the declared edition and actual `package.rust-version`; compile on that MSRV, with per-package lanes when workspace requirements differ. Clippy's MSRV-aware suggestions are not an old-compiler compatibility test. Pin blocking CI toolchains deliberately and review upgrades. [MSRV](https://doc.rust-lang.org/cargo/reference/rust-version.html).
- **Features:** test supported no-default and meaningful feature combinations, including isolated package use where workspace feature unification hides defects. Use `--all-features` only if those features can coexist on that platform/toolchain; it does not cover disabled-feature paths. [Cargo features](https://doc.rust-lang.org/cargo/reference/features.html).
- **Targets and `no_std`:** add explicit supported `--target` lanes and appropriate library/feature selection. Provision target libraries and native tools externally; cross-target behavior tests need a compatible runner/emulator. `--no-run` only compiles. A `no_std` library can legitimately have host tests requiring std. [Compilation targets](https://doc.rust-lang.org/cargo/commands/cargo-check.html#compilation-options).
- **Release profiles:** test/build optimized profiles when overflow, debug assertions or other profile-dependent behavior matters. Debug testing is not evidence of release behavior. [Cargo profiles](https://doc.rust-lang.org/cargo/reference/profiles.html).

## Separate optional layers

These are opt-in project lanes, **not script steps or universal Rust requirements**. Install/provision tools separately and choose actual supported tests, targets and policies.

| Layer | Prerequisites and useful scope | Limits and official documentation |
|---|---|---|
| **Miri** | A compatible, deliberately selected nightly with the Miri component; targeted supported tests via `cargo miri test`. Useful for unsafe ownership, provenance and initialization paths. | Slow interpreter; most FFI/network/platform operations are unsupported. Finds undefined behavior in exercised executions, not proof of soundness; scheduling/weak-memory coverage is incomplete. Unsupported operations are not automatically source defects. [Miri](https://github.com/rust-lang/miri#readme). |
| **Sanitizers** | Compatible nightly, supported sanitizer/target and linker setup; `rust-src` when rebuilding std. Coordinate instrumentation of native/FFI code. Choose ASan, MSan or TSan for the relevant memory/race risk. | Requirements differ by sanitizer; do not copy ASan flags to every lane. Uninstrumented code and unexercised paths weaken coverage; runtime overhead remains. Preserve existing flags when adding instrumentation. [Rust sanitizer support](https://doc.rust-lang.org/nightly/unstable-book/compiler-flags/sanitizer.html). |
| **Loom** | Integrate Loom replacement primitives and dedicated model tests for concurrent protocols; register custom cfgs where required. Loom itself is stable-capable, not inherently nightly-only. | Ordinary std/dependency primitives are invisible unless modeled. Bounds limit exploration; not every relaxed-memory reordering is modeled. It cannot simply be switched on for arbitrary existing tests. [Loom](https://docs.rs/loom/latest/loom/), [cfg checking](https://doc.rust-lang.org/rustc/check-cfg/cargo-specifics.html). |
| **Fuzzing** | Installed cargo-fuzz, compatible nightly, supported platform/C++ toolchain and meaningful existing harnesses, oracles and corpus; run selected targets and retain/minimize regression inputs. Forward intended features to the tested crate. | Bounded campaigns and absence of crashes are not correctness/security proofs. Harness quality and exercised configurations determine coverage. [Rust Fuzz setup](https://rust-fuzz.github.io/book/cargo-fuzz/setup.html), [guide](https://rust-fuzz.github.io/book/cargo-fuzz/guide.html). |
| **Advisories / dependency policy** | Installed `cargo audit` for a prepared lockfile, or `cargo deny --workspace --locked check` with reviewed policy and feature/target graph selection. Keep advisory data current; overlapping advisory tools need not both run. | Reported advisories are not unknown-vulnerability or reachability analysis; license/bans/source rules are organizational policy, not Rust law. Audit can generate a missing lockfile, so prepare it intentionally. Offline data can be stale; new advisories can change results without code changes. [RustSec cargo-audit](https://github.com/rustsec/rustsec/tree/main/cargo-audit#readme), [cargo-deny checks](https://embarkstudios.github.io/cargo-deny/checks/index.html). |
| **Semver compatibility** | Installed `cargo semver-checks`, compatible rustdoc/toolchain support and an explicitly chosen real released/revision baseline; relevant to public libraries. Select supported feature/target lanes. | Feature heuristics are not exhaustive; behavioral and some type/generic breaks still need review. No fabricated revision pins. [Tool documentation](https://github.com/obi1kenobi/cargo-semver-checks#readme), [Cargo SemVer](https://doc.rust-lang.org/cargo/reference/semver.html). |
| **Package readiness** | `cargo package --locked` for publishable packages, optionally inspecting contents with `cargo package --list`; satisfy package metadata/dependency requirements. | Verifies the extracted package, not just the checkout, and does not publish. Missing packaged files/path-only dependencies can expose different failures. Do not use `--no-verify` and claim the package build was checked. Unnecessary for ordinary internal apps. [Cargo package](https://doc.rust-lang.org/cargo/commands/cargo-package.html). |

Successful tooling is evidence only for the checks and configurations actually exercised; none of these layers proves general correctness, unsafe soundness or dependency safety.
