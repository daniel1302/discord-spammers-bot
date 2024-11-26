package main

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

const (
	DefaultRequestTimeout = 10 * time.Second
)

func commandWipe(
	logger *zap.Logger,
	message *discordgo.MessageCreate,
	discord *discordgo.Session,
	bot *DiscordBot,
	config ConfigCommandWipe,
	reportChannel string,
) {
	if bot.wipeInProgress.Load() {
		logger.Info("Wipe command is still running. Wait before it finishes")
		return
	}

	bot.wipeInProgress.Store(true)
	defer bot.wipeInProgress.Store(false)

	if !config.Enabled {
		return
	}

	if !strings.HasPrefix(message.Content, config.Command) {
		logger.Sugar().Debugf(
			"Message is not wipe command",
		)

		return
	}

	if message.ChannelID == "" || !slices.Contains(config.ActiveChannels, message.ChannelID) {
		logger.Sugar().Debugf(
			"Channel(%s) has not enabled wipe command",
			message.ChannelID,
		)

		return
	}

	if message.Author == nil {
		logger.Sugar().Debugf(
			"Author is unknown, wipe command cannot be executed",
		)

		return
	}

	// Ignore messages from bot itself
	if message.Author != nil && message.Author.ID == discord.State.User.ID {
		return
	}

	if !isUserWhitelisted(logger, discord, bot, config.WhitelistedRoles, message.Author.ID) {
		logger.Sugar().Debugf(
			"User %s(%s) is not allowed to execute wipe command",
			message.Author.Username,
			message.Author.ID,
		)

		return
	}

	errors := 5
	deletedMessages := 0

	wipeCtx, wipeCancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer wipeCancel()
	for {
		if errors < 1 {
			logger.Error("Too many errors during wipe command exiting")
			return
		}

		logger.Info("Getting messages to be deleted")
		messages, err := func() ([]*discordgo.Message, error) {
			ctx, cancel := context.WithTimeout(wipeCtx, DefaultRequestTimeout)
			defer cancel()

			return discord.ChannelMessages(
				message.ChannelID,
				1,
				"",
				"",
				"",
				discordgo.WithContext(ctx),
				discordgo.WithClient(DefaultHttpClient(DefaultRequestTimeout)),
			)
		}()

		if err != nil {
			errors--
			logger.Error("Failed to fetch messages", zap.Error(err))
			continue
		}

		// no more messages
		if len(messages) < 1 {
			break
		}

		m := messages[0]
		bot.wipedMessagesMut.Lock()
		bot.wipedMessages = append(bot.wipedMessages, m.ID)
		bot.wipedMessagesMut.Unlock()
		time.Sleep(100 * time.Millisecond)

		if err := func() error {
			// rl.Wait(wipeCtx)
			ctx, cancel := context.WithTimeout(wipeCtx, DefaultRequestTimeout)
			defer cancel()

			logger.Sugar().Infof("Deleting message: %s written %s", m.ID, m.Timestamp)
			return discord.ChannelMessageDelete(
				message.ChannelID,
				m.ID,
				discordgo.WithContext(ctx),
				discordgo.WithClient(DefaultHttpClient(DefaultRequestTimeout)),
				discordgo.WithRetryOnRatelimit(true),
				discordgo.WithRestRetries(5),
			)
		}(); err != nil {
			errors--
			logger.Error("Failed to delete message", zap.Error(err))
		}

		deletedMessages++
	}

	logMessage := fmt.Sprintf("Wipe channel command received\n=================================\nAuthor: <@%s>\nChannel: <#%s>\nMessages deleted: %d",
		message.Author.ID,
		message.ChannelID,
		deletedMessages,
	)
	logger.Info(logMessage)

	discord.ChannelMessageSend(reportChannel, logMessage)
}
