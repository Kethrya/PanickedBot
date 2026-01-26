package main

import (
	"errors"
	"os"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/jmoiron/sqlx"
)

type config struct {
	DiscordToken string
	DatabaseDSN  string
}

type GuildConfig struct {
	ResultsChannelID string `db:"results_channel_id"`
	CommandChannelID string `db:"command_channel_id"`
	AllowedRoleID    string `db:"allowed_role_id"`
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
		SELECT results_channel_id, command_channel_id, allowed_role_id
		FROM config
		WHERE discord_guild_id = ?
	`, guildID)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func isAllowedChannel(i *discordgo.InteractionCreate, cfg *GuildConfig) bool {
	// If no channel configured yet, allow anywhere (pre-setup)
	if cfg.ResultsChannelID == "" {
		return true
	}
	return i.ChannelID == cfg.ResultsChannelID
}

func resultChannel(i *discordgo.InteractionCreate, cfg *GuildConfig) string {
	if cfg.ResultsChannelID != "" {
		return cfg.ResultsChannelID
	}
	return i.ChannelID
}
