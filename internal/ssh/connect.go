package ssh

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"

	"github.com/tech-arch1tect/lssh/pkg/types"
)

func Connect(host *types.Host) error {
	args := []string{}

	if host.Port > 0 && host.Port != 22 {
		args = append(args, "-p", fmt.Sprintf("%d", host.Port))
	}

	username := host.User
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
