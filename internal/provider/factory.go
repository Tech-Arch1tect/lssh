package provider

import (
	"fmt"

	"github.com/tech-arch1tect/lssh/internal/cache"
)

type CacheConfig interface {
	IsCacheEnabled() bool
}

func NewProvider(config Config, appConfig CacheConfig) (Provider, error) {
	var baseProvider Provider
	var filepath string

	switch config.Type {
	case "json":
		fp, ok := config.Config["file"].(string)
		if !ok {
			return nil, fmt.Errorf("json provider requires 'file' config parameter")
		}
		filepath = fp
		baseProvider = NewJSONProvider(config.Name, filepath)
	case "ansible":
		fp, ok := config.Config["file"].(string)
		if !ok {
			return nil, fmt.Errorf("ansible provider requires 'file' config parameter")
		}
		filepath = fp
		baseProvider = NewAnsibleProvider(config.Name, filepath)
	default:
		return nil, fmt.Errorf("unknown provider type: %s", config.Type)
	}

	if appConfig.IsCacheEnabled() {
		return cache.NewCachedProvider(baseProvider, config.Type, filepath), nil
	}

	return baseProvider, nil
}
