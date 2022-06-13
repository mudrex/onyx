package optimus

import (
	"context"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/mudrex/onyx/pkg/config"
	"github.com/mudrex/onyx/pkg/filesystem"
	"github.com/mudrex/onyx/pkg/logger"
	"github.com/mudrex/onyx/pkg/utils"
)

type Job struct {
	Name   string `json:"name"`
	Config string `json:"config"`
}

type JobConfig map[string][]Job

type JobConfigLock struct {
	Checksum     string    `json:"checksum"`
	LockedConfig JobConfig `json:"job_config"`
}

func RefreshJobs(ctx context.Context, cfg aws.Config) error {
	return refreshConfig(ctx, cfg, config.Config.OptimusJobsConfig, "jobs")
}

func refreshConfig(ctx context.Context, cfg aws.Config, accessConfig string, flag string) error {
	configData, err := filesystem.ReadFile(accessConfig)
	if err != nil {
		return err
	}

	var loadedJobConfig JobConfig

	err = json.Unmarshal([]byte(configData), &loadedJobConfig)
	if err != nil {
		return err
	}

	// Load lock file
	var jobConfigLock JobConfigLock
	if filesystem.FileExists(accessConfig + ".lock") {
		configLockData, err := filesystem.ReadFile(accessConfig + ".lock")
		if err != nil {
			return err
		}

		err = json.Unmarshal([]byte(configLockData), &jobConfigLock)
		if err != nil {
			return err
		}
	}

	// fmt.Println(jobConfigLock)
	// fmt.Println(loadedJobConfig)

	// Verify checksum to prevent extra work
	if utils.GetSHA512Checksum([]byte(configData)) == jobConfigLock.Checksum {
		logger.Info("Config is upto date! There is nothing to do!")
		// return nil
	}

	if len(config.Config.OptimusSecretName) == 0 {
		return fmt.Errorf("optimus secret name not specified")
	}

	// secretString := secretsmanager.GetSecret(ctx, cfg, config.Config.OptimusSecretName)
	// err = json.Unmarshal([]byte(secretString), &optimusSecret)
	// if err != nil {
	// 	return err
	// }

	err = refreshJobs(ctx, cfg, loadedJobConfig, jobConfigLock.LockedConfig, optimusSecret)
	if err != nil {
		return err
	}

	loadedConfigBytes, err := json.MarshalIndent(loadedJobConfig, "", "    ")
	if err != nil {
		logger.Error("Unable to update config file")
		return err
	}

	filesystem.CreateFileWithData(accessConfig, string(loadedConfigBytes))

	jobConfigLock.LockedConfig = loadedJobConfig
	jobConfigLock.Checksum = utils.GetSHA512Checksum(loadedConfigBytes)

	loadedConfigLockBytes, err := json.MarshalIndent(jobConfigLock, "", "    ")
	if err != nil {
		logger.Error("Unable to update config file")
		return err
	}

	return filesystem.CreateFileWithData(accessConfig+".lock", string(loadedConfigLockBytes))
}

func refreshJobs(
	ctx context.Context,
	cfg aws.Config,
	currConfig JobConfig,
	lockedConfig JobConfig,
	secret OptimusSecret,
) error {
	fmt.Println("step1")
	// fmt.Println("Current Config", currConfig)
	// fmt.Println("Locked Config", lockedConfig)
	currConfigArray, err := getB64StringArray(currConfig) // re-using Config struct since the difference would be similar
	if err != nil {
		return err
	}

	fmt.Println(currConfigArray)

	// var lock Config
	// fmt.Println(getDiff(currConfig, lockedConfig), secret)

	// for _, job := range currConfig["jobs"] {

	// 	j, err := json.Marshal(job)
	// 	sEnc := b64.StdEncoding.EncodeToString(j)

	// 	b64array = append(b64array, sEnc)
	// 	// fmt.Printf("%T", j)

	// 	if err != nil {
	// 		return fmt.Errorf("unable to update jobs config file")
	// 	}

	// 	// print(j)
	// }
	// errAdd := addJobs(getDiff(currConfig, lockedConfig), secret)
	// if errAdd != nil {
	// 	logger.Error("Unable to add jobs to optimus")
	// 	return errAdd
	// }

	// errRemove := removeJobs(getDiff(lockedConfig, currConfig), secret)
	// if errRemove != nil {
	// 	logger.Error("Unable to remove roles from users")
	// 	return errRemove
	// }

	// getDiff(currConfig, lockedConfig)
	// fmt.Print("step1 done")
	// fmt.Print("%+v\n", currConfig)
	// fmt.Print("%+v\n", lockedConfig)
	return nil
}

func getB64StringArray(config JobConfig) ([]string, error) {

	var array []string
	for _, job := range config["jobs"] {

		j, err := json.Marshal(job)
		if err != nil {
			return []string{""}, err
		}
		sEnc := b64.StdEncoding.EncodeToString(j)

		array = append(array, sEnc)
		// print(j)
	}
	// fmt.Println(array)
	return array, nil
}

// func addJobs(add Job, secret) error {

// }

// func refreshUsers(
// 	ctx context.Context,
// 	cfg aws.Config,
// 	currConfig Config,
// 	lockedConfig Config,
// 	secret OptimusSecret,
// ) error {
// 	errAdd := addUsers(getDiff(currConfig, lockedConfig), secret)
// 	if errAdd != nil {
// 		logger.Error("Unable to add roles to users")
// 		return errAdd
// 	}

// 	errRemove := removeUsers(getDiff(lockedConfig, currConfig), secret)
// 	if errRemove != nil {
// 		logger.Error("Unable to remove roles from users")
// 		return errRemove
// 	}
// 	return nil
// }

// func addUsers(add Config, secret OptimusSecret) error {
// 	for username, roles := range add {
// 		for _, role := range roles {
// 			return sendRoleRequest("/role-strategy/strategy/assignRole", username, role, secret)
// 		}
// 	}
// 	return nil
// }

// func removeUsers(remove Config, secret OptimusSecret) error {
// 	for username, roles := range remove {
// 		for _, role := range roles {
// 			return sendRoleRequest("/role-strategy/strategy/unassignRole", username, role, secret)
// 		}
// 	}
// 	return nil
// }

// func sendRoleRequest(uri string, username string, role string, secret OptimusSecret) error {
// 	sid, err := utils.GetSidFromUsername(username)
// 	if err != nil {
// 		logger.Error("Unable to get sid from username %s", username)
// 		return err
// 	}

// 	// Sanitize URL
// 	url, err := url.Parse(secret.Host + uri)
// 	if err != nil {
// 		logger.Error("Unable to create url with host %s and uri %s", secret.Host, uri)
// 		return err
// 	}

// 	// Create HTTP Request
// 	client := &http.Client{}
// 	request, err := http.NewRequest("POST", url.String(), nil)
// 	if err != nil {
// 		logger.Error("Unable to create request with url %s", url.String())
// 		return err
// 	}
// 	request.SetBasicAuth(secret.Username, secret.Token)
// 	query := request.URL.Query()
// 	query.Add("type", "globalRoles")
// 	query.Add("roleName", role)
// 	query.Add("sid", sid)
// 	request.URL.RawQuery = query.Encode()
// 	logger.Info("(%s) Running for username: %s, sid: %s | %s", role, username, sid, request.URL.String())

// 	// Send HTTP request
// 	response, err := client.Do(request)

// 	if err != nil {
// 		logger.Error("Unable to add/remove role %s to/from user %s", role, username)
// 		return err
// 	}

// 	// Check for response code other than 200
// 	if response.StatusCode != 200 {
// 		logger.Error("Request failed with status code %s", response.Status)
// 		return errors.New("request failed with status code %s")
// 	}

// 	return nil
// }

// func refreshRoles(
// 	ctx context.Context,
// 	cfg aws.Config,
// 	currConfig Config,
// 	lockedConfig Config,
// 	secret OptimusSecret,
// ) error {
// 	logger.Info("Coming Soon!")
// 	return nil
// }

// func getDiff(config, configLock Config) Config {
// 	var diff Config = make(Config)
// 	fmt.Println(diff)

// 	for username, roles := range config {
// 		intersection := utils.GetIntersectionBetweenStringArrays(roles, configLock[username])
// 		diff[username] = utils.GetDifferenceBetweenStringArrays(roles, intersection)
// 	}

// 	fmt.Println("After diff", diff)

// 	return diff
// }
