package cmd

import (
	"context"

	"github.com/mudrex/onyx/pkg/config"
	initPkg "github.com/mudrex/onyx/pkg/core/init"
	"github.com/spf13/cobra"
)

var configCommand = &cobra.Command{
	Use:   "config",
	Short: "Sets up onyx config",
}

var configInitCommand = &cobra.Command{
	Use:     "init",
	Short:   "Sets up onyx config",
	Args:    cobra.NoArgs,
	Example: "onyx config init",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		return initPkg.Init(ctx, forceInit)
	},
}

var configSetCommand = &cobra.Command{
	Use:     "set <key> <value>",
	Short:   "Sets up onyx config",
	Example: "onyx config set somekey somevalue",
	Args:    cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return config.SetConfigKey(args[0], args[1])
	},
}

func init() {
	configCommand.AddCommand(configInitCommand, configSetCommand)

	configInitCommand.Flags().BoolVarP(&forceInit, "force", "f", false, "Force re initialization of config")
}
