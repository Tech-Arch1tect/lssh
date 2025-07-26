package provider

import (
	"fmt"
)

func NewProvider(config Config) (Provider, error) {
	switch config.Type {
	case "json":
		filepath, ok := config.Config["file"].(string)
		if !ok {
			return nil, fmt.Errorf("json provider requires 'file' config parameter")
		}
		return NewJSONProvider(config.Name, filepath), nil
	case "ansible":
		filepath, ok := config.Config["file"].(string)
		if !ok {
			return nil, fmt.Errorf("ansible provider requires 'file' config parameter")
		}
		return NewAnsibleProvider(config.Name, filepath), nil
	default:
		return nil, fmt.Errorf("unknown provider type: %s", config.Type)
	}
}
