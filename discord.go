package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"

	"go.uber.org/zap"

	"github.com/bwmarrin/discordgo"
)

func Run(logger *zap.Logger, config *Config) error {
	// create a session
	discord, err := discordgo.New("Bot " + config.BotToken)

	if err != nil {
		return fmt.Errorf("failed to initialize bot: %w", err)
	}

	// add a event handler
	discord.AddHandler(newMessageHandler(logger, config.ReportChannel, config.ModeratedKeywords))
	discord.AddHandler(deleteMessageHandler(logger, config.ReportChannel))

	// open session
	discord.Open()
	defer discord.Close() // close session, after function termination

	if err := registerChannelsModeration(logger, discord, config.ModeratedChannels); err != nil {
		return fmt.Errorf("failed to register channels for moderation: %w", err)
	}

	discord.State.MaxMessageCount = config.MessageKeepTrackCount

	// keep bot running untill there is NO os interruption (ctrl + C)
	logger.Info("Bot running....")
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

	return nil
}

func registerChannelsModeration(logger *zap.Logger, discord *discordgo.Session, channelsIds []string) error {
	for _, channelId := range channelsIds {
		channel, err := discord.Channel(channelId)
		if err != nil {
			return fmt.Errorf("cannot find channel %s: %w", channelId, err)
		}

		err = discord.State.ChannelAdd(channel)
		if err != nil {
			return fmt.Errorf("cannot add channel %s(%s) to the state world: %w\n", channel.Name, channel.ID, err)
		}

		logger.Sugar().Infof("channel %s(%s) added to the state word", channel.Name, channelId)
	}

	return nil
}

func newMessageHandler(logger *zap.Logger, administrationChannelId string, moderatedKeywords []string) interface{} {
	return func(discord *discordgo.Session, message *discordgo.MessageCreate) {
		if message.Author.ID == discord.State.User.ID {
			return
		}

		// We can do nothing when message content is empty
		if message.Content == "" {
			logger.Sugar().Warnf("cannot get message content for message id %s", message.ID)
			return
		}

		suspicious := false
		for _, keyword := range moderatedKeywords {
			if strings.Contains(strings.ToLower(message.Content), strings.ToLower(keyword)) {
				suspicious = true
			}
		}

		if !suspicious {
			return
		}

		logMessage := fmt.Sprintf("Suspicious message on the server\n================================\nAuthor: <@%s>\nChannel: <#%s>\nMessage: ```%s```",
			message.Author.ID,
			message.ChannelID,
			message.Content,
		)
		logger.Info(logMessage)

		discord.ChannelMessageSend(administrationChannelId, logMessage)
	}
}

func deleteMessageHandler(logger *zap.Logger, administrationChannelId string) interface{} {
	return func(discord *discordgo.Session, message *discordgo.MessageDelete) {
		// If BeforeDelete is empty, there is nothing more We can do, as We lost track of this message.
		if message.BeforeDelete == nil {
			logger.Sugar().Warnf("Message %s is not in the state", message.ID)
			return
		}

		if message.BeforeDelete.Author == nil {
			logger.Sugar().Warnf("Message author for %s is not in the state", message.ID)
			return
		}

		logMessage := fmt.Sprintf("New deleted message on the server\n=================================\nAuthor: <@%s>\nChannel: <#%s>\nMessage: ```%s```",
			message.BeforeDelete.Author.ID,
			message.BeforeDelete.ChannelID,
			message.BeforeDelete.Content,
		)
		logger.Info(logMessage)

		discord.ChannelMessageSend(administrationChannelId, logMessage)
	}

}
