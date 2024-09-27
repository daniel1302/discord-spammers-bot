package main

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

func isUserWhitelisted(logger *zap.Logger, discord *discordgo.Session, bot *DiscordBot, whiteListerRoles []string, authorId string) bool {
	if len(whiteListerRoles) > 0 {
		userDetails, err := cachedUser(logger, discord, bot, authorId)
		if err != nil {
			logger.Sugar().Warnf("failed to get cached user: %s", err.Error())
		}

		for _, roleName := range whiteListerRoles {
			if slices.Contains(userDetails.Roles, RoleName(roleName)) {

				return true
			}
		}
	}

	return false
}

func reportSuspiciousMessage(
	logger *zap.Logger,
	message *discordgo.MessageCreate,
	discord *discordgo.Session,
	bot *DiscordBot,
	config ConfigSuspiciousMessage,
	reportChannel string,
) {
	if !config.Enabled {
		return
	}

	// Ignore messages from bot itself
	if message.Author.ID == discord.State.User.ID {
		return
	}

	if message.Author != nil && isUserWhitelisted(logger, discord, bot, config.WhiteListedRoles, message.Author.ID) {
		logger.Sugar().Infof(
			"User %s(%s) has whitelisted role, message does not need to be reported",
			message.Author.Username,
			message.Author.ID,
		)
		return
	}

	// We can do nothing when message content is empty
	if message.Content == "" {
		logger.Sugar().Warnf("cannot get message content for message id %s", message.ID)
		return
	}

	suspicious := false
	for _, keyword := range config.Keywords {
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

	discord.ChannelMessageSend(reportChannel, logMessage)
}

func deleteInviteLinks(
	logger *zap.Logger,
	message *discordgo.MessageCreate,
	discord *discordgo.Session,
	bot *DiscordBot,
	config ConfigDeleteInviteLinks,
	reportChannel string,
) {
	if !config.Enabled {
		return
	}

	// Ignore messages from bot itself
	if message.Author.ID == discord.State.User.ID {
		return
	}

	if message.Author != nil && isUserWhitelisted(logger, discord, bot, config.WhiteListedRoles, message.Author.ID) {
		logger.Sugar().Infof(
			"User %s(%s) has whitelisted role, message does not need to be reported",
			message.Author.Username,
			message.Author.ID,
		)
		return
	}

	discordInviteLinkRegex := regexp.MustCompile(`(discord\.[a-z]{2,}|discordapp?\.[a-z]{2,})(.invite)?[\/\\]\w+`)
	if !discordInviteLinkRegex.MatchString(message.Content) {
		return
	}

	logMessage := fmt.Sprintf("Suspicious message on the server\n================================\nAuthor: <@%s>\nChannel: <#%s>\nMessage: ```%s```",
		message.Author.ID,
		message.ChannelID,
		message.Content,
	)
	logger.Info(logMessage)

	discord.ChannelMessageSend(reportChannel, logMessage)

	warnUserMessage := fmt.Sprintf(
		"<@%s> Ops, looks like you posted invitation to another discord server. It is against rules of this server. Please ask administrator to post invitation link for you.",
		message.Author.ID,
	)
	if _, err := discord.ChannelMessageSend(message.ChannelID, warnUserMessage); err != nil {
		logger.Sugar().Error("failed to send warn message after posting server invitation: %s", err.Error())
	}

	if err := discord.ChannelMessageDelete(message.ChannelID, message.ID); err != nil {
		logger.Sugar().Error("failed to send delete invitation with posted server invitation: %s", err.Error())
	}
}

func reportDeletedMessage(
	logger *zap.Logger,
	message *discordgo.MessageDelete,
	discord *discordgo.Session,
	config ConfigReportDeletedMessages,
	bot *DiscordBot,
	reportChannel string,
) {
	if !config.Enabled {
		return
	}

	if message.BeforeDelete.Author != nil && isUserWhitelisted(logger, discord, bot, config.WhiteListedRoles, message.BeforeDelete.Author.ID) {
		logger.Sugar().Infof(
			"User %s(%s) has whitelisted role, message does not need to be reported",
			message.BeforeDelete.Author.Username,
			message.BeforeDelete.Author.ID,
		)
		return
	}

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

	discord.ChannelMessageSend(reportChannel, logMessage)
}
