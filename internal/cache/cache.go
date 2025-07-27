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

	cp.saveToCache(cacheFile, groups)
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
