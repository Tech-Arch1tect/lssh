# LSSH - List SSH Servers

A TUI-based CLI tool for managing and connecting to SSH servers with a pluggable provider system. Navigate host lists with arrow keys, filter in real-time, and connect seamlessly.

## Features

- **Grid-based interface** with arrow key navigation
- **Real-time filtering** with `/` key (searches names and hostnames)
- **Multiple view modes**: All Hosts (flat) and By Group (hierarchical)
- **Pluggable providers**: JSON files, Ansible inventories, and extensible architecture
- **Automatic SSH connection** with user override support (press `u`)
- **Caching layer** for improved performance with remote providers with no extra effort from the user

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

## Navigation

- `↑/↓/j/k` or arrow keys: Navigate hosts/groups
- `←/→/h/l`: Navigate grid columns
- `Enter`: Connect to host or enter group
- `Tab`: Switch between "All Hosts" and "By Group" views
- `/`: Filter hosts (type to search)
- `u`: Override username for connection
- `Backspace/h`: Go back to previous view
- `q/Ctrl+C`: Quit
