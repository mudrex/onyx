package config

import (
	"encoding/json"

	"github.com/mudrex/onyx/pkg/utils"
)

type C struct {
	Region    string `json:"region"`
	SlackHook string `json:"slack_hook"`
}

var Config C

var Filename = ".onyx.json"

func Default() C {
	return C{
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
