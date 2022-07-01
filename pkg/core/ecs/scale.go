package ecs

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/applicationautoscaling"
	"github.com/aws/aws-sdk-go-v2/service/applicationautoscaling/types"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/mudrex/onyx/pkg/config"
	"github.com/mudrex/onyx/pkg/filesystem"
	"github.com/mudrex/onyx/pkg/logger"
)

// ServiceLimits defines the service limits
type ServiceLimits struct {
	// min count defined when creating a service
	// in normal scenario this should be used as min task count
	// in normal scenario this will also be used as the desired task count
	MinCount int32 `json:"min_count"`

	// max count defined when creating a service
	// in normal scenario this should be used as max task count
	MaxCount int32 `json:"max_count"`

	// ScaleUpMinCount is the required min count in an event of scale up
	ScaleUpMinCount int32 `json:"scale_up_min_count"`
}

var scaleUpDownMap = make(map[string]map[string]ServiceLimits)

func populateScaleUpDownMap() error {
	configData, err := filesystem.ReadFile(config.Config.ECSScaleUpConfig)
	if err != nil {
		return err
	}

	err = json.Unmarshal([]byte(configData), &scaleUpDownMap)
	if err != nil {
		return err
	}

	return nil
}

func Scale(ctx context.Context, cfg aws.Config, action string) error {
	err := populateScaleUpDownMap()
	if err != nil {
		return err
	}

	autoscalingHandler := applicationautoscaling.NewFromConfig(cfg)
	ecsHandler := ecs.NewFromConfig(cfg)

	for cluster, service := range scaleUpDownMap {
		for name, limits := range service {
			minCount := limits.ScaleUpMinCount
			desiredCount := limits.ScaleUpMinCount
			if action == "down" {
				minCount = limits.MinCount
				desiredCount = limits.MinCount
			}

			output, err := ecsHandler.DescribeServices(ctx, &ecs.DescribeServicesInput{
				Cluster:  aws.String(cluster),
				Services: []string{name},
			})
			if err != nil {
				logger.Error("Unable to describe service %s in %s", logger.Red(logger.Underline(name)), cluster)
				continue
			}

			if len(output.Services) == 0 {
				logger.Error("No service %s found in %s", logger.Red(logger.Underline(name)), cluster)
				continue
			}

			originalDesiredCount := output.Services[0].DesiredCount

			_, err = autoscalingHandler.RegisterScalableTarget(
				ctx,
				&applicationautoscaling.RegisterScalableTargetInput{
					ResourceId:        aws.String("service/" + cluster + "/" + name),
					ServiceNamespace:  types.ServiceNamespaceEcs,
					MinCapacity:       aws.Int32(minCount),
					MaxCapacity:       aws.Int32(limits.MaxCount),
					ScalableDimension: types.ScalableDimensionECSServiceDesiredCount,
				},
			)
			if err != nil {
				logger.Error("%s (%d) -> %s (%s) | autoscaling error: %s", logger.Bold(action), desiredCount, logger.Red(logger.Underline(name)), cluster, err.Error())
				continue
			}

			_, err = ecsHandler.UpdateService(ctx, &ecs.UpdateServiceInput{
				Cluster:      aws.String(cluster),
				Service:      aws.String(name),
				DesiredCount: aws.Int32(desiredCount),
			})
			if err != nil {
				logger.Error("%s (%d) -> %s (%s) | Error: %s", logger.Bold(action), desiredCount, logger.Red(logger.Underline(name)), cluster, err.Error())
			} else {
				logger.Success("Scaled %s %s from %d -> %d", action, logger.Red(logger.Underline(name)), originalDesiredCount, desiredCount)
			}
		}
	}

	logger.Success("Scale %s completed.", logger.Underline(action))

	return nil
}
