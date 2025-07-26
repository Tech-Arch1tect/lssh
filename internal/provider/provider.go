package provider

import (
	"context"

	"github.com/tech-arch1tect/lssh/pkg/types"
)

type Provider interface {
	Name() string
	GetHosts(ctx context.Context) ([]*types.Host, error)
}

type Config struct {
	Type   string                 `json:"type"`
	Name   string                 `json:"name"`
	Config map[string]interface{} `json:"config"`
}
