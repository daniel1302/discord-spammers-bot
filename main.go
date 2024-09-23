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

	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}

	if err := Run(logger, config); err != nil {
		panic(err)
	}
}
