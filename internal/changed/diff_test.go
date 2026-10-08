package changed

import (
	"os/exec"
	"strings"
	"testing"
)

func TestSinceSelectsOnlyOverlappingLines(t *testing.T) {
	root := initTrunk(t)
	writeFile(t, root, "sample.go", alphaBetaGamma(false))
	git(t, root, "add", ".")
	git(t, root, "commit", "-m", "base")
	base := revParse(t, root, "HEAD")

	writeFile(t, root, "sample.go", alphaBetaGamma(true))
	git(t, root, "add", ".")
	git(t, root, "commit", "-m", "edit beta")

	selection, err := Since(root, base)
	if err != nil {
		t.Fatal(err)
	}
	if selection.Base != base || selection.MergeBase != base {
		t.Fatalf("selection = %+v", selection)
	}
	if !selection.Overlaps("sample.go", 7, 10) {
		t.Fatal("edited Beta was not selected")
	}
	if selection.Overlaps("sample.go", 3, 5) || selection.Overlaps("sample.go", 12, 14) {
		t.Fatal("untouched units were selected")
	}
	assertFiles(t, selection.Paths(), "sample.go")
}

func TestSincePureDeletionSpansThePoint(t *testing.T) {
	root := initTrunk(t)
	writeFile(t, root, "sample.go", ""+
		"package sample\n\n"+
		"func Alpha() {\n\tprintln(\"a\")\n}\n\n"+
		"func Beta() {\n\tprintln(\"b\")\n\tprintln(\"drop\")\n\tprintln(\"c\")\n}\n\n"+
		"func Gamma() {\n\tprintln(\"g\")\n}\n")
	git(t, root, "add", ".")
	git(t, root, "commit", "-m", "base")
	base := revParse(t, root, "HEAD")
	writeFile(t, root, "sample.go", ""+
		"package sample\n\n"+
		"func Alpha() {\n\tprintln(\"a\")\n}\n\n"+
		"func Beta() {\n\tprintln(\"b\")\n\tprintln(\"c\")\n}\n\n"+
		"func Gamma() {\n\tprintln(\"g\")\n}\n")

	selection, err := Since(root, base)
	if err != nil {
		t.Fatal(err)
	}
	if !selection.Overlaps("sample.go", 7, 10) {
		t.Fatal("Beta does not span the deletion point")
	}
	if selection.Overlaps("sample.go", 3, 5) || selection.Overlaps("sample.go", 12, 14) {
		t.Fatal("pure deletion selected a neighboring unit")
	}
}

func TestSinceRenameKeepsDestination(t *testing.T) {
	root := initTrunk(t)
	writeFile(t, root, "sample.go", alphaBetaGamma(false))
	git(t, root, "add", ".")
	git(t, root, "commit", "-m", "base")
	base := revParse(t, root, "HEAD")
	git(t, root, "mv", "sample.go", "moved.go")
	writeFile(t, root, "moved.go", alphaBetaGamma(true))

	selection, err := Since(root, base)
	if err != nil {
		t.Fatal(err)
	}
	if selection.Overlaps("sample.go", 7, 10) || !selection.Overlaps("moved.go", 7, 10) {
		t.Fatalf("paths = %#v", selection.Paths())
	}
	if selection.Overlaps("moved.go", 3, 5) {
		t.Fatal("rename selected an untouched unit")
	}
}

func TestSinceUntrackedFileIsWhole(t *testing.T) {
	root := initTrunk(t)
	writeFile(t, root, "tracked.go", "package sample\n\nfunc Kept() {}\n")
	git(t, root, "add", ".")
	git(t, root, "commit", "-m", "base")
	writeFile(t, root, "new.go", "package sample\n\nfunc Fresh() {}\n")

	selection, err := Since(root, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if !selection.Overlaps("new.go", 1, 1) || !selection.Overlaps("new.go", 3, 3) {
		t.Fatal("untracked file was not wholly selected")
	}
	if selection.Overlaps("tracked.go", 3, 3) {
		t.Fatal("clean file was selected")
	}
}

func TestSinceCountsStagedAndUnstaged(t *testing.T) {
	root := initTrunk(t)
	writeFile(t, root, "sample.go", alphaBetaGamma(false))
	writeFile(t, root, "other.go", "package sample\n\nfunc Other() {}\n")
	git(t, root, "add", ".")
	git(t, root, "commit", "-m", "base")
	writeFile(t, root, "sample.go", alphaBetaGamma(true))
	git(t, root, "add", "sample.go")
	writeFile(t, root, "other.go", "package sample\n\nfunc Other() {\n\tprintln(\"x\")\n}\n")

	selection, err := Since(root, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if !selection.Overlaps("sample.go", 7, 10) || !selection.Overlaps("other.go", 3, 5) {
		t.Fatalf("paths = %#v", selection.Paths())
	}
	if selection.Overlaps("sample.go", 3, 5) {
		t.Fatal("staged edit selected an untouched unit")
	}
}

func TestSinceUnknownRefAndMissingMergeBase(t *testing.T) {
	root := initTrunk(t)
	git(t, root, "commit", "-m", "base", "--allow-empty")

	_, err := Since(root, "no-such-ref")
	if err == nil || !strings.Contains(err.Error(), "git fetch") {
		t.Fatalf("unknown ref error = %v", err)
	}
	git(t, root, "checkout", "--orphan", "other")
	git(t, root, "commit", "-m", "unrelated", "--allow-empty")
	_, err = Since(root, "trunk")
	if err == nil || !strings.Contains(err.Error(), "fetch-depth: 0") {
		t.Fatalf("missing merge base error = %v", err)
	}
}

func initTrunk(t *testing.T) string {
	t.Helper()
	requireGit(t)
	root := t.TempDir()
	git(t, root, "init", "--initial-branch=trunk")
	git(t, root, "config", "user.email", "jevlint@example.com")
	git(t, root, "config", "user.name", "jevlint")
	git(t, root, "config", "commit.gpgsign", "false")
	return root
}

func revParse(t *testing.T, dir string, ref string) string {
	t.Helper()
	command := exec.Command("git", "rev-parse", ref)
	command.Dir = dir
	output, err := command.Output()
	if err != nil {
		t.Fatalf("rev-parse %s: %v", ref, err)
	}
	return strings.TrimSpace(string(output))
}

func alphaBetaGamma(edited bool) string {
	beta := "\tprintln(\"b\")\n"
	if edited {
		beta += "\tprintln(\"changed\")\n"
	}
	return "package sample\n\n" +
		"func Alpha() {\n\tprintln(\"a\")\n}\n\n" +
		"func Beta() {\n" + beta + "}\n\n" +
		"func Gamma() {\n\tprintln(\"g\")\n}\n"
}

func TestParseInsertionHunk(t *testing.T) {
	files := parseDiff([]byte("diff --git a/src/cmd/ping.rs b/src/cmd/ping.rs\n+++ b/src/cmd/ping.rs\n@@ -61,0 +62 @@ impl Ping {\n+        let _smoke = 1;\n"))
	change, ok := files["src/cmd/ping.rs"]
	if !ok || change.whole || len(change.spans) != 1 || !change.spans[0].overlaps(55, 68) {
		t.Fatalf("parsed = %#v", files)
	}
}
