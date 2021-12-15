package ecs

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	ecsLib "github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/mudrex/onyx/pkg/core/ec2"
	"github.com/mudrex/onyx/pkg/logger"
	"github.com/mudrex/onyx/pkg/utils"
)

type ContainerInstance struct {
	Arn      *string
	Instance ec2.Instance
}

func Describe(ctx context.Context, cfg aws.Config, serviceName, nameFilter string) error {
	clusters, err := ListClusters(ctx, cfg, nameFilter)
	if err != nil {
		return err
	}

	for _, cluster := range *clusters {
		DescribeByCluster(ctx, cfg, cluster.Name, serviceName)
	}

	return nil
}

func DescribeByCluster(ctx context.Context, cfg aws.Config, clusterName, serviceName string) error {
	if serviceName == "" {
		logger.Warn("Service name is not provided. This results in large query, please consider narrowing your search.")
	}

	cluster := Cluster{
		Name: clusterName,
	}

	ecsHandler := ecsLib.NewFromConfig(cfg)
	// Fetch all services of the cluster
	err := cluster.GetServices(ctx, cfg, serviceName)
	if err != nil {
		return err
	}

	// Fetch tasks details of the required services
	allTasks := DescribeTasks(ctx, cfg, clusterName, &cluster.Services)

	// Filter only required container instances
	containerInstancesMap := make(map[string]*ContainerInstance)
	for _, task := range *allTasks {
		containerInstancesMap[*task.ContainerInstance.Arn] = &ContainerInstance{
			Arn: task.ContainerInstance.Arn,
		}
	}

	cluster.ContainerInstances = len(containerInstancesMap)

	containerInstancesArns := make([]string, 0)
	for containerInstanceArn := range containerInstancesMap {
		containerInstancesArns = append(containerInstancesArns, containerInstanceArn)
	}

	// Get the required container instances filteres from tasks in a cluster
	containerInstances, err := ecsHandler.DescribeContainerInstances(ctx, &ecsLib.DescribeContainerInstancesInput{
		ContainerInstances: containerInstancesArns,
		Cluster:            &clusterName,
	})

	if err != nil {
		return err
	}

	instanceIDsMap := make(map[string]ec2.Instance)
	for _, containerInstance := range containerInstances.ContainerInstances {
		instanceIDsMap[*containerInstance.Ec2InstanceId] = ec2.Instance{
			ID: *containerInstance.Ec2InstanceId,
		}

		containerInstancesMap[*containerInstance.ContainerInstanceArn] = &ContainerInstance{
			Arn:      containerInstance.ContainerInstanceArn,
			Instance: instanceIDsMap[*containerInstance.Ec2InstanceId],
		}
	}

	instanceIDs := make([]string, 0)
	for instanceID := range instanceIDsMap {
		instanceIDs = append(instanceIDs, instanceID)
	}

	instancesDetails, err := ec2.DescribeInstances(ctx, cfg, instanceIDs)
	if err != nil {
		return err
	}

	for _, instancesDetail := range *instancesDetails {
		instanceIDsMap[instancesDetail.ID] = instancesDetail
	}

	for containerInstanceArn, containerInstance := range containerInstancesMap {
		instanceID := containerInstance.Instance.ID

		containerInstance.Instance = instanceIDsMap[instanceID]
		containerInstancesMap[containerInstanceArn] = containerInstance
	}

	for taskArn, task := range *allTasks {
		task.ContainerInstance = containerInstancesMap[*task.ContainerInstance.Arn]
		(*allTasks)[taskArn] = task
	}

	taskPerTaskDefinition := make(map[string][]Task)
	for _, task := range *allTasks {
		if tasks, ok := taskPerTaskDefinition[task.TaskDefinitionArn]; ok {
			taskPerTaskDefinition[task.TaskDefinitionArn] = append(tasks, task)
		} else {
			taskPerTaskDefinition[task.TaskDefinitionArn] = []Task{task}
		}
	}

	allServices := &cluster.Services
	for i, service := range *allServices {
		service.Tasks = taskPerTaskDefinition[service.TaskDefinitionArn]
		(*allServices)[i] = service
	}

	cluster.Services = *allServices
	cluster.Print()

	// fmt.Println("Cluster name:", cluster.Name)
	// // fmt.Println("Registered container instances:", cluster.ContainerInstances)

	// for _, service := range cluster.Services {
	// 	fmt.Println("Service Name:", service.Name)
	// 	// fmt.Println("  Task Definition:", service.TaskDefinitionArn)
	// 	fmt.Println("  Tasks:")
	// 	for _, task := range service.Tasks {
	// 		// fmt.Println("    Arn:", *task.Arn)
	// 		fmt.Println("    IP:", task.ContainerInstance.Instance.PrivateIPv4)
	// 	}
	// }

	return nil
}

func RedeployService(ctx context.Context, cfg aws.Config, clusterName, serviceName string) error {
	cluster := Cluster{
		Name: clusterName,
	}

	serviceMap := make(map[string]bool)
	err := cluster.GetServices(ctx, cfg, serviceName)
	if err != nil {
		return err
	}

	cluster.FilterServicesByName(serviceName)

	fmt.Println("Cluster Name:", clusterName)
	fmt.Println("Select service(s) to restart:")
	for i, service := range cluster.Services {
		fmt.Println(logger.Bold(i), ":", service.Name)
	}

	indexes := utils.GetUserInput("Enter choice: ")
	if len(indexes) == 0 {
		return errors.New("invalid choice")
	}

	for _, index := range strings.Split(indexes, ",") {
		i, _ := strconv.ParseInt(strings.TrimSpace(index), 0, 32)
		serviceMap[cluster.Services[int(i)].Name] = true
	}

	services := make([]string, 0)
	for service := range serviceMap {
		services = append(services, service)
	}

	if len(services) == 0 {
		return errors.New("no services to restart")
	}

	for _, service := range services {
		ecsHandler := ecsLib.NewFromConfig(cfg)
		_, err := ecsHandler.UpdateService(ctx, &ecsLib.UpdateServiceInput{
			Cluster:            aws.String(clusterName),
			Service:            aws.String(service),
			ForceNewDeployment: true,
		})

		if err != nil {
			fmt.Println("Unable to restart " + service + ". Error: " + err.Error())
		} else {
			fmt.Println("Restarted " + service)
		}
	}

	return nil
}

func UpdateContainerAgent(ctx context.Context, cfg aws.Config) error {
	ecsHandler := ecsLib.NewFromConfig(cfg)
	clusterOutput, err := ecsHandler.ListClusters(ctx, &ecsLib.ListClustersInput{})
	if err != nil {
		return err
	}

	containerInstances := make(map[string][]string)
	for _, cluster := range clusterOutput.ClusterArns {
		containerInstancesOutput, err := ecsHandler.ListContainerInstances(ctx, &ecsLib.ListContainerInstancesInput{
			Cluster: aws.String(cluster),
		})
		if err != nil {
			logger.Error("Unable to get container instances for cluster %s. Error: %s", logger.Underline(cluster), err.Error())
			continue
		}

		containerInstances[cluster] = containerInstancesOutput.ContainerInstanceArns
	}

	for cluster, containerInstances := range containerInstances {
		for _, containerInstance := range containerInstances {
			_, err := ecsHandler.UpdateContainerAgent(ctx, &ecsLib.UpdateContainerAgentInput{
				Cluster:           aws.String(cluster),
				ContainerInstance: aws.String(containerInstance),
			})
			if err != nil {
				logger.Error("Unable to update container agent for cluster %s and container instance %s. Error: %s", logger.Underline(cluster), logger.Underline(containerInstance), err.Error())
				continue
			}
		}
	}

	return nil
}

func Revert(
	ctx context.Context,
	cfg aws.Config,
	cluster,
	service,
	tagToRevertTo string,
	revisionsToLookback int32,
) error {
	if revisionsToLookback > 50 {
		return errors.New("please limit your lookback to 50")
	}

	clusters, err := ListClusters(ctx, cfg, cluster)
	if err != nil {
		return err
	}

	for i, cluster := range *clusters {
		err = cluster.GetServices(ctx, cfg, service)
		if err != nil {
			return err
		}

		for j, service := range cluster.Services {
			service.GetTaskDefintions(ctx, cfg, revisionsToLookback)

			for _, taskDefinition := range service.TaskDefinitions {
				if strings.Contains(taskDefinition.Image, tagToRevertTo) {
					service.TaskDefinitionArn = taskDefinition.GetNameWithVersion()
					service.CanRevert = true
					break
				}
			}

			cluster.Services[j] = service
		}

		(*clusters)[i] = cluster
	}

	revertServices(ctx, cfg, clusters, tagToRevertTo, revisionsToLookback, true)

	shouldDo := logger.InfoScan("Choose y/n: ")
	if shouldDo != "y" {
		logger.Success("Nothing to do")
		return nil
	}

	revertServices(ctx, cfg, clusters, tagToRevertTo, revisionsToLookback, false)

	return nil
}

func revertServices(
	ctx context.Context,
	cfg aws.Config,
	clusters *[]Cluster,
	tagToRevertTo string,
	revisionsToLookback int32,
	dryRun bool,
) {
	ecsHandler := ecsLib.NewFromConfig(cfg)
	for _, cluster := range *clusters {
		for _, service := range cluster.Services {
			if !service.CanRevert {
				logger.Warn(
					"Tag %s not found in past %s revisions, wont revert %s/%s",
					logger.Bold(tagToRevertTo),
					logger.Underline(revisionsToLookback),
					logger.Italic(cluster.Name),
					logger.Italic(service.Name),
				)
				continue
			}

			if dryRun {
				logger.Info(
					"Will update %s/%s from %s -> %s",
					logger.Underline(cluster.Name),
					logger.Underline(service.Name),
					logger.Italic(service.OldTaskDefinitionArn),
					logger.Bold(service.TaskDefinitionArn),
				)
				continue
			}

			_, err := ecsHandler.UpdateService(ctx, &ecsLib.UpdateServiceInput{
				Cluster:            aws.String(cluster.Name),
				Service:            aws.String(service.Name),
				TaskDefinition:     aws.String(service.TaskDefinitionArn),
				ForceNewDeployment: true,
			})
			if err != nil {
				logger.Error(
					"Unable to update %s/%s from %s -> %s. Error: %s",
					logger.Underline(cluster.Name),
					logger.Underline(service.Name),
					logger.Italic(service.OldTaskDefinitionArn),
					logger.Bold(service.TaskDefinitionArn),
					err.Error(),
				)
				continue
			}

			logger.Success(
				"Updated %s/%s from %s -> %s",
				logger.Underline(cluster.Name),
				logger.Underline(service.Name),
				logger.Italic(service.OldTaskDefinitionArn),
				logger.Bold(service.TaskDefinitionArn),
			)
		}
	}
}
