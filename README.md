# LSSH - List SSH Servers

A TUI-based CLI tool for managing and connecting to SSH servers with a pluggable provider system. Navigate host lists with arrow keys, filter in real-time, and connect seamlessly.

## Features

- **Grid-based interface** with arrow key navigation
- **Real-time filtering** with `/` key (searches names and hostnames)
- **Multiple view modes**: All Hosts (flat) and By Group (hierarchical)
- **Bulk command execution** across multiple servers simultaneously
- **Pluggable providers**: JSON files, Ansible inventories, and extensible architecture
- **Automatic SSH connection** with user override support (press `u`)
- **Caching layer** for improved performance with remote providers with no extra effort from the user
- **Exclude patterns**: Hide groups/hosts using wildcard patterns with soft and hard exclusion modes

## Providers

- **JSON**: Simple JSON files with grouped host definitions
- **Ansible**: Read from Ansible inventory files and host/group variables

## Quick Start

Download the latest binary from [GitHub Releases](https://github.com/tech-arch1tect/lssh/releases) or build from source.

```bash
# Use with JSON provider (default)
export LSSH_HOSTS_FILE=example-provider-data/hosts.json
lssh

# Use with Ansible provider
export LSSH_PROVIDER_TYPE=ansible LSSH_HOSTS_FILE=example-provider-data/ansible.yml
lssh
```

## Build

```bash
go build -o lssh .
```

## Configuration

### Config File

Create a configuration file at `~/.config/lssh/config.json`:

```json
{
  "providers": [
    {
      "type": "json",
      "name": "default",
      "config": {
        "file": "hosts.json"
      }
    }
  ],
  "exclude_groups": ["Development", "test_*"],
  "hard_exclude_groups": ["staging"],
  "exclude_hosts": ["web-*", "backup-server", "*-temp"],
  "cache_enabled": true
}
```

### Environment Variables

Override configuration with environment variables:

- `LSSH_HOSTS_FILE`: Override hosts file location
- `LSSH_PROVIDER_TYPE`: Override provider type (json, ansible)
- `LSSH_EXCLUDE_GROUPS`: Comma-separated list of group patterns for soft exclusion
- `LSSH_HARD_EXCLUDE_GROUPS`: Comma-separated list of group patterns for hard exclusion
- `LSSH_EXCLUDE_HOSTS`: Comma-separated list of host patterns to exclude
- `LSSH_CACHE_ENABLED`: Enable/disable caching (true/false)
- `XDG_CONFIG_HOME`: Override config directory

```bash
# Example: Soft exclude groups (hide groups but keep duplicate hosts)
export LSSH_EXCLUDE_GROUPS="Development,test_*"
# Example: Hard exclude groups (remove groups and all duplicate hosts)
export LSSH_HARD_EXCLUDE_GROUPS="staging"
export LSSH_EXCLUDE_HOSTS="web-*,*-temp"
lssh
```

### Exclude Patterns

LSSH supports two types of group exclusions:

#### Soft Exclusion (`exclude_groups`)
- **Behavior**: Hides the specified groups from the interface
- **Duplicate hosts**: Hosts that appear in multiple groups remain accessible through non-excluded groups
- **Use case**: Declutter the group view while preserving access to all hosts

#### Hard Exclusion (`hard_exclude_groups`) 
- **Behavior**: Removes groups AND any hosts that appeared in those groups
- **Duplicate hosts**: Completely removes hosts from all views, even if they exist in non-excluded groups
- **Use case**: Completely remove unwanted hosts from the interface

#### Wildcard Pattern Support
- `*` matches any sequence of characters
- `web-*` matches `web-01`, `web-02`, etc.
- `*-temp` matches `server-temp`, `db-temp`, etc.
- `test_*` matches `test_group`, `test_env`, etc.
- Exact matches work without wildcards: `backup-server`

#### Examples
```bash
# Soft exclusion: Hide "Development" group but keep duplicate hosts
export LSSH_EXCLUDE_GROUPS="Development"

# Hard exclusion: Remove "Development" group and all its hosts everywhere
export LSSH_HARD_EXCLUDE_GROUPS="Development"

# Combined: Hide staging groups, completely remove test hosts
export LSSH_EXCLUDE_GROUPS="staging_*"
export LSSH_HARD_EXCLUDE_GROUPS="test_*"
```

Environment variables take precedence over config file settings.

## Navigation

### Basic Navigation
- `↑/↓/j/k` or arrow keys: Navigate hosts/groups
- `←/→/h/l`: Navigate grid columns
- `Enter`: Connect to host or enter group
- `Tab`: Switch between "All Hosts" and "By Group" views
- `/`: Filter hosts (type to search)
- `u`: Override username for connection
- `Backspace/h`: Go back to previous view
- `q/Ctrl+C`: Quit

### Bulk Commands
- `s`: Toggle bulk selection mode
- `Space`: Toggle host selection (shows checkboxes)
- `c`: Enter command to execute on selected hosts
- View real-time progress and results for all hosts
- Output is saved in ~/.lssh/logs/
