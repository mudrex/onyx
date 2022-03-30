package auth

import (
	"context"
	"encoding/json"
	"os"
	"strings"

	"github.com/mudrex/onyx/pkg/config"
	"github.com/mudrex/onyx/pkg/logger"
)

var servicesAccessList = make(map[string][]string)
var hostAccessList = make(map[string]map[string]bool)

func printAllowedServices() {
	services := make([]string, len(servicesAccessList))

	i := 0
	for service := range servicesAccessList {
		services[i] = logger.Bold(service)
		i++
	}

	logger.Info("Allowed services: %s", strings.Join(services, ", "))
}

func CheckUserAccessForService(ctx context.Context, username, serviceName string) (bool, error) {
	file, _ := os.Open(config.Config.ServicesAccessConfig)
	defer file.Close()
	decoder := json.NewDecoder(file)
	err := decoder.Decode(&servicesAccessList)
	if err != nil {
		panic(err)
	}

	if _, ok := servicesAccessList[serviceName]; !ok {
		logger.Error(
			"%s doesn't exist in service list, please get it added in %s in keyhouse repository if required.",
			logger.Bold(serviceName),
			logger.Underline("<env>/entry-server/services-access.json"),
		)

		printAllowedServices()
		return false, nil
	}

	for service, users := range servicesAccessList {
		if strings.Contains(serviceName, service) {
			for _, user := range users {
				if username == user {
					return true, nil
				}
			}
		}
	}

	return false, nil
}

func CheckUserAccessForHostShell(ctx context.Context, username, host string) (bool, error) {
	file, _ := os.Open(config.Config.HostsAccessConfig)
	defer file.Close()
	decoder := json.NewDecoder(file)
	err := decoder.Decode(&hostAccessList)
	if err != nil {
		panic(err)
	}

	if _, ok := hostAccessList[host]; !ok {
		logger.Warn("%s doesn't exist in allowed list. Please get it added.", host)

		// TODO: temp allow access to all hosts except a few critical
		return true, nil
	}

	if allAccess, ok := hostAccessList[host]["all"]; ok {
		// if this host has access for "all" enabled, allow for everyone
		if allAccess {
			return true, nil
		}

		if access, ok := hostAccessList[host][username]; ok {
			return access, nil
		}

		return false, nil
	}

	if access, ok := hostAccessList[host][username]; ok {
		return access, nil
	}

	return false, nil
}
