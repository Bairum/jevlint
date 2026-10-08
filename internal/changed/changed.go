package changed

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Files lists working-tree paths changed since HEAD. Deletes are skipped,
// renames keep the destination, and paths outside projectRoot are ignored.
func Files(projectRoot string) ([]string, error) {
	selection, err := Since(projectRoot, "HEAD")
	if err != nil {
		return nil, err
	}
	return selection.Paths(), nil
}

// Span is an inclusive line range in the working tree.
type Span struct {
	Start uint
	End   uint
	// Point is a pure deletion after Start. A unit spans it when the unit
	// covers both Start and the following line.
	Point bool
}

type fileChange struct {
	whole bool
	spans []Span
}

// Selection is the working-tree lines changed since Base.
type Selection struct {
	Base      string
	MergeBase string
	files     map[string]fileChange
}

// Since selects lines changed between merge-base(ref, HEAD) and the working
// tree, including staged, unstaged, and untracked non-ignored files. A whole
// untracked file counts as changed. No branch name is assumed.
func Since(projectRoot string, ref string) (Selection, error) {
	root, err := filepath.Abs(projectRoot)
	if err != nil {
		return Selection{}, fmt.Errorf("resolve project root: %w", err)
	}
	if _, err := exec.LookPath("git"); err != nil {
		return Selection{}, fmt.Errorf("git is required for --changed")
	}
	if _, err := gitOutput(root, "rev-parse", "--show-toplevel"); err != nil {
		return Selection{}, err
	}
	if ref == "HEAD" {
		if _, err := gitOutput(root, "rev-parse", "--verify", "--end-of-options", "HEAD^{commit}"); err != nil {
			if unknownRevision(err) {
				return untrackedSelection(root, ref, "")
			}
			return Selection{}, err
		}
	}
	if err := verifyRef(root, ref); err != nil {
		return Selection{}, err
	}
	mergeBase, err := mergeBase(root, ref)
	if err != nil {
		return Selection{}, err
	}
	return untrackedSelection(root, ref, mergeBase)
}

func untrackedSelection(root string, ref string, mergeBase string) (Selection, error) {
	untracked, err := gitOutput(root, "ls-files", "--others", "--exclude-standard", "-z", "--", ".")
	if err != nil {
		return Selection{}, err
	}
	selection := Selection{Base: ref, MergeBase: mergeBase, files: map[string]fileChange{}}
	if mergeBase != "" {
		diff, err := gitOutput(
			root,
			"diff", "-U0", "--no-color", "--no-ext-diff", "-M", "--relative",
			mergeBase, "--",
		)
		if err != nil {
			return Selection{}, err
		}
		selection.files = parseDiff(diff)
	}
	for _, path := range bytes.Split(untracked, []byte{0}) {
		if len(path) == 0 {
			continue
		}
		name := string(path)
		change := selection.files[name]
		change.whole = true
		selection.files[name] = change
	}
	return confine(root, selection)
}

// Paths returns the changed project-relative paths in order.
func (selection Selection) Paths() []string {
	paths := make([]string, 0, len(selection.files))
	for path := range selection.files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

// Empty reports whether nothing in the selection changed.
func (selection Selection) Empty() bool {
	return len(selection.files) == 0
}

// Restrict keeps changes under the requested paths. An empty request keeps all.
func (selection Selection) Restrict(requested []string) Selection {
	if len(requested) == 0 {
		return selection
	}
	next := Selection{
		Base:      selection.Base,
		MergeBase: selection.MergeBase,
		files:     make(map[string]fileChange),
	}
	for path, change := range selection.files {
		if matchesRequested(path, requested) {
			next.files[path] = change
		}
	}
	return next
}

// Overlaps reports whether a unit's inclusive line span should be evaluated.
func (selection Selection) Overlaps(path string, start, end uint) bool {
	change, ok := selection.files[path]
	if !ok {
		return false
	}
	if change.whole {
		return true
	}
	for _, span := range change.spans {
		if span.overlaps(start, end) {
			return true
		}
	}
	return false
}

func (span Span) overlaps(start, end uint) bool {
	if end < start {
		return false
	}
	if span.Point {
		if span.Start == 0 {
			return start <= 1 && end >= 1
		}
		return start <= span.Start && end >= span.Start+1
	}
	return start <= span.End && span.Start <= end
}

// existingProjectFiles keeps the changed files that still exist under the project root.
func existingProjectFiles(root string, gitRoot []byte, gitPaths []string) ([]string, error) {
	boundary, err := os.OpenRoot(root)
	if err != nil {
		return nil, fmt.Errorf("open project root: %w", err)
	}
	defer boundary.Close()
	seen := make(map[string]struct{})
	files := make([]string, 0)
	for _, gitPath := range gitPaths {
		absolute := filepath.Join(string(gitRoot), filepath.FromSlash(gitPath))
		relative, ok := underRoot(root, absolute)
		if !ok {
			continue
		}
		info, statErr := boundary.Lstat(relative)
		if statErr != nil {
			if os.IsNotExist(statErr) {
				continue
			}
			return nil, fmt.Errorf("inspect dirty path %q: %w", relative, statErr)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			info, statErr = boundary.Stat(relative)
			if statErr != nil {
				continue
			}
		}
		if !info.Mode().IsRegular() {
			continue
		}
		if _, exists := seen[relative]; exists {
			continue
		}
		seen[relative] = struct{}{}
		files = append(files, relative)
	}
	sort.Strings(files)
	return files, nil
}

// Relativize turns requested paths into project-relative paths, rejecting escapes.
func Relativize(projectRoot string, requested []string) ([]string, error) {
	root, err := filepath.Abs(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve project root: %w", err)
	}
	paths := make([]string, 0, len(requested))
	for _, requestedPath := range requested {
		absolute := requestedPath
		if !filepath.IsAbs(requestedPath) {
			absolute = filepath.Join(root, requestedPath)
		}
		relative, ok := underRoot(root, filepath.Clean(absolute))
		if !ok {
			return nil, fmt.Errorf("source path %q is outside project root", requestedPath)
		}
		paths = append(paths, relative)
	}
	return paths, nil
}

// Intersect keeps the files that fall under one of the requested paths.
func Intersect(files []string, requested []string) []string {
	if len(requested) == 0 {
		return files
	}
	matched := make([]string, 0)
	for _, file := range files {
		if matchesRequested(file, requested) {
			matched = append(matched, file)
		}
	}
	return matched
}

// matchesRequested reports whether a file is the requested path or sits under it.
func matchesRequested(file string, requested []string) bool {
	cleaned := filepath.Clean(file)
	for _, request := range requested {
		request = filepath.Clean(request)
		if request == "." {
			return true
		}
		if cleaned == request {
			return true
		}
		prefix := request + string(filepath.Separator)
		if strings.HasPrefix(cleaned, prefix) {
			return true
		}
	}
	return false
}

// underRoot returns a path relative to the root when it is inside the root.
func underRoot(root string, absolute string) (string, bool) {
	relative, err := filepath.Rel(root, absolute)
	if err != nil {
		return "", false
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", false
	}
	return relative, true
}

func verifyRef(dir string, ref string) error {
	_, err := gitOutput(dir, "rev-parse", "--verify", "--end-of-options", ref+"^{commit}")
	if err == nil {
		return nil
	}
	if unknownRevision(err) {
		return fmt.Errorf("unknown git ref %q; fetch it with git fetch and retry", ref)
	}
	return err
}

func unknownRevision(err error) bool {
	message := err.Error()
	return strings.Contains(message, "Needed a single revision") ||
		strings.Contains(message, "Not a valid object name") ||
		strings.Contains(message, "unknown revision") ||
		strings.Contains(message, "ambiguous argument")
}

func mergeBase(dir string, ref string) (string, error) {
	output, err := gitOutput(dir, "merge-base", "--end-of-options", ref, "HEAD")
	if err != nil {
		return "", fmt.Errorf("no merge base between %q and HEAD; the clone may be shallow, so fetch more history (actions/checkout fetch-depth: 0)", ref)
	}
	base := string(bytes.TrimSpace(output))
	if base == "" {
		return "", fmt.Errorf("no merge base between %q and HEAD; the clone may be shallow, so fetch more history (actions/checkout fetch-depth: 0)", ref)
	}
	return base, nil
}

func confine(root string, selection Selection) (Selection, error) {
	kept, err := existingProjectFiles(root, []byte(root), selection.Paths())
	if err != nil {
		return Selection{}, err
	}
	allowed := make(map[string]struct{}, len(kept))
	for _, path := range kept {
		allowed[path] = struct{}{}
	}
	next := Selection{
		Base:      selection.Base,
		MergeBase: selection.MergeBase,
		files:     make(map[string]fileChange, len(kept)),
	}
	for path, change := range selection.files {
		if _, ok := allowed[path]; ok {
			next.files[path] = change
		}
	}
	return next, nil
}

var hunkNew = regexp.MustCompile(`^@@ -\d+(?:,\d+)? \+(\d+)(?:,(\d+))? @@`)

func parseDiff(data []byte) map[string]fileChange {
	files := make(map[string]fileChange)
	var current string
	var skip, sawRename, sawHunk bool
	flush := func() {
		if current == "" || skip {
			return
		}
		if sawRename && !sawHunk {
			change := files[current]
			change.whole = true
			files[current] = change
		}
	}
	for _, line := range bytes.Split(data, []byte{'\n'}) {
		switch {
		case bytes.HasPrefix(line, []byte("diff --git ")):
			flush()
			current, skip, sawRename, sawHunk = "", false, false, false
		case bytes.HasPrefix(line, []byte("deleted file mode")):
			skip = true
		case bytes.HasPrefix(line, []byte("rename to ")):
			current = gitPath(line[len("rename to "):])
			sawRename = true
		case bytes.HasPrefix(line, []byte("+++ ")):
			path, ok := diffDest(line[len("+++ "):])
			if !ok {
				skip = true
				continue
			}
			current = path
		case bytes.HasPrefix(line, []byte("@@ ")):
			if skip || current == "" {
				continue
			}
			span, ok := parseHunk(line)
			if !ok {
				continue
			}
			sawHunk = true
			change := files[current]
			change.whole = false
			change.spans = append(change.spans, span)
			files[current] = change
		}
	}
	flush()
	return files
}

func parseHunk(line []byte) (Span, bool) {
	match := hunkNew.FindSubmatch(bytes.TrimRight(line, "\r"))
	if match == nil {
		return Span{}, false
	}
	start, err := strconv.ParseUint(string(match[1]), 10, 32)
	if err != nil {
		return Span{}, false
	}
	if len(match[2]) == 0 {
		return Span{Start: uint(start), End: uint(start)}, true
	}
	count, err := strconv.ParseUint(string(match[2]), 10, 32)
	if err != nil {
		return Span{}, false
	}
	if count == 0 {
		return Span{Start: uint(start), Point: true}, true
	}
	return Span{Start: uint(start), End: uint(start) + uint(count) - 1}, true
}

func diffDest(raw []byte) (string, bool) {
	line := string(bytes.TrimRight(raw, "\r"))
	if tab := strings.IndexByte(line, '\t'); tab >= 0 {
		line = line[:tab]
	}
	line = strings.TrimRight(line, " ")
	if line == "/dev/null" {
		return "", false
	}
	path := gitPathText(line)
	path = strings.TrimPrefix(path, "b/")
	if path == "" || path == "/dev/null" {
		return "", false
	}
	return path, true
}

func gitPath(raw []byte) string {
	return gitPathText(string(bytes.TrimRight(raw, "\r")))
}

func gitPathText(value string) string {
	value = strings.TrimRight(value, " \t")
	if strings.HasPrefix(value, "\"") {
		if unquoted, err := strconv.Unquote(value); err == nil {
			return unquoted
		}
	}
	return value
}

// gitOutput runs a git command in a directory and returns its output.
func gitOutput(dir string, args ...string) ([]byte, error) {
	command := exec.Command("git", args...)
	command.Dir = dir
	output, err := command.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			message := strings.TrimSpace(string(exitErr.Stderr))
			if message == "" {
				message = strings.TrimSpace(string(output))
			}
			if strings.Contains(message, "not a git repository") {
				return nil, fmt.Errorf("--changed requires a git repository")
			}
			return nil, fmt.Errorf("git %s: %s", args[0], message)
		}
		return nil, fmt.Errorf("git %s: %w", args[0], err)
	}
	if args[0] == "rev-parse" {
		return bytes.TrimSpace(output), nil
	}
	return output, nil
}
