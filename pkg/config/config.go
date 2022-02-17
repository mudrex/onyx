package config

import (
	"encoding/json"
	"fmt"

	"github.com/mudrex/onyx/pkg/logger"
	"github.com/mudrex/onyx/pkg/utils"
)

type C struct {
	Region    string `json:"region"`
	SlackHook string `json:"slack_hook"`
}

var Config C

var Filename = ".onyx.json"

func Default() *C {
	return &C{
		Region: "us-east-1",
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
	case "slack_hook":
		loadedConfig.SlackHook = value
	default:
		return fmt.Errorf("unrecognized key %s", logger.Underline(key))
	}

	finalConfig, err := json.Marshal(loadedConfig)
	if err != nil {
		return err
	}

	return utils.CreateFileWithData(Filename, string(finalConfig))
}
