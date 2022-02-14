package utils

import (
	"errors"
	"fmt"
	"os"
)

func configFilePath(configFile string) string {
	dirname, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	return fmt.Sprintf("%s/%s", dirname, configFile)
}

func ReadFile(filename string) (string, error) {
	dat, err := os.ReadFile(configFilePath(filename))
	if err != nil {
		return "", err
	}

	return string(dat), nil
}

func FileExists(filename string) bool {
	if _, err := os.Stat(configFilePath(filename)); errors.Is(err, os.ErrNotExist) {
		return false
	}

	return true
}

func CreateFileWithData(filename, data string) error {
	f, err := os.OpenFile(configFilePath(filename), os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	defer f.Close()

	_, err = f.WriteString(data)
	if err != nil {
		return err
	}

	return nil
}
