"""Scorer math for repeated eval JSON (no model calls)."""

import contextlib
import importlib.util
import io
from pathlib import Path
import unittest
import score_evals


spec = importlib.util.spec_from_file_location("check_rust_packs", Path(__file__).with_name("check-rust-packs.py"))
runner = importlib.util.module_from_spec(spec)
spec.loader.exec_module(runner)


def case(name, expected, probability, rule="rule"):
    body = {
        "rule": rule,
        "name": name,
        "file": f"{name}.rs",
        "expected": expected,
        "actual": expected,
        "matched": True,
    }
    if probability is not None:
        body["failProbability"] = probability
    return body


def report(cases, pack="root"):
    return {"pack": pack, "cases": cases}


class ScoreTests(unittest.TestCase):
    def test_auc_ties_use_mean_fail_probability(self):
        # Identical repeats: fail 0.9, 0.5 against pass 0.5, 0.1.
        # Pairwise: 1 + 1 + 0.5 + 1 over 4 = 0.875. A tied score is half a win.
        repeats = [
            report([
                case("a", "fail", 0.9),
                case("b", "fail", 0.5),
                case("c", "pass", 0.5),
                case("d", "pass", 0.1),
            ]),
            report([
                case("a", "fail", 0.9),
                case("b", "fail", 0.5),
                case("c", "pass", 0.5),
                case("d", "pass", 0.1),
            ]),
        ]
        summary = score_evals.score_reports(repeats, threshold=0.5)
        self.assertEqual(summary["packs"]["root"]["auc"], 0.875)
        self.assertEqual(summary["packs"]["root"]["rules"]["rule"]["auc"], 0.875)
        self.assertEqual(score_evals.auc([0.4, 0.4], [1, 0]), 0.5)

        # Mean, not max: fail mean 0.3 loses to pass mean 0.4; max would win.
        split = score_evals.score_reports([
            report([case("bug", "fail", 0.0), case("clean", "pass", 0.4)]),
            report([case("bug", "fail", 0.6), case("clean", "pass", 0.4)]),
        ])
        self.assertEqual(split["packs"]["root"]["auc"], 0)
        self.assertIn("rule", score_evals.format_table(summary))

    def test_threshold_fits_at_specificity_boundary(self):
        # 10 passes: 9 at 0.1 and 1 at 0.5. Fails at 0.5 and 0.9.
        # score >= 0.5 false-positives the pass sitting on the boundary, so
        # specificity is exactly 0.90 and that point is kept (recall 1).
        passes = [case(f"p{index}", "pass", 0.1) for index in range(9)]
        passes.append(case("boundary", "pass", 0.5))
        fails = [case("f0", "fail", 0.5), case("f1", "fail", 0.9)]
        summary = score_evals.score_reports(
            [report(passes + fails)], threshold=0.8, specificity_target=0.9,
        )
        metrics = summary["packs"]["root"]
        self.assertEqual(metrics["specificity"], 1)
        self.assertEqual(metrics["recall"], 0.5)
        self.assertEqual(metrics["fittedThreshold"], 0.5)
        self.assertEqual(metrics["recallAtTarget"], 1)

        # Target 1.0 cannot use the shared 0.4 score: >= would count the pass.
        blocked = score_evals.score_reports([report([
            case("clean", "pass", 0.4),
            case("tied", "fail", 0.4),
            case("clear", "fail", 0.9),
        ])], specificity_target=1)
        self.assertEqual(blocked["packs"]["root"]["fittedThreshold"], 0.9)
        self.assertEqual(blocked["packs"]["root"]["recallAtTarget"], 0.5)

    def test_missing_results_are_inconclusive(self):
        summary = score_evals.score_reports([
            report([
                case("bug", "fail", 0.8),
                case("clean", "pass", 0.2),
                case("gone", "pass", 0.1),
            ]),
            report([
                case("bug", "fail", 0.6),
                case("clean", "pass", 0.2),
            ]),
        ])
        metrics = summary["packs"]["root"]
        self.assertEqual(metrics["inconclusive"], 1)
        self.assertEqual(metrics["failCases"], 1)
        self.assertEqual(metrics["passCases"], 2)
        self.assertEqual(metrics["auc"], 1)
        self.assertEqual(metrics["specificity"], 1)
        invalid = score_evals.score_reports([report([
            case("bug", "fail", None),
            case("clean", "pass", 0.2),
        ])])
        self.assertEqual(invalid["packs"]["root"]["inconclusive"], 1)
        self.assertIsNone(invalid["packs"]["root"]["auc"])

    def test_flip_rate_and_mean_abs_delta(self):
        summary = score_evals.score_reports([
            report([case("unstable", "fail", 0.25), case("stable", "pass", 0.25)]),
            report([case("unstable", "fail", 0.75), case("stable", "pass", 0.25)]),
        ], threshold=0.5)
        metrics = summary["packs"]["root"]
        self.assertEqual(metrics["flipRate"], 0.5)
        self.assertEqual(metrics["meanAbsDelta"], 0.25)
        self.assertEqual(metrics["recall"], 1)
        self.assertEqual(metrics["specificity"], 1)

    def test_unit_max_and_unnamed_cases(self):
        units_only = score_evals.score_reports([report([{
            "rule": "rule", "name": "bug", "file": "bug.rs", "expected": "fail",
            "units": [{"failProbability": 0.2}, {"failProbability": 0.7}],
        }, {
            "rule": "rule", "name": "clean", "file": "clean.rs", "expected": "pass",
            "units": [{"failProbability": 0.1}],
        }])])
        self.assertEqual(units_only["packs"]["root"]["auc"], 1)
        unnamed = score_evals.score_reports([
            {"cases": [{"rule": "database-joins", "file": "good.go", "expected": "pass", "failProbability": 0.2}]},
            {"cases": [{"rule": "database-joins", "file": "good.go", "expected": "pass", "failProbability": 0.4}]},
        ])
        self.assertEqual(unnamed["packs"]["root"]["passCases"], 1)
        self.assertEqual(unnamed["packs"]["root"]["meanAbsDelta"], 0.2)

    def test_pack_flag_writes_min_fail_probability(self):
        parser = runner.argument_parser()
        args = parser.parse_args([])
        self.assertEqual(args.min_fail_probability, 0.5)
        self.assertFalse(any(action.dest == "min_confidence" for action in parser._actions))
        self.assertEqual(runner.eval_config([{"id": "rule"}], 0.5)["minFailProbability"], 0.5)
        with self.assertRaises(SystemExit):
            with contextlib.redirect_stderr(io.StringIO()):
                parser.parse_args(["--min-confidence", "0.8"])

if __name__ == "__main__":
    unittest.main()
