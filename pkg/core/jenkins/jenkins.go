package jenkins

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/mudrex/onyx/pkg/config"
	"github.com/mudrex/onyx/pkg/core/secretsmanager"
	"github.com/mudrex/onyx/pkg/filesystem"
	"github.com/mudrex/onyx/pkg/logger"
	"github.com/mudrex/onyx/pkg/utils"
)

type JenkinsSecret struct {
	Username string `json:"username"`
	Token    string `json:"password"`
	Host     string `json:"host"`
}

type Config map[string][]string

type ConfigLock struct {
	Checksum     string `json:"checksum"`
	LockedConfig Config `json:"locked_config"`
}

var jenkinsSecret = JenkinsSecret{}

func RefreshUsers(ctx context.Context, cfg aws.Config) error {
	return refresh(ctx, cfg, config.Config.JenkinsUsersConfig)
}

func RefreshRoles(ctx context.Context, cfg aws.Config) error {
	return refresh(ctx, cfg, config.Config.JenkinsRolesConfig)
}

func refresh(ctx context.Context, cfg aws.Config, accessConfig string) error {
	configData, err := filesystem.ReadFile(accessConfig)
	if err != nil {
		return err
	}

	var loadedConfig Config

	err = json.Unmarshal([]byte(configData), &loadedConfig)
	if err != nil {
		return err
	}
	print(loadedConfig)

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

	if len(config.Config.JenkinsSecretName) == 0 {
		return fmt.Errorf("jenkins secret name not specified")
	}

	secretString := secretsmanager.GetSecret(ctx, cfg, config.Config.JenkinsSecretName)
	err = json.Unmarshal([]byte(secretString), &jenkinsSecret)
	if err != nil {
		return err
	}

	refreshUsers(ctx, cfg, loadedConfig, configLock.LockedConfig, jenkinsSecret)

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

func refreshUsers(
	ctx context.Context,
	cfg aws.Config,
	currConfig Config,
	lockedConfig Config,
	secret JenkinsSecret,
) error {
	add, _ := getDiff(currConfig, lockedConfig)
	err := addUsers(add, secret)
	if err != nil {
		logger.Error("Unable to add roles to users")
		return err
	}
	//removeUsers(ctx, cfg, remove, secret)
	return nil
}

func addUsers(add Config, secret JenkinsSecret) error {
	for sid, roles := range add {
		for _, role := range roles {
			assignRole(sid, role, secret)
		}
	}
	return nil
}

func assignRole(sid string, role string, secret JenkinsSecret) error {
	url := secret.Host + "/role-strategy/strategy/assignRole"
	client := &http.Client{}
	request, err := http.NewRequest("POST", url, nil)
	if err != nil {
		logger.Error("Unable create request with url %s", url)
		return err
	}
	request.SetBasicAuth(secret.Username, secret.Token)
	query := request.URL.Query()
	query.Add("type", "globalRoles")
	query.Add("roleName", role)
	query.Add("sid", sid)
	request.URL.RawQuery = query.Encode()
	logger.Info("(%s) Running for %s | %s", role, sid, request.URL.String())

	response, err := client.Do(request)
	if err != nil {
		logger.Error("Unable to add role %s to user %s", role, sid)
		return err
	}
	logger.Info(response.Status)
	return nil
}

// func run(
// 	ctx context.Context,
// 	cfg aws.Config,
// 	diff map[string]map[string]map[string][]string,
// 	isGrant bool,
// 	secret Database,
// ) {
// 	permission := "GRANT"
// 	if !isGrant {
// 		permission = "REVOKE"
// 	}

// 	for username, accessMap := range diff {
// 		queries := builtQueriesForUser(secret.DBName, username, accessMap, isGrant)
// 		if len(queries) == 0 {
// 			logger.Success("Nothing to do for %s", username)
// 			continue
// 		}

// 		logger.Info("(%s) Running for %s", permission, username)

// 		_, stderr, err := runQuery(ctx, cfg, strings.Join(queries, ";"))
// 		if err != nil {
// 			logger.Error("  %s; \n%s", strings.Join(queries, ";"), stderr)

// 			// TODO: what to do with failed queries?
// 			// should they not go in access.lock?
// 			// how to extract failure query from string of queries
// 		} else {
// 			logger.Success("  %s;", strings.Join(queries, ";"))
// 		}
// 	}
// }

func getDiff(config, configLock Config) (Config, Config) {
	var addDiff Config = make(Config)
	var removeDiff Config = make(Config)

	for sid, roles := range config {
		intersection := utils.GetIntersectionBetweenStringArrays(roles, configLock[sid])
		addDiff[sid] = utils.GetDifferenceBetweenStringArrays(roles, intersection)
		removeDiff[sid] = utils.GetDifferenceBetweenStringArrays(configLock[sid], intersection)
	}

	return addDiff, removeDiff
}

// 	return diff, users
// }

// func builtQueriesForUser(dbname string, username string, accessMap map[string]map[string][]string, isGrant bool) []string {
// 	queries := make([]string, 0)

// 	permission := "GRANT"
// 	permissionHelper := "TO"
// 	if !isGrant {
// 		permission = "REVOKE"
// 		permissionHelper = "FROM"
// 	}

// 	for tableName, grants := range accessMap {
// 		if permission == "GRANT" {
// 			if _, ok := CriticalTables[tableName]; ok {
// 				logger.Warn("%s is being granted %v access to %s", username, grants, tableName)
// 				notifier.Notify(
// 					config.Config.SlackHook,
// 					fmt.Sprintf(":bangbang: %s is being granted %v access to %s", username, grants, tableName),
// 				)
// 			}
// 		}

// 		for grant, columns := range grants {
// 			if len(columns) == 0 {
// 				logger.Warn("Skipping %s on %s, no columns present", grant, tableName)
// 				continue
// 			}

// 			if len(columns) == 1 && columns[0] == "*" {
// 				if isGrant {
// 					logger.Warn("%s demands %s on all columns %s.%s", username, grant, dbname, tableName)
// 				}

// 				queries = append(queries, fmt.Sprintf("%s %s on %s.%s %s '%s'@'%%'", permission, grant, dbname, tableName, permissionHelper, username))
// 				continue
// 			}

// 			queries = append(queries, fmt.Sprintf("%s %s (%s) on %s.%s %s '%s'@'%%'", permission, grant, strings.Join(columns, ", "), dbname, tableName, permissionHelper, username))
// 		}
// 	}

// 	return queries
// }

// func runQuery(ctx context.Context, cfg aws.Config, query string) (string, string, error) {
// 	if databaseSecret.Host == "" {
// 		logger.Info("Fetching DB credentials")

// 		secretString := secretsmanager.GetSecret(ctx, cfg, config.Config.RDSSecretName)
// 		err := json.Unmarshal([]byte(secretString), &databaseSecret)
// 		if err != nil {
// 			return "", "", err
// 		}
// 	}

// 	c := fmt.Sprintf(
// 		"mysql -h %s -u %s -p'%s' -e \"%s;\"",
// 		databaseSecret.Host,
// 		databaseSecret.Username,
// 		databaseSecret.Password,
// 		query,
// 	)

// 	var stdout bytes.Buffer
// 	var stderr bytes.Buffer
// 	cmd := exec.Command("bash", "-c", c)
// 	cmd.Stdout = &stdout
// 	cmd.Stderr = &stderr

// 	return stdout.String(), stderr.String(), cmd.Run()
// }

// func createUsers(ctx context.Context, cfg aws.Config, usernames []string) error {
// 	for _, username := range usernames {
// 		if username == "" {
// 			continue
// 		}

// 		newPassword := utils.GetRandomStringWithSymbols(40)
// 		_, _, err := runQuery(ctx, cfg, fmt.Sprintf("CREATE USER '%s'@'%%' IDENTIFIED BY '%s'", username, newPassword))
// 		if err != nil {
// 			return err
// 		}

// 		logger.Info("Created user %s", username)

// 		// TODO: send mail to user with db password
// 	}

// 	return nil
// }

// func dropUsers(ctx context.Context, cfg aws.Config, usernames []string) error {
// 	for _, username := range usernames {
// 		if username == "" {
// 			continue
// 		}

// 		_, _, err := runQuery(ctx, cfg, fmt.Sprintf("DROP USER '%s'@'%%'", username))
// 		if err != nil {
// 			return err
// 		}

// 		logger.Warn("Dropped user %s", username)
// 	}

// 	return nil
// }
