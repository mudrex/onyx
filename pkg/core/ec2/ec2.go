package ec2

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	ec2Lib "github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/mudrex/onyx/pkg/logger"
)

type Instance struct {
	ID          string
	PublicIPv4  string
	PrivateIPv4 string
}

func GetInstanceIDsByNameTag(ctx context.Context, cfg aws.Config, name string) ([]string, error) {
	ec2Handler := ec2Lib.NewFromConfig(cfg)

	ec2DetailsOutput, err := ec2Handler.DescribeInstances(ctx, &ec2Lib.DescribeInstancesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("tag:Name"),
				Values: []string{name},
			},
		},
	})

	if err != nil {
		return nil, err
	}

	instances := make([]string, 0)
	for _, reservation := range ec2DetailsOutput.Reservations {
		for _, instance := range reservation.Instances {
			instances = append(instances, aws.ToString(instance.InstanceId))
		}
	}

	return instances, err
}

func DescribeInstances(ctx context.Context, cfg aws.Config, instanceIDs []string) (*[]Instance, error) {
	ec2Handler := ec2Lib.NewFromConfig(cfg)

	instances := make([]Instance, 0)
	ec2DetailsOutput, err := ec2Handler.DescribeInstances(ctx, &ec2Lib.DescribeInstancesInput{
		InstanceIds: instanceIDs,
	})
	if err != nil {
		return &instances, err
	}

	for _, reservation := range ec2DetailsOutput.Reservations {
		for _, instance := range reservation.Instances {
			instances = append(instances, Instance{
				ID:          aws.ToString(instance.InstanceId),
				PrivateIPv4: aws.ToString(instance.PrivateIpAddress),
				PublicIPv4:  aws.ToString(instance.PublicIpAddress),
			})
		}
	}

	return &instances, nil
}

func StopInstance(ctx context.Context, cfg aws.Config, instanceID, instanceName string) error {
	if instanceName != "" {
		instanceIDs, err := GetInstanceIDsByNameTag(ctx, cfg, instanceName)
		if err != nil {
			return err
		}

		if len(instanceIDs) == 0 {
			logger.Success("Nothing to do")
			return nil
		}

		instanceID = instanceIDs[0]
	}

	ec2Handler := ec2Lib.NewFromConfig(cfg)
	_, err := ec2Handler.StopInstances(ctx, &ec2Lib.StopInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return err
	}

	logger.Success("Stopped instance %s", instanceID)
	return nil
}

func StartInstance(ctx context.Context, cfg aws.Config, instanceID, instanceName string) error {
	if instanceName != "" {
		instanceIDs, err := GetInstanceIDsByNameTag(ctx, cfg, instanceName)
		if err != nil {
			return err
		}

		if len(instanceIDs) == 0 {
			logger.Success("Nothing to do")
			return nil
		}

		instanceID = instanceIDs[0]
	}

	ec2Handler := ec2Lib.NewFromConfig(cfg)
	_, err := ec2Handler.StartInstances(ctx, &ec2Lib.StartInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return err
	}

	logger.Success("Started instance %s", instanceID)
	return nil
}
