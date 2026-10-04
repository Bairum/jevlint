package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// A worktree batch includes its files and the roots of nested worktrees.
type gitIgnoreWorktree struct {
	root       string
	parent     *gitIgnoreWorktree
	parentPath string
	paths      map[string][]string
	ignored    map[string]bool
	checked    bool
	blocked    bool
}

// removeGitIgnored applies Git's ignore rules only to directory-discovered files.
func removeGitIgnored(ctx context.Context, root string, discovered, explicit map[string]struct{}) error {
	for file := range explicit {
		delete(discovered, file)
	}
	if len(discovered) == 0 {
		return nil
	}
	physicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return fmt.Errorf("locate project root for Git ignores: %w", err)
	}
	directories := make(map[string]string)
	owners := make(map[string]*gitIgnoreWorktree)
	batches := make(map[*gitIgnoreWorktree]struct{})
	for file := range discovered {
		if err := ctx.Err(); err != nil {
			return err
		}
		directory := filepath.Dir(file)
		physicalDirectory, ok := directories[directory]
		if !ok {
			physicalDirectory, err = filepath.EvalSymlinks(directory)
			if err != nil {
				return fmt.Errorf("locate source directory for Git ignores: %w", err)
			}
			directories[directory] = physicalDirectory
		}
		// Resolve directory aliases, not terminal file symlinks: Git can check
		// a symlink itself, but refuses paths beyond a directory symlink.
		physicalFile := filepath.Join(physicalDirectory, filepath.Base(file))
		if _, err := relativeProjectPath(physicalRoot, physicalFile); err != nil {
			return err
		}
		owner, err := owningGitWorktree(physicalDirectory, owners)
		if err != nil {
			return err
		}
		if owner == nil {
			continue
		}
		relative, err := relativeProjectPath(owner.root, physicalFile)
		if err != nil {
			return err
		}
		owner.paths[relative] = append(owner.paths[relative], file)
		batches[owner] = struct{}{}
	}
	for batch := range batches {
		if err := checkWorktreeIgnores(ctx, batch); err != nil {
			return err
		}
		for path, files := range batch.paths {
			if batch.blocked || batch.ignored[path] {
				for _, file := range files {
					delete(discovered, file)
				}
			}
		}
	}
	return nil
}

// owningGitWorktree caches directory ancestry and recognizes both .git files
// and directories. Metadata failures must not become a non-Git fallback.
func owningGitWorktree(directory string, owners map[string]*gitIgnoreWorktree) (*gitIgnoreWorktree, error) {
	if owner, ok := owners[directory]; ok {
		return owner, nil
	}
	_, err := os.Lstat(filepath.Join(directory, ".git"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("inspect Git metadata: %w", err)
	}
	metadata := err == nil
	var parent *gitIgnoreWorktree
	if ancestor := filepath.Dir(directory); ancestor != directory {
		parent, err = owningGitWorktree(ancestor, owners)
		if err != nil {
			return nil, err
		}
	}
	if !metadata {
		owners[directory] = parent
		return parent, nil
	}
	owner := &gitIgnoreWorktree{
		root: directory, parent: parent,
		paths: make(map[string][]string), ignored: make(map[string]bool),
	}
	if parent != nil {
		relative, err := relativeProjectPath(parent.root, directory)
		if err != nil {
			return nil, err
		}
		owner.parentPath = relative + "/"
		parent.paths[owner.parentPath] = nil
	}
	owners[directory] = owner
	return owner, nil
}

// checkWorktreeIgnores checks parents first: an ignored nested repository must
// not bypass ancestor ignores merely because it has its own Git metadata.
func checkWorktreeIgnores(ctx context.Context, batch *gitIgnoreWorktree) error {
	if batch.checked {
		return nil
	}
	if batch.parent != nil {
		if err := checkWorktreeIgnores(ctx, batch.parent); err != nil {
			return err
		}
		if batch.parent.blocked || batch.parent.ignored[batch.parentPath] {
			batch.blocked, batch.checked = true, true
			return nil
		}
	}
	var input bytes.Buffer
	for path := range batch.paths {
		input.WriteString(path)
		input.WriteByte(0)
	}
	// Do not use --no-index: tracked files and submodule roots remain allowed.
	cmd := exec.CommandContext(ctx, "git", "check-ignore", "-z", "--stdin")
	cmd.Dir = batch.root
	cmd.Stdin = &input
	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		var exit *exec.ExitError
		if !errors.As(err, &exit) || exit.ExitCode() != 1 {
			return fmt.Errorf("cannot apply Git ignores in %q: %w", batch.root, err)
		}
	}
	for _, relative := range bytes.Split(output, []byte{0}) {
		if len(relative) != 0 {
			batch.ignored[string(relative)] = true
		}
	}
	batch.checked = true
	return nil
}
