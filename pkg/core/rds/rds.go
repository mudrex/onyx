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
	"github.com/mudrex/onyx/pkg/filesystem"
	"github.com/mudrex/onyx/pkg/logger"
	"github.com/mudrex/onyx/pkg/notifier"
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

var databaseSecret = Database{}

var CriticalTables = make(map[string]bool)

func RefreshUserAccess(ctx context.Context, cfg aws.Config) error {
	return refreshAccess(ctx, cfg, config.Config.RDSAccessConfig)
}

func RefreshServicesAccess(ctx context.Context, cfg aws.Config) error {
	return refreshAccess(ctx, cfg, config.Config.RDSServicesAccessConfig)
}

func refreshAccess(ctx context.Context, cfg aws.Config, accessConfig string) error {
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
		logger.Info("Nothing to do")
		return nil
	}

	if len(config.Config.RDSSecretName) == 0 {
		return fmt.Errorf("RDS secret name not specified")
	}

	// Load critical table list
	if filesystem.FileExists(config.Config.RDSCriticalTablesConfig) {
		var criticalTablesList []string
		criticalTablesListData, err := filesystem.ReadFile(config.Config.RDSCriticalTablesConfig)
		if err != nil {
			return err
		}

		err = json.Unmarshal([]byte(criticalTablesListData), &criticalTablesList)
		if err != nil {
			return err
		}

		for _, table := range criticalTablesList {
			CriticalTables[table] = true
		}
	}

	secretString := secretsmanager.GetSecret(ctx, cfg, config.Config.RDSSecretName)

	err = json.Unmarshal([]byte(secretString), &databaseSecret)
	if err != nil {
		return err
	}

	grantPermissions, usersToAdd := getDiff(loadedConfig, configLock.LockedConfig, true)
	err = createUsers(ctx, cfg, usersToAdd)
	if err != nil {
		return err
	}

	revokePermission, usersToRemove := getDiff(configLock.LockedConfig, loadedConfig, false)
	err = dropUsers(ctx, cfg, usersToRemove)
	if err != nil {
		return err
	}

	run(ctx, cfg, grantPermissions, true, databaseSecret)  // grant
	run(ctx, cfg, revokePermission, false, databaseSecret) // revoke

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

func run(
	ctx context.Context,
	cfg aws.Config,
	diff map[string]map[string]map[string][]string,
	isGrant bool,
	secret Database,
) {
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

		_, stderr, err := runQuery(ctx, cfg, strings.Join(queries, ";"))
		if err != nil {
			logger.Error("  %s; \n%s", strings.Join(queries, ";"), stderr)

			// TODO: what to do with failed queries?
			// should they not go in access.lock?
			// how to extract failure query from string of queries
		} else {
			logger.Success("  %s;", strings.Join(queries, ";"))
		}
	}
}

func getDiff(config, configLock Config, isGrant bool) (map[string]map[string]map[string][]string, []string) {
	diff := make(map[string]map[string]map[string][]string)
	users := make([]string, 0)

	for username, tableGrants := range config {
		if _, ok := configLock[username]; !ok {
			if isGrant {
				users = append(users, username)
				diff[username] = tableGrants
			} else {
				users = append(users, username)
			}

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

	return diff, users
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
		if permission == "GRANT" {
			if _, ok := CriticalTables[tableName]; ok {
				logger.Warn("%s is being granted %v access to %s", username, grants, tableName)
				notifier.Notify(
					config.Config.SlackHook,
					fmt.Sprintf(":bangbang: %s is being granted %v access to %s", username, grants, tableName),
				)
			}
		}

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

func runQuery(ctx context.Context, cfg aws.Config, query string) (string, string, error) {
	if databaseSecret.Host == "" {
		logger.Info("Fetching DB credentials")

		secretString := secretsmanager.GetSecret(ctx, cfg, config.Config.RDSSecretName)
		err := json.Unmarshal([]byte(secretString), &databaseSecret)
		if err != nil {
			return "", "", err
		}
	}

	c := fmt.Sprintf(
		"mysql -h %s -u %s -p'%s' -e \"%s;\"",
		databaseSecret.Host,
		databaseSecret.Username,
		databaseSecret.Password,
		query,
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command("bash", "-c", c)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	return stdout.String(), stderr.String(), cmd.Run()
}

func createUsers(ctx context.Context, cfg aws.Config, usernames []string) error {
	for _, username := range usernames {
		if username == "" {
			continue
		}

		newPassword := utils.GetRandomStringWithSymbols(40)
		_, _, err := runQuery(ctx, cfg, fmt.Sprintf("CREATE USER '%s'@'%%' IDENTIFIED BY '%s'", username, newPassword))
		if err != nil {
			return err
		}

		logger.Info("Created user %s", username)

		// TODO: send mail to user with db password
	}

	return nil
}

func dropUsers(ctx context.Context, cfg aws.Config, usernames []string) error {
	for _, username := range usernames {
		if username == "" {
			continue
		}

		_, _, err := runQuery(ctx, cfg, fmt.Sprintf("DROP USER '%s'@'%%'", username))
		if err != nil {
			return err
		}

		logger.Warn("Dropped user %s", username)
	}

	return nil
}
