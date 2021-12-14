package cmd

import (
	"context"
	"errors"
	"log"

	"bitbucket.org/mudrex/onyx/pkg/core/sandstorm"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/spf13/cobra"
)

var sandstormCommand = &cobra.Command{
	Use:     "sandstorm <env> <init|revert>",
	Short:   "Starts or stops entire ecs infra",
	Args:    cobra.MinimumNArgs(1),
	Example: "onyx sandstorm init\nonyx sandstorm revert",
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.New("Disabled")
		cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-east-1"))
		if err != nil {
			log.Fatalf("unable to load SDK config, %v", err)
		}
		ctx := context.Background()
		if args[0] != "staging" && args[0] != "production" {
			return errors.New("Invalid env: " + args[0])
		}

		if args[1] != "init" && args[1] != "revert" {
			return errors.New("Invalid type: " + args[1])
		}

		sandstorm.Process(ctx, cfg, args[0], args[1])

		return nil
	},
}
