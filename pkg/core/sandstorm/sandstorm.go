package sandstorm

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/applicationautoscaling"
	"github.com/aws/aws-sdk-go-v2/service/applicationautoscaling/types"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/mudrex/onyx/pkg/logger"
)

type Service struct {
	Name         string
	ClusterName  string
	DesiredCount int32
	MinCount     int32
	MaxCount     int32
}

var staging = []Service{}

var production = []Service{
	{
		Name:         "",
		DesiredCount: 2,
		MinCount:     2,
		MaxCount:     5,
		ClusterName:  "",
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
