package cmd

import (
	"context"
	"errors"
	"log"

	"github.com/aws/aws-sdk-go-v2/config"
	configPkg "github.com/mudrex/onyx/pkg/config"
	"github.com/mudrex/onyx/pkg/core/ecs"
	"github.com/spf13/cobra"
)

var ecsClusterName string
var ecsServiceName string
var tagToRevertTo string
var revisionsToLookback int32
var tailLogs int32

var ecsCommand = &cobra.Command{
	Use:   "ecs",
	Short: "Actions to be performed on ECS clusters",
}

var ecsDescribeCommand = &cobra.Command{
	Use:   "describe --cluster <cluster-name> [--service <service-name>]",
	Short: "Describes the given ECS cluster tasks.",
	Long:  `Lists down the private IP's of the ec2 instances the tasks of the cluster are running on, filtered by service name if provided.`,
	Args:  cobra.NoArgs,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return configPkg.LoadConfig()
	},
	Example: "onyx ecs describe --cluster staging-api-cluster \nonyx ecs describe --cluster staging-api-cluster --service some-service",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(configPkg.GetRegion()))
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

var ecsSpawnShellCommand = &cobra.Command{
	Use:   "spawn-shell --cluster <cluster-name> --service <service-name>",
	Short: "SpawnShells the given service.",
	Long:  `For the given cluster and service name pair, onyx spawns the docker shell bypassing the host instance's shell`,
	Args:  cobra.NoArgs,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return configPkg.LoadConfig()
	},
	Example: "onyx ecs spawn-shell --cluster staging-api-cluster --service some-service",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(configPkg.GetRegion()))
		if err != nil {
			log.Fatalf("unable to load SDK config, %v", err)
		}
		ctx := context.Background()

		if ecsClusterName == "" {
			return errors.New("empty cluster name")
		}

		if ecsServiceName == "" {
			return errors.New("empty service name")
		}

		return ecs.SpawnServiceShell(ctx, cfg, ecsServiceName, ecsClusterName)
	},
}

var ecsListAccessCommand = &cobra.Command{
	Use:   "list-access",
	Short: "Lists access for the user",
	Args:  cobra.NoArgs,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return configPkg.LoadConfig()
	},
	Example: "onyx ecs list-access",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		return ecs.ListAccess(ctx)
	},
}

var ecsTailLogsCommand = &cobra.Command{
	Use:   "tail-logs --cluster <cluster-name> --service <service-name> [--tail n]",
	Short: "Tail logs for a service container",
	Long:  `For the given cluster and service name pair, onyx tails the docker instance`,
	Args:  cobra.NoArgs,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return configPkg.LoadConfig()
	},
	Example: "onyx ecs tail-logs --cluster staging-api-cluster --service some-service --tail 100",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(configPkg.GetRegion()))
		if err != nil {
			log.Fatalf("unable to load SDK config, %v", err)
		}
		ctx := context.Background()

		if ecsClusterName == "" {
			return errors.New("empty cluster name")
		}

		if ecsServiceName == "" {
			return errors.New("empty service name")
		}

		return ecs.TailContainerLogs(ctx, cfg, ecsServiceName, ecsClusterName, tailLogs)
	},
}

var ecsRestartServiceCommand = &cobra.Command{
	Use:     "restart --cluster <cluster-name> [--service <service-name>]",
	Short:   "Forces new deployment of ECS services",
	Long:    `Triggers redployment of the chosen services of a cluster. If service name is provided it restarts only the exact matching input, else fails.`,
	Example: "onyx ecs restart --cluster staging-api-cluster\nonyx ecs restart --cluster staging-api-cluster --service backtest_services",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return configPkg.LoadConfig()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(configPkg.GetRegion()))
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
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return configPkg.LoadConfig()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(configPkg.GetRegion()))
		if err != nil {
			log.Fatalf("unable to load SDK config, %v", err)
		}
		ctx := context.Background()

		return ecs.UpdateContainerAgent(ctx, cfg)
	},
}

var ecsRevertToCommand = &cobra.Command{
	Use:   "revert",
	Short: "",
	Long:  `Reverts the service to the tag provided. It looks for the given tag in last n revisions of the task definition family and reverts to that state`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return configPkg.LoadConfig()
	},
	Example: "onyx ecs revert --cluster production --service user --tag v0.0.12 --past 10",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(configPkg.GetRegion()))
		if err != nil {
			log.Fatalf("unable to load SDK config, %v", err)
		}
		ctx := context.Background()

		return ecs.Revert(ctx, cfg, ecsClusterName, ecsServiceName, tagToRevertTo, revisionsToLookback)
	},
}

func init() {
	ecsCommand.AddCommand(ecsDescribeCommand, ecsRestartServiceCommand, ecsUpdateContainerInstanceCommand, ecsRevertToCommand, ecsSpawnShellCommand, ecsTailLogsCommand, ecsListAccessCommand)

	ecsRestartServiceCommand.Flags().StringVarP(&ecsClusterName, "cluster", "c", "", "Cluster Name (required)")
	ecsRestartServiceCommand.MarkFlagRequired("cluster")
	ecsRestartServiceCommand.Flags().StringVarP(&ecsServiceName, "service", "s", "", "Service Name")

	ecsDescribeCommand.Flags().StringVarP(&ecsClusterName, "cluster", "c", "", "Cluster Name (required)")
	ecsDescribeCommand.Flags().StringVarP(&ecsServiceName, "service", "s", "", "Filters tasks belonging to the service name provided. Returns the best matching service tasks. (required)")
	ecsDescribeCommand.MarkFlagRequired("service")
	ecsDescribeCommand.MarkFlagRequired("cluster")

	ecsSpawnShellCommand.Flags().StringVarP(&ecsClusterName, "cluster", "c", "", "Cluster Name (required)")
	ecsSpawnShellCommand.Flags().StringVarP(&ecsServiceName, "service", "s", "", "Filters tasks belonging to the service name provided. Returns the best matching service tasks. (required)")
	ecsSpawnShellCommand.MarkFlagRequired("service")
	ecsSpawnShellCommand.MarkFlagRequired("cluster")

	ecsTailLogsCommand.Flags().StringVarP(&ecsClusterName, "cluster", "c", "", "Cluster Name (required)")
	ecsTailLogsCommand.Flags().StringVarP(&ecsServiceName, "service", "s", "", "Filters tasks belonging to the service name provided. Returns the best matching service tasks. (required)")
	ecsTailLogsCommand.Flags().Int32VarP(&tailLogs, "tail", "t", 10, "")
	ecsTailLogsCommand.MarkFlagRequired("service")
	ecsTailLogsCommand.MarkFlagRequired("cluster")

	ecsRevertToCommand.Flags().StringVarP(&ecsClusterName, "cluster", "c", "", "Cluster Name (required)")
	ecsRevertToCommand.Flags().StringVarP(&ecsServiceName, "service", "s", "", "Filters tasks belonging to the service name provided. Returns the best matching service tasks.")
	ecsRevertToCommand.Flags().StringVarP(&tagToRevertTo, "tag", "", "", "Tag to which the service will be reverted")
	ecsRevertToCommand.MarkFlagRequired("cluster")
	ecsRevertToCommand.MarkFlagRequired("service")
	ecsRevertToCommand.MarkFlagRequired("tag")
	ecsRevertToCommand.Flags().Int32VarP(&revisionsToLookback, "past", "", 5, "Revisions to look back the tag in. Max lookback is 50")
}
