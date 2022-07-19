package cmd

import (
	"context"

	configPkg "github.com/mudrex/onyx/pkg/config"
	"github.com/mudrex/onyx/pkg/core/ssh"
	"github.com/spf13/cobra"
)

var sshCommand = &cobra.Command{
	Use:   "ssh",
	Short: "Role based ssh into remote machines",
}

var sshDoCommand = &cobra.Command{
	Use:     "do <user>@<ip>",
	Short:   "Spawns up the remote machine shell",
	Example: "onyx ssh do ec2-user@11.11.11.11\nonyx ssh do ubuntu@11.11.11.11",
	Args:    cobra.ExactArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return configPkg.LoadConfig()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		return ssh.Do(ctx, args[0])
	},
}

func init() {
	sshCommand.AddCommand(sshDoCommand)
}
