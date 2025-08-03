package cache

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/tech-arch1tect/lssh/pkg/provider"
	"github.com/tech-arch1tect/lssh/pkg/types"
)

type CachedProvider struct {
	provider     provider.Provider
	providerType string
	filePath     string
	cacheDir     string
	ttl          time.Duration
}

type cacheEntry struct {
	Groups    []*types.Group `json:"groups"`
	Timestamp time.Time      `json:"timestamp"`
}

func NewCachedProvider(p provider.Provider, providerType, filePath string) *CachedProvider {
	cacheDir := getCacheDir()
	ttl := getCacheTTL()

	return &CachedProvider{
		provider:     p,
		providerType: providerType,
		filePath:     filePath,
		cacheDir:     cacheDir,
		ttl:          ttl,
	}
}

func (cp *CachedProvider) Name() string {
	return cp.provider.Name()
}

func (cp *CachedProvider) GetGroups(ctx context.Context) ([]*types.Group, error) {
	cacheKey := cp.getCacheKey()
	cacheFile := filepath.Join(cp.cacheDir, cacheKey+".json")

	if entry, err := cp.loadFromCache(cacheFile); err == nil {
		if time.Since(entry.Timestamp) < cp.ttl {
			return entry.Groups, nil
		}
	}

	groups, err := cp.provider.GetGroups(ctx)
	if err != nil {
		return nil, err
	}

	totalHosts := 0
	for _, group := range groups {
		totalHosts += len(group.AllHosts())
	}

	if totalHosts > 0 {
		cp.saveToCache(cacheFile, groups)
	}

	return groups, nil
}

func (cp *CachedProvider) getCacheKey() string {
	keyData := fmt.Sprintf("%s:%s:%s", cp.providerType, cp.filePath, cp.provider.Name())
	h := sha256.New()
	h.Write([]byte(keyData))
	return fmt.Sprintf("lssh_%x", h.Sum(nil)[:8])
}

func (cp *CachedProvider) loadFromCache(cacheFile string) (*cacheEntry, error) {
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil, err
	}

	var entry cacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, err
	}

	return &entry, nil
}

func (cp *CachedProvider) saveToCache(cacheFile string, groups []*types.Group) {
	if err := os.MkdirAll(cp.cacheDir, 0755); err != nil {
		return
	}

	entry := cacheEntry{
		Groups:    groups,
		Timestamp: time.Now(),
	}

	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return
	}

	os.WriteFile(cacheFile, data, 0644)
}

func getCacheDir() string {
	if cacheDir := os.Getenv("LSSH_CACHE_DIR"); cacheDir != "" {
		return cacheDir
	}

	if xdgCache := os.Getenv("XDG_CACHE_HOME"); xdgCache != "" {
		return filepath.Join(xdgCache, "lssh")
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	return filepath.Join(homeDir, ".cache", "lssh")
}

func getCacheTTL() time.Duration {
	ttlStr := os.Getenv("LSSH_CACHE_TTL")
	if ttlStr == "" {
		return 24 * time.Hour
	}

	if hours, err := strconv.Atoi(ttlStr); err == nil {
		return time.Duration(hours) * time.Hour
	}

	if duration, err := time.ParseDuration(ttlStr); err == nil {
		return duration
	}

	return 24 * time.Hour
}

func ClearCache() error {
	cacheDir := getCacheDir()

	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read cache directory: %w", err)
	}

	var deleteErrors []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			filePath := filepath.Join(cacheDir, entry.Name())
			if err := os.Remove(filePath); err != nil {
				deleteErrors = append(deleteErrors, fmt.Sprintf("failed to delete %s: %v", entry.Name(), err))
			}
		}
	}

	if len(deleteErrors) > 0 {
		return fmt.Errorf("some cache files could not be deleted:\n%s", filepath.Join(deleteErrors...))
	}

	return nil
}
