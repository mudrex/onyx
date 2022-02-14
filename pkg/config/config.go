package config

import (
	"encoding/json"
	"fmt"

	"github.com/mudrex/onyx/pkg/utils"
)

var Config map[string]string

var Filename = ".onyx.json"

func Default() string {
	return `{"region": "us-east-1"}`
}

func LoadConfig() error {
	data, err := utils.ReadFile(Filename)
	if err != nil {
		return err
	}

	json.Unmarshal([]byte(data), &Config)
	fmt.Println(Config)
	return nil
}

func GetRegion() string {
	if region, ok := Config["region"]; ok {
		return region
	}

	return "us-east-1"
}
