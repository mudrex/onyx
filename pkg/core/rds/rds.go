package rds

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/mudrex/onyx/pkg/config"
	"github.com/mudrex/onyx/pkg/core/secretsmanager"
	"github.com/mudrex/onyx/pkg/logger"
	"github.com/mudrex/onyx/pkg/utils"
)

type Table struct {
	Grants  []string `json:"grants"`
	Columns []string `json:"columns"`
	Applied bool     `json:"applied"`
}

type Database struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Host     string `json:"host"`
	DBName   string `json:"dbname"`
}

type Config map[string]map[string]Table

func RefreshAccess(ctx context.Context, cfg aws.Config) error {
	configData, err := utils.ReadFile(config.Config.RDSAccessConfig)
	if err != nil {
		return err
	}

	var loadedConfig Config

	err = json.Unmarshal([]byte(configData), &loadedConfig)
	if err != nil {
		return err
	}

	if len(config.Config.RDSSecretName) == 0 {
		return fmt.Errorf("RDS secret name not specified")
	}

	secretString := secretsmanager.GetSecret(ctx, cfg, config.Config.RDSSecretName)

	var secret Database

	err = json.Unmarshal([]byte(secretString), &secret)
	if err != nil {
		return err
	}

	for username, accessMap := range loadedConfig {
		queries := builtQueriesForUser(secret.DBName, username, accessMap)
		if len(queries) == 0 {
			logger.Success("Nothing to do for %s", username)
			continue
		}

		logger.Info("Running %s for %s", strings.Join(queries, ";"), username)

		c := fmt.Sprintf(
			"mysql -h %s -u %s -p'%s' -e \"%s;\"",
			secret.Host,
			secret.Username,
			secret.Password,
			strings.Join(queries, ";"),
		)

		var stdout bytes.Buffer
		var stderr bytes.Buffer
		cmd := exec.Command("bash", "-c", c)
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err := cmd.Run()

		fmt.Println(stdout.String())
		fmt.Println(stderr.String())

		if err != nil {
			return err
		}

		logger.Success("Successfully ran %s for %s", strings.Join(queries, ";"), username)

		for tableName, tableGrants := range accessMap {
			tableGrants.Applied = true
			accessMap[tableName] = tableGrants
		}
	}

	loadedConfigBytes, err := json.MarshalIndent(loadedConfig, "", "    ")
	if err != nil {
		logger.Error("Unable to update config file")
		return err
	}

	return utils.CreateFileWithData(config.Config.RDSAccessConfig, string(loadedConfigBytes))
}

func builtQueriesForUser(dbname string, username string, accessMap map[string]Table) []string {
	queries := make([]string, 0)

	for table, tableGrants := range accessMap {
		if !tableGrants.Applied {
			if len(tableGrants.Columns) > 0 {
				for _, grant := range tableGrants.Grants {
					queries = append(queries, fmt.Sprintf("GRANT %s (%s) on %s.%s to '%s'@'%%'", grant, strings.Join(tableGrants.Columns, ", "), dbname, table, username))
				}
			} else if len(tableGrants.Grants) > 1 {
				queries = append(queries, fmt.Sprintf("GRANT %s on %s.%s to '%s'@'%%'", strings.ToUpper(strings.Join(tableGrants.Grants, ", ")), dbname, table, username))
			} else {
				queries = append(queries, fmt.Sprintf("GRANT %s on %s.%s to '%s'@'%%'", strings.ToUpper(tableGrants.Grants[0]), dbname, table, username))
			}
		}
	}

	return queries
}
