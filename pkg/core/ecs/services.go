package ecs

type Service struct {
	Arn               *string
	Name              string
	TaskDefinitionArn string
	Tasks             []Task
}
