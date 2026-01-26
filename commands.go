package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jmoiron/sqlx"

	"PanickedBot/internal/discord"
)

func setupCommand() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
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
}

func getCommands() []*discordgo.ApplicationCommand {
	return []*discordgo.ApplicationCommand{
		{Name: "ping", Description: "health check"},
		setupCommand(),
	}
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

func createInteractionHandler(database *sqlx.DB) func(s *discordgo.Session, i *discordgo.InteractionCreate) {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Type != discordgo.InteractionApplicationCommand {
			return
		}

		cmdName := i.ApplicationCommandData().Name

		if cmdName == "setup" {
			handleSetup(s, i, database)
			return
		}

		if i.GuildID == "" {
			discord.RespondText(s, i, "This bot only works in servers.")
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

	}
}
