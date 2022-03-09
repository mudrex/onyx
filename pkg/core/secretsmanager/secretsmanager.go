package secretsmanager

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

func GetSecret(ctx context.Context, cfg aws.Config, name string) (value string) {
	svc := secretsmanager.NewFromConfig(cfg)

	result, err := svc.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(name),
	})
	if err != nil {
		return ""
	}

	return aws.ToString(result.SecretString)
}
