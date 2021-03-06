package iam

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"

	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	configPkg "github.com/mudrex/onyx/pkg/config"
	"github.com/mudrex/onyx/pkg/logger"
	"github.com/mudrex/onyx/pkg/utils"
)

func Whoami() (string, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(configPkg.GetRegion()))
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}
	ctx := context.Background()

	iamHandler := iam.NewFromConfig(cfg)
	output, err := iamHandler.GetUser(ctx, &iam.GetUserInput{})
	if err != nil {
		return "", err
	}

	return *output.User.UserName, nil
}

func CreateUser(userName, path string) error {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(configPkg.GetRegion()))
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}
	ctx := context.Background()
	iamHandler := iam.NewFromConfig(cfg)

	output, err := iamHandler.CreateUser(ctx, &iam.CreateUserInput{
		UserName: aws.String(userName),
		Path:     aws.String(path),
	})
	if err != nil {
		return err
	}

	newPassword := utils.GetRandomStringWithSymbols(40)
	_, err = iamHandler.CreateLoginProfile(ctx, &iam.CreateLoginProfileInput{
		UserName:              output.User.UserName,
		Password:              aws.String(newPassword),
		PasswordResetRequired: true,
	})
	if err != nil {
		return err
	}

	logger.Success("Create new user %s with password: %s", logger.Bold(userName), logger.Underline(newPassword))

	iamHandler.AddUserToGroup(ctx, &iam.AddUserToGroupInput{
		UserName:  output.User.UserName,
		GroupName: aws.String("ConsoleAccess"),
	})

	iamHandler.AddUserToGroup(ctx, &iam.AddUserToGroupInput{
		UserName:  output.User.UserName,
		GroupName: aws.String("CICDLevel1"),
	})

	iamHandler.AddUserToGroup(ctx, &iam.AddUserToGroupInput{
		UserName:  output.User.UserName,
		GroupName: aws.String("SecurityGroupsLevel2"),
	})

	return nil
}

func DeleteUser(userName string) error {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(configPkg.GetRegion()))
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}
	ctx := context.Background()
	iamHandler := iam.NewFromConfig(cfg)

	groups, err := iamHandler.ListGroupsForUser(ctx, &iam.ListGroupsForUserInput{
		UserName: aws.String(userName),
	})
	if err != nil {
		return err
	}

	for _, group := range groups.Groups {
		iamHandler.RemoveUserFromGroup(ctx, &iam.RemoveUserFromGroupInput{
			UserName:  aws.String(userName),
			GroupName: group.GroupName,
		})
	}

	_, err = iamHandler.DeleteLoginProfile(ctx, &iam.DeleteLoginProfileInput{
		UserName: aws.String(userName),
	})
	if err != nil {
		return err
	}

	_, err = iamHandler.DeleteUser(ctx, &iam.DeleteUserInput{
		UserName: aws.String(userName),
	})

	if err != nil {
		return err
	}

	return nil
}

func CheckExpiredAccessKeys() error {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(configPkg.GetRegion()))
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}
	ctx := context.Background()
	iamHandler := iam.NewFromConfig(cfg)

	allUsers, err := iamHandler.ListUsers(ctx, &iam.ListUsersInput{
		PathPrefix: aws.String("/tech/"),
	})
	if err != nil {
		return err
	}

	olderAccessKeys := make([]string, 0)
	dormantAccessKeys := make([]string, 0)

	for _, user := range allUsers.Users {
		accessKeys, err := iamHandler.ListAccessKeys(ctx, &iam.ListAccessKeysInput{
			UserName: user.UserName,
		})
		if err != nil {
			return err
		}

		for _, accessKey := range accessKeys.AccessKeyMetadata {
			lastUsed, err := iamHandler.GetAccessKeyLastUsed(ctx, &iam.GetAccessKeyLastUsedInput{
				AccessKeyId: accessKey.AccessKeyId,
			})
			if err != nil {
				return err
			}

			if aws.ToTime(accessKey.CreateDate).Unix() < time.Now().AddDate(0, -3, 0).Unix() {
				olderAccessKeys = append(olderAccessKeys, aws.ToString(user.UserName))
			}

			if aws.ToTime(lastUsed.AccessKeyLastUsed.LastUsedDate).Unix() < time.Now().AddDate(0, -1, 0).Unix() {
				dormantAccessKeys = append(dormantAccessKeys, aws.ToString(user.UserName))
			}
		}
	}

	if len(olderAccessKeys) > 0 {
		fmt.Println("Older access keys:", strings.Join(olderAccessKeys, ", "))
	}

	if len(dormantAccessKeys) > 0 {
		fmt.Println("Dormant access keys:", strings.Join(dormantAccessKeys, ", "))
	}

	return nil
}

func DuplicateSecret() error {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-east-1"))
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}
	ctx := context.Background()
	secretsmanagerHandler := secretsmanager.NewFromConfig(cfg)

	secretName := "-"
	finalSecretName := "-"

	output, err := secretsmanagerHandler.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretName),
	})
	if err != nil {
		return err
	}

	do, err := secretsmanagerHandler.DescribeSecret(ctx, &secretsmanager.DescribeSecretInput{
		SecretId: aws.String(secretName),
	})
	if err != nil {
		return err
	}

	tags := []types.Tag{
		{
			Key:   aws.String("Name"),
			Value: aws.String(finalSecretName),
		},
	}

	for _, tag := range do.Tags {
		if aws.ToString(tag.Key) != "Name" {
			tags = append(tags, tag)
		}
	}

	_, err = secretsmanagerHandler.CreateSecret(
		ctx,
		&secretsmanager.CreateSecretInput{
			Name:         aws.String(finalSecretName),
			Description:  do.Description,
			SecretString: output.SecretString,
			Tags:         tags,
		},
	)
	if err != nil {
		return err
	}

	return nil
}
