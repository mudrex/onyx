package cmd

import (
	"bitbucket.org/mudrex/onyx/pkg/core/iam"
	"bitbucket.org/mudrex/onyx/pkg/logger"
	"github.com/spf13/cobra"
)

var iamCommand = &cobra.Command{
	Use:   "iam",
	Short: "Actions to be performed on IAM namespace",
}

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Returns the user making requests",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, err := iam.Whoami()
		if err != nil {
			return err
		}

		logger.Info(name)

		return nil
	},
}

var newUserCmd = &cobra.Command{
	Use:   "create-user",
	Short: "Creates a new user with minimal permissions required to access console",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return iam.CreateUser(args[0], args[1])
	},
}

var deleteUserCmd = &cobra.Command{
	Use:   "delete-user",
	Short: "Deletes a user",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return iam.DeleteUser(args[0])
	},
}

var expiredAccessKeysCmd = &cobra.Command{
	Use:   "check-expired-access-keys",
	Short: "Checks expired access keys",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return iam.CheckExpiredAccessKeys()
	},
}

func init() {
	iamCommand.AddCommand(whoamiCmd, newUserCmd, deleteUserCmd, expiredAccessKeysCmd)
}
