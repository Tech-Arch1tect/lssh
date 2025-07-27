package provider

import (
	"context"

	"github.com/tech-arch1tect/lssh/pkg/types"
)

type Provider interface {
	Name() string
	GetGroups(ctx context.Context) ([]*types.Group, error)
}
