package auth

import (
	"context"
	"encoding/json"
	"os"
	"os/user"
	"strings"
)

var accessList = make(map[string][]string)

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
