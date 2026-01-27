package commands

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jmoiron/sqlx"

	"PanickedBot/internal"
	"PanickedBot/internal/discord"
)

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

	// Get or create member record
	m, err := internal.GetMemberByDiscordUserID(dbx, i.GuildID, i.Member.User.ID)
	if err == sql.ErrNoRows {
		// Create new member - use Discord username as default BDO name
		memberID, err := internal.CreateMember(dbx, i.GuildID, i.Member.User.ID, i.Member.User.Username)
		if err != nil {
			log.Printf("updateself create error: %v", err)
			discord.RespondEphemeral(s, i, "Failed to create your member record. Please try again.")
			return
		}

		// Get the newly created member
		m, err = internal.GetMemberByDiscordUserID(dbx, i.GuildID, i.Member.User.ID)
		if err != nil {
			log.Printf("updateself lookup after create error: %v", err)
			discord.RespondEphemeral(s, i, "Failed to update your information. Please try again.")
			return
		}

		log.Printf("Created new member ID %d for user %s", memberID, i.Member.User.Username)
	} else if err != nil {
		log.Printf("updateself lookup error: %v", err)
		discord.RespondEphemeral(s, i, "Failed to update your information. Please try again.")
		return
	}

	// Build update fields
	fields := internal.UpdateFields{}
	if familyName != "" {
		fields.FamilyName = &familyName
	}
	if class != "" {
		fields.Class = &class
	}
	if spec != "" {
		fields.Spec = &spec
	}

	err = internal.UpdateMember(dbx, m.ID, fields)
	if err != nil {
		log.Printf("updateself error: %v", err)
		discord.RespondEphemeral(s, i, "Failed to update your information. Please try again.")
		return
	}

	discord.RespondText(s, i, "Your information has been updated successfully.")
}

func handleGear(s *discordgo.Session, i *discordgo.InteractionCreate, dbx *sqlx.DB, cfg *GuildConfig) {
	if !hasGuildMemberPermission(i, cfg) {
		discord.RespondEphemeral(s, i, "You need guild member role to use this command.")
		return
	}

	// Parse options
	var ap, aap, dp int64

	for _, opt := range i.ApplicationCommandData().Options {
		switch opt.Name {
		case "ap":
			ap = opt.IntValue()
		case "aap":
			aap = opt.IntValue()
		case "dp":
			dp = opt.IntValue()
		}
	}

	// Get or create member record
	m, err := internal.GetMemberByDiscordUserID(dbx, i.GuildID, i.Member.User.ID)
	if err == sql.ErrNoRows {
		// Create new member - use Discord username as default BDO name
		memberID, err := internal.CreateMember(dbx, i.GuildID, i.Member.User.ID, i.Member.User.Username)
		if err != nil {
			log.Printf("gear create error: %v", err)
			discord.RespondEphemeral(s, i, "Failed to create your member record. Please try again.")
			return
		}

		// Get the newly created member
		m, err = internal.GetMemberByDiscordUserID(dbx, i.GuildID, i.Member.User.ID)
		if err != nil {
			log.Printf("gear lookup after create error: %v", err)
			discord.RespondEphemeral(s, i, "Failed to update your gear stats. Please try again.")
			return
		}

		log.Printf("Created new member ID %d for user %s", memberID, i.Member.User.Username)
	} else if err != nil {
		log.Printf("gear lookup error: %v", err)
		discord.RespondEphemeral(s, i, "Failed to update your gear stats. Please try again.")
		return
	}

	// Convert int64 to int for the update fields
	apInt := int(ap)
	aapInt := int(aap)
	dpInt := int(dp)

	// Build update fields
	fields := internal.UpdateFields{
		AP:  &apInt,
		AAP: &aapInt,
		DP:  &dpInt,
	}

	err = internal.UpdateMember(dbx, m.ID, fields)
	if err != nil {
		log.Printf("gear update error: %v", err)
		discord.RespondEphemeral(s, i, "Failed to update your gear stats. Please try again.")
		return
	}

	// Calculate and display GS
	gs := (apInt + aapInt) / 2 + dpInt
	discord.RespondText(s, i, fmt.Sprintf("Your gear stats have been updated successfully.\nAP: %d | AAP: %d | DP: %d | GS: %d", apInt, aapInt, dpInt, gs))
}

func handleUpdateMember(s *discordgo.Session, i *discordgo.InteractionCreate, dbx *sqlx.DB, cfg *GuildConfig) {
	if !hasOfficerPermission(s, i, cfg) {
		discord.RespondEphemeral(s, i, "You need officer role or admin permission to use this command.")
		return
	}

	// Parse options
	var targetUser *discordgo.User
	var familyName, class, spec, teamsStr string
	var meetsCap *bool
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
		case "teams":
			teamsStr = opt.StringValue()
			hasUpdates = true
		case "meets_cap":
			val := opt.BoolValue()
			meetsCap = &val
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

	// Get or create member record
	m, err := internal.GetMemberByDiscordUserID(dbx, i.GuildID, targetUser.ID)
	if err == sql.ErrNoRows {
		// Create new member - use Discord username as default BDO name
		memberID, err := internal.CreateMember(dbx, i.GuildID, targetUser.ID, targetUser.Username)
		if err != nil {
			log.Printf("updatemember create error: %v", err)
			discord.RespondEphemeral(s, i, "Failed to create member record. Please try again.")
			return
		}

		// Get the newly created member
		m, err = internal.GetMemberByDiscordUserID(dbx, i.GuildID, targetUser.ID)
		if err != nil {
			log.Printf("updatemember lookup after create error: %v", err)
			discord.RespondEphemeral(s, i, "Failed to update member information. Please try again.")
			return
		}

		log.Printf("Created new member ID %d for user %s", memberID, targetUser.Username)
	} else if err != nil {
		log.Printf("updatemember lookup error: %v", err)
		discord.RespondEphemeral(s, i, "Failed to update member information. Please try again.")
		return
	}

	// Look up team IDs if team names provided
	var teamIDs []int64
	if teamsStr != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Split comma-separated team names
		teamNames := strings.Split(teamsStr, ",")
		for idx, name := range teamNames {
			teamNames[idx] = strings.TrimSpace(name)
		}

		// Look up each team
		for _, teamName := range teamNames {
			if teamName == "" {
				continue
			}

			var teamData struct {
				ID       int64 `db:"id"`
				IsActive bool  `db:"is_active"`
			}
			err := dbx.GetContext(ctx, &teamData, `
				SELECT id, is_active FROM `+"teams"+`
				WHERE discord_guild_id = ? AND display_name = ?
			`, i.GuildID, teamName)

			if err == sql.ErrNoRows {
				discord.RespondEphemeral(s, i, "Team '"+teamName+"' not found.")
				return
			} else if err != nil {
				log.Printf("updatemember team lookup error: %v", err)
				discord.RespondEphemeral(s, i, "Failed to update member information. Please try again.")
				return
			}

			if !teamData.IsActive {
				discord.RespondEphemeral(s, i, "Team '"+teamName+"' is not active.")
				return
			}

			teamIDs = append(teamIDs, teamData.ID)
		}
	}

	// Build update fields
	fields := internal.UpdateFields{}
	if familyName != "" {
		fields.FamilyName = &familyName
	}
	if class != "" {
		fields.Class = &class
	}
	if spec != "" {
		fields.Spec = &spec
	}
	if len(teamIDs) > 0 {
		fields.TeamIDs = teamIDs
	}
	if meetsCap != nil {
		fields.MeetsCap = meetsCap
	}

	err = internal.UpdateMember(dbx, m.ID, fields)
	if err != nil {
		log.Printf("updatemember error: %v", err)
		discord.RespondEphemeral(s, i, "Failed to update member information. Please try again.")
		return
	}

	// Update team assignments if provided
	if len(teamIDs) > 0 {
		err = internal.AssignMemberToTeams(dbx, m.ID, teamIDs)
		if err != nil {
			log.Printf("updatemember team assignment error: %v", err)
			discord.RespondEphemeral(s, i, "Failed to assign teams. Please try again.")
			return
		}
	}

	discord.RespondText(s, i, "Member information updated successfully.")
}

func handleInactive(s *discordgo.Session, i *discordgo.InteractionCreate, dbx *sqlx.DB, cfg *GuildConfig) {
	if !hasOfficerPermission(s, i, cfg) {
		discord.RespondEphemeral(s, i, "You need officer role or admin permission to use this command.")
		return
	}

	// Parse options
	var targetUser *discordgo.User
	var familyName string

	for _, opt := range i.ApplicationCommandData().Options {
		switch opt.Name {
		case "member":
			targetUser = opt.UserValue(s)
		case "family_name":
			familyName = opt.StringValue()
		}
	}

	// Must provide either member or family_name
	if targetUser == nil && familyName == "" {
		discord.RespondEphemeral(s, i, "Please provide either a Discord member or family name.")
		return
	}

	// Get member record - use functions that include inactive members
	var m *internal.Member
	var err error

	if targetUser != nil {
		m, err = internal.GetMemberByDiscordUserIDIncludingInactive(dbx, i.GuildID, targetUser.ID)
	} else {
		m, err = internal.GetMemberByFamilyNameIncludingInactive(dbx, i.GuildID, familyName)
	}

	if err == sql.ErrNoRows {
		discord.RespondEphemeral(s, i, "Member not found.")
		return
	} else if err != nil {
		log.Printf("inactive lookup error: %v", err)
		discord.RespondEphemeral(s, i, "Failed to mark member as inactive. Please try again.")
		return
	}

	// Set inactive
	err = internal.SetMemberActive(dbx, m.ID, false)
	if err != nil {
		log.Printf("inactive error: %v", err)
		discord.RespondEphemeral(s, i, "Failed to mark member as inactive. Please try again.")
		return
	}

	discord.RespondText(s, i, "Member marked as inactive successfully.")
}

func handleActive(s *discordgo.Session, i *discordgo.InteractionCreate, dbx *sqlx.DB, cfg *GuildConfig) {
	if !hasOfficerPermission(s, i, cfg) {
		discord.RespondEphemeral(s, i, "You need officer role or admin permission to use this command.")
		return
	}

	// Parse options
	var targetUser *discordgo.User
	var familyName string

	for _, opt := range i.ApplicationCommandData().Options {
		switch opt.Name {
		case "member":
			targetUser = opt.UserValue(s)
		case "family_name":
			familyName = opt.StringValue()
		}
	}

	// Must provide either member or family_name
	if targetUser == nil && familyName == "" {
		discord.RespondEphemeral(s, i, "Please provide either a Discord member or family name.")
		return
	}

	// Get member record - use functions that include inactive members
	var m *internal.Member
	var err error

	if targetUser != nil {
		m, err = internal.GetMemberByDiscordUserIDIncludingInactive(dbx, i.GuildID, targetUser.ID)
	} else {
		m, err = internal.GetMemberByFamilyNameIncludingInactive(dbx, i.GuildID, familyName)
	}

	if err == sql.ErrNoRows {
		discord.RespondEphemeral(s, i, "Member not found.")
		return
	} else if err != nil {
		log.Printf("active lookup error: %v", err)
		discord.RespondEphemeral(s, i, "Failed to mark member as active. Please try again.")
		return
	}

	// Set active
	err = internal.SetMemberActive(dbx, m.ID, true)
	if err != nil {
		log.Printf("active error: %v", err)
		discord.RespondEphemeral(s, i, "Failed to mark member as active. Please try again.")
		return
	}

	discord.RespondText(s, i, "Member marked as active successfully.")
}
