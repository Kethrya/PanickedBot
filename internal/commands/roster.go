package commands

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/jmoiron/sqlx"

	"PanickedBot/internal"
	"PanickedBot/internal/discord"
)

// calculateGS calculates Gear Score as (AP+AAP)/2+DP
// Assumes 0 for any nil values
func calculateGS(ap, aap, dp *int) int {
	apVal := 0
	if ap != nil {
		apVal = *ap
	}
	aapVal := 0
	if aap != nil {
		aapVal = *aap
	}
	dpVal := 0
	if dp != nil {
		dpVal = *dp
	}
	return (apVal+aapVal)/2 + dpVal
}

// getDisplayNameForRoster returns the display name for a roster member
// Uses the provided guildMembersMap for efficient lookups, falls back to cached data
func getDisplayNameForRoster(guildMembersMap map[string]*discordgo.Member, member *internal.Member) string {
	// Try to get from the guild members map if we have a user ID
	if member.DiscordUserID != nil && *member.DiscordUserID != "" {
		if guildMember, ok := guildMembersMap[*member.DiscordUserID]; ok {
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
		// If not found in guild members map, fall back to cached display name if available
		if member.DisplayName != nil && *member.DisplayName != "" {
			return *member.DisplayName
		}
		return *member.DiscordUserID
	}

	// If no Discord user ID, try cached display name
	if member.DisplayName != nil && *member.DisplayName != "" {
		return *member.DisplayName
	}

	// Final fallback: use family name
	return member.FamilyName
}

// truncateString truncates a string to maxLen characters, adding "..." if truncated
func truncateString(s string, maxLen int) string {
	// Count runes, not bytes, to avoid splitting multi-byte UTF-8 characters
	runes := []rune(s)
	if len(runes) > maxLen {
		if maxLen > 3 {
			return string(runes[:maxLen-3]) + "..."
		}
		return string(runes[:maxLen])
	}
	return s
}

func handleGetRoster(s *discordgo.Session, i *discordgo.InteractionCreate, dbx *sqlx.DB, cfg *GuildConfig) {
	if !hasOfficerPermission(s, i, cfg) {
		discord.RespondEphemeral(s, i, "You need officer role or admin permission to use this command.")
		return
	}

	// Get all roster members
	members, err := internal.GetAllRosterMembers(dbx, i.GuildID)
	if err != nil {
		log.Printf("getroster error: %v", err)
		discord.RespondEphemeral(s, i, "Failed to retrieve roster members. Please try again.")
		return
	}

	if len(members) == 0 {
		discord.RespondEphemeral(s, i, "No active roster members found.")
		return
	}

	// Batch fetch all guild members to avoid N+1 API calls
	// This fetches all members in the guild efficiently
	guildMembersMap := make(map[string]*discordgo.Member)
	after := ""
	for {
		guildMembers, err := s.GuildMembers(i.GuildID, after, 1000)
		if err != nil {
			log.Printf("getroster: failed to fetch guild members: %v", err)
			// Continue with cached data if Discord API fails
			break
		}
		if len(guildMembers) == 0 {
			break
		}
		for _, gm := range guildMembers {
			guildMembersMap[gm.User.ID] = gm
		}
		if len(guildMembers) < 1000 {
			break
		}
		// For pagination, use the last member's ID as the 'after' parameter
		after = guildMembers[len(guildMembers)-1].User.ID
	}

	// Sort members by GS (higher first)
	sort.Slice(members, func(i, j int) bool {
		gsI := calculateGS(members[i].AP, members[i].AAP, members[i].DP)
		gsJ := calculateGS(members[j].AP, members[j].AAP, members[j].DP)
		return gsI > gsJ
	})

	// Build response message with aligned columns
	var response strings.Builder
	response.WriteString("**Guild Roster Members**\n```\n")

	// Header
	header := fmt.Sprintf("%-20s %-20s %-15s %-12s %6s %-9s\n", "Name", "Family Name", "Class", "Spec", "GS", "Meets Cap")
	response.WriteString(header)
	response.WriteString(strings.Repeat("-", 90) + "\n")

	// Data rows
	for _, member := range members {
		discordName := truncateString(getDisplayNameForRoster(guildMembersMap, &member), 20)

		familyName := truncateString(member.FamilyName, 20)

		class := ""
		if member.Class != nil && *member.Class != "" {
			class = truncateString(*member.Class, 15)
		}

		spec := ""
		if member.Spec != nil && *member.Spec != "" {
			spec = truncateString(*member.Spec, 12)
		}

		gs := calculateGS(member.AP, member.AAP, member.DP)
		gsStr := ""
		if gs > 0 {
			gsStr = fmt.Sprintf("%d", gs)
		}

		meetsCapStr := "false"
		if member.MeetsCap {
			meetsCapStr = "true"
		}

		response.WriteString(fmt.Sprintf("%-20s %-20s %-15s %-12s %6s %-9s\n", discordName, familyName, class, spec, gsStr, meetsCapStr))
	}

	response.WriteString("```")

	// Discord has a 2000 character limit for messages
	responseText := response.String()
	if len(responseText) > 2000 {
		// If too long, show fewer rows
		var truncatedResponse strings.Builder
		truncatedResponse.WriteString("**Guild Roster Members** (showing first entries)\n```\n")
		truncatedResponse.WriteString(header)
		truncatedResponse.WriteString(strings.Repeat("-", 90) + "\n")

		currentLen := truncatedResponse.Len()
		const closingLen = 3 // length of "```"

		for _, member := range members {
			discordName := truncateString(getDisplayNameForRoster(guildMembersMap, &member), 20)

			familyName := truncateString(member.FamilyName, 20)

			class := ""
			if member.Class != nil && *member.Class != "" {
				class = truncateString(*member.Class, 15)
			}

			spec := ""
			if member.Spec != nil && *member.Spec != "" {
				spec = truncateString(*member.Spec, 12)
			}

			gs := calculateGS(member.AP, member.AAP, member.DP)
			gsStr := ""
			if gs > 0 {
				gsStr = fmt.Sprintf("%d", gs)
			}

			meetsCapStr := "false"
			if member.MeetsCap {
				meetsCapStr = "true"
			}

			line := fmt.Sprintf("%-20s %-20s %-15s %-12s %6s %-9s\n", discordName, familyName, class, spec, gsStr, meetsCapStr)

			// Check if adding this line would exceed the limit
			if currentLen+len(line)+closingLen > 1990 {
				break
			}
			truncatedResponse.WriteString(line)
			currentLen += len(line)
		}
		truncatedResponse.WriteString("```")
		discord.RespondText(s, i, truncatedResponse.String())
	} else {
		discord.RespondText(s, i, responseText)
	}
}
