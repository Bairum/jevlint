package packs

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// packIDPattern matches pack identifiers. A pack id becomes a filesystem path
// component under the user cache and a lookup key in the project config, so it
// is restricted to a conservative set of characters instead of accepting
// arbitrary strings.
var packIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// validatePackID rejects pack identifiers that are unsafe as path components.
func validatePackID(id string) error {
	if !packIDPattern.MatchString(id) {
		return fmt.Errorf("invalid pack id %q", id)
	}
	return nil
}

// validateRelativeName rejects a pack declared path that is empty, absolute, or
// climbs out of its directory.
func validateRelativeName(name string) error {
	if name == "" {
		return fmt.Errorf("path is empty")
	}
	if filepath.IsAbs(name) || filepath.VolumeName(name) != "" {
		return fmt.Errorf("path %q must be relative", name)
	}
	cleaned := filepath.Clean(name)
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return fmt.Errorf("path %q escapes its directory", name)
	}
	return nil
}

// safeJoin joins root and a relative path while guaranteeing that the result
// stays inside root. It rejects absolute paths and any "../" traversal.
func safeJoin(root string, relative string) (string, error) {
	if err := validateRelativeName(relative); err != nil {
		return "", err
	}
	root = filepath.Clean(root)
	joined := filepath.Join(root, filepath.Clean(relative))
	inside, err := filepath.Rel(root, joined)
	if err != nil {
		return "", fmt.Errorf("path %q is invalid: %w", relative, err)
	}
	if inside == ".." || strings.HasPrefix(inside, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q escapes its directory", relative)
	}
	return joined, nil
}

// rejectSymlinks fails when root or any entry below it is a symbolic link.
// Packs are plain files; following a link would let a pack read or copy files
// outside its checkout even though the link path itself is lexically contained.
// The ".git" directory is skipped because it is never copied into the cache.
func rejectSymlinks(root string) error {
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			relative, err := filepath.Rel(root, path)
			if err != nil {
				return fmt.Errorf("pack contains an unsafe symbolic link: %w", err)
			}
			return fmt.Errorf("pack contains a symbolic link %q", relative)
		}
		if path != root && entry.IsDir() && entry.Name() == ".git" {
			return filepath.SkipDir
		}
		return nil
	})
}
