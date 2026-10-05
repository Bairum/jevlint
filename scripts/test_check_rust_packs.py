"""Behavioral checks for repeated Rust fixture calibration (no model calls)."""

import importlib.util
from pathlib import Path
import unittest


spec = importlib.util.spec_from_file_location("check_rust_packs", Path(__file__).with_name("check-rust-packs.py"))
runner = importlib.util.module_from_spec(spec)
spec.loader.exec_module(runner)


def calibration(expected, outcomes):
    fixtures = [{"name": name, "rule": "rule", "file": f"{name}.rs", "expect": expect}
                for name, expect in expected.items()]
    packs = [("rust-core", [], None, fixtures)]
    runs = []
    for index, answers in enumerate(outcomes, 1):
        cases = [{"name": name, "actual": actual, "decisions": {"abstain": 0, "belowFloor": 0}}
                 for name, actual in answers.items()]
        runs.append({"pack": "rust-core", "run": index, "cases": cases, "errors": []})
    return packs, runs


class CalibrationTests(unittest.TestCase):
    def test_majority_flip_and_fixture_metrics(self):
        packs, runs = calibration(
            {"detected": "fail", "missed": "fail", "clean": "pass", "unstable": "pass"},
            [
                {"detected": "fail", "missed": "pass", "clean": "pass", "unstable": "fail"},
                {"detected": "fail", "missed": "pass", "clean": "pass", "unstable": "pass"},
                {"detected": "pass", "missed": "pass", "clean": "pass", "unstable": "pass"},
            ],
        )
        runs[0]["cases"][0]["decisions"] = {"abstain": 2, "belowFloor": 1}
        summary = runner.summarize(packs, runs, 3)
        self.assertEqual([case["majority"] for case in summary["cases"]], ["fail", "pass", "pass", "pass"])
        self.assertEqual(summary["flipRate"], 0.5)
        self.assertEqual(summary["flippedCases"], 2)
        self.assertEqual(summary["mismatches"], 5)
        self.assertEqual(summary["rules"]["rule"]["recall"], 0.5)
        self.assertEqual(summary["rules"]["rule"]["specificity"], 1)
        self.assertEqual((summary["abstain"], summary["belowFloor"]), (2, 1))

    def test_ties_and_missing_results_cannot_pass(self):
        packs, runs = calibration({"fixture": "fail"}, [{"fixture": "fail"}, {"fixture": "pass"}])
        summary = runner.summarize(packs, runs, 2)
        self.assertEqual(summary["cases"][0]["majority"], "inconclusive")
        self.assertEqual(summary["rules"]["rule"]["recall"], 0)
        self.assertIsNone(summary["rules"]["rule"]["specificity"])
        self.assertTrue(runner.calibration_failed(summary))
        packs, runs = calibration({"fixture": "fail"}, [{"fixture": "fail"}, {}, {}])
        runs[1]["errors"] = ["eval exited 2"]
        summary = runner.summarize(packs, runs, 3)
        self.assertEqual(summary["cases"][0]["runs"][1]["actual"], None)
        self.assertEqual(summary["inconclusive"], 2)
        self.assertEqual(summary["cases"][0]["majority"], "inconclusive")
        self.assertTrue(runner.calibration_failed(summary))

    def test_each_failure_condition_and_stable_success(self):
        packs, runs = calibration({"bug": "fail", "clean": "pass"},
                                  [{"bug": "fail", "clean": "pass"}] * 3)
        summary = runner.summarize(packs, runs, 3)
        self.assertFalse(runner.calibration_failed(summary))
        for key in ("mismatches", "inconclusive", "flippedCases", "errors"):
            with self.subTest(key=key):
                failing = dict(summary)
                failing[key] = ["error"] if key == "errors" else 1
                self.assertTrue(runner.calibration_failed(failing))


if __name__ == "__main__":
    unittest.main()
