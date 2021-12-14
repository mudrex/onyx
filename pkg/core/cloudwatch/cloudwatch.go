package cloudwatch

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	cloudwatchLib "github.com/aws/aws-sdk-go-v2/service/cloudwatchevents"
)

func DisableRule(ctx context.Context, cfg aws.Config, name string) error {
	cloudwatchHandler := cloudwatchLib.NewFromConfig(cfg)
	_, err := cloudwatchHandler.DisableRule(ctx, &cloudwatchLib.DisableRuleInput{
		Name: aws.String(name),
	})
	return err
}

func EnableRule(ctx context.Context, cfg aws.Config, name string) error {
	cloudwatchHandler := cloudwatchLib.NewFromConfig(cfg)
	_, err := cloudwatchHandler.EnableRule(ctx, &cloudwatchLib.EnableRuleInput{
		Name: aws.String(name),
	})
	return err
}
