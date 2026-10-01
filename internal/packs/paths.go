package packs

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// packIDPartPattern matches one component of a pack id. A pack id is
// "owner/name" and each component becomes a path segment under the user cache,
// so components are restricted to a conservative set of characters instead of
// accepting arbitrary strings.
var packIDPartPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// SplitPackID splits and validates an "owner/name" pack identifier.
func SplitPackID(id string) (string, string, error) {
	owner, name, ok := strings.Cut(id, "/")
	if !ok || !packIDPartPattern.MatchString(owner) || !packIDPartPattern.MatchString(name) {
		return "", "", fmt.Errorf("invalid pack id %q; want owner/name", id)
	}
	return owner, name, nil
}

// validatePackID rejects pack identifiers that are not a safe owner/name pair.
func validatePackID(id string) error {
	_, _, err := SplitPackID(id)
	return err
}

// validateIDPart rejects a single path component used in the pack cache.
func validateIDPart(kind string, value string) error {
	if !packIDPartPattern.MatchString(value) {
		return fmt.Errorf("invalid pack %s %q", kind, value)
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
