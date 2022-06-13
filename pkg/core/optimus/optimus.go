package optimus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/mudrex/onyx/pkg/config"
	"github.com/mudrex/onyx/pkg/filesystem"
	"github.com/mudrex/onyx/pkg/logger"
	"github.com/mudrex/onyx/pkg/utils"
)

type OptimusSecret struct {
	Username string `json:"username"`
	Token    string `json:"token"`
	Host     string `json:"host"`
}

type Config map[string][]string

type ConfigLock struct {
	Checksum     string `json:"checksum"`
	LockedConfig Config `json:"locked_config"`
}

var optimusSecret = OptimusSecret{}

func RefreshUsers(ctx context.Context, cfg aws.Config) error {
	return refresh(ctx, cfg, config.Config.OptimusUsersConfig, "users")
}

func RefreshRoles(ctx context.Context, cfg aws.Config) error {
	return refresh(ctx, cfg, config.Config.OptimusRolesConfig, "roles")
}

// func RefreshJobs(ctx context.Context, cfg aws.Config) error {
// 	return refresh(ctx, cfg, config.Config.OptimusJobsConfig, "jobs")
// }

func refresh(ctx context.Context, cfg aws.Config, accessConfig string, flag string) error {
	configData, err := filesystem.ReadFile(accessConfig)
	if err != nil {
		return err
	}

	var loadedConfig Config

	err = json.Unmarshal([]byte(configData), &loadedConfig)
	if err != nil {
		return err
	}

	// Load lock file
	var configLock ConfigLock
	if filesystem.FileExists(accessConfig + ".lock") {
		configLockData, err := filesystem.ReadFile(accessConfig + ".lock")
		if err != nil {
			return err
		}

		err = json.Unmarshal([]byte(configLockData), &configLock)
		if err != nil {
			return err
		}
	}

	// Verify checksum to prevent extra work
	if utils.GetSHA512Checksum([]byte(configData)) == configLock.Checksum {
		logger.Info("Config is upto date! There is nothing to do!")
		return nil
	}

	if len(config.Config.OptimusSecretName) == 0 {
		return fmt.Errorf("optimus secret name not specified")
	}

	// secretString := secretsmanager.GetSecret(ctx, cfg, config.Config.OptimusSecretName)
	// err = json.Unmarshal([]byte(secretString), &optimusSecret)
	// if err != nil {
	// 	return err
	// }

	switch flag {
	case "users":
		err = refreshUsers(ctx, cfg, loadedConfig, configLock.LockedConfig, optimusSecret)
		if err != nil {
			return err
		}

	case "roles":
		refreshRoles(ctx, cfg, loadedConfig, configLock.LockedConfig, optimusSecret)
		// case "jobs":
		// 	refreshJobs(ctx, cfg, loadedConfig, configLock.LockedConfig, optimusSecret)
	}

	loadedConfigBytes, err := json.MarshalIndent(loadedConfig, "", "    ")
	if err != nil {
		logger.Error("Unable to update config file")
		return err
	}

	filesystem.CreateFileWithData(accessConfig, string(loadedConfigBytes))

	configLock.LockedConfig = loadedConfig
	configLock.Checksum = utils.GetSHA512Checksum(loadedConfigBytes)

	loadedConfigLockBytes, err := json.MarshalIndent(configLock, "", "    ")
	if err != nil {
		logger.Error("Unable to update config file")
		return err
	}

	return filesystem.CreateFileWithData(accessConfig+".lock", string(loadedConfigLockBytes))
}

// func refreshJobs(
// 	ctx context.Context,
// 	cfg aws.Config,
// 	currConfig Config,
// 	lockedConfig Config,
// 	secret OptimusSecret,
// ) error {
// 	fmt.Print("step1")
// 	// getDiff(currConfig, lockedConfig)
// 	// fmt.Print("step1 done")
// 	// fmt.Print("%+v\n", currConfig)
// 	// fmt.Print("%+v\n", lockedConfig)

// 	return nil
// }

func refreshUsers(
	ctx context.Context,
	cfg aws.Config,
	currConfig Config,
	lockedConfig Config,
	secret OptimusSecret,
) error {
	errAdd := addUsers(getDiff(currConfig, lockedConfig), secret)
	if errAdd != nil {
		logger.Error("Unable to add roles to users")
		return errAdd
	}

	errRemove := removeUsers(getDiff(lockedConfig, currConfig), secret)
	if errRemove != nil {
		logger.Error("Unable to remove roles from users")
		return errRemove
	}
	return nil
}

func addUsers(add Config, secret OptimusSecret) error {
	fmt.Println("add users", add)
	for username, roles := range add {
		for _, role := range roles {
			return sendRoleRequest("/role-strategy/strategy/assignRole", username, role, secret)
		}
	}
	return nil
}

func removeUsers(remove Config, secret OptimusSecret) error {
	for username, roles := range remove {
		for _, role := range roles {
			return sendRoleRequest("/role-strategy/strategy/unassignRole", username, role, secret)
		}
	}
	return nil
}

func sendRoleRequest(uri string, username string, role string, secret OptimusSecret) error {
	sid, err := utils.GetSidFromUsername(username)
	if err != nil {
		logger.Error("Unable to get sid from username %s", username)
		return err
	}

	// Sanitize URL
	url, err := url.Parse(secret.Host + uri)
	if err != nil {
		logger.Error("Unable to create url with host %s and uri %s", secret.Host, uri)
		return err
	}

	// Create HTTP Request
	client := &http.Client{}
	request, err := http.NewRequest("POST", url.String(), nil)
	if err != nil {
		logger.Error("Unable to create request with url %s", url.String())
		return err
	}
	request.SetBasicAuth(secret.Username, secret.Token)
	query := request.URL.Query()
	query.Add("type", "globalRoles")
	query.Add("roleName", role)
	query.Add("sid", sid)
	request.URL.RawQuery = query.Encode()
	logger.Info("(%s) Running for username: %s, sid: %s | %s", role, username, sid, request.URL.String())

	// Send HTTP request
	response, err := client.Do(request)

	if err != nil {
		logger.Error("Unable to add/remove role %s to/from user %s", role, username)
		return err
	}

	// Check for response code other than 200
	if response.StatusCode != 200 {
		logger.Error("Request failed with status code %s", response.Status)
		return errors.New("request failed with status code %s")
	}

	return nil
}

func refreshRoles(
	ctx context.Context,
	cfg aws.Config,
	currConfig Config,
	lockedConfig Config,
	secret OptimusSecret,
) error {
	logger.Info("Coming Soon!")
	return nil
}

func getDiff(config, configLock Config) Config {
	var diff Config = make(Config)
	fmt.Println("before diff ", diff)

	for username, roles := range config {
		fmt.Println("config lock", username, configLock[username])
		intersection := utils.GetIntersectionBetweenStringArrays(roles, configLock[username])
		diff[username] = utils.GetDifferenceBetweenStringArrays(roles, intersection)
		print(diff[username])
	}

	fmt.Println("after diff ", diff)

	return diff
}
