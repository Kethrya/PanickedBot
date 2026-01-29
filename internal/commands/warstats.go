package commands

import (
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/jmoiron/sqlx"

	"PanickedBot/internal/db"
	"PanickedBot/internal/discord"
)

// formatWarStatLine formats a single war stat entry as a string
func formatWarStatLine(stat db.WarStats) string {
	familyName := truncateString(stat.FamilyName, 20)

	totalWarsStr := "N/A"
	mostRecentStr := "N/A"
	killsStr := "N/A"
	deathsStr := "N/A"
	kdStr := "N/A"

	if stat.TotalWars > 0 {
		totalWarsStr = fmt.Sprintf("%d", stat.TotalWars)
		killsStr = fmt.Sprintf("%d", stat.TotalKills)
		deathsStr = fmt.Sprintf("%d", stat.TotalDeaths)

		if stat.MostRecentWar != nil {
			mostRecentStr = stat.MostRecentWar.Format("2006-01-02")
		}

		// Calculate K/D ratio
		if stat.TotalDeaths > 0 {
			kd := float64(stat.TotalKills) / float64(stat.TotalDeaths)
			kdStr = fmt.Sprintf("%.2f", kd)
		} else if stat.TotalKills > 0 {
			kdStr = fmt.Sprintf("%.2f", float64(stat.TotalKills))
		} else {
			kdStr = "0.00"
		}
	}

	return fmt.Sprintf("%-20s %12s %-15s %8s %8s %8s\n",
		familyName, totalWarsStr, mostRecentStr, killsStr, deathsStr, kdStr)
}

func handleWarStats(s *discordgo.Session, i *discordgo.InteractionCreate, dbx *sqlx.DB, cfg *GuildConfig) {
	if !hasOfficerPermission(s, i, cfg) {
		discord.RespondEphemeral(s, i, "You need officer role or admin permission to use this command.")
		return
	}

	// Get war statistics
	stats, err := db.GetWarStats(dbx, i.GuildID)
	if err != nil {
		log.Printf("warstats error: %v", err)
		discord.RespondEphemeral(s, i, "Failed to retrieve war statistics. Please try again.")
		return
	}

	if len(stats) == 0 {
		discord.RespondEphemeral(s, i, "No active roster members found.")
		return
	}

	// Build response message with aligned columns
	var response strings.Builder
	response.WriteString("**War Statistics**\n```\n")

	// Header
	response.WriteString(fmt.Sprintf("%-20s %12s %-15s %8s %8s %8s\n",
		"Family Name", "Total Wars", "Most Recent", "Kills", "Deaths", "K/D"))
	response.WriteString(strings.Repeat("-", 85) + "\n")

	// Data rows
	for _, stat := range stats {
		response.WriteString(formatWarStatLine(stat))
	}

	response.WriteString("```")

	// Discord has a 2000 character limit for messages
	responseText := response.String()
	if len(responseText) > 2000 {
		// If too long, show fewer rows
		var truncatedResponse strings.Builder
		truncatedResponse.WriteString("**War Statistics** (showing first entries)\n```\n")
		header := fmt.Sprintf("%-20s %12s %-15s %8s %8s %8s\n",
			"Family Name", "Total Wars", "Most Recent", "Kills", "Deaths", "K/D")
		truncatedResponse.WriteString(header)
		truncatedResponse.WriteString(strings.Repeat("-", 85) + "\n")

		currentLen := truncatedResponse.Len()
		const closingLen = 3 // length of "```"

		for _, stat := range stats {
			line := formatWarStatLine(stat)

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
