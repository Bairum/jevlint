package packs

import (
	"os/exec"
	"testing"
)

func TestInstallNonDefaultBranch(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is required for pack ref tests")
	}
	repo := writePackRepo(t, `{"version":1,"id":"owner/pack"}`, validRulesJSON)
	packsRunGit(t, repo, "branch", "feature")
	pin, err := gitOutput(repo, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	cache := t.TempDir()
	installed, err := Install(repo+"@feature", func() (string, error) { return cache, nil })
	if err != nil || installed.SHA != pin || installed.Ref != "feature" {
		t.Fatalf("Install() = %#v, %v; want feature at %s", installed, err, pin)
	}
}
