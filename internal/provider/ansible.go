package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"os/user"

	"github.com/tech-arch1tect/lssh/pkg/types"
)

type AnsibleProvider struct {
	name     string
	filepath string
}

func NewAnsibleProvider(name, filepath string) *AnsibleProvider {
	return &AnsibleProvider{
		name:     name,
		filepath: filepath,
	}
}

func (p *AnsibleProvider) Name() string {
	return p.name
}

func (p *AnsibleProvider) GetGroups(ctx context.Context) ([]*types.Group, error) {
	cmd := exec.CommandContext(ctx, "ansible-inventory", "-i", p.filepath, "--list")
	cmd.Env = nil
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("ansible-inventory command failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("failed to run ansible-inventory: %w", err)
	}

	var inventory map[string]interface{}
	if err := json.Unmarshal(output, &inventory); err != nil {
		return nil, fmt.Errorf("failed to parse ansible-inventory output: %w", err)
	}

	currentUser, _ := user.Current()
	defaultUser := currentUser.Username

	var hostVars map[string]map[string]interface{}
	if metaInterface, exists := inventory["_meta"]; exists {
		if metaMap, ok := metaInterface.(map[string]interface{}); ok {
			if hostVarsInterface, exists := metaMap["hostvars"]; exists {
				if hvMap, ok := hostVarsInterface.(map[string]interface{}); ok {
					hostVars = make(map[string]map[string]interface{})
					for hostname, vars := range hvMap {
						if varMap, ok := vars.(map[string]interface{}); ok {
							hostVars[hostname] = varMap
						}
					}
				}
			}
		}
	}

	var groups []*types.Group
	for groupName, groupData := range inventory {
		if groupName == "_meta" || groupName == "all" || groupName == "ungrouped" {
			continue
		}

		groupMap, ok := groupData.(map[string]interface{})
		if !ok {
			continue
		}

		group := &types.Group{
			Name:  groupName,
			Hosts: []*types.Host{},
		}

		if hostsInterface, exists := groupMap["hosts"]; exists {
			if hostsList, ok := hostsInterface.([]interface{}); ok {
				for _, hostInterface := range hostsList {
					if hostname, ok := hostInterface.(string); ok {
						host := &types.Host{
							Name:     hostname,
							Hostname: hostname,
							User:     defaultUser,
						}

						if hostVars != nil {
							if vars, exists := hostVars[hostname]; exists {
								if ansibleHost, ok := vars["ansible_host"].(string); ok && ansibleHost != "" {
									host.Hostname = ansibleHost
								}
								if ansibleUser, ok := vars["ansible_user"].(string); ok && ansibleUser != "" {
									host.User = ansibleUser
								}
								if ansiblePort, ok := vars["ansible_port"]; ok {
									if portFloat, ok := ansiblePort.(float64); ok {
										host.Port = int(portFloat)
									}
								}
							}
						}

						group.Hosts = append(group.Hosts, host)
					}
				}
			}
		}

		if len(group.Hosts) > 0 {
			groups = append(groups, group)
		}
	}

	return groups, nil
}
