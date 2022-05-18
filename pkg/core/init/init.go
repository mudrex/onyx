package init

import (
	"context"

	"github.com/mudrex/onyx/pkg/config"
	"github.com/mudrex/onyx/pkg/filesystem"
	"github.com/mudrex/onyx/pkg/logger"
)

func change(ctx context.Context) error {
	return nil
}

func Init(ctx context.Context, force bool) error {
	if !filesystem.FileExists(config.Filename) {
		err := filesystem.CreateFileWithData(config.Filename, config.Config.ToString())
		if err != nil {
			return err
		}

		logger.Success("Created %s file", logger.Underline(config.Filename))
		return nil
	}

	if force {
		logger.Info("Overwriting %s file.", logger.Underline(config.Filename))

		err := filesystem.CreateFileWithData(config.Filename, config.Default().ToString())
		if err != nil {
			return err
		}

		logger.Success("Created %s file", logger.Underline(config.Filename))
		return nil
	}

	logger.Info("Detected %s file. Nothing to do", logger.Underline(config.Filename))

	return nil
}
