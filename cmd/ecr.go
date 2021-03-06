package cmd

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/config"
	configPkg "github.com/mudrex/onyx/pkg/config"
	"github.com/mudrex/onyx/pkg/core/ecr"
	"github.com/spf13/cobra"
)

var preserveImages int32

var ecrCommand = &cobra.Command{
	Use:   "ecr",
	Short: "Actions to be performed on ECR",
}

var ecrCleanupCommand = &cobra.Command{
	Use:   "cleanup",
	Short: "Cleans up older tags pushed on ECR repository",
	Args:  cobra.MaximumNArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return configPkg.LoadConfig()
	},
	Example: "onyx ecr cleanup [service-name] --preserve <preserve-images>",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(configPkg.GetRegion()))
		if err != nil {
			log.Fatalf("unable to load SDK config, %v", err)
		}
		ctx := context.Background()

		if len(args) > 0 {
			return ecr.Cleanup(ctx, cfg, args[0], preserveImages)
		}

		return ecr.Cleanup(ctx, cfg, "", preserveImages)
	},
}

func init() {
	ecrCleanupCommand.Flags().Int32VarP(&preserveImages, "preserve", "", 4, "Number of images to not delete")

	ecrCommand.AddCommand(ecrCleanupCommand)
}
