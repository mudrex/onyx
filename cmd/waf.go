package cmd

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/mudrex/onyx/pkg/core/waf"
	"github.com/spf13/cobra"
)

var wafCommand = &cobra.Command{
	Use:   "waf",
	Short: "Actions to be performed on waf",
}

var ipSetName string
var cidrs string

var wafUpdateIpSetCommand = &cobra.Command{
	Use:     "update-ip-set",
	Short:   "",
	Args:    cobra.MaximumNArgs(1),
	Example: "onyx cw disable SomRule",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-east-1"))
		if err != nil {
			log.Fatalf("unable to load SDK config, %v", err)
		}
		ctx := context.Background()

		return waf.UpdateIPSet(ctx, cfg, ipSetName, cidrs)
	},
}

func init() {
	wafCommand.AddCommand(wafUpdateIpSetCommand)

	wafUpdateIpSetCommand.Flags().StringVarP(&ipSetName, "ip-set", "c", "", "Ip set name (requried)")
	wafUpdateIpSetCommand.Flags().StringVarP(&cidrs, "cidrs", "s", "", "CIDRs to add")
	wafUpdateIpSetCommand.MarkFlagRequired("ip-set")
	wafUpdateIpSetCommand.MarkFlagRequired("cidrs")
}
