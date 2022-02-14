package ecs

import (
	"context"
	"regexp"

	"github.com/aws/aws-sdk-go-v2/aws"
	ecsLib "github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	configPkg "github.com/mudrex/onyx/pkg/config"
)

type Service struct {
	Arn                  *string
	Name                 string
	TaskDefinitionArn    string
	OldTaskDefinitionArn string
	Tasks                []Task
	ClusterName          string
	TaskDefinitions      []TaskDefinition
	CanRevert            bool
}

func (s *Service) GetTaskDefintions(ctx context.Context, cfg aws.Config, revisionsToLookback int32) {
	ecsHandler := ecsLib.NewFromConfig(cfg)

	re := regexp.MustCompile(`arn:aws:ecs:` + configPkg.GetRegion() + `:\d+:task-definition/(.*)+`)
	s.OldTaskDefinitionArn = re.ReplaceAllString(s.TaskDefinitionArn, "${1}")

	m := regexp.MustCompile(`arn:aws:ecs:` + configPkg.GetRegion() + `:\d+:task-definition/(.*)+:\d+`)
	tdName := m.ReplaceAllString(s.TaskDefinitionArn, "${1}")

	o1, err := ecsHandler.ListTaskDefinitions(ctx, &ecsLib.ListTaskDefinitionsInput{
		FamilyPrefix: aws.String(tdName),
		MaxResults:   aws.Int32(revisionsToLookback),
		Sort:         types.SortOrderDesc,
	})

	if err != nil {
		return
	}

	s.TaskDefinitions = make([]TaskDefinition, 0)

	for _, td := range o1.TaskDefinitionArns {
		o, err := ecsHandler.DescribeTaskDefinition(ctx, &ecsLib.DescribeTaskDefinitionInput{
			TaskDefinition: aws.String(re.ReplaceAllString(td, "${1}")),
		})

		if err != nil {
			return
		}

		s.TaskDefinitions = append(s.TaskDefinitions, TaskDefinition{
			Arn:     o.TaskDefinition.TaskDefinitionArn,
			Name:    tdName,
			Version: o.TaskDefinition.Revision,
			Image:   *o.TaskDefinition.ContainerDefinitions[0].Image,
		})
	}
}
