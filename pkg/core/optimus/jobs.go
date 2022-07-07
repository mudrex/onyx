package optimus

import (
	"bytes"
	"context"
	b64 "encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/mudrex/onyx/pkg/config"
	"github.com/mudrex/onyx/pkg/core/secretsmanager"
	"github.com/mudrex/onyx/pkg/filesystem"
	"github.com/mudrex/onyx/pkg/logger"
	"github.com/mudrex/onyx/pkg/utils"
)

type Job struct {
	Name   string `json:"name"`
	Config string `json:"config"`
	Repo   string `json:"repository"`
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

	// Verify checksum to prevent extra work
	if utils.GetSHA512Checksum([]byte(configData)) == jobConfigLock.Checksum {
		logger.Info("Config is upto date! There is nothing to do!")
		// return nil
	}

	if len(config.Config.OptimusSecretName) == 0 {
		return fmt.Errorf("optimus secret name not specified")
	}

	secretString := secretsmanager.GetSecret(ctx, cfg, config.Config.OptimusSecretName)
	err = json.Unmarshal([]byte(secretString), &optimusSecret)
	if err != nil {
		return err
	}

	err = refreshJobs(ctx, cfg, loadedJobConfig, jobConfigLock.LockedConfig, optimusSecret, accessConfig)
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
	accessConfig string,
) error {
	currConfigArray, err := getB64StringArray(currConfig) // re-using Config struct since the difference would be similar
	if err != nil {
		return err
	}

	lockedConfigArray, err := getB64StringArray(lockedConfig)
	if err != nil {
		return err
	}

	var jobDiff []string

	// functions to get array of jobs to add to optimus
	intersection := utils.GetIntersectionBetweenStringArrays(currConfigArray, lockedConfigArray)
	jobDiff = utils.GetDifferenceBetweenStringArrays(currConfigArray, intersection)

	errAdd := addJobs(jobDiff, secret, accessConfig)
	if errAdd != nil {
		logger.Error("Unable to add job")
		return errAdd
	}

	// functions to get array of jobs to delete from optimus
	intersection = utils.GetIntersectionBetweenStringArrays(lockedConfigArray, currConfigArray)
	jobDiff = utils.GetDifferenceBetweenStringArrays(lockedConfigArray, intersection)

	errRemove := removeJobs(jobDiff, secret, accessConfig)
	if errRemove != nil {
		logger.Error("Unable to remove job")
		return errRemove
	}

	return nil

}

func addJobs(add []string, secret OptimusSecret, accessConfig string) error {

	for _, b64strings := range add {

		sDec, _ := b64.StdEncoding.DecodeString(b64strings)
		var job Job
		err := json.Unmarshal(sDec, &job)
		if err != nil {
			logger.Error("Unable to marshall object")
			return err
		}

		errRequest := sendJobRequest(job, secret, accessConfig)
		if errRequest != nil {
			logger.Error("Unable to send request")
			return errRequest
		}

	}

	return nil
}

func removeJobs(remove []string, secret OptimusSecret, accessConfig string) error {
	client := &http.Client{}

	for _, b64strings := range remove {
		url := secret.Host
		sDec, _ := b64.StdEncoding.DecodeString(b64strings)
		var job Job
		err := json.Unmarshal(sDec, &job)
		if err != nil {
			logger.Error("Unable to marshall object")
			return err
		}

		route := strings.FieldsFunc(job.Name, Split)

		for i := 1; i < len(route); i++ {

			url = url + "/job/" + route[i]
		}
		request, err := http.NewRequest("DELETE", url+"/", nil)
		if err != nil {
			logger.Error("Unable to remove job")
			return err
		}
		request.SetBasicAuth(secret.Username, secret.Token)

		response, err := client.Do(request)
		if err != nil {
			return err
		}

		switch response.StatusCode {
		case 204:
			fmt.Println("[remove] ", response.StatusCode, job.Name+" removed successfully")
		default:
			fmt.Println("[remove] ", response.StatusCode, "Unable to remove "+job.Name)
			return errors.New("unable to remove job")
		}

	}
	return nil
}

func sendJobRequest(job Job, secret OptimusSecret, accessConfig string) error {
	route := strings.FieldsFunc(job.Name, Split)
	folderXMLLocation := strings.Replace(accessConfig, "jobs.json", "config/folder.xml", 1)
	folderXML, err := filesystem.ReadFile(folderXMLLocation)
	if err != nil {
		return err
	}
	jobsConfig := strings.Replace(accessConfig, "jobs.json", job.Config, 1)
	configXML, err := filesystem.ReadFile(jobsConfig)
	if err != nil {
		return err
	}

	configXML = strings.Replace(configXML, "_INSERT_PIPELINE_NAME_HERE_", route[len(route)-1], 1)
	configXML = strings.Replace(configXML, "_INSERT_REPO_NAME_HERE_", job.Repo, 1)

	url := secret.Host
	client := &http.Client{}

	for i := 1; i < len(route); i++ {

		url = url + "/createItem"
		var xml_payload = new(bytes.Buffer)
		if i == len(route)-1 {
			xml_payload = bytes.NewBuffer([]byte(configXML))
		} else {
			xml_payload = bytes.NewBuffer([]byte(folderXML))
		}
		request, err := http.NewRequest("POST", url, xml_payload)
		if err != nil {
			logger.Error("Unable to create url")
			return err
		}
		query := request.URL.Query()
		query.Add("name", route[i])
		request.URL.RawQuery = query.Encode()
		request.Header.Set("Content-Type", "application/xml")
		request.SetBasicAuth(secret.Username, secret.Token)
		response, err := client.Do(request)
		if err != nil {
			return err
		}

		if i == len(route)-1 {
			switch response.StatusCode {
			case 200:
				fmt.Println("[add] ", response.StatusCode, job.Name+" added successfully")
			case 400:
				fmt.Println("[add] ", response.StatusCode, job.Name+" already exists")
			default:
				fmt.Println("[add]", response.StatusCode, "Error adding "+job.Name)
				return errors.New("unable to add job")
			}
		}

		url = strings.Replace(url, "/createItem", "/job/"+route[i], 1)
	}
	return nil

}

func Split(r rune) bool {
	return r == '/'
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
	}
	return array, nil
}
