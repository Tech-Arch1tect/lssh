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

func (p *JSONProvider) GetHosts(ctx context.Context) ([]*types.Host, error) {
	data, err := os.ReadFile(p.filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read JSON file %s: %w", p.filepath, err)
	}

	var hosts []*types.Host
	if err := json.Unmarshal(data, &hosts); err != nil {
		return nil, fmt.Errorf("failed to parse JSON file %s: %w", p.filepath, err)
	}

	return hosts, nil
}
