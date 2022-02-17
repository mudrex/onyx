package ssh

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/mudrex/onyx/pkg/auth"
	"github.com/mudrex/onyx/pkg/config"
	"github.com/mudrex/onyx/pkg/logger"
	"github.com/mudrex/onyx/pkg/notifier"
)

var accessList = map[string]map[string]int{}

func Do(ctx context.Context, userHost string) error {
	host := strings.Split(userHost, "@")[1]

	if config.Config.VPCCidr != "" {
		_, cidr, err := net.ParseCIDR(config.Config.VPCCidr)
		if err != nil {
			return err
		}

		if !cidr.Contains(net.ParseIP(host)) {
			return fmt.Errorf("%s is not a private IP. Aborting. This act will be reported", host)
		}
	}

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

	notifier.Notify(
		config.Config.SlackHook,
		fmt.Sprintf("[ssh/do] *%s* logged in to _%s_", username, userHost),
	)

	sshCmdDockerShell := fmt.Sprintf("sudo ssh -t -i %s %s", config.Config.PrivateKey, userHost)
	out1 := exec.Command("bash", "-c", sshCmdDockerShell)
	out1.Stdin = os.Stdin
	out1.Stdout = os.Stdout
	out1.Stderr = os.Stderr
	out1.Run()

	syscall.Setuid(userUID)
	logger.Success("Exiting safely")

	return nil
}
