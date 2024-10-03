package main

import (
	"fmt"
	"net/http"
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
			return false
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
	if message.Author != nil && message.Author.ID == discord.State.User.ID {
		return
	}

	if message.Author != nil && isUserWhitelisted(logger, discord, bot, config.WhiteListedRoles, message.Author.ID) {
		logger.Sugar().Debugf(
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
			break
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
	// reportChannel string,
) {
	if !config.Enabled {
		return
	}

	// Ignore messages from bot itself
	if message.Author != nil && message.Author.ID == discord.State.User.ID {
		return
	}

	if message.Author != nil && isUserWhitelisted(logger, discord, bot, config.WhiteListedRoles, message.Author.ID) {
		logger.Sugar().Debugf(
			"User %s(%s) has whitelisted role, message does not need to be reported",
			message.Author.Username,
			message.Author.ID,
		)
		return
	}

	if !shouldMessageBeDeleted(logger, message.Content) {
		return
	}

	warnUserMessage := fmt.Sprintf(
		config.WarnMessage,
		message.Author.ID,
	)

	if _, err := discord.ChannelMessageSend(message.ChannelID, warnUserMessage); err != nil {
		logger.Sugar().Error("failed to send warn message after posting server invitation: %s", err.Error())
	}

	if err := discord.ChannelMessageDelete(message.ChannelID, message.ID); err != nil {
		logger.Sugar().Error("failed to send delete invitation with posted server invitation: %s", err.Error())
	}
}

func isDiscordInvitation(message string) bool {
	// Example matches:
	//	- discord.com/invite\ZnZ3nxZMuq
	//	- discordapp.com/invite\ZnZ3nxZMuq
	inviteRegex := regexp.MustCompile(`(https?:\/\/)?(www\.)?((discord(app)?\.com[/\\]invite)|(discord\.gg))[/\\]\w+`)
	return inviteRegex.MatchString(message)
}

func shouldMessageBeDeleted(logger *zap.Logger, message string) bool {
	if isDiscordInvitation(message) {
		return true
	}

	// Some of the spammers send custom domains that returns only 301 Location: discord.com/invite/xxxx
	// Example: https:/%20@@dis.army/chat/21312
	urlRegex := regexp.MustCompile(`https:/\/?([^\s]+)`)
	foundUrl := urlRegex.FindStringSubmatch(message)
	if len(foundUrl) > 0 {
		resp, err := http.Get(fmt.Sprintf("https://%s", foundUrl[1]))
		if err != nil {
			logger.Sugar().Infof("checking if message with link(%s) should be deleted, but cannot open page: %s", foundUrl[0], err.Error())
			return false
		}

		// Follow all the redirects, etc and then check the latest Request URL
		latestUrl := resp.Request.URL.String()
		if isDiscordInvitation(latestUrl) {
			return true
		}
	}

	return false
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

	// If BeforeDelete is empty, there is nothing more We can do, as We lost track of this message.
	// Sometimes message is deleted too fast to process it
	if message.BeforeDelete == nil {
		logger.Sugar().Warnf("Message %s is not in the state bot state", message.ID)
		return
	}

	if message.BeforeDelete.Author != nil && isUserWhitelisted(logger, discord, bot, config.WhiteListedRoles, message.BeforeDelete.Author.ID) {
		logger.Sugar().Debugf(
			"User %s(%s) has whitelisted role, message does not need to be reported",
			message.BeforeDelete.Author.Username,
			message.BeforeDelete.Author.ID,
		)
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
