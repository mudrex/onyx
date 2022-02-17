package cmd

import (
	"context"

	initPkg "github.com/mudrex/onyx/pkg/core/init"
	"github.com/spf13/cobra"
)

var forceInit bool

var initCommand = &cobra.Command{
	Use:     "init",
	Short:   "Sets up onyx config",
	Example: "onyx init",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		return initPkg.Init(ctx, forceInit)
	},
}

func init() {
	initCommand.Flags().BoolVarP(&forceInit, "force", "f", false, "Force re initialization of config")
}
