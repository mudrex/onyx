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

type Database struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Host     string `json:"host"`
	DBName   string `json:"dbname"`
}

type Config map[string]map[string]map[string][]string

type ConfigLock struct {
	Checksum     string `json:"checksum"`
	LockedConfig Config `json:"locked_config"`
}

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

	var configLock ConfigLock
	if utils.FileExists(config.Config.RDSAccessConfig + ".lock") {
		configLockData, err := utils.ReadFile(config.Config.RDSAccessConfig + ".lock")
		if err != nil {
			return err
		}

		err = json.Unmarshal([]byte(configLockData), &configLock)
		if err != nil {
			return err
		}
	}

	if utils.GetSHA512Checksum([]byte(configData)) == configLock.Checksum {
		logger.Info("Nothing to do")
		return nil
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

	run(getDiff(loadedConfig, configLock.LockedConfig), true, secret)  // grant
	run(getDiff(configLock.LockedConfig, loadedConfig), false, secret) // revoke

	loadedConfigBytes, err := json.MarshalIndent(loadedConfig, "", "    ")
	if err != nil {
		logger.Error("Unable to update config file")
		return err
	}

	utils.CreateFileWithData(config.Config.RDSAccessConfig, string(loadedConfigBytes))

	configLock.LockedConfig = loadedConfig
	configLock.Checksum = utils.GetSHA512Checksum(loadedConfigBytes)

	loadedConfigLockBytes, err := json.MarshalIndent(configLock, "", "    ")
	if err != nil {
		logger.Error("Unable to update config file")
		return err
	}

	return utils.CreateFileWithData(config.Config.RDSAccessConfig+".lock", string(loadedConfigLockBytes))
}

func run(diff map[string]map[string]map[string][]string, isGrant bool, secret Database) {
	permission := "GRANT"
	if !isGrant {
		permission = "REVOKE"
	}

	for username, accessMap := range diff {
		queries := builtQueriesForUser(secret.DBName, username, accessMap, isGrant)
		if len(queries) == 0 {
			logger.Success("Nothing to do for %s", username)
			continue
		}

		logger.Info("(%s) Running for %s", permission, username)

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

		if err != nil {
			logger.Error("  %s; \n%s", strings.Join(queries, ";"), stderr.String())

			// TODO: what to do with failed queries?
			// should they not go in access.lock?
			// how to extract failure query from string of queries
		} else {
			logger.Success("  %s;", strings.Join(queries, ";"))
		}
	}
}

func getDiff(config, configLock Config) map[string]map[string]map[string][]string {
	diff := make(map[string]map[string]map[string][]string)
	for username, tableGrants := range config {
		if _, ok := configLock[username]; !ok {
			diff[username] = tableGrants
			continue
		}

		lockedTables := configLock[username]
		for tableName, grants := range tableGrants {
			if _, ok := lockedTables[tableName]; !ok {
				if tableMap, ok := diff[username]; !ok {
					diff[username] = map[string]map[string][]string{
						tableName: grants,
					}
				} else {
					tableMap[tableName] = grants
					diff[username] = tableMap
				}
				continue
			}

			for grant, columns := range grants {
				if lockedColumns, ok := lockedTables[tableName][grant]; !ok {
					if _, ok := diff[username]; !ok {
						diff[username] = make(map[string]map[string][]string)
					}

					if _, ok := diff[username][tableName]; !ok {
						diff[username][tableName] = make(map[string][]string)
					}

					diff[username][tableName][grant] = columns
				} else {
					if !utils.AreStringArrayEqual(columns, lockedColumns) {
						columnsToAdd := utils.GetStringAMinusB(columns, lockedColumns)
						if len(columnsToAdd) == 0 {
							continue
						}

						if _, ok := diff[username]; !ok {
							diff[username] = make(map[string]map[string][]string)
						}

						if _, ok := diff[username][tableName]; !ok {
							diff[username][tableName] = make(map[string][]string)
						}

						if _, ok := diff[username][tableName][grant]; !ok {
							diff[username][tableName][grant] = make([]string, 0)
						}

						diff[username][tableName][grant] = columnsToAdd
					}
				}
			}
		}
	}

	return diff
}

func builtQueriesForUser(dbname string, username string, accessMap map[string]map[string][]string, isGrant bool) []string {
	queries := make([]string, 0)

	permission := "GRANT"
	permissionHelper := "TO"
	if !isGrant {
		permission = "REVOKE"
		permissionHelper = "FROM"
	}

	for tableName, grants := range accessMap {
		for grant, columns := range grants {
			if len(columns) == 0 {
				logger.Warn("Skipping %s on %s, no columns present", grant, tableName)
				continue
			}

			if len(columns) == 1 && columns[0] == "*" {
				if isGrant {
					logger.Warn("%s demands %s on all columns %s.%s", username, grant, dbname, tableName)
				}

				queries = append(queries, fmt.Sprintf("%s %s on %s.%s %s '%s'@'%%'", permission, grant, dbname, tableName, permissionHelper, username))
				continue
			}

			queries = append(queries, fmt.Sprintf("%s %s (%s) on %s.%s %s '%s'@'%%'", permission, grant, strings.Join(columns, ", "), dbname, tableName, permissionHelper, username))
		}
	}

	return queries
}
