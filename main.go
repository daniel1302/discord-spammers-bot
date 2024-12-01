package main

import (
	"fmt"
	"os"

	"go.uber.org/zap"
)

func main() {
	args := os.Args[1:]

	if len(args) < 1 {
		panic(fmt.Sprintf("invalid config path. usage: %s <config-path>", os.Args[0]))
	}

	config, err := ReadConfigFile(args[0])
	if err != nil {
		panic(fmt.Sprintf("failed to parse config file: %s", err.Error()))
	}

	level := zap.NewAtomicLevelAt(zap.InfoLevel)
	if config.Debug {
		level = zap.NewAtomicLevelAt(zap.DebugLevel)
	}
	zapConfig := zap.NewProductionConfig()
	zapConfig.Level = level

	logger, err := zapConfig.Build()
	logger.Level()
	if err != nil {
		panic(err)
	}

	if err := Run(logger, config); err != nil {
		panic(err)
	}
}
