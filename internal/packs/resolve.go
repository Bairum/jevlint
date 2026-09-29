package packs

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"jevlint/internal/config"
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
	checkout, err := cloneSpec(parsed)
	if err != nil {
		return config.PackRef{}, err
	}
	defer os.RemoveAll(filepath.Dir(checkout.dir))

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
	checkout, err := cloneSpec(parsed)
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
	dir string
	sha string
}

func cloneSpec(spec Spec) (checkout, error) {
	parent, err := os.MkdirTemp("", "jevlint-pack-")
	if err != nil {
		return checkout{}, fmt.Errorf("create pack checkout: %w", err)
	}
	dir := filepath.Join(parent, "repo")
	if err := runGit("", "clone", spec.Source, dir); err != nil {
		os.RemoveAll(parent)
		return checkout{}, err
	}
	if spec.Ref != "" {
		if err := runGit(dir, "checkout", spec.Ref); err != nil {
			os.RemoveAll(parent)
			return checkout{}, err
		}
	}
	sha, err := gitOutput(dir, "rev-parse", "HEAD")
	if err != nil {
		os.RemoveAll(parent)
		return checkout{}, err
	}
	return checkout{dir: dir, sha: sha}, nil
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
