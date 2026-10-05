package changed

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFilesSkipsEscapingSymlinks(t *testing.T) {
	root := initRepo(t)
	outside := t.TempDir()
	writeFile(t, outside, "secret.go", "package secret\nfunc ExternalSecret() {}\n")
	writeFile(t, root, "inside.go", "package sample\nfunc Internal() {}\n")
	if err := os.Symlink(filepath.Join(outside, "secret.go"), filepath.Join(root, "external.go")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "linked")); err != nil {
		t.Fatal(err)
	}
	files, err := Files(root)
	if err != nil {
		t.Fatal(err)
	}
	assertFiles(t, files, "inside.go")
}

func TestFilesKeepsConfinedRegularSymlinkTargets(t *testing.T) {
	root := initRepo(t)
	writeFile(t, root, "inside.go", "package sample\nfunc Internal() {}\n")
	if err := os.Symlink("inside.go", filepath.Join(root, "internal.go")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("absent.go", filepath.Join(root, "missing.go")); err != nil {
		t.Fatal(err)
	}
	files, err := Files(root)
	if err != nil {
		t.Fatal(err)
	}
	assertFiles(t, files, "inside.go", "internal.go")
}

func TestRelativizeRejectsOutsideRoot(t *testing.T) {
	root := t.TempDir()
	for _, path := range []string{"../outside.go", filepath.Join(filepath.Dir(root), "outside.go")} {
		if _, err := Relativize(root, []string{path}); err == nil {
			t.Fatalf("outside-root request %q was not rejected", path)
		}
	}
}

func TestFilesSkipsDeletedPathsUnderEscapingDirectorySymlink(t *testing.T) {
	root := initRepo(t)
	writeFile(t, root, "nested/file.go", "package sample\n")
	writeFile(t, root, "inside.go", "package sample\n")
	git(t, root, "add", ".")
	git(t, root, "commit", "-m", "initial")

	outside := t.TempDir()
	writeFile(t, outside, "file.go", "package secret\nfunc ExternalSecret() {}\n")
	if err := os.Remove(filepath.Join(root, "nested", "file.go")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, "nested")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "nested")); err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, "inside.go", "package sample\nfunc Internal() {}\n")

	files, err := Files(root)
	if err != nil {
		t.Fatal(err)
	}
	assertFiles(t, files, "inside.go")
}

func TestFilesKeepsRecreatedStagedDeletion(t *testing.T) {
	root := initRepo(t)
	writeFile(t, root, "restored.go", "package sample\n")
	git(t, root, "add", ".")
	git(t, root, "commit", "-m", "initial")
	git(t, root, "rm", "restored.go")
	writeFile(t, root, "restored.go", "package sample\nfunc Restored() {}\n")

	files, err := Files(root)
	if err != nil {
		t.Fatal(err)
	}
	assertFiles(t, files, "restored.go")
}
