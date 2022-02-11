package ecs

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/mudrex/onyx/pkg/logger"
	"github.com/mudrex/onyx/pkg/utils"
)

func spawnRemoteDockerContainerShell(ctx context.Context, host, serviceName string) error {
	userUID := syscall.Getuid()

	err := syscall.Setuid(0)
	if err != nil {
		return err
	}

	// TODO: disable StrictHostKeyChecking
	cmd := fmt.Sprintf("sudo ssh -i /opt/gatekeeper/keys/services.pem ec2-user@%s 'docker ps | grep \"%s\" | cut -d\" \" -f1 | head -1'", host, serviceName)
	out, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return err
	}

	containerID := strings.Trim(string(out), "\n")

	logger.Info("Spawning shell for %s on instance %s", logger.Underline(serviceName), host)

	sshCmdDockerShell := fmt.Sprintf("sudo ssh -t -i /opt/gatekeeper/keys/services.pem ec2-user@%s 'docker exec -it %s bash'", host, containerID)
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

func SpawnServiceShell(ctx context.Context, cfg aws.Config, serviceName, clusterName string) error {
	servicesHosts, err := getServicesHosts(ctx, cfg, serviceName, clusterName)
	if err != nil {
		return err
	}

	if len(servicesHosts) == 0 {
		return fmt.Errorf("no service %s found", logger.Underline(serviceName))
	}

	if len(servicesHosts) == 1 {
		for _, hosts := range servicesHosts {
			return spawnRemoteDockerContainerShell(ctx, hosts[0], serviceName)
		}
	}

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
		return errors.New(logger.Bold("Invalid choice"))
	}

	for _, index := range strings.Split(choices, ",") {
		i, _ := strconv.ParseInt(strings.TrimSpace(index), 0, 32)
		if i >= 0 && i < int64(len(services)) {
			return spawnRemoteDockerContainerShell(ctx, servicesHosts[services[i]][0], serviceName)
		}
	}

	return nil
}
