package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

const cacheValid = 5 * time.Minute

const RoleUnknown = RoleName("UNKNOWN")

type UserID string
type RoleID string
type RoleName string

type ServerUser struct {
	ID       UserID
	Username string
	Roles    []RoleName

	validUntil time.Time
}

type DiscordBot struct {
	m sync.RWMutex

	Config Config

	guildsIDs     []string
	applicationId string

	cachedUsers      map[UserID]ServerUser
	cachedRoles      map[RoleID]RoleName
	cachedRolesReady bool
}

func NewDiscordBot(config Config) *DiscordBot {
	return &DiscordBot{
		Config:      config,
		cachedUsers: map[UserID]ServerUser{},
		guildsIDs:   []string{},
	}
}

func (b *DiscordBot) CacheRoles(ctx context.Context, logger *zap.Logger, discord *discordgo.Session) {
	t := time.NewTicker(cacheValid)

	for {
		// wait for list of the guilds
		if len(b.GuildsIDs()) < 1 {
			continue
		}

		b.m.Lock()
		b.cachedRoles = cachedRoles(logger, discord, b.guildsIDs)

		b.cachedRolesReady = true
		b.m.Unlock()

		select {
		case <-t.C:
			continue
		case <-ctx.Done():
			return
		}
	}
}

func (b *DiscordBot) WaitUntilReady(ctx context.Context) error {
	t := time.NewTicker(100 * time.Millisecond)

	for {
		select {
		case <-t.C:
			if b.Ready() {
				return nil
			}
		case <-ctx.Done():
			return fmt.Errorf("context gets invalid")
		}
	}
}

func (b *DiscordBot) Ready() bool {
	b.m.RLock()
	defer b.m.RUnlock()

	return len(b.applicationId) > 0 && len(b.guildsIDs) > 0 && b.cachedRolesReady
}

func (b *DiscordBot) GuildsIDs() []string {
	b.m.RLock()
	defer b.m.RUnlock()

	return b.guildsIDs
}

func (b *DiscordBot) UpdateGuildsIDs(guilds []string) {
	b.m.Lock()
	defer b.m.Unlock()

	b.guildsIDs = guilds
}

func (b *DiscordBot) AddCachedUser(id UserID, details ServerUser) {
	b.m.Lock()
	defer b.m.Unlock()

	details.validUntil = time.Now().Add(cacheValid)

	b.cachedUsers[id] = details
}

func (b *DiscordBot) CachedUser(id UserID) *ServerUser {
	details, found := b.cachedUsers[id]
	if !found {
		return nil
	}

	// cache is invalid
	if !details.validUntil.After(time.Now()) {
		return nil
	}

	return &details
}

func (b *DiscordBot) UpdateApplicationId(id string) {
	b.applicationId = id
}

func (b *DiscordBot) CachedRole(id RoleID) RoleName {
	b.m.RLock()
	defer b.m.RUnlock()
	roleName, found := b.cachedRoles[id]
	if !found {
		return RoleUnknown
	}

	return roleName
}
