package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jmoiron/sqlx"

	"PanickedBot/internal/discord"
)

// BDO class choices
func getClassChoices() []*discordgo.ApplicationCommandOptionChoice {
	classes := []string{
		"Archer", "Berserker", "Corsair", "Dark Knight", "Drakania",
		"Guardian", "Hashashin", "Kunoichi", "Lahn", "Maegu",
		"Maehwa", "Musa", "Mystic", "Ninja", "Nova",
		"Ranger", "Sage", "Scholar", "Seraph", "Shai",
		"Sorceress", "Striker", "Tamer", "Valkyrie", "Warrior",
		"Witch", "Wizard", "Wukong", "Woosa",
	}
	
	choices := make([]*discordgo.ApplicationCommandOptionChoice, len(classes))
	for i, class := range classes {
		choices[i] = &discordgo.ApplicationCommandOptionChoice{
			Name:  class,
			Value: strings.ToLower(class),
		}
	}
	return choices
}

// Spec choices
func getSpecChoices() []*discordgo.ApplicationCommandOptionChoice {
	return []*discordgo.ApplicationCommandOptionChoice{
		{Name: "Succession", Value: "succession"},
		{Name: "Awakening", Value: "awakening"},
		{Name: "Ascension", Value: "ascension"},
	}
}

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

func getCommands() []*discordgo.ApplicationCommand {
	return []*discordgo.ApplicationCommand{
		{Name: "ping", Description: "health check"},
		setupCommand(),
		{
			Name:        "addgroup",
			Description: "Add a new group (officer role required)",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "name",
					Description: "Group name",
					Required:    true,
				},
			},
		},
		{
			Name:        "deletegroup",
			Description: "Delete an existing group (officer role required)",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "name",
					Description: "Group name to delete",
					Required:    true,
				},
			},
		},
		{
			Name:        "updateself",
			Description: "Update your own member information (guild member role required)",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "family_name",
					Description: "Your family name in BDO",
					Required:    false,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "class",
					Description: "Your BDO class",
					Required:    false,
					Choices:     getClassChoices(),
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "spec",
					Description: "Your class specialization",
					Required:    false,
					Choices:     getSpecChoices(),
				},
			},
		},
		{
			Name:        "updatemember",
			Description: "Update another member's information (officer role required)",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "member",
					Description: "Discord member to update",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "family_name",
					Description: "Member's family name in BDO",
					Required:    false,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "class",
					Description: "Member's BDO class",
					Required:    false,
					Choices:     getClassChoices(),
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "spec",
					Description: "Member's class specialization",
					Required:    false,
					Choices:     getSpecChoices(),
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "group",
					Description: "Group name to assign the member to",
					Required:    false,
				},
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
	   nullIfEmpty(officerRoleID), nullIfEmpty(guildMemberRoleID), nullIfEmpty(mercenaryRoleID))
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

// hasOfficerPermission checks if user has officer role or admin permissions
func hasOfficerPermission(s *discordgo.Session, i *discordgo.InteractionCreate, cfg *GuildConfig) bool {
	// Check admin permission first
	perms, err := s.UserChannelPermissions(i.Member.User.ID, i.ChannelID)
	if err == nil && ((perms&discordgo.PermissionManageGuild) != 0 || (perms&discordgo.PermissionAdministrator) != 0) {
		return true
	}
	
	// Check officer role if configured
	if cfg.OfficerRoleID != "" {
		for _, roleID := range i.Member.Roles {
			if roleID == cfg.OfficerRoleID {
				return true
			}
		}
	}
	
	return false
}

// hasGuildMemberPermission checks if user has guild member role
func hasGuildMemberPermission(i *discordgo.InteractionCreate, cfg *GuildConfig) bool {
	if cfg.GuildMemberRoleID == "" {
		return false
	}
	
	for _, roleID := range i.Member.Roles {
		if roleID == cfg.GuildMemberRoleID {
			return true
		}
	}
	
	return false
}

func handleAddGroup(s *discordgo.Session, i *discordgo.InteractionCreate, dbx *sqlx.DB, cfg *GuildConfig) {
	if !hasOfficerPermission(s, i, cfg) {
		discord.RespondEphemeral(s, i, "You need officer role or admin permission to use this command.")
		return
	}
	
	// Parse options
	var groupName string
	for _, opt := range i.ApplicationCommandData().Options {
		if opt.Name == "name" {
			groupName = opt.StringValue()
		}
	}
	
	if groupName == "" {
		discord.RespondEphemeral(s, i, "Group name is required.")
		return
	}
	
	// Generate code from name (lowercase, replace spaces with underscores)
	code := strings.ToLower(strings.ReplaceAll(groupName, " ", "_"))
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	_, err := dbx.ExecContext(ctx, `
		INSERT INTO `+"`groups`"+` (discord_guild_id, code, display_name, is_active)
		VALUES (?, ?, ?, 1)
	`, i.GuildID, code, groupName)
	
	if err != nil {
		if strings.Contains(err.Error(), "Duplicate entry") {
			discord.RespondEphemeral(s, i, "A group with that name already exists.")
		} else {
			log.Printf("add group error: %v", err)
			discord.RespondEphemeral(s, i, "DB error creating group.")
		}
		return
	}
	
	discord.RespondText(s, i, "Group **"+groupName+"** created successfully.")
}

func handleDeleteGroup(s *discordgo.Session, i *discordgo.InteractionCreate, dbx *sqlx.DB, cfg *GuildConfig) {
	if !hasOfficerPermission(s, i, cfg) {
		discord.RespondEphemeral(s, i, "You need officer role or admin permission to use this command.")
		return
	}
	
	// Parse options
	var groupName string
	for _, opt := range i.ApplicationCommandData().Options {
		if opt.Name == "name" {
			groupName = opt.StringValue()
		}
	}
	
	if groupName == "" {
		discord.RespondEphemeral(s, i, "Group name is required.")
		return
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	result, err := dbx.ExecContext(ctx, `
		UPDATE `+"`groups`"+` 
		SET is_active = 0
		WHERE discord_guild_id = ? AND display_name = ? AND is_active = 1
	`, i.GuildID, groupName)
	
	if err != nil {
		log.Printf("delete group error: %v", err)
		discord.RespondEphemeral(s, i, "DB error deleting group.")
		return
	}
	
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		discord.RespondEphemeral(s, i, "Group not found or already deleted.")
		return
	}
	
	discord.RespondText(s, i, "Group **"+groupName+"** deleted successfully.")
}

func handleUpdateSelf(s *discordgo.Session, i *discordgo.InteractionCreate, dbx *sqlx.DB, cfg *GuildConfig) {
	if !hasGuildMemberPermission(i, cfg) {
		discord.RespondEphemeral(s, i, "You need guild member role to use this command.")
		return
	}
	
	// Parse options
	var familyName, class, spec string
	hasUpdates := false
	
	for _, opt := range i.ApplicationCommandData().Options {
		switch opt.Name {
		case "family_name":
			familyName = opt.StringValue()
			hasUpdates = true
		case "class":
			class = opt.StringValue()
			hasUpdates = true
		case "spec":
			spec = opt.StringValue()
			hasUpdates = true
		}
	}
	
	if !hasUpdates {
		discord.RespondEphemeral(s, i, "Please provide at least one field to update.")
		return
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// Check if member exists
	var memberID int64
	err := dbx.GetContext(ctx, &memberID, `
		SELECT id FROM roster_members 
		WHERE discord_guild_id = ? AND discord_user_id = ? AND is_active = 1
	`, i.GuildID, i.Member.User.ID)
	
	if err == sql.ErrNoRows {
		discord.RespondEphemeral(s, i, "You are not registered as a guild member yet. Contact an officer to add you.")
		return
	} else if err != nil {
		log.Printf("updateself lookup error: %v", err)
		discord.RespondEphemeral(s, i, "DB error looking up your member record.")
		return
	}
	
	// Build dynamic update query
	updates := []string{}
	args := []interface{}{}
	
	if familyName != "" {
		updates = append(updates, "family_name = ?")
		args = append(args, familyName)
	}
	if class != "" {
		updates = append(updates, "class = ?")
		args = append(args, class)
	}
	if spec != "" {
		updates = append(updates, "spec = ?")
		args = append(args, spec)
	}
	
	args = append(args, memberID)
	
	_, err = dbx.ExecContext(ctx, `
		UPDATE roster_members 
		SET `+strings.Join(updates, ", ")+`
		WHERE id = ?
	`, args...)
	
	if err != nil {
		log.Printf("updateself error: %v", err)
		discord.RespondEphemeral(s, i, "DB error updating your information.")
		return
	}
	
	discord.RespondText(s, i, "Your information has been updated successfully.")
}

func handleUpdateMember(s *discordgo.Session, i *discordgo.InteractionCreate, dbx *sqlx.DB, cfg *GuildConfig) {
	if !hasOfficerPermission(s, i, cfg) {
		discord.RespondEphemeral(s, i, "You need officer role or admin permission to use this command.")
		return
	}
	
	// Parse options
	var targetUser *discordgo.User
	var familyName, class, spec, groupName string
	hasUpdates := false
	
	for _, opt := range i.ApplicationCommandData().Options {
		switch opt.Name {
		case "member":
			targetUser = opt.UserValue(s)
		case "family_name":
			familyName = opt.StringValue()
			hasUpdates = true
		case "class":
			class = opt.StringValue()
			hasUpdates = true
		case "spec":
			spec = opt.StringValue()
			hasUpdates = true
		case "group":
			groupName = opt.StringValue()
			hasUpdates = true
		}
	}
	
	if targetUser == nil {
		discord.RespondEphemeral(s, i, "Member is required.")
		return
	}
	
	if !hasUpdates {
		discord.RespondEphemeral(s, i, "Please provide at least one field to update.")
		return
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// Check if member exists
	var memberID int64
	err := dbx.GetContext(ctx, &memberID, `
		SELECT id FROM roster_members 
		WHERE discord_guild_id = ? AND discord_user_id = ? AND is_active = 1
	`, i.GuildID, targetUser.ID)
	
	if err == sql.ErrNoRows {
		discord.RespondEphemeral(s, i, "That user is not registered as a guild member yet.")
		return
	} else if err != nil {
		log.Printf("updatemember lookup error: %v", err)
		discord.RespondEphemeral(s, i, "DB error looking up member record.")
		return
	}
	
	// Look up group ID if group name provided
	var groupID *int64
	if groupName != "" {
		var groupData struct {
			ID       int64 `db:"id"`
			IsActive bool  `db:"is_active"`
		}
		err := dbx.GetContext(ctx, &groupData, `
			SELECT id, is_active FROM `+"`groups`"+` 
			WHERE discord_guild_id = ? AND display_name = ?
		`, i.GuildID, groupName)
		
		if err == sql.ErrNoRows {
			discord.RespondEphemeral(s, i, "Group '"+groupName+"' not found.")
			return
		} else if err != nil {
			log.Printf("updatemember group lookup error: %v", err)
			discord.RespondEphemeral(s, i, "DB error looking up group.")
			return
		}
		
		if !groupData.IsActive {
			discord.RespondEphemeral(s, i, "Group '"+groupName+"' is not active.")
			return
		}
		
		groupID = &groupData.ID
	}
	
	// Build dynamic update query
	updates := []string{}
	args := []interface{}{}
	
	if familyName != "" {
		updates = append(updates, "family_name = ?")
		args = append(args, familyName)
	}
	if class != "" {
		updates = append(updates, "class = ?")
		args = append(args, class)
	}
	if spec != "" {
		updates = append(updates, "spec = ?")
		args = append(args, spec)
	}
	if groupID != nil {
		updates = append(updates, "group_id = ?")
		args = append(args, *groupID)
	}
	
	args = append(args, memberID)
	
	_, err = dbx.ExecContext(ctx, `
		UPDATE roster_members 
		SET `+strings.Join(updates, ", ")+`
		WHERE id = ?
	`, args...)
	
	if err != nil {
		log.Printf("updatemember error: %v", err)
		discord.RespondEphemeral(s, i, "DB error updating member information.")
		return
	}
	
	discord.RespondText(s, i, "Member information updated successfully.")
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

		case "addgroup":
			handleAddGroup(s, i, database, cfg)

		case "deletegroup":
			handleDeleteGroup(s, i, database, cfg)

		case "updateself":
			handleUpdateSelf(s, i, database, cfg)

		case "updatemember":
			handleUpdateMember(s, i, database, cfg)

		default:
			discord.RespondEphemeral(s, i, "Unknown command.")
		}

	}
}
