package cmd

import (
	"context"
	"log"

	"bitbucket.org/mudrex/onyx/pkg/core/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/spf13/cobra"
)

var cloudwatchCommand = &cobra.Command{
	Use:   "cw",
	Short: "Actions to be performed on Cloudwatch",
}

var cloudwatchDisableRuleCommand = &cobra.Command{
	Use:     "disable <name>",
	Short:   "Disables cloudwatch rule",
	Args:    cobra.MaximumNArgs(1),
	Example: "onyx cw disable SomRule",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-east-1"))
		if err != nil {
			log.Fatalf("unable to load SDK config, %v", err)
		}
		ctx := context.Background()

		return cloudwatch.DisableRule(ctx, cfg, args[0])
	},
}

var cloudwatchEnableRuleCommand = &cobra.Command{
	Use:     "enable <name>",
	Short:   "Enables cloudwatch rule",
	Args:    cobra.MaximumNArgs(1),
	Example: "onyx cw disable SomRule",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-east-1"))
		if err != nil {
			log.Fatalf("unable to load SDK config, %v", err)
		}
		ctx := context.Background()

		return cloudwatch.EnableRule(ctx, cfg, args[0])
	},
}

func init() {
	cloudwatchCommand.AddCommand(cloudwatchDisableRuleCommand, cloudwatchEnableRuleCommand)
}
