package types

import (
	"fmt"
	"os/user"
)

type Host struct {
	Name     string `json:"name"`
	Hostname string `json:"hostname"`
	Port     int    `json:"port,omitempty"`
	User     string `json:"user,omitempty"`
}

func (h *Host) Address() string {
	if h.Port > 0 && h.Port != 22 {
		return fmt.Sprintf("%s:%d", h.Hostname, h.Port)
	}
	return h.Hostname
}

func (h *Host) SSHCommand() string {
	addr := h.Address()
	username := h.User
	if username == "" {
		if currentUser, err := user.Current(); err == nil {
			username = currentUser.Username
		}
	}
	if username != "" {
		return fmt.Sprintf("ssh %s@%s", username, addr)
	}
	return fmt.Sprintf("ssh %s", addr)
}
