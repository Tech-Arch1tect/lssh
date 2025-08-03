package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/tech-arch1tect/lssh/pkg/types"
)

type JSONProvider struct {
	name     string
	filepath string
}

func NewJSONProvider(name, filepath string) *JSONProvider {
	return &JSONProvider{
		name:     name,
		filepath: filepath,
	}
}

func (p *JSONProvider) Name() string {
	return p.name
}

func (p *JSONProvider) GetGroups(ctx context.Context) ([]*types.Group, error) {
	data, err := os.ReadFile(p.filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read JSON file %s: %w", p.filepath, err)
	}

	var groups []*types.Group
	if err := json.Unmarshal(data, &groups); err != nil {
		return nil, fmt.Errorf("failed to parse JSON file %s: %w", p.filepath, err)
	}

	totalHosts := 0
	for _, group := range groups {
		totalHosts += len(group.AllHosts())
	}

	if totalHosts == 0 {
		return nil, fmt.Errorf("no hosts found in JSON file %s", p.filepath)
	}

	return groups, nil
}
