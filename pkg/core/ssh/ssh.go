package ssh

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/mudrex/onyx/pkg/auth"
	"github.com/mudrex/onyx/pkg/logger"
)

var accessList = map[string]map[string]int{}

func Do(ctx context.Context, userHost string) error {
	host := strings.Split(userHost, "@")[1]

	username, isAuthorized, err := auth.CheckUserAccessForHostShell(ctx, host)
	if err != nil {
		return err
	}

	if !isAuthorized {
		logger.Error("%s is not authorized to access %s", logger.Underline(username), logger.Bold(userHost))
		return nil
	}

	userUID := syscall.Getuid()

	err = syscall.Setuid(0)
	if err != nil {
		return err
	}

	logger.Info("Spawning shell for %s", logger.Underline(userHost))

	sshCmdDockerShell := fmt.Sprintf("sudo ssh -t -i /opt/gatekeeper/keys/services.pem %s", userHost)
	out1 := exec.Command("bash", "-c", sshCmdDockerShell)
	out1.Stdin = os.Stdin
	out1.Stdout = os.Stdout
	out1.Stderr = os.Stderr
	out1.Run()

	syscall.Setuid(userUID)
	logger.Success("Exiting safely")

	return nil
}
