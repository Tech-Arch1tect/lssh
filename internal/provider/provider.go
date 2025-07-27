package provider

import (
	pkgprovider "github.com/tech-arch1tect/lssh/pkg/provider"
)

type Provider = pkgprovider.Provider

type Config struct {
	Type   string                 `json:"type"`
	Name   string                 `json:"name"`
	Config map[string]interface{} `json:"config"`
}
