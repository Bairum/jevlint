package evaluation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileCacheRoundTripUsesPrivateAtomicStorage(t *testing.T) {
	t.Parallel()

	cache, err := newFileCacheAt(filepath.Join(t.TempDir(), "cache"))
	if err != nil {
		t.Fatalf("newFileCacheAt() error = %v", err)
	}
	hit := CacheHit{
		Model: "jev-test",
		Answers: map[string]QuestionAnswer{
			"rule": {
				Type:          answerTypeChoice,
				Choice:        "fail",
				Probabilities: map[string]float64{"fail": 0.9, "pass": 0.1},
			},
		},
	}
	if !cache.Put("key", hit) {
		t.Fatal("Put() = false")
	}

	rootInfo, err := os.Stat(cache.root)
	if err != nil {
		t.Fatalf("stat cache root: %v", err)
	}
	if permissions := rootInfo.Mode().Perm(); permissions != 0o700 {
		t.Fatalf("cache root permissions = %o, want 700", permissions)
	}
	entryInfo, err := os.Stat(filepath.Join(cache.root, "key"+cacheEntryExtension))
	if err != nil {
		t.Fatalf("stat cache entry: %v", err)
	}
	if permissions := entryInfo.Mode().Perm(); permissions != 0o600 {
		t.Fatalf("cache entry permissions = %o, want 600", permissions)
	}

	data, err := os.ReadFile(filepath.Join(cache.root, "key"+cacheEntryExtension))
	if err != nil {
		t.Fatalf("read cache entry: %v", err)
	}
	if strings.Contains(string(data), "source") {
		t.Fatalf("cache entry contains source data: %s", data)
	}
	temporary, err := filepath.Glob(filepath.Join(cache.root, ".write-*"))
	if err != nil {
		t.Fatalf("glob temporary entries: %v", err)
	}
	if len(temporary) != 0 {
		t.Fatalf("temporary cache entries = %v", temporary)
	}

	got, ok := cache.Get("key")
	answer := got.Answers["rule"]
	if !ok || got.Model != hit.Model || answer.Choice != "fail" || answer.Probabilities["fail"] != 0.9 {
		t.Fatalf("Get() = %#v, %v", got, ok)
	}
	got.Answers["rule"] = QuestionAnswer{
		Type:          answerTypeChoice,
		Choice:        "pass",
		Probabilities: map[string]float64{"pass": 1, "fail": 0},
	}
	again, ok := cache.Get("key")
	if !ok || again.Answers["rule"].Choice != "fail" {
		t.Fatalf("Get() returned shared answers: %#v, %v", again, ok)
	}
}

func TestFileCacheRejectsCorruptInvalidAndOldEntries(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"corrupt": `not json`,
		"old version": `{
			"version": 1,
			"model": "jev-test",
			"answers": {"rule": {"type": "choice", "probabilities": {"pass": 1, "fail": 0}}}
		}`,
		"invalid answer": `{
			"version": 2,
			"model": "jev-test",
			"answers": {"rule": {"type": "choice", "probabilities": {"maybe": 1}}}
		}`,
		"empty answers": `{
			"version": 2,
			"model": "jev-test",
			"answers": {}
		}`,
		"missing model": `{
			"version": 2,
			"answers": {"rule": {"type": "choice", "probabilities": {"pass": 1, "fail": 0}}}
		}`,
	}
	for name, contents := range tests {
		name, contents := name, contents
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cache, err := newFileCacheAt(filepath.Join(t.TempDir(), "cache"))
			if err != nil {
				t.Fatalf("newFileCacheAt() error = %v", err)
			}
			path := filepath.Join(cache.root, "key"+cacheEntryExtension)
			if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
				t.Fatalf("write cache entry: %v", err)
			}
			if hit, ok := cache.Get("key"); ok || hit.Answers != nil {
				t.Fatalf("Get() = %#v, %v; want miss", hit, ok)
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatalf("invalid cache entry still exists: %v", err)
			}
		})
	}
}

func TestFileCacheScopesAndClearsEntries(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	first, err := newFileCacheAt(filepath.Join(root, "first"))
	if err != nil {
		t.Fatalf("newFileCacheAt() error = %v", err)
	}
	second, err := newFileCacheAt(filepath.Join(root, "second"))
	if err != nil {
		t.Fatalf("newFileCacheAt() error = %v", err)
	}
	hit := CacheHit{
		Model: "jev-test",
		Answers: map[string]QuestionAnswer{
			"rule": {
				Type:          answerTypeChoice,
				Choice:        "pass",
				Probabilities: map[string]float64{"pass": 1, "fail": 0},
			},
		},
	}
	if !first.Put("same-key", hit) {
		t.Fatal("first Put() = false")
	}
	if _, ok := second.Get("same-key"); ok {
		t.Fatal("second project cache read first project entry")
	}
	if err := first.Clear(); err != nil {
		t.Fatalf("Clear() error = %v", err)
	}
	if _, ok := first.Get("same-key"); ok {
		t.Fatal("Get() hit after Clear()")
	}
	if _, err := os.Stat(first.root); err != nil {
		t.Fatalf("cache root missing after Clear(): %v", err)
	}
}

func TestNewFileCacheScopesByAbsoluteProjectRoot(t *testing.T) {
	cacheHome := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheHome)

	firstProject := t.TempDir()
	secondProject := t.TempDir()
	first, err := NewFileCache(firstProject, os.UserCacheDir)
	if err != nil {
		t.Fatalf("NewFileCache() error = %v", err)
	}
	same, err := NewFileCache(filepath.Join(firstProject, "."), os.UserCacheDir)
	if err != nil {
		t.Fatalf("NewFileCache() error = %v", err)
	}
	second, err := NewFileCache(secondProject, os.UserCacheDir)
	if err != nil {
		t.Fatalf("NewFileCache() error = %v", err)
	}
	if first.root != same.root {
		t.Fatalf("same project roots differ: %q and %q", first.root, same.root)
	}
	if first.root == second.root {
		t.Fatalf("different projects share cache root %q", first.root)
	}
	if !strings.HasPrefix(first.root, cacheHome) {
		t.Fatalf("cache root = %q, want it under %q", first.root, cacheHome)
	}
}
