package ssh

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"time"

	"github.com/tech-arch1tect/lssh/pkg/types"
)

func Connect(host *types.Host) error {
	return ConnectWithUser(host, "")
}

func ConnectWithUser(host *types.Host, customUser string) error {
	args := []string{}

	if host.Port > 0 && host.Port != 22 {
		args = append(args, "-p", fmt.Sprintf("%d", host.Port))
	}

	username := customUser
	if username == "" {
		username = host.User
	}
	if username == "" {
		currentUser, err := user.Current()
		if err != nil {
			return fmt.Errorf("failed to get current user: %w", err)
		}
		username = currentUser.Username
	}
	target := fmt.Sprintf("%s@%s", username, host.Hostname)
	args = append(args, target)

	cmd := exec.Command("ssh", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("ssh connection failed: %w", err)
	}
	return nil
}

func ExecuteCommand(ctx context.Context, host *types.Host, command string) (string, error) {
	return ExecuteCommandWithUser(ctx, host, command, "")
}

func ExecuteCommandWithUser(ctx context.Context, host *types.Host, command, customUser string) (string, error) {
	args := []string{}

	if host.Port > 0 && host.Port != 22 {
		args = append(args, "-p", fmt.Sprintf("%d", host.Port))
	}

	username := customUser
	if username == "" {
		username = host.User
	}
	if username == "" {
		currentUser, err := user.Current()
		if err != nil {
			return "", fmt.Errorf("failed to get current user: %w", err)
		}
		username = currentUser.Username
	}

	target := fmt.Sprintf("%s@%s", username, host.Hostname)
	args = append(args, target, command)

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ssh", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.String()
	if stderr.Len() > 0 {
		output += "\nSTDERR:\n" + stderr.String()
	}

	if err != nil {
		return output, fmt.Errorf("ssh command failed: %w", err)
	}

	return output, nil
}
