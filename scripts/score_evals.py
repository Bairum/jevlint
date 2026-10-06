#!/usr/bin/env python3
"""Score repeated jevlint eval JSON reports by failProbability."""

import argparse
import json
from pathlib import Path
import subprocess
import sys


def probability(value):
    if isinstance(value, bool) or not isinstance(value, (int, float)):
        return None
    if value != value or value < 0 or value > 1:
        return None
    return float(value)


def auc(scores, labels):
    """Mann-Whitney AUC. Higher score is the fail class. Ties count 0.5."""
    paired = sorted(zip(scores, labels), key=lambda item: item[0])
    n_pos = sum(labels)
    n_neg = len(labels) - n_pos
    if not n_pos or not n_neg:
        return None
    rank_sum = 0.0
    index = 0
    while index < len(paired):
        end = index
        while end + 1 < len(paired) and paired[end + 1][0] == paired[index][0]:
            end += 1
        rank_sum += ((index + 1 + end + 1) / 2) * sum(label for _, label in paired[index:end + 1])
        index = end + 1
    return (rank_sum - n_pos * (n_pos + 1) / 2) / (n_pos * n_neg)


def counts_at(scores, labels, threshold):
    true_pos = false_neg = true_neg = false_pos = 0
    for score, label in zip(scores, labels):
        predicted = score >= threshold
        if label:
            true_pos += predicted
            false_neg += not predicted
        else:
            false_pos += predicted
            true_neg += not predicted
    recall = true_pos / (true_pos + false_neg) if true_pos + false_neg else None
    specificity = true_neg / (true_neg + false_pos) if true_neg + false_pos else None
    return recall, specificity


def fit_threshold(scores, labels, target):
    """Highest recall with specificity >= target. Ties keep the higher threshold.

    Candidates are the observed scores (predict fail when score >= threshold).
    A point above every score predicts no failures; its threshold is null.
    """
    if not scores or not any(labels) or not any(not label for label in labels):
        return None, None
    points = []
    for threshold in sorted(set(scores)):
        recall, specificity = counts_at(scores, labels, threshold)
        points.append((threshold, recall, specificity))
    points.append((None, 0.0, 1.0))
    feasible = [
        (recall, threshold if threshold is not None else float("inf"), threshold)
        for threshold, recall, specificity in points
        if specificity is not None and specificity + 1e-12 >= target
    ]
    if not feasible:
        return None, None
    best_recall = max(item[0] for item in feasible)
    tied = [item for item in feasible if item[0] == best_recall]
    tied.sort(key=lambda item: item[1], reverse=True)
    return tied[0][2], best_recall


def case_key(case):
    rule = case.get("rule") or ""
    name = case.get("name") or ""
    return rule, name or case.get("file") or ""


def case_probability(case):
    direct = probability(case.get("failProbability")) if "failProbability" in case else None
    if direct is not None:
        return direct
    values = [
        probability(unit.get("failProbability"))
        for unit in case.get("units") or []
        if isinstance(unit, dict)
    ]
    values = [value for value in values if value is not None]
    return max(values) if values else None


def align_cases(reports):
    """One record per case. probabilities[i] is repeat i, None if missing."""
    order = []
    seen = {}
    for report in reports:
        for case in report.get("cases") or []:
            key = case_key(case)
            if key not in seen:
                seen[key] = {
                    "rule": key[0],
                    "name": case.get("name") or case.get("file") or "",
                    "file": case.get("file") or "",
                    "expected": case.get("expected"),
                    "probabilities": [],
                }
                order.append(seen[key])
            elif seen[key]["expected"] is None:
                seen[key]["expected"] = case.get("expected")
    width = len(reports)
    for record in order:
        record["probabilities"] = [None] * width
    for index, report in enumerate(reports):
        for case in report.get("cases") or []:
            record = seen.get(case_key(case))
            if record is None:
                continue
            record["probabilities"][index] = case_probability(case)
    return order


def _noise(probabilities, threshold):
    observed = [value for value in probabilities if value is not None]
    decisions = [value >= threshold for value in observed]
    flipped = len(observed) >= 2 and any(decision != decisions[0] for decision in decisions)
    deltas = [
        abs(left - right)
        for left, right in zip(probabilities, probabilities[1:])
        if left is not None and right is not None
    ]
    return flipped, (sum(deltas) / len(deltas) if deltas else None)


def _group_metrics(records, threshold, target, rule_thresholds):
    conclusive = []
    inconclusive = 0
    flips = comparable = 0
    deltas = []
    fail_cases = pass_cases = 0
    for record in records:
        if record["expected"] == "fail":
            fail_cases += 1
        elif record["expected"] == "pass":
            pass_cases += 1
        else:
            inconclusive += 1
            continue
        present = [value for value in record["probabilities"] if value is not None]
        rule_threshold = (rule_thresholds or {}).get(record["rule"], threshold)
        flipped, delta = _noise(record["probabilities"], rule_threshold)
        if len(present) >= 2:
            comparable += 1
            flips += flipped
        if delta is not None:
            deltas.append(delta)
        if len(present) != len(record["probabilities"]) or not present:
            inconclusive += 1
            record["inconclusive"] = True
            record["mean"] = None
            continue
        record["inconclusive"] = False
        record["mean"] = sum(present) / len(present)
        record["flipped"] = flipped
        conclusive.append(record)
    scores = [record["mean"] for record in conclusive]
    labels = [record["expected"] == "fail" for record in conclusive]
    # Recall/specificity use each case's configured threshold, not one shared cut.
    true_pos = false_neg = true_neg = false_pos = 0
    for record in conclusive:
        rule_threshold = (rule_thresholds or {}).get(record["rule"], threshold)
        predicted = record["mean"] >= rule_threshold
        if record["expected"] == "fail":
            true_pos += predicted
            false_neg += not predicted
        else:
            false_pos += predicted
            true_neg += not predicted
    recall = true_pos / (true_pos + false_neg) if true_pos + false_neg else None
    specificity = true_neg / (true_neg + false_pos) if true_neg + false_pos else None
    fitted, recall_at_target = fit_threshold(scores, labels, target)
    return {
        "failCases": fail_cases,
        "passCases": pass_cases,
        "inconclusive": inconclusive,
        "auc": auc(scores, labels),
        "recall": recall,
        "specificity": specificity,
        "recallAtTarget": recall_at_target,
        "fittedThreshold": fitted,
        "flipRate": flips / comparable if comparable else 0,
        "meanAbsDelta": sum(deltas) / len(deltas) if deltas else None,
    }


def score_reports(reports, threshold=0.5, specificity_target=0.9, rule_thresholds=None):
    """Score one or more eval JSON reports. Repeats are consecutive reports of one pack."""
    by_pack = {}
    for report in reports:
        pack = report.get("pack") or "root"
        by_pack.setdefault(pack, []).append(report)
    packs = {}
    for pack in sorted(by_pack):
        records = align_cases(by_pack[pack])
        for record in records:
            record["pack"] = pack
        rules = {}
        for record in records:
            rules.setdefault(record["rule"], []).append(record)
        pack_metrics = _group_metrics(records, threshold, specificity_target, rule_thresholds)
        pack_metrics["rules"] = {
            rule: _group_metrics(rules[rule], threshold, specificity_target, rule_thresholds)
            for rule in sorted(rules)
        }
        packs[pack] = pack_metrics
    repeat = max((len(items) for items in by_pack.values()), default=0)
    return {
        "threshold": threshold,
        "specificityTarget": specificity_target,
        "repeat": repeat,
        "packs": packs,
    }


def format_table(summary):
    rows = [(
        "pack", "rule", "fail", "pass", "inc", "auc", "recall", "spec",
        "rec@tgt", "fitted", "flip", "|dp|",
    )]

    def cell(value, digits=3):
        if value is None:
            return "—"
        return f"{value:.{digits}f}"

    def add(pack, rule, metrics):
        rows.append((
            pack, rule or "*",
            str(metrics["failCases"]), str(metrics["passCases"]), str(metrics["inconclusive"]),
            cell(metrics["auc"]), cell(metrics["recall"]), cell(metrics["specificity"]),
            cell(metrics["recallAtTarget"]), cell(metrics["fittedThreshold"], 2),
            cell(metrics["flipRate"]), cell(metrics["meanAbsDelta"]),
        ))

    for pack, metrics in summary["packs"].items():
        add(pack, "", metrics)
        for rule, rule_metrics in metrics["rules"].items():
            add(pack, rule, rule_metrics)
    widths = [max(len(row[index]) for row in rows) for index in range(len(rows[0]))]
    lines = [
        f"threshold={summary['threshold']:.2f}  specificityTarget={summary['specificityTarget']:.2f}  repeats={summary['repeat']}",
    ]
    for row in rows:
        lines.append("  ".join(text.ljust(widths[index]) for index, text in enumerate(row)))
    return "\n".join(lines)


def thresholds_from_config(config):
    if "minConfidence" in config or any("minConfidence" in rule for rule in config.get("rules") or []):
        raise ValueError('minConfidence was replaced by minFailProbability; see README "Reporting threshold"')
    threshold = config.get("minFailProbability", 0.5)
    overrides = {
        rule["id"]: rule["minFailProbability"]
        for rule in config.get("rules") or []
        if "minFailProbability" in rule
    }
    return threshold, overrides


def run_evals(executable, config_path, eval_path, repeat):
    """Run jevlint eval --format json --verbose --refresh-cache `repeat` times."""
    reports = []
    errors = []
    command = [
        executable, "eval", "--config", str(config_path), "--evals", str(eval_path),
        "--format", "json", "--verbose", "--refresh-cache",
    ]
    for run in range(1, repeat + 1):
        print(f"Running real model evals ({run}/{repeat})", flush=True)
        try:
            process = subprocess.run(command, capture_output=True, text=True)
        except OSError as error:
            errors.append({"run": run, "message": str(error)})
            reports.append({"cases": []})
            continue
        if process.returncode not in (0, 1):
            errors.append({
                "run": run,
                "message": f"eval exited {process.returncode}: {process.stderr.strip()[:500]}",
            })
        try:
            reports.append(json.loads(process.stdout))
        except json.JSONDecodeError as error:
            errors.append({"run": run, "message": f"eval JSON: {error}"})
            reports.append({"cases": []})
    return reports, errors


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("reports", nargs="*", type=Path, help="eval JSON reports, one per repeat")
    parser.add_argument("--config", type=Path, help="jevlint config; with --evals, run the scorer on live evals")
    parser.add_argument("--evals", type=Path, help="jevlint-evals.json for --config")
    parser.add_argument("--jevlint", default="jevlint", help="jevlint executable (default: jevlint)")
    parser.add_argument("--repeat", type=int, default=1, help="uncached eval repeats when using --config (default: 1)")
    parser.add_argument("--pack", default="root", help="pack label for a live run (default: root)")
    parser.add_argument("--threshold", type=float, help="configured minFailProbability (default: config value or 0.5)")
    parser.add_argument("--target-specificity", type=float, default=0.9, help="fitted-threshold specificity target (default: 0.90)")
    parser.add_argument("--summary", type=Path, help="write the JSON summary here; the table always goes to stdout")
    args = parser.parse_args()
    if args.repeat < 1:
        parser.error("--repeat must be positive")
    if not 0 <= args.target_specificity <= 1:
        parser.error("--target-specificity must be between 0 and 1")
    if (args.config is None) != (args.evals is None):
        parser.error("--config and --evals are used together")
    if args.config is None and not args.reports:
        parser.error("pass eval JSON reports, or --config and --evals")
    rule_thresholds = None
    threshold = 0.5 if args.threshold is None else args.threshold
    errors = []
    if args.config is not None:
        config = json.loads(args.config.read_text(encoding="utf-8"))
        config_threshold, rule_thresholds = thresholds_from_config(config)
        if args.threshold is None:
            threshold = config_threshold
        reports, errors = run_evals(args.jevlint, args.config.resolve(), args.evals.resolve(), args.repeat)
        for report in reports:
            report["pack"] = args.pack
    else:
        reports = [json.loads(path.read_text(encoding="utf-8")) for path in args.reports]
    if not 0 <= threshold <= 1:
        parser.error("threshold must be between 0 and 1")
    summary = score_reports(
        reports,
        threshold=threshold,
        specificity_target=args.target_specificity,
        rule_thresholds=rule_thresholds,
    )
    if errors:
        summary["errors"] = errors
    print(format_table(summary), flush=True)
    text = json.dumps(summary, indent=2) + "\n"
    if args.summary:
        args.summary.parent.mkdir(parents=True, exist_ok=True)
        args.summary.write_text(text, encoding="utf-8")
        print(f"Summary: {args.summary.resolve()}", flush=True)
    else:
        print(text, end="")
    return 1 if errors else 0


if __name__ == "__main__":
    try:
        sys.exit(main())
    except (OSError, ValueError, KeyError, TypeError, subprocess.CalledProcessError) as error:
        print(f"score-evals: {error}", file=sys.stderr)
        sys.exit(1)
