# Bairum/design

Opt-in design-taste rules. They are not in the default set because they judge taste, or they need context outside the unit.

The pack declares no languages. Each generic rule keeps the `include` and `kinds` it had in the root config (`**/*`, and `field`, `comment`, `function`, and/or `type`) and applies to whatever languages the project enables. `rust-readability-mixed-levels-of-abstraction` stays Rust-only (`**/*.rs`, function units, its own `sourceMatch` filter) and carries the rust-readability guidance it was measured with. Its three Rust fixtures are evaluated with the pack but are not compiled by `scripts/check-rust-packs.py`, which only covers Rust-only packs.

```sh
jevlint plugin install /path/to/jevlint#packs/design
```

## Why opt-in

On the 9-repo labelled set, at the shipped 0.80 floor:

- `wrapper-without-value`: no reports at 0.80; 0 correct / 16 false at 0.60.
- `stringly-typed-control-flow`: 0 / 3 at 0.60.
- the other rules in this pack: no reports on that set. That is unmeasured, not evidence they are useless.

`rust-readability-mixed-levels-of-abstraction` on a held-out private Rust workspace: 1 actionable / 6 false reports, all false positives on FFI adapter functions. On adapter-heavy Rust, exclude FFI or adapter paths or leave this pack uninstalled.

## Rules

`boolean-property-naming`, `coincidental-abstraction`, `constructor-side-effects`, `duplicate-domain-logic`, `excessive-optional-configuration`, `hidden-global-dependency`, `invalid-state-representation`, `mixed-levels-of-abstraction`, `parallel-data-structures`, `redundant-type-properties`, `rust-readability-mixed-levels-of-abstraction` (Rust only, info), `speculative-generalization`, `stringly-typed-control-flow`, `temporal-coupling`, `unnecessary-complexity`, `wrapper-without-value`.

Fixtures and cases moved with the rules. `jevlint eval --packs` scores them after install.
