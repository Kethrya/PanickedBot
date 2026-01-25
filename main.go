package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jmoiron/sqlx"

	"PanickedBot/internal/db"
	"PanickedBot/internal/discord"
)

func main() {
	cfg, err := loadConfigFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	database, err := db.Open(db.Config{
		DSN:             cfg.DatabaseDSN,
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 30 * time.Minute,
	})
	if err != nil {
		log.Fatalf("db open: %v", err)
	}
	defer database.Close()

	if err := database.PingContext(context.Background()); err != nil {
		log.Fatalf("db ping: %v", err)
	}

	dg, err := discordgo.New("Bot " + cfg.DiscordToken)
	if err != nil {
		log.Fatalf("discord session: %v", err)
	}
	dg.Identify.Intents = discordgo.IntentsGuilds

	setupCommand := &discordgo.ApplicationCommand{
		Name:        "setup",
		Description: "Configure bot channels and permissions for this server",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionChannel,
				Name:        "command_channel",
				Description: "Channel where commands are allowed",
				Required:    true,
				ChannelTypes: []discordgo.ChannelType{
					discordgo.ChannelTypeGuildText,
					discordgo.ChannelTypeGuildNews,
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionChannel,
				Name:        "results_channel",
				Description: "Channel where results will be posted (defaults to command_channel)",
				Required:    false,
				ChannelTypes: []discordgo.ChannelType{
					discordgo.ChannelTypeGuildText,
					discordgo.ChannelTypeGuildNews,
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionRole,
				Name:        "allowed_role",
				Description: "Role allowed to use restricted commands (leave empty to require admin perms)",
				Required:    false,
			},
		},
	}

	commands := []*discordgo.ApplicationCommand{
		{Name: "ping", Description: "health check"},
		setupCommand,
	}

	dg.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Type != discordgo.InteractionApplicationCommand {
			return
		}

		cmdName := i.ApplicationCommandData().Name

		if cmdName == "setup" {
			handleSetup(s, i, database)
			return
		}

		if i.GuildID == "" {
			respondText(s, i, "This bot only works in servers.")
			return
		}

		cfg, err := loadGuildConfig(database, i.GuildID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				discord.RespondEphemeral(s, i, "Guild is not set up yet. Run /setup first.")
				return
			}
			log.Printf("load guild config: %v", err)
			discord.RespondEphemeral(s, i, "Internal error loading guild config.")
			return
		}

		// Channel guard (command channel)
		if cfg.CommandChannelID != "" && i.ChannelID != cfg.CommandChannelID {
			discord.RespondEphemeral(
				s,
				i,
				"Use this command in <#"+cfg.CommandChannelID+">.",
			)
			return
		}

		switch i.ApplicationCommandData().Name {

		case "ping":
			discord.RespondText(s, i, "pong")

		default:
			discord.RespondEphemeral(s, i, "Unknown command.")
		}

	})

	if err := dg.Open(); err != nil {
		log.Fatalf("discord open: %v", err)
	}
	defer dg.Close()

	appID := dg.State.User.ID

	registered := make([]*discordgo.ApplicationCommand, 0, len(commands))
	for _, cmd := range commands {
		rc, err := dg.ApplicationCommandCreate(appID, "", cmd)
		if err != nil {
			log.Fatalf("command create (%s): %v", cmd.Name, err)
		}
		registered = append(registered, rc)
		log.Printf("registered global /%s", cmd.Name)
	}

	if err := ensureGuildRows(database, dg.State.Guilds); err != nil {
		log.Printf("bootstrap guild rows warning: %v", err)
	}

	log.Printf("bot ready (app=%s)", appID)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	_ = registered
}

type config struct {
	DiscordToken string
	DatabaseDSN  string
}

type GuildConfig struct {
	ResultsChannelID string `db:"results_channel_id"`
	CommandChannelID string `db:"command_channel_id"`
	AllowedRoleID    string `db:"allowed_role_id"`
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

func respondText(s *discordgo.Session, i *discordgo.InteractionCreate, msg string) {
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
		},
	})
}

func ensureGuildRows(dbx *sqlx.DB, guildStates []*discordgo.Guild) error {
	if len(guildStates) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := dbx.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// Insert guild rows if missing; do not overwrite name.
	// MariaDB: INSERT IGNORE is easiest here.
	stmt := `INSERT IGNORE INTO guilds (discord_guild_id, name) VALUES (?, ?)`
	for _, g := range guildStates {
		if g == nil || g.ID == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, stmt, g.ID, g.Name); err != nil {
			return err
		}
	}

	return tx.Commit()
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

func handleSetup(s *discordgo.Session, i *discordgo.InteractionCreate, dbx *sqlx.DB) {
	// Must be in a server
	if i.GuildID == "" {
		discord.RespondEphemeral(s, i, "This command can only be used in a server.")
		return
	}

	// Require Manage Server or Administrator to run setup
	perms, err := s.UserChannelPermissions(i.Member.User.ID, i.ChannelID)
	if err != nil {
		discord.RespondEphemeral(s, i, "Could not verify permissions.")
		return
	}
	if (perms&discordgo.PermissionManageGuild) == 0 && (perms&discordgo.PermissionAdministrator) == 0 {
		discord.RespondEphemeral(s, i, "You need Manage Server or Administrator permission to run /setup.")
		return
	}

	// Parse options
	var commandChannelID string
	var resultsChannelID string
	var allowedRoleID string

	for _, opt := range i.ApplicationCommandData().Options {
		switch opt.Name {
		case "command_channel":
			commandChannelID = opt.ChannelValue(nil).ID
		case "results_channel":
			resultsChannelID = opt.ChannelValue(nil).ID
		case "allowed_role":
			allowedRoleID = opt.RoleValue(nil, i.GuildID).ID
		}
	}

	if commandChannelID == "" {
		discord.RespondEphemeral(s, i, "command_channel is required.")
		return
	}
	if resultsChannelID == "" {
		resultsChannelID = commandChannelID
	}

	// Fetch guild name (best effort)
	guildName := ""
	if g, err := s.State.Guild(i.GuildID); err == nil && g != nil {
		guildName = g.Name
	} else if g, err := s.Guild(i.GuildID); err == nil && g != nil {
		guildName = g.Name
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := dbx.BeginTxx(ctx, nil)
	if err != nil {
		discord.RespondEphemeral(s, i, "DB error starting transaction.")
		return
	}
	defer func() { _ = tx.Rollback() }()

	// Upsert guild row (keeps latest name if provided)
	_, err = tx.ExecContext(ctx, `
		INSERT INTO guilds (discord_guild_id, name)
		VALUES (?, ?)
		ON DUPLICATE KEY UPDATE
			name = COALESCE(VALUES(name), name)
	`, i.GuildID, nullIfEmpty(guildName))
	if err != nil {
		discord.RespondEphemeral(s, i, "DB error writing guild.")
		return
	}

	// Upsert config row
	_, err = tx.ExecContext(ctx, `
		INSERT INTO config (discord_guild_id, command_channel_id, results_channel_id, allowed_role_id)
		VALUES (?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			command_channel_id = VALUES(command_channel_id),
			results_channel_id = VALUES(results_channel_id),
			allowed_role_id    = VALUES(allowed_role_id)
	`, i.GuildID, commandChannelID, resultsChannelID, nullIfEmpty(allowedRoleID))
	if err != nil {
		discord.RespondEphemeral(s, i, "DB error writing config.")
		return
	}

	if err := tx.Commit(); err != nil {
		discord.RespondEphemeral(s, i, "DB error committing config.")
		return
	}

	// Respond ephemerally with what was set
	msg := "Setup saved.\n" +
		"Command channel: <#" + commandChannelID + ">\n" +
		"Results channel: <#" + resultsChannelID + ">"

	if allowedRoleID != "" {
		msg += "\nAllowed role: <@&" + allowedRoleID + ">"
	} else {
		msg += "\nAllowed role: (none — admin perms required for restricted commands)"
	}

	discord.RespondEphemeral(s, i, msg)
}

func nullIfEmpty(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}
