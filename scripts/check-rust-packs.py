#!/usr/bin/env python3
"""Compile Rust pack fixtures; optionally run real, uncached model evals."""

import argparse
from datetime import datetime, timezone
import json
import os
from pathlib import Path
import subprocess
import sys
from tempfile import TemporaryDirectory, gettempdir

_SCRIPT_DIR = Path(__file__).resolve().parent
if str(_SCRIPT_DIR) not in sys.path:
    sys.path.insert(0, str(_SCRIPT_DIR))
import score_evals


PACKS = ("rust-core", "rust-core-advisory", "rust-performance", "rust-readability")


def pack_file(pack, root, relative):
    """Resolve document references without allowing paths or symlinks out of pack."""
    if not isinstance(relative, str) or not relative or Path(relative).is_absolute():
        raise ValueError(f"{pack.name}: expected a nonempty relative file path")
    path = (root / relative).resolve(strict=True)
    if not path.is_relative_to(pack) or not path.is_file():
        raise ValueError(f"{pack.name}: file must stay inside its pack: {relative}")
    return path


def load_json(path):
    return json.loads(path.read_text(encoding="utf-8"))


def load_packs(repo, selected=PACKS):
    packs = []
    fixtures = set()
    for name in selected:
        pack = (repo / "packs" / name).resolve(strict=True)
        manifest = load_json(pack_file(pack, pack, "pack.json"))
        if manifest["version"] != 1 or manifest["languages"] != ["rust"]:
            raise ValueError(f"{name}: expected a version 1 Rust pack")
        rules = load_json(pack_file(pack, pack, manifest.get("rules", "rules.json")))["rules"]
        guidance = pack_file(pack, pack, manifest["guidance"]).read_text(encoding="utf-8")
        if not guidance.strip():
            raise ValueError(f"{name}: guidance must be nonempty")
        for rule in rules:
            if not rule.get("guidance"):
                rule["guidance"] = guidance
        eval_path = pack_file(pack, pack, manifest.get("evals", "jevlint-evals.json"))
        evals = load_json(eval_path)
        rule_ids = {rule["id"] for rule in rules}
        if not rules or len(rule_ids) != len(rules):
            raise ValueError(f"{name}: rules must be nonempty and have unique IDs")
        if evals["version"] != 1 or not evals["cases"]:
            raise ValueError(f"{name}: expected nonempty version 1 eval cases")
        names = [case["name"] for case in evals["cases"]]
        if len(set(names)) != len(names) or not all(names):
            raise ValueError(f"{name}: eval case names must be nonempty and unique")
        for case in evals["cases"]:
            if case["rule"] not in rule_ids or case["expect"] not in ("pass", "fail"):
                raise ValueError(f"{name}: invalid eval rule or expectation: {case['name']}")
            fixture = pack_file(pack, eval_path.parent, case["file"])
            if fixture.suffix != ".rs":
                raise ValueError(f"{name}: eval fixture must be Rust: {case['file']}")
            fixtures.add(fixture)
        # Also compile any fixtures not yet referenced by an eval case.
        for fixture in pack.glob("fixtures/**/*.rs"):
            fixtures.add(pack_file(pack, pack, str(fixture.relative_to(pack))))
        packs.append((name, rules, eval_path, evals["cases"]))
    return packs, sorted(fixtures)


def eval_config(rules, threshold):
    return {"languages": {"rust": {}}, "minFailProbability": threshold, "rules": rules}


def argument_parser():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--eval", action="store_true", help="also run real model evals (requires configured provider credentials)")
    parser.add_argument("--jevlint", default="jevlint", help="real jevlint executable for --eval (default: jevlint)")
    parser.add_argument("--pack", choices=PACKS, action="append", help="check only this pack (repeatable)")
    parser.add_argument("--repeat", type=int, default=1, help="number of real uncached eval runs (default: 1)")
    parser.add_argument("--summary", type=Path, help="JSON summary path (default: timestamped file in the system temporary directory)")
    parser.add_argument("--min-fail-probability", type=float, default=0.5, help="minFailProbability written into the eval config (default: 0.5)")
    parser.add_argument("--target-specificity", type=float, default=0.9, help="fitted-threshold specificity target (default: 0.90)")
    return parser


def run_eval(executable, config_path, eval_path, cases, name, run):
    """Keep every run, including failed commands, so calibration cannot hide errors."""
    result = {"pack": name, "run": run, "cases": [], "errors": []}
    try:
        process = subprocess.run([
            executable, "eval", "--config", str(config_path),
            "--evals", str(eval_path), "--format", "json", "--verbose", "--refresh-cache",
        ], capture_output=True, text=True)
        result.update(returncode=process.returncode, stderr=process.stderr)
        if process.returncode not in (0, 1):
            result["errors"].append(f"eval exited {process.returncode}")
        report = json.loads(process.stdout)
        expected = {case["name"]: case for case in cases}
        seen = set()
        for case in report.get("cases") or []:
            fixture = expected.get(case.get("name"))
            if fixture is None or case["name"] in seen:
                raise ValueError("unexpected or duplicate eval case")
            if case.get("rule") != fixture["rule"] or case.get("expected") != fixture["expect"]:
                raise ValueError("eval case rule or expectation differs from fixture")
            seen.add(case["name"])
            result["cases"].append(case)
        if seen != expected.keys():
            result["errors"].append("eval did not return every fixture case")
        result["report"] = report
    except (OSError, ValueError, KeyError, TypeError, json.JSONDecodeError) as error:
        result["errors"].append(str(error))
    return result


def outcome_gate(packs, runs):
    """Same pass/fail gate as before: mismatches, missing outcomes, flips, errors."""
    mismatches = inconclusive = flips = 0
    for name, _, _, fixtures in packs:
        pack_runs = [run for run in runs if run["pack"] == name]
        for fixture in fixtures:
            counts = {}
            for run in pack_runs:
                case = next(
                    (item for item in run.get("cases") or [] if item.get("name") == fixture["name"]),
                    None,
                )
                actual = case.get("actual") if case else None
                mismatches += actual in ("pass", "fail") and actual != fixture["expect"]
                inconclusive += actual not in ("pass", "fail")
                counts[actual] = counts.get(actual, 0) + 1
            if len(counts) > 1:
                flips += 1
    return {
        "mismatches": mismatches,
        "inconclusive": inconclusive,
        "flippedCases": flips,
        "errors": [
            {"pack": run["pack"], "run": run["run"], "message": error}
            for run in runs for error in run.get("errors") or []
        ],
    }


def calibration_failed(summary):
    return bool(summary["mismatches"] or summary["inconclusive"] or
                summary["flippedCases"] or summary["errors"])


def main():
    parser = argument_parser()
    args = parser.parse_args()
    if args.repeat < 1:
        parser.error("--repeat must be positive")
    if not 0 <= args.min_fail_probability <= 1:
        parser.error("--min-fail-probability must be between 0 and 1")
    if not 0 <= args.target_specificity <= 1:
        parser.error("--target-specificity must be between 0 and 1")
    if not args.eval and (args.repeat != 1 or args.summary is not None):
        parser.error("--repeat and --summary require --eval")
    repo = Path(__file__).resolve().parent.parent
    packs, fixtures = load_packs(repo, tuple(dict.fromkeys(args.pack)) if args.pack else PACKS)

    with TemporaryDirectory(prefix="jevlint-rust-packs-") as directory:
        temporary = Path(directory)
        manifest = [
            '[package]', 'name = "jevlint-rust-pack-fixtures"', 'version = "0.0.0"',
            'edition = "2021"', 'autoexamples = false', '',
            '[workspace]', '',
        ]
        for index, fixture in enumerate(fixtures):
            manifest.extend([
                '[[example]]', f'name = "fixture_{index:03d}"',
                f'path = {json.dumps(str(fixture), ensure_ascii=False)}',
                'crate-type = ["lib"]', '',
            ])
        cargo_manifest = temporary / "Cargo.toml"
        cargo_manifest.write_text("\n".join(manifest), encoding="utf-8")
        # Check only: intentionally faulty/unsafe fixtures must never be executed.
        subprocess.run(
            ["cargo", "check", "--all-targets", "--manifest-path", str(cargo_manifest)],
            cwd=temporary, check=True, env=os.environ,
        )
        print(f"Compiled {len(fixtures)} isolated Rust 2021 fixture targets across {len(packs)} packs.", flush=True)
        if not args.eval:
            print("Only fixture compilation checked; model outcomes remain unverified.")
            return
        runs = []
        reports = []
        overrides = {}
        for name, rules, eval_path, cases in packs:
            for rule in rules:
                if "minFailProbability" in rule:
                    overrides[rule["id"]] = rule["minFailProbability"]
            config_path = temporary / f"{name}.json"
            config_path.write_text(
                json.dumps(eval_config(rules, args.min_fail_probability)),
                encoding="utf-8",
            )
            for run in range(1, args.repeat + 1):
                print(f"Running real model evals: {name} ({run}/{args.repeat})", flush=True)
                result = run_eval(args.jevlint, config_path, eval_path, cases, name, run)
                runs.append(result)
                report = result.get("report") or {"cases": result["cases"]}
                report = dict(report)
                report["pack"] = name
                report["cases"] = result["cases"]
                reports.append(report)
        expected = {
            name: cases for name, _, _, cases in packs
        }
        summary = score_evals.score_reports(
            reports,
            threshold=args.min_fail_probability,
            specificity_target=args.target_specificity,
            rule_thresholds=overrides or None,
            expected=expected,
        )
        gate = outcome_gate(packs, runs)
        if any(metrics["inconclusive"] for metrics in summary["packs"].values()) and not gate["inconclusive"]:
            gate["inconclusive"] = sum(metrics["inconclusive"] for metrics in summary["packs"].values())
        summary.update(gate)
        print(score_evals.format_table(summary), flush=True)
        timestamp = datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%S.%fZ")
        summary_path = args.summary or Path(gettempdir()) / f"rust-pack-evals-{timestamp}.json"
        summary_path.parent.mkdir(parents=True, exist_ok=True)
        summary_path.write_text(json.dumps(summary, indent=2) + "\n", encoding="utf-8")
        print(f"Calibration summary: {summary_path.resolve()}", flush=True)
        if calibration_failed(gate):
            print("Calibration failed: mismatches, inconclusive outcomes, flips, or eval errors.", file=sys.stderr)
            return 1
        print("Fixture compilation and repeated real model evals passed for the selected packs.")
        return 0


if __name__ == "__main__":
    try:
        sys.exit(main())
    except (OSError, ValueError, KeyError, TypeError, subprocess.CalledProcessError) as error:
        print(f"check-rust-packs: {error}", file=sys.stderr)
        sys.exit(error.returncode if isinstance(error, subprocess.CalledProcessError) and error.returncode > 0 else 1)
