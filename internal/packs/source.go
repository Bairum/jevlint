package packs

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type Spec struct {
	Source string
	Ref    string
	Path   string
	// treeSegments is a GitHub tree URL's "<ref>/<path>" tail. Ref names may
	// contain slashes, so Ref and Path are a first-segment guess until the
	// clone resolves them.
	treeSegments []string
}

func ParseSpec(raw string) (Spec, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return Spec{}, fmt.Errorf("pack source is required")
	}
	if parsed, err := parseGitHubTreeURL(value); err == nil {
		return parsed, nil
	}
	source, ref, path := splitSourceRefPath(value)
	if isLocalSource(source) {
		absolute, err := filepath.Abs(strings.TrimPrefix(source, "file://"))
		if err != nil {
			return Spec{}, fmt.Errorf("resolve pack source: %w", err)
		}
		if _, err := os.Stat(absolute); err != nil {
			return Spec{}, fmt.Errorf("pack source %q: %w", raw, err)
		}
		return Spec{Source: absolute, Ref: ref, Path: path}, nil
	}
	if !strings.Contains(source, "://") && strings.Count(source, "/") == 1 {
		source = "https://github.com/" + source + ".git"
	}
	return Spec{Source: source, Ref: ref, Path: path}, nil
}

func parseGitHubTreeURL(raw string) (Spec, error) {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host != "github.com" {
		return Spec{}, fmt.Errorf("not a github tree url")
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) < 4 || parts[2] != "tree" {
		return Spec{}, fmt.Errorf("not a github tree url")
	}
	source := "https://github.com/" + parts[0] + "/" + parts[1] + ".git"
	ref := parts[3]
	path := strings.Join(parts[4:], "/")
	return Spec{Source: source, Ref: ref, Path: path, treeSegments: parts[3:]}, nil
}

func splitSourceRefPath(raw string) (string, string, string) {
	source := raw
	var path string
	if hash := strings.Index(source, "#"); hash >= 0 {
		path = strings.TrimPrefix(source[hash+1:], "/")
		source = source[:hash]
	}
	ref := ""
	if at := strings.LastIndex(source, "@"); at > 0 {
		ref = source[at+1:]
		source = source[:at]
	}
	return source, ref, path
}

func isLocalSource(source string) bool {
	if strings.HasPrefix(source, "file://") {
		return true
	}
	if strings.Contains(source, "://") {
		return false
	}
	return strings.HasPrefix(source, "/") ||
		strings.HasPrefix(source, ".") ||
		filepath.IsAbs(source)
}
