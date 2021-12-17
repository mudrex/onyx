package cloudwatch

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	cloudwatchEventsLib "github.com/aws/aws-sdk-go-v2/service/cloudwatchevents"
)

func DisableRule(ctx context.Context, cfg aws.Config, name string) error {
	cloudwatchHandler := cloudwatchEventsLib.NewFromConfig(cfg)
	_, err := cloudwatchHandler.DisableRule(ctx, &cloudwatchEventsLib.DisableRuleInput{
		Name: aws.String(name),
	})
	return err
}

func EnableRule(ctx context.Context, cfg aws.Config, name string) error {
	cloudwatchHandler := cloudwatchEventsLib.NewFromConfig(cfg)
	_, err := cloudwatchHandler.EnableRule(ctx, &cloudwatchEventsLib.EnableRuleInput{
		Name: aws.String(name),
	})
	return err
}
