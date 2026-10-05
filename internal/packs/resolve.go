package packs

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/codegirl-007/jevlint/internal/config"
)

func Resolve(
	refs []config.PackRef,
	userCacheDir func() (string, error),
) ([]Loaded, error) {
	if len(refs) == 0 {
		return nil, nil
	}
	userCache, err := userCacheDir()
	if err != nil {
		return nil, fmt.Errorf("resolve user cache directory: %w", err)
	}
	loaded := make([]Loaded, 0, len(refs))
	for _, ref := range refs {
		item, err := resolveRef(ref, userCache)
		if err != nil {
			return nil, err
		}
		loaded = append(loaded, item)
	}
	return loaded, nil
}

func resolveRef(ref config.PackRef, userCache string) (Loaded, error) {
	dir, err := CacheDir(userCache, ref.SHA, ref.ID)
	if err != nil {
		return Loaded{}, err
	}
	if _, err := os.Stat(filepath.Join(dir, ManifestFile)); err != nil {
		if err := fetchRef(ref, dir); err != nil {
			return Loaded{}, err
		}
	}
	item, err := LoadDir(dir)
	if err != nil {
		return Loaded{}, err
	}
	if item.Manifest.ID != ref.ID {
		return Loaded{}, fmt.Errorf(
			"pack id %q does not match pin %q",
			item.Manifest.ID,
			ref.ID,
		)
	}
	item.Ref = ref
	return item, nil
}

func Install(
	spec string,
	userCacheDir func() (string, error),
) (config.PackRef, error) {
	parsed, err := ParseSpec(spec)
	if err != nil {
		return config.PackRef{}, err
	}
	userCache, err := userCacheDir()
	if err != nil {
		return config.PackRef{}, fmt.Errorf("resolve user cache directory: %w", err)
	}
	checkout, err := cloneSpec(parsed, "")
	if err != nil {
		return config.PackRef{}, err
	}
	defer os.RemoveAll(filepath.Dir(checkout.dir))

	parsed = checkout.spec
	packDir := checkout.dir
	if parsed.Path != "" {
		packDir, err = safeJoin(checkout.dir, parsed.Path)
		if err != nil {
			return config.PackRef{}, fmt.Errorf("pack path: %w", err)
		}
	}
	item, err := LoadDir(packDir)
	if err != nil {
		return config.PackRef{}, err
	}
	ref := config.PackRef{
		ID:     item.Manifest.ID,
		Source: parsed.Source,
		Path:   parsed.Path,
		Ref:    parsed.Ref,
		SHA:    checkout.sha,
	}
	dest, err := CacheDir(userCache, ref.SHA, ref.ID)
	if err != nil {
		return config.PackRef{}, err
	}
	if err := replaceDir(dest, packDir); err != nil {
		return config.PackRef{}, err
	}
	return ref, nil
}

func fetchRef(ref config.PackRef, dest string) error {
	parsed := Spec{Source: ref.Source, Ref: ref.SHA, Path: ref.Path}
	checkout, err := cloneSpec(parsed, ref.SHA)
	if err != nil {
		return fmt.Errorf("fetch pack %q: %w", ref.ID, err)
	}
	defer os.RemoveAll(filepath.Dir(checkout.dir))
	packDir := checkout.dir
	if ref.Path != "" {
		packDir, err = safeJoin(checkout.dir, ref.Path)
		if err != nil {
			return fmt.Errorf("fetch pack %q: %w", ref.ID, err)
		}
	}
	return replaceDir(dest, packDir)
}

type checkout struct {
	dir  string
	sha  string
	spec Spec
}

func cloneSpec(spec Spec, pin string) (checkout, error) {
	parent, err := os.MkdirTemp("", "jevlint-pack-")
	if err != nil {
		return checkout{}, fmt.Errorf("create pack checkout: %w", err)
	}
	dir := filepath.Join(parent, "repo")
	if err := runGit("", "clone", spec.Source, dir); err != nil {
		os.RemoveAll(parent)
		return checkout{}, err
	}
	if len(spec.treeSegments) > 0 {
		spec.Ref, spec.Path, err = resolveTreeRef(dir, spec.treeSegments)
		if err != nil {
			os.RemoveAll(parent)
			return checkout{}, err
		}
	}
	ref := "HEAD"
	if pin != "" {
		ref = pin
	} else if spec.Ref != "" {
		// Resolve the friendly ref ourselves: git checkout's own lookup
		// prefers a tag over a same-named remote branch, unlike resolveCommit.
		resolved, ok := resolveCommit(dir, spec.Ref)
		if !ok {
			os.RemoveAll(parent)
			return checkout{}, fmt.Errorf("pack ref %q: no branch, tag or commit with that name", spec.Ref)
		}
		ref = resolved
	}
	// Peel to an actual commit before checkout; a SHA-shaped branch is not a pin.
	commit, err := gitOutput(dir, "rev-parse", "--verify", "--end-of-options", ref+"^{commit}")
	if err != nil {
		os.RemoveAll(parent)
		return checkout{}, err
	}
	if pin != "" && !strings.EqualFold(commit, pin) {
		os.RemoveAll(parent)
		return checkout{}, fmt.Errorf("pack commit %s does not match pin %s", commit, pin)
	}
	if err := runGit(dir, "checkout", "--detach", commit); err != nil {
		os.RemoveAll(parent)
		return checkout{}, err
	}
	sha, err := gitOutput(dir, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		os.RemoveAll(parent)
		return checkout{}, err
	}
	if !strings.EqualFold(sha, commit) || (pin != "" && !strings.EqualFold(sha, pin)) {
		os.RemoveAll(parent)
		return checkout{}, fmt.Errorf("pack checkout %s does not match commit %s", sha, commit)
	}
	return checkout{dir: dir, sha: sha, spec: spec}, nil
}

// resolveTreeRef splits a GitHub tree URL's "<ref>/<path>" tail. Branch and
// tag names may contain slashes, so the longest leading run of segments that
// names a branch, tag or commit in the clone is the ref, as on GitHub.
func resolveTreeRef(dir string, segments []string) (string, string, error) {
	for end := len(segments); end > 0; end-- {
		ref := strings.Join(segments[:end], "/")
		if _, ok := resolveCommit(dir, ref); ok {
			return ref, strings.Join(segments[end:], "/"), nil
		}
	}
	return "", "", fmt.Errorf("github tree url: no branch, tag or commit named by %q", strings.Join(segments, "/"))
}

// resolveCommit returns the commit a ref names in a fresh clone, preferring a
// remote branch, then a tag, then any other revision such as a commit SHA.
func resolveCommit(dir string, ref string) (string, bool) {
	if strings.HasPrefix(ref, "-") {
		return "", false
	}
	for _, name := range []string{"refs/remotes/origin/" + ref, "refs/tags/" + ref, ref} {
		if commit, err := gitOutput(dir, "rev-parse", "--verify", "--quiet", "--end-of-options", name+"^{commit}"); err == nil {
			return commit, true
		}
	}
	return "", false
}

func runGit(dir string, args ...string) error {
	command := exec.Command("git", args...)
	if dir != "" {
		command.Dir = dir
	}
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git %s: %w\n%s", args[0], err, output)
	}
	return nil
}

func gitOutput(dir string, args ...string) (string, error) {
	command := exec.Command("git", args...)
	command.Dir = dir
	output, err := command.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", args[0], err)
	}
	return string(trimNewline(output)), nil
}

func trimNewline(value []byte) []byte {
	for len(value) > 0 && (value[len(value)-1] == '\n' || value[len(value)-1] == '\r') {
		value = value[:len(value)-1]
	}
	return value
}

func replaceDir(dest string, source string) error {
	if err := rejectSymlinks(source); err != nil {
		return err
	}
	if err := os.RemoveAll(dest); err != nil {
		return fmt.Errorf("clear pack cache: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o700); err != nil {
		return fmt.Errorf("create pack cache: %w", err)
	}
	return copyDir(source, dest)
}

func copyDir(source string, dest string) error {
	return filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if relative == ".git" || strings.HasPrefix(relative, ".git"+string(os.PathSeparator)) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("pack contains a symbolic link %q", relative)
		}
		target := filepath.Join(dest, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o700)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o600)
	})
}
