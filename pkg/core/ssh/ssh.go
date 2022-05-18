package ssh

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/mudrex/onyx/pkg/audit"
	"github.com/mudrex/onyx/pkg/auth"
	"github.com/mudrex/onyx/pkg/config"
	"github.com/mudrex/onyx/pkg/logger"
	"github.com/mudrex/onyx/pkg/notifier"
	"github.com/mudrex/onyx/pkg/utils"
)

var accessList = map[string]map[string]int{}

func Do(ctx context.Context, userHost string) error {
	sshUser := strings.Split(userHost, "@")[0]
	host := strings.Split(userHost, "@")[1]

	username := utils.GetUser()

	if config.Config.VPCCidr != "" {
		_, cidr, err := net.ParseCIDR(config.Config.VPCCidr)
		if err != nil {
			return err
		}

		if !cidr.Contains(net.ParseIP(host)) {
			log := fmt.Sprintf(":bangbang: [ssh/do] *%s* attempted ssh via public ip: _%s_", username, host)
			notifier.Notify(
				config.Config.SlackHook,
				log,
			)
			audit.Log(ctx, log)

			return fmt.Errorf("%s is not a private IP. Aborting. %s", logger.Underline(host), logger.Red("This act will be reported"))
		}
	}

	isAuthorized, err := auth.CheckUserAccessForHostShell(ctx, username, host)
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

	willUserAbleToSSH := utils.CheckIfUserAbleToLogin(config.Config.PrivateKey, host, sshUser)
	if !willUserAbleToSSH {
		return fmt.Errorf("unable to ssh %s", userHost)
	}

	logger.Info("Spawning shell for %s", logger.Underline(userHost))

	log := fmt.Sprintf("[ssh/do] *%s* logged in to _%s_", username, userHost)
	notifier.Notify(
		config.Config.SlackHook,
		log,
	)

	audit.Log(ctx, log)

	sshCmdDockerShell := fmt.Sprintf("ssh -t -i %s %s", config.Config.PrivateKey, userHost)
	out1 := exec.Command("bash", "-c", sshCmdDockerShell)
	out1.Stdin = os.Stdin
	out1.Stdout = os.Stdout
	out1.Stderr = os.Stderr
	out1.Run()

	syscall.Setuid(userUID)
	logger.Success("Exiting safely")

	return nil
}
