package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"go.uber.org/zap"

	"github.com/bwmarrin/discordgo"
)

func Run(logger *zap.Logger, config *Config) error {
	// create a session
	discord, err := discordgo.New("Bot " + config.BotToken)

	if err != nil {
		return fmt.Errorf("failed to initialize bot: %w", err)
	}
	bot := NewDiscordBot(*config)

	// add a event handler
	discord.AddHandler(readyHandler(logger, bot))
	discord.AddHandler(newMessageHandler(logger, bot, *config))
	discord.AddHandler(deleteMessageHandler(logger, *config, bot))

	// open session
	discord.Open()
	defer discord.Close() // close session, after function termination

	if err := registerChannelsModeration(logger, discord, config.ModeratedChannels); err != nil {
		return fmt.Errorf("failed to register channels for moderation: %w", err)
	}

	discord.State.MaxMessageCount = config.MessageKeepTrackCount
	if config.DiscordAPIDebug {
		discord.LogLevel = discordgo.LogDebug
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	appCtx, appCtxCancel := context.WithCancel(context.Background())
	defer appCtxCancel()
	go bot.CacheRoles(appCtx, logger, discord)
	go bot.ClearCachedWipedMessageIDs(appCtx, logger)

	// Wait until bot is ready
	if err := bot.WaitUntilReady(ctx); err != nil {
		return fmt.Errorf("bot is not ready until: %w", err)
	}

	// keep bot running until there is NO os interruption (ctrl + C)
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
			return fmt.Errorf("cannot add channel %s(%s) to the state world: %w", channel.Name, channel.ID, err)
		}

		logger.Sugar().Infof("channel %s(%s) added to the state word", channel.Name, channelId)
	}

	return nil
}

func newMessageHandler(logger *zap.Logger, bot *DiscordBot, config Config) interface{} {
	return func(discord *discordgo.Session, message *discordgo.MessageCreate) {
		reportSuspiciousMessage(logger.Named("Moderation.ReportSuspiciousMessage"), message, discord, bot, config.Features.SuspiciousMessage, config.ReportChannel)

		deleteInviteLinks(logger.Named("Moderation.DeleteInviteLinks"), message, discord, bot, config.Features.DeleteInviteLinks)
		commandWipe(logger.Named("Command.Wipe"), message, discord, bot, config.Commands.Wipe, config.ReportChannel)
	}
}

func deleteMessageHandler(logger *zap.Logger, config Config, bot *DiscordBot) interface{} {
	return func(discord *discordgo.Session, message *discordgo.MessageDelete) {
		reportDeletedMessage(logger.Named("Moderation.ReportDeletedMessage"), message, discord, config.Features.ReportDeletedMessages, bot, config.ReportChannel)
	}
}

func readyHandler(logger *zap.Logger, bot *DiscordBot) interface{} {
	return func(discord *discordgo.Session, ready *discordgo.Ready) {
		guilds := []string{}
		for _, guild := range ready.Guilds {
			guilds = append(guilds, guild.ID)
		}
		if len(guilds) > 0 {
			bot.UpdateGuildsIDs(guilds)
		} else {
			logger.Warn("failed to get list of guilds from the ready event")
		}

		if ready.Application != nil && len(ready.Application.ID) > 0 {
			bot.UpdateApplicationId(ready.Application.ID)
		} else {
			logger.Warn("failed to get application id from the ready event")
		}
	}
}

func cachedUser(logger *zap.Logger, discord *discordgo.Session, bot *DiscordBot, UserId string) (*ServerUser, error) {
	userDetails := bot.CachedUser(UserID(UserId))

	if userDetails != nil {
		return userDetails, nil
	}

	for _, guildId := range bot.GuildsIDs() {
		user, err := discord.GuildMember(
			guildId,
			UserId,
			discordgo.WithRestRetries(5),
			discordgo.WithRetryOnRatelimit(true),
		)
		if err != nil {
			logger.Sugar().Debugf("user %s does not belong to guild %s: %s", UserId, guildId, err.Error())
			continue
		}

		if user == nil {
			logger.Sugar().Warnf("user %s does belong to guild %s, but details not available", UserId, guildId)
			continue
		}

		roles := []RoleName{}
		for _, roleId := range user.Roles {
			roles = append(roles, bot.CachedRole(RoleID(roleId)))
		}

		userDetails := ServerUser{
			ID:       UserID(UserId),
			Username: user.User.Username,
			Roles:    roles,
		}

		bot.AddCachedUser(UserID(UserId), userDetails)

		return &userDetails, nil
	}

	return nil, fmt.Errorf("failed to find user in the server")
}

func cachedRoles(logger *zap.Logger, discord *discordgo.Session, guildIds []string) map[RoleID]RoleName {
	roles := map[RoleID]RoleName{}

	for _, guildId := range guildIds {
		guild, err := discord.Guild(guildId)
		if err != nil {
			logger.Sugar().Warnf("failed to get information about guild id %s: %s", guildId, err.Error())
			continue
		}

		for _, role := range guild.Roles {
			roles[RoleID(role.ID)] = RoleName(role.Name)
		}
	}

	return roles
}
