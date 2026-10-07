# Bairum/design

Opt-in design-taste rules. They are not in the default set because they judge taste, or they need context outside the unit.

The pack declares no languages. Each rule keeps the `include` and `kinds` it had in the root config (`**/*`, and `field`, `comment`, `function`, and/or `type`). It applies to whatever languages the project enables.

```sh
jevlint plugin install /path/to/jevlint#packs/design
```

## Why opt-in

On the 9-repo labelled set, at the shipped 0.80 floor:

- `wrapper-without-value`: no reports at 0.80; 0 correct / 16 false at 0.60.
- `stringly-typed-control-flow`: 0 / 3 at 0.60.
- the other rules in this pack: no reports on that set. That is unmeasured, not evidence they are useless.

`mixed-levels-of-abstraction` on a held-out private Rust workspace: 1 actionable / 6 false reports.

## Rules

`boolean-property-naming`, `coincidental-abstraction`, `constructor-side-effects`, `duplicate-domain-logic`, `excessive-optional-configuration`, `hidden-global-dependency`, `invalid-state-representation`, `mixed-levels-of-abstraction`, `parallel-data-structures`, `redundant-type-properties`, `speculative-generalization`, `stringly-typed-control-flow`, `temporal-coupling`, `unnecessary-complexity`, `wrapper-without-value`.

Fixtures and cases moved with the rules. `jevlint eval --packs` scores them after install.
