package config

import (
	"encoding/json"
	"fmt"

	"github.com/mudrex/onyx/pkg/logger"
	"github.com/mudrex/onyx/pkg/utils"
)

type C struct {
	Region                  string `json:"region"`
	Environment             string `json:"environment"`
	SlackHook               string `json:"slack_hook"`
	VPCCidr                 string `json:"vpc_cidr"`
	PrivateKey              string `json:"private_key"`
	HostsAccessConfig       string `json:"hosts_access_config"`
	ServicesAccessConfig    string `json:"services_access_config"`
	RDSAccessConfig         string `json:"rds_access_config"`
	RDSServicesAccessConfig string `json:"rds_services_access_config"`
	RDSCriticalTablesConfig string `json:"rds_critical_tables_config"`
	RDSSecretName           string `json:"rds_secret_name"`
}

var Config C

var Filename = ".onyx.json"

func Default() *C {
	return &C{
		Region:               "us-east-1",
		PrivateKey:           "/opt/gatekeeper/keys/services.pem",
		VPCCidr:              "10.10.0.0/16",
		HostsAccessConfig:    "/opt/gatekeeper/hosts-access.json",
		ServicesAccessConfig: "/opt/gatekeeper/services-access.json",
		Environment:          "staging",
	}
}

func (c *C) ToString() string {
	dataByte, err := json.Marshal(c)
	if err != nil {
		return ""
	}

	return string(dataByte)
}

func LoadConfig() error {
	if !utils.FileExists(Filename) {
		logger.Fatal("%s doesn't exist. Please create one with %s", logger.Bold(Filename), logger.Underline("onyx init"))
	}

	data, err := utils.ReadFile(Filename)
	if err != nil {
		return err
	}

	json.Unmarshal([]byte(data), &Config)
	return nil
}

func GetRegion() string {
	if Config.Region == "" {
		return "us-east-1"
	}

	return Config.Region
}

func SetConfigKey(key, value string) error {
	configData, err := utils.ReadFile(Filename)
	if err != nil {
		return err
	}

	var loadedConfig C

	err = json.Unmarshal([]byte(configData), &loadedConfig)
	if err != nil {
		return err
	}

	switch key {

	case "region":
		loadedConfig.Region = value
	case "environment":
		loadedConfig.Environment = value
	case "slack_hook":
		loadedConfig.SlackHook = value
	case "vpc_cidr":
		loadedConfig.VPCCidr = value
	case "private_key":
		loadedConfig.PrivateKey = value
	case "hosts_access_config":
		loadedConfig.HostsAccessConfig = value
	case "services_access_config":
		loadedConfig.ServicesAccessConfig = value
	case "rds_access_config":
		loadedConfig.RDSAccessConfig = value
	case "rds_secret_name":
		loadedConfig.RDSSecretName = value
	case "rds_services_access_config":
		loadedConfig.RDSServicesAccessConfig = value
	case "rds_critical_tables_config":
		loadedConfig.RDSCriticalTablesConfig = value
	default:
		return fmt.Errorf("unrecognized key %s", logger.Underline(key))
	}

	finalConfig, err := json.Marshal(loadedConfig)
	if err != nil {
		return err
	}

	return utils.CreateFileWithData(Filename, string(finalConfig))
}
