package main

import (
	"errors"
	"os"
	"strings"

	"github.com/jmoiron/sqlx"
)

type config struct {
	DiscordToken string
	DatabaseDSN  string
}

type GuildConfig struct {
	OfficerRoleID      string `db:"officer_role_id"`
	GuildMemberRoleID  string `db:"guild_member_role_id"`
	MercenaryRoleID    string `db:"mercenary_role_id"`
	CommandChannelID   string `db:"command_channel_id"`
	ResultsChannelID   string `db:"results_channel_id"`
}

func loadConfigFromEnv() (config, error) {
	get := func(key string) string { return strings.TrimSpace(os.Getenv(key)) }
	c := config{
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

func loadGuildConfig(db *sqlx.DB, guildID string) (*GuildConfig, error) {
	var cfg GuildConfig
	err := db.Get(&cfg, `
		SELECT officer_role_id, guild_member_role_id, mercenary_role_id, 
		       command_channel_id, results_channel_id
		FROM config
		WHERE discord_guild_id = ?
	`, guildID)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
