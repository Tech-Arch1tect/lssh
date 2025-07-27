package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tech-arch1tect/lssh/internal/config"
	"github.com/tech-arch1tect/lssh/internal/provider"
	"github.com/tech-arch1tect/lssh/internal/ssh"
	"github.com/tech-arch1tect/lssh/internal/tui"
)

func main() {
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

	model := tui.NewModel(providers)
	p := tea.NewProgram(model, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}

	for {
		if m, ok := finalModel.(tui.Model); ok {
			if choice := m.Choice(); choice != nil {
				fmt.Printf("Connecting to %s (%s)...\n", choice.Name, choice.Hostname)
				sshErr := ssh.Connect(choice)
				
				if sshErr != nil {
					model = tui.NewModelWithError(providers, sshErr)
				} else {
					model = tui.NewModel(providers)
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
