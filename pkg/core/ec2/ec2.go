package ec2

import (
	"context"

	"bitbucket.org/mudrex/onyx/pkg/logger"
	"github.com/aws/aws-sdk-go-v2/aws"
	ec2Lib "github.com/aws/aws-sdk-go-v2/service/ec2"
)

type Instance struct {
	ID          string
	PublicIPv4  string
	PrivateIPv4 string
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

func StopInstance(ctx context.Context, cfg aws.Config, instanceID string) error {
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

func StartInstance(ctx context.Context, cfg aws.Config, instanceID string) error {
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
