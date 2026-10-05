#!/usr/bin/env python3
"""Compile Rust pack fixtures; optionally run real, uncached model evals."""

import argparse
from collections import Counter
from datetime import datetime, timezone
import json
import os
from pathlib import Path
import subprocess
import sys
from tempfile import TemporaryDirectory, gettempdir


PACKS = ("rust-core", "rust-core-advisory", "rust-tokio", "rust-api", "rust-unsafe", "rust-performance", "rust-readability")
TOKIO = '{ version = "=1.53.2", default-features = false, features = ["rt", "rt-multi-thread", "sync", "time", "io-util", "macros"] }'
PYO3 = '{ version = "=0.29.2", default-features = false, features = ["macros", "abi3-py38", "extension-module"] }'


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
        for case in report["cases"]:
            fixture = expected.get(case["name"])
            if fixture is None or case["name"] in seen:
                raise ValueError("unexpected or duplicate eval case")
            if case["rule"] != fixture["rule"] or case["expected"] != fixture["expect"]:
                raise ValueError("eval case rule or expectation differs from fixture")
            if case["actual"] not in ("pass", "fail", "inconclusive"):
                raise ValueError("invalid eval outcome")
            for key in ("pass", "fail", "skip", "abstain", "reported", "belowFloor"):
                count = case["decisions"][key]
                if type(count) is not int or count < 0:
                    raise ValueError(f"invalid decision count: {key}")
            seen.add(case["name"])
            result["cases"].append(case)
        if seen != expected.keys():
            result["errors"].append("eval did not return every fixture case")
    except (OSError, ValueError, KeyError, TypeError) as error:
        result["errors"].append(str(error))
    return result


def summarize(packs, runs, repeat):
    cases = []
    rules = {}
    mismatches = inconclusive = abstain = below_floor = flips = 0
    for name, _, _, fixtures in packs:
        pack_runs = [run for run in runs if run["pack"] == name]
        for fixture in fixtures:
            samples = []
            for run in pack_runs:
                case = next((case for case in run["cases"] if case["name"] == fixture["name"]), None)
                actual = case["actual"] if case else None
                samples.append({"run": run["run"], "actual": actual})
                mismatches += actual in ("pass", "fail") and actual != fixture["expect"]
                inconclusive += actual not in ("pass", "fail")
                if case:
                    abstain += case["decisions"]["abstain"]
                    below_floor += case["decisions"]["belowFloor"]
            counts = Counter(sample["actual"] for sample in samples)
            outcome, votes = counts.most_common(1)[0]
            majority = outcome if outcome is not None and votes > repeat / 2 else "inconclusive"
            flipped = len(counts) > 1
            flips += flipped
            cases.append({
                "pack": name, "name": fixture["name"], "rule": fixture["rule"],
                "file": fixture["file"], "expected": fixture["expect"], "runs": samples,
                "majority": majority, "flipped": flipped,
                "mismatches": sum(sample["actual"] in ("pass", "fail") and
                                  sample["actual"] != fixture["expect"] for sample in samples),
                "inconclusive": sum(sample["actual"] not in ("pass", "fail") for sample in samples),
            })
            metric = rules.setdefault(fixture["rule"], {
                "pack": name, "failCases": 0, "detected": 0, "passCases": 0, "passed": 0,
            })
            if fixture["expect"] == "fail":
                metric["failCases"] += 1
                metric["detected"] += majority == "fail"
            else:
                metric["passCases"] += 1
                metric["passed"] += majority == "pass"
    for metric in rules.values():
        metric["recall"] = metric["detected"] / metric["failCases"] if metric["failCases"] else None
        metric["specificity"] = metric["passed"] / metric["passCases"] if metric["passCases"] else None
    errors = [{"pack": run["pack"], "run": run["run"], "message": error}
              for run in runs for error in run["errors"]]
    return {
        "repeat": repeat, "runs": runs, "cases": cases, "rules": rules,
        "flippedCases": flips, "flipRate": flips / len(cases) if cases else 0,
        "mismatches": mismatches, "inconclusive": inconclusive,
        "abstain": abstain, "belowFloor": below_floor, "errors": errors,
    }


def calibration_failed(summary):
    return bool(summary["mismatches"] or summary["inconclusive"] or
                summary["flippedCases"] or summary["errors"])


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--eval", action="store_true", help="also run real model evals (requires configured provider credentials)")
    parser.add_argument("--jevlint", default="jevlint", help="real jevlint executable for --eval (default: jevlint)")
    parser.add_argument("--pack", choices=PACKS, action="append", help="check only this pack (repeatable)")
    parser.add_argument("--repeat", type=int, default=1, help="number of real uncached eval runs (default: 1)")
    parser.add_argument("--summary", type=Path, help="JSON summary path (default: timestamped file in the system temporary directory)")
    parser.add_argument("--min-confidence", type=float, default=0.8, help="confidence floor used for evals, matching the recommended consumer config (default: 0.8)")
    args = parser.parse_args()
    if args.repeat < 1:
        parser.error("--repeat must be positive")
    if not args.eval and (args.repeat != 1 or args.summary is not None):
        parser.error("--repeat and --summary require --eval")
    repo = Path(__file__).resolve().parent.parent
    packs, fixtures = load_packs(repo, tuple(dict.fromkeys(args.pack)) if args.pack else PACKS)

    with TemporaryDirectory(prefix="jevlint-rust-packs-") as directory:
        temporary = Path(directory)
        manifest = [
            '[package]', 'name = "jevlint-rust-pack-fixtures"', 'version = "0.0.0"',
            'edition = "2021"', 'autoexamples = false', '',
            '[workspace]', '', '[dependencies]', f'tokio = {TOKIO}', f'pyo3 = {PYO3}', '',
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
            cwd=temporary, check=True, env={**os.environ, "PYO3_NO_PYTHON": "1"},
        )
        print(f"Compiled {len(fixtures)} isolated Rust 2021 fixture targets across {len(packs)} packs.", flush=True)
        if not args.eval:
            print("Only fixture compilation checked; model outcomes remain unverified.")
            return
        runs = []
        for name, rules, eval_path, cases in packs:
            config_path = temporary / f"{name}.json"
            config_path.write_text(json.dumps({"languages": {"rust": {}}, "minConfidence": args.min_confidence, "rules": rules}), encoding="utf-8")
            for run in range(1, args.repeat + 1):
                print(f"Running real model evals: {name} ({run}/{args.repeat})", flush=True)
                runs.append(run_eval(args.jevlint, config_path, eval_path, cases, name, run))
        summary = summarize(packs, runs, args.repeat)
        timestamp = datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%S.%fZ")
        summary_path = args.summary or Path(gettempdir()) / f"rust-pack-evals-{timestamp}.json"
        summary_path.parent.mkdir(parents=True, exist_ok=True)
        summary_path.write_text(json.dumps(summary, indent=2) + "\n", encoding="utf-8")
        print(f"Calibration summary: {summary_path.resolve()}", flush=True)
        if calibration_failed(summary):
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
