package sandstorm

import (
	"context"

	"bitbucket.org/mudrex/onyx/pkg/logger"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/applicationautoscaling"
	"github.com/aws/aws-sdk-go-v2/service/applicationautoscaling/types"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
)

type Service struct {
	Name         string
	ClusterName  string
	DesiredCount int32
	MinCount     int32
	MaxCount     int32
}

var staging = []Service{
	{
		Name:         "user_services",
		DesiredCount: 1,
		MinCount:     1,
		MaxCount:     2,
		ClusterName:  "staging-api-cluster",
	},
	{
		Name:         "wallet_services",
		DesiredCount: 1,
		MinCount:     1,
		MaxCount:     2,
		ClusterName:  "staging-api-cluster",
	},
	{
		Name:         "backtest_services",
		DesiredCount: 1,
		MinCount:     1,
		MaxCount:     2,
		ClusterName:  "staging-api-cluster",
	},
	{
		Name:         "data_websocket_services",
		DesiredCount: 1,
		MinCount:     1,
		MaxCount:     1,
		ClusterName:  "staging-api-cluster",
	},
	{
		Name:         "wallet_services_beat",
		DesiredCount: 1,
		MinCount:     1,
		MaxCount:     1,
		ClusterName:  "staging-celery-cluster",
	},
	{
		Name:         "backtest_services_beat",
		DesiredCount: 1,
		MinCount:     1,
		MaxCount:     1,
		ClusterName:  "staging-celery-cluster",
	},
	{
		Name:         "alert_services_beat",
		DesiredCount: 1,
		MinCount:     1,
		MaxCount:     1,
		ClusterName:  "staging-celery-cluster",
	},
	{
		Name:         "central_services_beat",
		DesiredCount: 1,
		MinCount:     1,
		MaxCount:     1,
		ClusterName:  "staging-celery-cluster",
	},
	{
		Name:         "data_services_beat",
		DesiredCount: 1,
		MinCount:     1,
		MaxCount:     1,
		ClusterName:  "staging-celery-cluster",
	},
	{
		Name:         "backtest_services_celery_1",
		DesiredCount: 1,
		MinCount:     1,
		MaxCount:     1,
		ClusterName:  "staging-celery-cluster",
	},
	{
		Name:         "backtest_services_celery_2",
		DesiredCount: 1,
		MinCount:     1,
		MaxCount:     2,
		ClusterName:  "staging-celery-cluster",
	},
	{
		Name:         "backtest_services_celery_3",
		DesiredCount: 1,
		MinCount:     1,
		MaxCount:     2,
		ClusterName:  "staging-celery-cluster",
	},
	{
		Name:         "backtest_services_celery_4",
		DesiredCount: 1,
		MinCount:     1,
		MaxCount:     2,
		ClusterName:  "staging-celery-cluster",
	},
	{
		Name:         "alert_services_celery",
		DesiredCount: 1,
		MinCount:     1,
		MaxCount:     2,
		ClusterName:  "staging-celery-cluster",
	},
	{
		Name:         "wallet_services_celery",
		DesiredCount: 1,
		MinCount:     1,
		MaxCount:     2,
		ClusterName:  "staging-celery-cluster",
	},
	{
		Name:         "data_services_celery_1",
		DesiredCount: 1,
		MinCount:     1,
		MaxCount:     2,
		ClusterName:  "staging-celery-cluster",
	},
	{
		Name:         "data_services_celery_2",
		DesiredCount: 1,
		MinCount:     1,
		MaxCount:     1,
		ClusterName:  "staging-celery-cluster",
	},
	{
		Name:         "data_services_celery_3",
		DesiredCount: 1,
		MinCount:     1,
		MaxCount:     1,
		ClusterName:  "staging-celery-cluster",
	},
	{
		Name:         "central_services_celery",
		DesiredCount: 1,
		MinCount:     1,
		MaxCount:     2,
		ClusterName:  "staging-celery-cluster",
	},
	{
		Name:         "strategy_manager_v2",
		DesiredCount: 1,
		MinCount:     1,
		MaxCount:     1,
		ClusterName:  "staging-api-cluster",
	},
	{
		Name:         "socket_services",
		DesiredCount: 1,
		MinCount:     1,
		MaxCount:     2,
		ClusterName:  "staging-api-cluster",
	},
}

var production = []Service{
	{
		Name:         "user_services",
		DesiredCount: 2,
		MinCount:     2,
		MaxCount:     5,
		ClusterName:  "production-cluster",
	},
	{
		Name:         "wallet_services",
		DesiredCount: 2,
		MinCount:     2,
		MaxCount:     5,
		ClusterName:  "production-cluster",
	},
	{
		Name:         "backtest_services",
		DesiredCount: 2,
		MinCount:     2,
		MaxCount:     10,
		ClusterName:  "production-cluster",
	},
	{
		Name:         "data_websocket_services",
		DesiredCount: 1,
		MinCount:     1,
		MaxCount:     1,
		ClusterName:  "production-cluster",
	},
	{
		Name:         "wallet_services_beat",
		DesiredCount: 1,
		MinCount:     1,
		MaxCount:     1,
		ClusterName:  "production-celery-cluster",
	},
	{
		Name:         "backtest_services_beat",
		DesiredCount: 1,
		MinCount:     1,
		MaxCount:     1,
		ClusterName:  "production-celery-cluster",
	},
	{
		Name:         "alert_services_beat",
		DesiredCount: 1,
		MinCount:     1,
		MaxCount:     1,
		ClusterName:  "production-celery-cluster",
	},
	{
		Name:         "central_services_beat",
		DesiredCount: 1,
		MinCount:     1,
		MaxCount:     1,
		ClusterName:  "production-celery-cluster",
	},
	{
		Name:         "data_services_beat",
		DesiredCount: 1,
		MinCount:     1,
		MaxCount:     1,
		ClusterName:  "production-celery-cluster",
	},
	{
		Name:         "backtest_services_celery_1",
		DesiredCount: 6,
		MinCount:     6,
		MaxCount:     6,
		ClusterName:  "production-celery-cluster",
	},
	{
		Name:         "backtest_services_celery_2",
		DesiredCount: 1,
		MinCount:     1,
		MaxCount:     5,
		ClusterName:  "production-celery-cluster",
	},
	{
		Name:         "backtest_services_celery_3",
		DesiredCount: 1,
		MinCount:     1,
		MaxCount:     5,
		ClusterName:  "production-celery-cluster",
	},
	{
		Name:         "backtest_services_celery_4",
		DesiredCount: 2,
		MinCount:     2,
		MaxCount:     15,
		ClusterName:  "production-celery-cluster",
	},
	{
		Name:         "alert_services_celery",
		DesiredCount: 1,
		MinCount:     1,
		MaxCount:     5,
		ClusterName:  "production-celery-cluster",
	},
	{
		Name:         "wallet_services_celery",
		DesiredCount: 1,
		MinCount:     1,
		MaxCount:     5,
		ClusterName:  "production-celery-cluster",
	},
	{
		Name:         "data_services_celery",
		DesiredCount: 3,
		MinCount:     3,
		MaxCount:     10,
		ClusterName:  "production-celery-cluster",
	},
	{
		Name:         "central_services_celery",
		DesiredCount: 2,
		MinCount:     1,
		MaxCount:     5,
		ClusterName:  "production-celery-cluster",
	},
	{
		Name:         "strategy_manager_1",
		DesiredCount: 1,
		MinCount:     1,
		MaxCount:     1,
		ClusterName:  "production-cluster",
	},
	{
		Name:         "socket_services",
		DesiredCount: 1,
		MinCount:     1,
		MaxCount:     5,
		ClusterName:  "production-cluster",
	},
}

func Process(ctx context.Context, cfg aws.Config, env, event string) {
	ecsHandler := ecs.NewFromConfig(cfg)
	autoscalingHandler := applicationautoscaling.NewFromConfig(cfg)

	logger.Info("Running sandstorm %s on %s", logger.Bold(event), logger.Bold(env))

	var serviceList []Service
	if env == "staging" {
		serviceList = staging
	} else if env == "production" {
		serviceList = production
	} else {
		return
	}

	if event == "revert" {
		serviceList = reverseArray(serviceList)
	}

	for _, service := range serviceList {
		desiredCount := service.DesiredCount
		minCount := service.MinCount
		if event == "init" {
			minCount = 0
			desiredCount = 0
		}

		_, err := autoscalingHandler.RegisterScalableTarget(ctx, &applicationautoscaling.RegisterScalableTargetInput{
			ResourceId:        aws.String("service/" + service.ClusterName + "/" + service.Name),
			ServiceNamespace:  types.ServiceNamespaceEcs,
			MinCapacity:       aws.Int32(minCount),
			MaxCapacity:       aws.Int32(service.MaxCount),
			ScalableDimension: types.ScalableDimensionECSServiceDesiredCount,
			SuspendedState: &types.SuspendedState{
				DynamicScalingInSuspended:  aws.Bool(false),
				DynamicScalingOutSuspended: aws.Bool(event == "init"),
				ScheduledScalingSuspended:  aws.Bool(event == "init"),
			},
		})
		if err != nil {
			logger.Error("%s (%d) -> %s (%s) | autoscaling error: %s", logger.Bold(event), desiredCount, logger.Red(logger.Underline(service.Name)), service.ClusterName, err.Error())
			continue
		}

		_, err = ecsHandler.UpdateService(ctx, &ecs.UpdateServiceInput{
			Cluster:      aws.String(service.ClusterName),
			Service:      aws.String(service.Name),
			DesiredCount: aws.Int32(desiredCount),
		})
		if err != nil {
			logger.Error("%s (%d) -> %s (%s) | Error: %s", logger.Bold(event), desiredCount, logger.Red(logger.Underline(service.Name)), service.ClusterName, err.Error())
		} else {
			logger.Success("%s (%d) -> %s (%s)", logger.Bold(event), desiredCount, logger.Underline(service.Name), service.ClusterName)
		}
	}
}

func reverseArray(arr []Service) []Service {
	for i, j := 0, len(arr)-1; i < j; i, j = i+1, j-1 {
		arr[i], arr[j] = arr[j], arr[i]
	}
	return arr
}
