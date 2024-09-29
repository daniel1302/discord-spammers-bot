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

	Features ConfigFeatures `toml:"features"`

	ModeratedChannels []string `toml:"moderated_channels"`
	ModeratedKeywords []string `toml:"moderated_keywords"`
}

type ConfigSuspiciousMessage struct {
	Enabled          bool     `toml:"enabled"`
	Keywords         []string `toml:"keywords"`
	WhiteListedRoles []string `toml:"whitelisted_roles"`
}

type ConfigReportDeletedMessages struct {
	Enabled          bool     `toml:"enabled"`
	WhiteListedRoles []string `toml:"whitelisted_roles"`
}

type ConfigDeleteInviteLinks struct {
	Enabled          bool     `toml:"enabled"`
	WhiteListedRoles []string `toml:"whitelisted_roles"`
	WarnMessage      string   `toml:"warn_message"`
}

type ConfigFeatures struct {
	SuspiciousMessage ConfigSuspiciousMessage `toml:"suspicious_messages"`

	ReportDeletedMessages ConfigReportDeletedMessages `toml:"report_deleted_messages"`

	DeleteInviteLinks ConfigDeleteInviteLinks `toml:"delete_invite_links"`
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
