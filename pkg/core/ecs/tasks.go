package ecs

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	ecsLib "github.com/aws/aws-sdk-go-v2/service/ecs"
)

type Task struct {
	Arn               *string
	TaskDefinitionArn string
	ContainerInstance *ContainerInstance
	Service           *Service
}

func DescribeTasks(ctx context.Context, cfg aws.Config, clusterName string, services *[]Service) *map[string]Task {
	ecsHandler := ecsLib.NewFromConfig(cfg)

	tasksArns := make([]string, 0)
	for _, service := range *services {
		allTasksOutput, err := ecsHandler.ListTasks(ctx, &ecsLib.ListTasksInput{
			Cluster:     &clusterName,
			ServiceName: aws.String(service.Name),
		})
		if err == nil {
			tasksArns = append(tasksArns, allTasksOutput.TaskArns...)
		}
	}

	detailedTasks, err := ecsHandler.DescribeTasks(ctx, &ecsLib.DescribeTasksInput{
		Cluster: &clusterName,
		Tasks:   tasksArns,
	})

	allTasks := make(map[string]Task)
	if err != nil {
		return &allTasks
	}

	for _, task := range detailedTasks.Tasks {
		if task.ContainerInstanceArn != nil {
			allTasks[*task.TaskArn] = Task{
				Arn:               task.TaskArn,
				TaskDefinitionArn: *task.TaskDefinitionArn,
				ContainerInstance: &ContainerInstance{
					Arn: task.ContainerInstanceArn,
				},
				Service: &Service{},
			}
		}
	}

	return &allTasks
}
