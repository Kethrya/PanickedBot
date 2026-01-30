package commands

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"

	"PanickedBot/internal"
	"PanickedBot/internal/db"
	"PanickedBot/internal/discord"
)

// getDiscordDisplayName fetches the Discord display name for a user ID
func getDiscordDisplayName(s *discordgo.Session, guildID string, userID string) string {
	// Try to get the guild member to fetch their current display name
	guildMember, err := s.GuildMember(guildID, userID)
	if err == nil && guildMember != nil {
		// Priority order: server nickname > global display name > username
		if guildMember.Nick != "" {
			return guildMember.Nick
		} else if guildMember.User != nil {
			if guildMember.User.GlobalName != "" {
				return guildMember.User.GlobalName
			} else if guildMember.User.Username != "" {
				return guildMember.User.Username
			}
		}
	}

	// Fallback to user ID if we can't fetch the member
	return userID
}

func handleUpdateSelf(s *discordgo.Session, i *discordgo.InteractionCreate, dbx *db.DB, cfg *GuildConfig) {
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

	// Validate and normalize class name if provided
	if class != "" {
		normalizedClass, valid := validateClassName(class)
		if !valid {
			discord.RespondEphemeral(s, i, "Invalid class name. Please provide a valid Black Desert Online class.")
			return
		}
		class = normalizedClass
	}

	// Get display name from Discord
	displayName := getDiscordDisplayName(s, i.GuildID, i.Member.User.ID)

	// Get or create member record
	m, err := getOrCreateMember(dbx, i.GuildID, i.Member.User.ID, i.Member.User.Username, "updateself")
	if err != nil {
		discord.RespondEphemeral(s, i, "Failed to update your information. Please try again.")
		return
	}

	// Build update fields including display name
	fields := internal.UpdateFields{
		DisplayName: &displayName,
	}
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

func handleGear(s *discordgo.Session, i *discordgo.InteractionCreate, dbx *db.DB, cfg *GuildConfig) {
	if !hasGuildMemberPermission(i, cfg) {
		discord.RespondEphemeral(s, i, "You need guild member role to use this command.")
		return
	}

	// Parse options
	var targetUser *discordgo.User
	var ap, aap, dp int64

	for _, opt := range i.ApplicationCommandData().Options {
		switch opt.Name {
		case "member":
			targetUser = opt.UserValue(s)
		case "ap":
			ap = opt.IntValue()
		case "aap":
			aap = opt.IntValue()
		case "dp":
			dp = opt.IntValue()
		}
	}

	// Determine if user is updating themselves or another member
	isOfficer := hasOfficerPermission(s, i, cfg)
	var userIDToUpdate string
	var usernameToUpdate string

	if targetUser != nil {
		// User specified a member to update
		// Check if they're trying to update someone else
		if targetUser.ID != i.Member.User.ID && !isOfficer {
			discord.RespondEphemeral(s, i, "Only officers can update another member's gear stats.")
			return
		}
		userIDToUpdate = targetUser.ID
		usernameToUpdate = targetUser.Username
	} else {
		// User is updating their own stats
		userIDToUpdate = i.Member.User.ID
		usernameToUpdate = i.Member.User.Username
	}

	// Validate non-negative values
	if ap < 0 {
		discord.RespondEphemeral(s, i, "AP cannot be negative.")
		return
	}
	if aap < 0 {
		discord.RespondEphemeral(s, i, "AAP cannot be negative.")
		return
	}
	if dp < 0 {
		discord.RespondEphemeral(s, i, "DP cannot be negative.")
		return
	}

	// Get display name from Discord
	displayName := getDiscordDisplayName(s, i.GuildID, userIDToUpdate)

	// Get or create member record
	m, err := getOrCreateMember(dbx, i.GuildID, userIDToUpdate, usernameToUpdate, "gear")
	if err != nil {
		discord.RespondEphemeral(s, i, "Failed to update gear stats. Please try again.")
		return
	}

	// Convert int64 to int for the update fields
	apInt := int(ap)
	aapInt := int(aap)
	dpInt := int(dp)

	// Build update fields including display name
	fields := internal.UpdateFields{
		AP:          &apInt,
		AAP:         &aapInt,
		DP:          &dpInt,
		DisplayName: &displayName,
	}

	err = internal.UpdateMember(dbx, m.ID, fields)
	if err != nil {
		log.Printf("gear update error: %v", err)
		discord.RespondEphemeral(s, i, "Failed to update gear stats. Please try again.")
		return
	}

	// Calculate and display GS
	gs := (apInt+aapInt)/2 + dpInt

	// Create appropriate response message
	var responseMsg string
	if targetUser != nil && targetUser.ID != i.Member.User.ID {
		// Officer updated another member
		responseMsg = fmt.Sprintf("Gear stats updated successfully for %s.\nAP: %d | AAP: %d | DP: %d | GS: %d", displayName, apInt, aapInt, dpInt, gs)
	} else {
		// User updated their own stats
		responseMsg = fmt.Sprintf("Your gear stats have been updated successfully.\nAP: %d | AAP: %d | DP: %d | GS: %d", apInt, aapInt, dpInt, gs)
	}

	discord.RespondText(s, i, responseMsg)
}

func handleUpdateMember(s *discordgo.Session, i *discordgo.InteractionCreate, dbx *db.DB, cfg *GuildConfig) {
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

	// Validate and normalize class name if provided
	if class != "" {
		normalizedClass, valid := validateClassName(class)
		if !valid {
			discord.RespondEphemeral(s, i, "Invalid class name. Please provide a valid Black Desert Online class.")
			return
		}
		class = normalizedClass
	}

	// Get display name from Discord
	displayName := getDiscordDisplayName(s, i.GuildID, targetUser.ID)

	// Get or create member record
	m, err := getOrCreateMember(dbx, i.GuildID, targetUser.ID, targetUser.Username, "updatemember")
	if err != nil {
		discord.RespondEphemeral(s, i, "Failed to update member information. Please try again.")
		return
	}

	// Look up team IDs if team names provided
	var teamIDs []int64
	if teamsStr != "" {
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

			team, err := db.GetTeamByName(dbx, i.GuildID, teamName)
			if err == sql.ErrNoRows {
				discord.RespondEphemeral(s, i, "Team '"+teamName+"' not found.")
				return
			} else if err != nil {
				log.Printf("updatemember team lookup error: %v", err)
				discord.RespondEphemeral(s, i, "Failed to update member information. Please try again.")
				return
			}

			if !team.IsActive {
				discord.RespondEphemeral(s, i, "Team '"+teamName+"' is not active.")
				return
			}

			teamIDs = append(teamIDs, team.ID)
		}
	}

	// Build update fields including display name
	fields := internal.UpdateFields{
		DisplayName: &displayName,
	}
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

func handleInactive(s *discordgo.Session, i *discordgo.InteractionCreate, dbx *db.DB, cfg *GuildConfig) {
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

func handleActive(s *discordgo.Session, i *discordgo.InteractionCreate, dbx *db.DB, cfg *GuildConfig) {
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
