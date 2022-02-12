package auth

import (
	"context"
	"encoding/json"
	"os"
	"os/user"
	"strings"
)

var accessList = make(map[string][]string)
var hostAccessList = make(map[string]map[string]bool)

func CheckUserAccessForService(ctx context.Context, serviceName string) (string, bool, error) {
	currUser, err := user.Current()
	if err != nil {
		return "", false, err
	}

	file, _ := os.Open("/opt/gatekeeper/services-access.json")
	defer file.Close()
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&accessList)
	if err != nil {
		panic(err)
	}

	for service, users := range accessList {
		if strings.Contains(serviceName, service) {
			for _, user := range users {
				if currUser.Username == user {
					return currUser.Username, true, nil
				}
			}
		}
	}

	return currUser.Username, false, nil
}

func CheckUserAccessForHostShell(ctx context.Context, host string) (string, bool, error) {
	currUser, err := user.Current()
	if err != nil {
		return "", false, err
	}

	file, _ := os.Open("/opt/gatekeeper/hosts-access.json")
	defer file.Close()
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&hostAccessList)
	if err != nil {
		panic(err)
	}

	// if host is not listed in config, allow access to everyone
	if _, ok := hostAccessList[host]; !ok {
		return currUser.Username, true, nil
	}

	if allAccess, ok := hostAccessList[host]["all"]; ok {
		// if this host has access for "all" enabled, allow for everyone
		if allAccess {
			return currUser.Username, true, nil
		}

		if access, ok := hostAccessList[host][currUser.Username]; ok {
			return currUser.Username, access, nil
		}

		return currUser.Username, false, nil
	}

	if access, ok := hostAccessList[host][currUser.Username]; ok {
		return currUser.Username, access, nil
	}

	return currUser.Username, true, nil
}
