package commands

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"

	"PanickedBot/internal"
	"PanickedBot/internal/db"
	"PanickedBot/internal/discord"
)

func handleAttendance(s *discordgo.Session, i *discordgo.InteractionCreate, dbx *db.DB, cfg *GuildConfig) {
	if !hasOfficerPermission(s, i, cfg) {
		discord.RespondEphemeral(s, i, "You need officer role or admin permission to use this command.")
		return
	}

	// Parse options
	var weeksBack int64 = 4 // Default to 4 weeks
	for _, opt := range i.ApplicationCommandData().Options {
		if opt.Name == "weeks" {
			weeksBack = opt.IntValue()
		}
	}

	if weeksBack < 1 {
		discord.RespondEphemeral(s, i, "Weeks must be at least 1.")
		return
	}

	if weeksBack > 52 {
		discord.RespondEphemeral(s, i, "Weeks cannot exceed 52.")
		return
	}

	// Defer response since checking all members can take time
	err := discord.DeferResponse(s, i)
	if err != nil {
		log.Printf("attendance defer error: %v", err)
		return
	}

	// Create attendance checker
	checker := internal.NewAttendanceChecker(dbx)

	// Check all members attendance
	results, err := checker.CheckAllMembersAttendance(i.GuildID, int(weeksBack))
	if err != nil {
		log.Printf("attendance error: %v", err)
		discord.FollowUpEphemeral(s, i, "Failed to check attendance. Please try again.")
		return
	}

	// Filter to only members with issues
	membersWithIssues := make([]internal.MemberAttendance, 0)
	for _, result := range results {
		if result.HasAttendanceIssue() {
			membersWithIssues = append(membersWithIssues, result)
		}
	}

	// Build response message
	var message strings.Builder
	message.WriteString(fmt.Sprintf("**Attendance Report (Last %d weeks)**\n\n", weeksBack))

	if len(membersWithIssues) == 0 {
		message.WriteString("✅ No members have attendance issues!")
	} else {
		message.WriteString(fmt.Sprintf("⚠️ **%d members with attendance issues:**\n\n", len(membersWithIssues)))
		
		for _, result := range membersWithIssues {
			message.WriteString(fmt.Sprintf("**%s**\n", result.FamilyName))
			message.WriteString(fmt.Sprintf("  • Missed %d of %d weeks\n", len(result.MissedWeeks), result.TotalWeeks))
			message.WriteString(fmt.Sprintf("  • Attended %d weeks\n", result.AttendedWeeks))
			
			if len(result.MissedWeeks) <= 5 {
				// Show missed weeks if not too many
				message.WriteString("  • Missed weeks: ")
				weekStrs := make([]string, len(result.MissedWeeks))
				for idx, week := range result.MissedWeeks {
					weekStrs[idx] = week.StartDate.Format("02-01-06")
				}
				message.WriteString(strings.Join(weekStrs, ", "))
				message.WriteString("\n")
			}
			message.WriteString("\n")
		}
	}

	// Send follow-up response
	discord.FollowUpText(s, i, message.String())
}

func handleCheckAttendance(s *discordgo.Session, i *discordgo.InteractionCreate, dbx *db.DB, cfg *GuildConfig) {
	if !hasOfficerPermission(s, i, cfg) {
		discord.RespondEphemeral(s, i, "You need officer role or admin permission to use this command.")
		return
	}

	// Parse options
	var targetUser *discordgo.User
	var familyName string
	var weeksBack int64 = 4 // Default to 4 weeks

	for _, opt := range i.ApplicationCommandData().Options {
		switch opt.Name {
		case "member":
			targetUser = opt.UserValue(s)
		case "family_name":
			familyName = opt.StringValue()
		case "weeks":
			weeksBack = opt.IntValue()
		}
	}

	// Require either member or family_name
	if targetUser == nil && familyName == "" {
		discord.RespondEphemeral(s, i, "Please provide either a Discord member or family name.")
		return
	}

	if weeksBack < 1 {
		discord.RespondEphemeral(s, i, "Weeks must be at least 1.")
		return
	}

	if weeksBack > 52 {
		discord.RespondEphemeral(s, i, "Weeks cannot exceed 52.")
		return
	}

	// Defer response for consistency and to handle potential long operations
	err := discord.DeferResponse(s, i)
	if err != nil {
		log.Printf("checkattendance defer error: %v", err)
		return
	}

	// Get member record
	var member *internal.Member
	var memberErr error

	if targetUser != nil {
		member, memberErr = internal.GetMemberByDiscordUserID(dbx, i.GuildID, targetUser.ID)
	} else {
		member, memberErr = internal.GetMemberByFamilyName(dbx, i.GuildID, familyName)
	}

	if memberErr != nil {
		if errors.Is(memberErr, sql.ErrNoRows) {
			discord.FollowUpEphemeral(s, i, "Member not found.")
			return
		}
		log.Printf("checkattendance lookup error: %v", memberErr)
		discord.FollowUpEphemeral(s, i, "Failed to retrieve member information. Please try again.")
		return
	}

	// Create attendance checker
	checker := internal.NewAttendanceChecker(dbx)

	// Check member attendance
	result, err := checker.CheckMemberAttendance(i.GuildID, member.ID, int(weeksBack))
	if err != nil {
		log.Printf("checkattendance error: %v", err)
		discord.FollowUpEphemeral(s, i, "Failed to check attendance. Please try again.")
		return
	}

	// Build response message
	var message strings.Builder
	message.WriteString(fmt.Sprintf("**Attendance Report for %s**\n", result.FamilyName))
	message.WriteString(fmt.Sprintf("Member since: %s\n\n", formatDateYYMMDD(result.CreatedAt)))
	message.WriteString(fmt.Sprintf("**Last %d weeks:**\n", weeksBack))
	message.WriteString(fmt.Sprintf("• Total weeks: %d\n", result.TotalWeeks))
	message.WriteString(fmt.Sprintf("• Attended: %d weeks\n", result.AttendedWeeks))
	message.WriteString(fmt.Sprintf("• Missed: %d weeks\n\n", len(result.MissedWeeks)))

	if len(result.MissedWeeks) == 0 {
		message.WriteString("✅ No missed weeks!")
	} else {
		message.WriteString("⚠️ **Missed weeks:**\n")
		for _, week := range result.MissedWeeks {
			message.WriteString(fmt.Sprintf("• Week of %s\n", formatDateYYMMDD(week.StartDate)))
		}
	}

	// Send follow-up response
	discord.FollowUpText(s, i, message.String())
}
