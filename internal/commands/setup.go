package commands

import (
	"context"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jmoiron/sqlx"

	"PanickedBot/internal"
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
				Name:        "officer_role",
				Description: "Role allowed to manage members, wars, etc. (leave empty to require admin perms)",
				Required:    false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionRole,
				Name:        "guild_member_role",
				Description: "Role required for members to update their own information",
				Required:    false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionRole,
				Name:        "mercenary_role",
				Description: "Role for mercenary members",
				Required:    false,
			},
		},
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
	var officerRoleID string
	var guildMemberRoleID string
	var mercenaryRoleID string

	for _, opt := range i.ApplicationCommandData().Options {
		switch opt.Name {
		case "command_channel":
			commandChannelID = opt.ChannelValue(nil).ID
		case "results_channel":
			resultsChannelID = opt.ChannelValue(nil).ID
		case "officer_role":
			officerRoleID = opt.RoleValue(nil, i.GuildID).ID
		case "guild_member_role":
			guildMemberRoleID = opt.RoleValue(nil, i.GuildID).ID
		case "mercenary_role":
			mercenaryRoleID = opt.RoleValue(nil, i.GuildID).ID
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
		discord.RespondEphemeral(s, i, "Failed to save configuration. Please try again.")
		return
	}
	defer func() { _ = tx.Rollback() }()

	// Upsert guild row (keeps latest name if provided)
	_, err = tx.ExecContext(ctx, `
		INSERT INTO guilds (discord_guild_id, name)
		VALUES (?, ?)
		ON DUPLICATE KEY UPDATE
			name = COALESCE(VALUES(name), name)
	`, i.GuildID, internal.NullIfEmpty(guildName))
	if err != nil {
		discord.RespondEphemeral(s, i, "Failed to save configuration. Please try again.")
		return
	}

	// Upsert config row
	_, err = tx.ExecContext(ctx, `
		INSERT INTO config (discord_guild_id, command_channel_id, results_channel_id,
		                    officer_role_id, guild_member_role_id, mercenary_role_id)
		VALUES (?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			command_channel_id   = VALUES(command_channel_id),
			results_channel_id   = VALUES(results_channel_id),
			officer_role_id      = VALUES(officer_role_id),
			guild_member_role_id = VALUES(guild_member_role_id),
			mercenary_role_id    = VALUES(mercenary_role_id)
	`, i.GuildID, commandChannelID, resultsChannelID,
		internal.NullIfEmpty(officerRoleID), internal.NullIfEmpty(guildMemberRoleID), internal.NullIfEmpty(mercenaryRoleID))
	if err != nil {
		discord.RespondEphemeral(s, i, "Failed to save configuration. Please try again.")
		return
	}

	if err := tx.Commit(); err != nil {
		discord.RespondEphemeral(s, i, "Failed to save configuration. Please try again.")
		return
	}

	// Respond ephemerally with what was set
	msg := "Setup saved.\n" +
		"Command channel: <#" + commandChannelID + ">\n" +
		"Results channel: <#" + resultsChannelID + ">"

	if officerRoleID != "" {
		msg += "\nOfficer role: <@&" + officerRoleID + ">"
	} else {
		msg += "\nOfficer role: (none — admin perms required for officer commands)"
	}

	if guildMemberRoleID != "" {
		msg += "\nGuild member role: <@&" + guildMemberRoleID + ">"
	}

	if mercenaryRoleID != "" {
		msg += "\nMercenary role: <@&" + mercenaryRoleID + ">"
	}

	discord.RespondEphemeral(s, i, msg)
}
