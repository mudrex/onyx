package ecs

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
	"syscall"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/mudrex/onyx/pkg/auth"
	"github.com/mudrex/onyx/pkg/config"
	"github.com/mudrex/onyx/pkg/logger"
	"github.com/mudrex/onyx/pkg/notifier"
	"github.com/mudrex/onyx/pkg/utils"
)

func spawnRemoteDockerContainerShell(ctx context.Context, host, serviceName, shell string) error {
	userUID := syscall.Getuid()

	err := syscall.Setuid(0)
	if err != nil {
		return err
	}

	cmd := fmt.Sprintf("sudo ssh -i %s -o StrictHostKeyChecking=no ec2-user@%s 'docker ps | grep \"%s\" | cut -d\" \" -f1 | head -1'", config.Config.PrivateKey, host, serviceName)
	out, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return err
	}

	containerID := strings.Trim(string(out), "\n")

	logger.Info("Spawning shell for %s on instance %s", logger.Underline(serviceName), host)

	notifier.Notify(
		config.Config.SlackHook,
		fmt.Sprintf("[ecs/spawn-shell] *%s* logged in to _%s_ for %s", utils.GetUser(), host, serviceName),
	)

	sshCmdDockerShell := fmt.Sprintf("sudo ssh -t -i %s -o StrictHostKeyChecking=no ec2-user@%s 'docker exec -it %s %s'", config.Config.PrivateKey, host, containerID, shell)
	out1 := exec.Command("bash", "-c", sshCmdDockerShell)
	out1.Stdin = os.Stdin
	out1.Stdout = os.Stdout
	out1.Stderr = os.Stderr
	out1.Run()

	syscall.Setuid(userUID)
	logger.Success("Exiting safely")

	return nil
}

func getServicesHosts(ctx context.Context, cfg aws.Config, serviceName, clusterName string) (map[string][]string, error) {
	servicesHosts := make(map[string][]string)
	clusters, err := ListClusters(ctx, cfg, clusterName)
	if err != nil {
		return servicesHosts, err
	}

	cluster, err := DescribeByCluster(ctx, cfg, (*clusters)[0].Name, serviceName)
	if err != nil {
		return servicesHosts, err
	}

	for _, service := range cluster.Services {
		if strings.Contains(service.Name, serviceName) {
			hosts := make([]string, 0)
			for _, task := range service.Tasks {
				hosts = append(hosts, task.ContainerInstance.Instance.PrivateIPv4)
			}

			servicesHosts[service.Name] = hosts
		}
	}

	for service, hosts := range servicesHosts {
		if len(hosts) == 0 {
			delete(servicesHosts, service)
		}
	}

	return servicesHosts, nil
}

func SpawnServiceShell(ctx context.Context, cfg aws.Config, serviceName, clusterName, shell string) error {
	if !strings.Contains(clusterName, config.Config.Environment) {
		logger.Error("You are in %s environment but you are trying to access %s environment", logger.Underline(config.Config.Environment), clusterName)
		return nil
	}

	currUser, err := user.Current()
	if err != nil {
		return err
	}

	username := currUser.Username

	isAuthorized, err := auth.CheckUserAccessForService(ctx, username, serviceName)
	if err != nil {
		return err
	}

	if !isAuthorized {
		logger.Error("%s is not authorized to access %s", logger.Underline(username), logger.Bold(serviceName))
		return nil
	}

	servicesHosts, err := getServicesHosts(ctx, cfg, serviceName, clusterName)
	if err != nil {
		return err
	}

	if len(servicesHosts) == 0 {
		return fmt.Errorf("no service %s found", logger.Underline(serviceName))
	}

	if len(servicesHosts) == 1 {
		for _, hosts := range servicesHosts {
			return spawnRemoteDockerContainerShell(ctx, hosts[0], serviceName, shell)
		}
	}

	host, err := getServiceHost(servicesHosts)
	if err != nil {
		return err
	}

	return spawnRemoteDockerContainerShell(ctx, host, serviceName, shell)
}

func getServiceHost(servicesHosts map[string][]string) (string, error) {
	services := make([]string, 0)
	for service := range servicesHosts {
		services = append(services, service)
	}

	logger.Info("Select service to connect to")
	for i, service := range services {
		fmt.Println(logger.Bold(i), ":", service)
	}

	choices := utils.GetUserInput("Enter Choice: ")

	if len(choices) == 0 {
		return "", errors.New(logger.Bold("Invalid choice"))
	}

	for _, index := range strings.Split(choices, ",") {
		i, _ := strconv.ParseInt(strings.TrimSpace(index), 0, 32)
		if i >= 0 && i < int64(len(services)) {
			return servicesHosts[services[i]][0], nil
		}
	}

	return "", nil
}

func tailContainerLogs(ctx context.Context, host, serviceName string, tailLogs int32) error {
	userUID := syscall.Getuid()

	err := syscall.Setuid(0)
	if err != nil {
		return err
	}

	// TODO: disable StrictHostKeyChecking
	cmd := fmt.Sprintf("sudo ssh -i %s -o StrictHostKeyChecking=no ec2-user@%s 'docker ps --format '{{.Names}}' | grep \"%s\" | cut -d\" \" -f1 | head -1'", config.Config.PrivateKey, host, strings.ReplaceAll(serviceName, "_", "-"))
	out, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return err
	}

	containerID := strings.Trim(string(out), "\n")

	logger.Info("Spawning shell for %s on instance %s", logger.Underline(serviceName), host)

	notifier.Notify(
		config.Config.SlackHook,
		fmt.Sprintf("[ecs/tail-logs] *%s* tailed logs for %s on _%s_", utils.GetUser(), serviceName, host),
	)

	sshCmdDockerShell := fmt.Sprintf("sudo ssh -t -i %s -o StrictHostKeyChecking=no ec2-user@%s 'docker logs -f %s --tail %d'", config.Config.PrivateKey, host, containerID, tailLogs)
	out1 := exec.Command("bash", "-c", sshCmdDockerShell)
	out1.Stdin = os.Stdin
	out1.Stdout = os.Stdout
	out1.Stderr = os.Stderr
	out1.Run()

	syscall.Setuid(userUID)
	logger.Success("Exiting safely")

	return nil
}

func TailContainerLogs(ctx context.Context, cfg aws.Config, serviceName, clusterName string, tailLogs int32) error {
	if !strings.Contains(clusterName, config.Config.Environment) {
		logger.Error("You are in %s environment but you are trying to access %s environment", logger.Underline(config.Config.Environment), clusterName)
		return nil
	}

	currUser, err := user.Current()
	if err != nil {
		return err
	}

	username := currUser.Username

	isAuthorized, err := auth.CheckUserAccessForService(ctx, username, serviceName)
	if err != nil {
		return err
	}

	if !isAuthorized {
		logger.Error("%s is not authorized to access %s", logger.Underline(username), logger.Bold(serviceName))
		return nil
	}

	servicesHosts, err := getServicesHosts(ctx, cfg, serviceName, clusterName)
	if err != nil {
		return err
	}

	if len(servicesHosts) == 0 {
		return fmt.Errorf("no service %s found", logger.Underline(serviceName))
	}

	if len(servicesHosts) == 1 {
		for _, hosts := range servicesHosts {
			return tailContainerLogs(ctx, hosts[0], serviceName, tailLogs)
		}
	}

	host, err := getServiceHost(servicesHosts)
	if err != nil {
		return err
	}

	return tailContainerLogs(ctx, host, serviceName, tailLogs)
}

func ListAccess(ctx context.Context) error {
	currUser, err := user.Current()
	if err != nil {
		return err
	}

	username := currUser.Username

	auth.GetListOfAllowedServices(username)
	return nil
}
