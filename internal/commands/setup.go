package commands

import (
	"github.com/bwmarrin/discordgo"
	"github.com/jmoiron/sqlx"

	"PanickedBot/internal"
	"PanickedBot/internal/db"
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
				Description: "Channel where commands and results will be posted",
				Required:    true,
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
	var officerRoleID string
	var guildMemberRoleID string
	var mercenaryRoleID string

	for _, opt := range i.ApplicationCommandData().Options {
		switch opt.Name {
		case "command_channel":
			commandChannelID = opt.ChannelValue(nil).ID
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

	// Fetch guild name (best effort)
	guildName := ""
	if g, errGet := s.State.Guild(i.GuildID); errGet == nil && g != nil {
		guildName = g.Name
	} else if g, errGet := s.Guild(i.GuildID); errGet == nil && g != nil {
		guildName = g.Name
	}

	// Upsert guild and config
	err = db.UpsertGuildAndConfig(
		dbx,
		i.GuildID,
		guildName,
		commandChannelID,
		internal.NullIfEmptyPtr(officerRoleID),
		internal.NullIfEmptyPtr(guildMemberRoleID),
		internal.NullIfEmptyPtr(mercenaryRoleID),
	)
	if err != nil {
		discord.RespondEphemeral(s, i, "Failed to save configuration. Please try again.")
		return
	}

	// Respond ephemerally with what was set
	msg := "Setup saved.\n" +
		"Command/Results channel: <#" + commandChannelID + ">"

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
