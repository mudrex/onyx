package ecs

import (
	"strings"
)

func extractServiceNameFromServiceArns(clusterName string, serviceArns []string) (services []string) {
	for _, serviceArn := range serviceArns {
		a := strings.Split(serviceArn, "/")
		services = append(services, a[len(a)-1])
	}

	return
}
