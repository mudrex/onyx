package cmd

import (
	"context"
	"errors"
	"log"

	"github.com/aws/aws-sdk-go-v2/config"
	configPkg "github.com/mudrex/onyx/pkg/config"
	"github.com/mudrex/onyx/pkg/core/jenkins"
	"github.com/spf13/cobra"
)

var jenkinsCommand = &cobra.Command{
	Use:   "jenkins",
	Short: "Access control actions to be performed on Jenkins",
}

var jenkinsRefreshCommand = &cobra.Command{
	Use:   "refresh",
	Short: "Refresh from config file.",
	Args:  cobra.ExactArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return configPkg.LoadConfig()
	},
	Example: "onyx jenkins refresh",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(configPkg.GetRegion()))
		if err != nil {
			log.Fatalf("unable to load SDK config, %v", err)
		}
		ctx := context.Background()

		if args[0] == "users" {
			return jenkins.RefreshUsers(ctx, cfg)
		} else if args[0] == "roles" {
			return jenkins.RefreshRoles(ctx, cfg)
		}

		return errors.New("Invalid type " + args[0])
	},
}

func init() {
	jenkinsCommand.AddCommand(jenkinsRefreshCommand)
}
