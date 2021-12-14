package cmd

import (
	"context"
	"errors"
	"log"

	"bitbucket.org/mudrex/onyx/pkg/core/ecs"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/spf13/cobra"
)

var ecsClusterName string
var ecsServiceName string

var ecsCommand = &cobra.Command{
	Use:   "ecs",
	Short: "Actions to be performed on ECS clusters",
}

var ecsDescribeCommand = &cobra.Command{
	Use:     "describe --cluster <cluster-name> [--service <service-name>]",
	Short:   "Describes the given ECS cluster tasks.",
	Long:    `Lists down the private IP's of the ec2 instances the tasks of the cluster are running on, filtered by service name if provided.`,
	Args:    cobra.NoArgs,
	Example: "onyx ecs describe --cluster staging-api-cluster \nonyx ecs describe --cluster staging-api-cluster --service some-service",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-east-1"))
		if err != nil {
			log.Fatalf("unable to load SDK config, %v", err)
		}
		ctx := context.Background()

		if ecsClusterName == "" {
			return errors.New("empty cluster name")
		}

		return ecs.Describe(ctx, cfg, ecsServiceName, ecsClusterName)
	},
}

var ecsRestartServiceCommand = &cobra.Command{
	Use:     "restart --cluster <cluster-name> [--service <service-name>]",
	Short:   "Forces new deployment of ECS services",
	Long:    `Triggers redployment of the chosen services of a cluster. If service name is provided it restarts only the exact matching input, else fails.`,
	Example: "onyx ecs restart --cluster staging-api-cluster\nonyx ecs restart --cluster staging-api-cluster --service backtest_services",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-east-1"))
		if err != nil {
			log.Fatalf("unable to load SDK config, %v", err)
		}
		ctx := context.Background()

		return ecs.RedeployService(ctx, cfg, ecsClusterName, ecsServiceName)
	},
}

var ecsUpdateContainerInstanceCommand = &cobra.Command{
	Use:     "update-agent",
	Short:   "Updates container agents for all attached container instances",
	Long:    ``,
	Example: "onyx ecs update-agent",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-east-1"))
		if err != nil {
			log.Fatalf("unable to load SDK config, %v", err)
		}
		ctx := context.Background()

		return ecs.UpdateContainerAgent(ctx, cfg)
	},
}

func init() {
	ecsCommand.AddCommand(ecsDescribeCommand, ecsRestartServiceCommand, ecsUpdateContainerInstanceCommand)

	ecsRestartServiceCommand.Flags().StringVarP(&ecsClusterName, "cluster", "c", "", "Cluster Name (required)")
	ecsRestartServiceCommand.MarkFlagRequired("cluster")
	ecsRestartServiceCommand.Flags().StringVarP(&ecsServiceName, "service", "s", "", "Service Name")

	ecsDescribeCommand.Flags().StringVarP(&ecsClusterName, "cluster", "c", "", "Cluster Name (required)")
	ecsDescribeCommand.Flags().StringVarP(&ecsServiceName, "service", "s", "", "Filters tasks belonging to the service name provided. Returns the best matching service tasks.")
}
