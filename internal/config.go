package internal

import (
	"errors"
	"os"
	"strings"

	"PanickedBot/internal/db"
)

// Config holds application configuration
type Config struct {
	DiscordToken string
	DatabaseDSN  string
}

// GuildConfig represents guild-specific configuration
type GuildConfig struct {
	OfficerRoleID     string `db:"officer_role_id"`
	GuildMemberRoleID string `db:"guild_member_role_id"`
	MercenaryRoleID   string `db:"mercenary_role_id"`
	CommandChannelID  string `db:"command_channel_id"`
}

// LoadConfigFromEnv loads configuration from environment variables
func LoadConfigFromEnv() (Config, error) {
	get := func(key string) string { return strings.TrimSpace(os.Getenv(key)) }
	c := Config{
		DiscordToken: get("DISCORD_BOT_TOKEN"),
		DatabaseDSN:  get("DATABASE_DSN"),
	}
	if c.DiscordToken == "" {
		return c, errors.New("DISCORD_BOT_TOKEN is not set")
	}
	if c.DatabaseDSN == "" {
		return c, errors.New("DATABASE_DSN is not set")
	}
	return c, nil
}

// LoadGuildConfig loads guild-specific configuration from database
func LoadGuildConfig(dbx *db.DB, guildID string) (*GuildConfig, error) {
	var cfg GuildConfig
	err := dbx.Get(&cfg, `
		SELECT officer_role_id, guild_member_role_id, mercenary_role_id, 
		       command_channel_id
		FROM config
		WHERE discord_guild_id = ?
	`, guildID)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
