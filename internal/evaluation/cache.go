package evaluation

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	cacheEntryVersion   = 2
	cacheEntryExtension = ".json"
)

// CacheHit is the parsed answers and answering model stored for one request.
type CacheHit struct {
	Model   string
	Answers map[string]QuestionAnswer
}

// ResultCache stores per-question answers under string keys.
type ResultCache interface {
	Get(string) (CacheHit, bool)
	Put(string, CacheHit) bool
	Clear() error
}

// FileCache stores results in files under one directory.
type FileCache struct {
	root string
}

// cacheEntry is the stored form of one set of answers.
type cacheEntry struct {
	Version   int                       `json:"version"`
	CreatedAt time.Time                 `json:"createdAt"`
	Model     string                    `json:"model"`
	Answers   map[string]QuestionAnswer `json:"answers"`
}

// NewFileCache builds a cache for a project under the user cache directory.
func NewFileCache(
	projectRoot string,
	userCacheDir func() (string, error),
) (*FileCache, error) {
	userCache, err := userCacheDir()
	if err != nil {
		return nil, fmt.Errorf("resolve user cache directory: %w", err)
	}
	absoluteRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve project cache scope: %w", err)
	}
	scope := sha256.Sum256([]byte(filepath.Clean(absoluteRoot)))
	return newFileCacheAt(filepath.Join(
		userCache,
		"jevlint",
		fmt.Sprintf("v%d", cacheEntryVersion),
		fmt.Sprintf("%x", scope),
	))
}

// newFileCacheAt builds a cache that stores files under a directory.
func newFileCacheAt(root string) (*FileCache, error) {
	cache := &FileCache{root: root}
	if err := cache.ensureRoot(); err != nil {
		return nil, err
	}
	return cache, nil
}

// Get reads the answers stored under a key.
func (cache *FileCache) Get(key string) (CacheHit, bool) {
	path := filepath.Join(cache.root, key+cacheEntryExtension)
	data, err := os.ReadFile(path)
	if err != nil {
		return CacheHit{}, false
	}
	entry, err := decodeCacheEntry(data)
	if err != nil {
		if removeErr := os.Remove(path); removeErr != nil && !os.IsNotExist(removeErr) {
			return CacheHit{}, false
		}
		return CacheHit{}, false
	}
	return CacheHit{Model: entry.Model, Answers: cloneAnswers(entry.Answers)}, true
}

// decodeCacheEntry reads and checks one stored entry.
func decodeCacheEntry(data []byte) (cacheEntry, error) {
	var entry cacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return cacheEntry{}, fmt.Errorf("decode cache entry: %w", err)
	}
	if entry.Version != cacheEntryVersion {
		return cacheEntry{}, fmt.Errorf("unsupported cache entry version %d", entry.Version)
	}
	if len(entry.Answers) == 0 || entry.Model == "" {
		return cacheEntry{}, fmt.Errorf("invalid cache entry results")
	}
	for _, answer := range entry.Answers {
		if err := answer.validate(); err != nil {
			return cacheEntry{}, fmt.Errorf("invalid cache entry results")
		}
	}
	return entry, nil
}

// Put writes the answers under a key.
func (cache *FileCache) Put(key string, hit CacheHit) bool {
	if len(hit.Answers) == 0 || hit.Model == "" {
		return false
	}
	for _, answer := range hit.Answers {
		if err := answer.validate(); err != nil {
			return false
		}
	}
	if err := cache.ensureRoot(); err != nil {
		return false
	}
	data, err := json.Marshal(cacheEntry{
		Version:   cacheEntryVersion,
		CreatedAt: time.Now().UTC(),
		Model:     hit.Model,
		Answers:   hit.Answers,
	})
	if err != nil {
		return false
	}
	return writeAtomicCacheFile(
		cache.root,
		filepath.Join(cache.root, key+cacheEntryExtension),
		data,
	) == nil
}

// writeAtomicCacheFile writes a file by writing a temporary file and renaming it.
func writeAtomicCacheFile(directory string, destination string, data []byte) error {
	temporary, err := os.CreateTemp(directory, ".write-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	succeeded := false
	defer func() {
		_ = temporary.Close()
		if !succeeded {
			_ = os.Remove(temporaryPath)
		}
	}()
	if err := temporary.Chmod(0o600); err != nil {
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		return err
	}
	if err := temporary.Sync(); err != nil {
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, destination); err != nil {
		return err
	}
	succeeded = true
	return nil
}

// Clear removes all stored results and recreates the directory.
func (cache *FileCache) Clear() error {
	if err := os.RemoveAll(cache.root); err != nil {
		return fmt.Errorf("clear evaluation cache: %w", err)
	}
	return cache.ensureRoot()
}

// ensureRoot creates the cache directory with private permissions.
func (cache *FileCache) ensureRoot() error {
	if err := os.MkdirAll(cache.root, 0o700); err != nil {
		return fmt.Errorf("create evaluation cache: %w", err)
	}
	if err := os.Chmod(cache.root, 0o700); err != nil {
		return fmt.Errorf("secure evaluation cache: %w", err)
	}
	return nil
}
