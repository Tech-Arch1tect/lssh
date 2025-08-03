package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tech-arch1tect/lssh/internal/cache"
	"github.com/tech-arch1tect/lssh/internal/config"
	"github.com/tech-arch1tect/lssh/internal/provider"
	"github.com/tech-arch1tect/lssh/internal/ssh"
	"github.com/tech-arch1tect/lssh/internal/tui"
)

func main() {
	clearCache := flag.Bool("clear-cache", false, "Clear all cached provider data")
	flag.Parse()

	if *clearCache {
		if err := cache.ClearCache(); err != nil {
			fmt.Fprintf(os.Stderr, "Error clearing cache: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Cache cleared successfully")
		return
	}

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	var providers []provider.Provider
	for _, providerConfig := range cfg.Providers {
		p, err := provider.NewProvider(providerConfig, cfg)
		if err != nil {
			return fmt.Errorf("failed to create provider %s: %w", providerConfig.Name, err)
		}
		providers = append(providers, p)
	}

	model := tui.NewModel(providers, cfg)
	p := tea.NewProgram(model, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}

	for {
		if m, ok := finalModel.(tui.Model); ok {
			if choice := m.Choice(); choice != nil {
				customUser := m.CustomUsername()
				if customUser != "" {
					fmt.Printf("Connecting to %s (%s) as %s...\n", choice.Name, choice.Hostname, customUser)
				} else {
					fmt.Printf("Connecting to %s (%s)...\n", choice.Name, choice.Hostname)
				}
				sshErr := ssh.ConnectWithUser(choice, customUser)

				if sshErr != nil {
					model = tui.NewModelWithError(providers, cfg, sshErr)
				} else {
					model = tui.NewModel(providers, cfg)
				}

				p = tea.NewProgram(model, tea.WithAltScreen())
				finalModel, err = p.Run()
				if err != nil {
					return fmt.Errorf("failed to run TUI: %w", err)
				}
				continue
			}
		}
		break
	}

	return nil
}
