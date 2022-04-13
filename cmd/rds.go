package cmd

import (
	"context"
	"errors"
	"log"

	"github.com/aws/aws-sdk-go-v2/config"
	configPkg "github.com/mudrex/onyx/pkg/config"
	"github.com/mudrex/onyx/pkg/core/rds"
	"github.com/spf13/cobra"
)

var rdsCommand = &cobra.Command{
	Use:   "rds",
	Short: "Actions to be performed on RDS",
}

var rdsRefreshAccessCommand = &cobra.Command{
	Use:   "refresh-access",
	Short: "Refresh access from access file",
	Args:  cobra.ExactArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return configPkg.LoadConfig()
	},
	Example: "onyx rds refresh-access",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(configPkg.GetRegion()))
		if err != nil {
			log.Fatalf("unable to load SDK config, %v", err)
		}
		ctx := context.Background()

		if args[0] == "users" {
			return rds.RefreshUserAccess(ctx, cfg)
		} else if args[0] == "services" {
			return rds.RefreshServicesAccess(ctx, cfg)
		}

		return errors.New("Invalid type " + args[0])
	},
}

func init() {
	rdsCommand.AddCommand(rdsRefreshAccessCommand)
}
