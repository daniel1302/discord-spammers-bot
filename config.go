package main

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

type Config struct {
	BotToken      string `toml:"bot_token"`
	ReportChannel string `toml:"report_channel"`

	MessageKeepTrackCount int `toml:"messages_keep_track_count"`

	ModeratedChannels []string `toml:"moderated_channels"`
	ModeratedKeywords []string `toml:"moderated_keywords"`
}

func ReadConfigFile(configFilePath string) (*Config, error) {
	configBytes, err := os.ReadFile(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	config := &Config{}

	if _, err = toml.Decode(string(configBytes), &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return config, nil
}
