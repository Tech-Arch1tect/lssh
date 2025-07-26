package ssh

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"syscall"

	"github.com/tech-arch1tect/lssh/pkg/types"
)

func Connect(host *types.Host) error {
	args := []string{"ssh"}

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

	sshPath, err := exec.LookPath("ssh")
	if err != nil {
		return fmt.Errorf("ssh command not found: %w", err)
	}

	env := os.Environ()
	err = syscall.Exec(sshPath, args, env)
	if err != nil {
		return fmt.Errorf("failed to execute ssh: %w", err)
	}

	return nil
}
