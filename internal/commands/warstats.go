package commands

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

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
			mostRecentStr = stat.MostRecentWar.Format("02-01-06")
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

func handleWarStats(s *discordgo.Session, i *discordgo.InteractionCreate, dbx *db.DB, cfg *GuildConfig) {
	if !hasOfficerPermission(s, i, cfg) {
		discord.RespondEphemeral(s, i, "You need officer role or admin permission to use this command.")
		return
	}

	// Check if a date parameter was provided
	var dateStr string
	options := i.ApplicationCommandData().Options
	if len(options) > 0 {
		dateStr = options[0].StringValue()
	}

	// If date is provided, show stats for that specific war
	if dateStr != "" {
		handleWarStatsByDate(s, i, dbx, dateStr)
		return
	}

	// Otherwise, show stats for all wars (original behavior)
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

// handleWarStatsByDate shows war statistics for a specific date
func handleWarStatsByDate(s *discordgo.Session, i *discordgo.InteractionCreate, dbx *db.DB, dateStr string) {
	// Parse the date in Eastern timezone
	est := getEasternLocation()
	warDate, err := time.ParseInLocation("02-01-06", dateStr, est)
	if err != nil {
		discord.RespondEphemeral(s, i, "Invalid date format. Please use DD-MM-YY format (e.g., 15-01-25).")
		return
	}

	// Get war statistics for this specific date
	stats, err := db.GetWarStatsByDate(dbx, i.GuildID, warDate)
	if err != nil {
		log.Printf("warstats by date error: %v", err)
		discord.RespondEphemeral(s, i, "Failed to retrieve war statistics. Please try again.")
		return
	}

	if len(stats) == 0 {
		discord.RespondEphemeral(s, i, fmt.Sprintf("No war data found for date %s.", dateStr))
		return
	}

	// Calculate totals
	var totalKills, totalDeaths int
	for _, stat := range stats {
		totalKills += stat.Kills
		totalDeaths += stat.Deaths
	}

	// Calculate overall K/D ratio
	var overallKD string
	if totalDeaths > 0 {
		kd := float64(totalKills) / float64(totalDeaths)
		overallKD = fmt.Sprintf("%.2f", kd)
	} else if totalKills > 0 {
		overallKD = fmt.Sprintf("%.2f", float64(totalKills))
	} else {
		overallKD = "0.00"
	}

	// Build response message with aligned columns
	var response strings.Builder
	response.WriteString(fmt.Sprintf("**War Statistics for %s**\n```\n", dateStr))

	// Header
	response.WriteString(fmt.Sprintf("%-20s %10s %10s %10s\n",
		"Family Name", "Kills", "Deaths", "K/D"))
	response.WriteString(strings.Repeat("-", 55) + "\n")

	// Data rows
	for _, stat := range stats {
		familyName := truncateString(stat.FamilyName, 20)
		
		// Calculate K/D ratio for this member
		var kdStr string
		if stat.Deaths > 0 {
			kd := float64(stat.Kills) / float64(stat.Deaths)
			kdStr = fmt.Sprintf("%.2f", kd)
		} else if stat.Kills > 0 {
			kdStr = fmt.Sprintf("%.2f", float64(stat.Kills))
		} else {
			kdStr = "0.00"
		}

		response.WriteString(fmt.Sprintf("%-20s %10d %10d %10s\n",
			familyName, stat.Kills, stat.Deaths, kdStr))
	}

	// Add totals line
	response.WriteString(strings.Repeat("-", 55) + "\n")
	response.WriteString(fmt.Sprintf("%-20s %10d %10d %10s\n",
		"TOTAL", totalKills, totalDeaths, overallKD))

	response.WriteString("```")

	discord.RespondText(s, i, response.String())
}

func handleWarResults(s *discordgo.Session, i *discordgo.InteractionCreate, dbx *db.DB, cfg *GuildConfig) {
	if !hasOfficerPermission(s, i, cfg) {
		discord.RespondEphemeral(s, i, "You need officer role or admin permission to use this command.")
		return
	}

	// Get war results
	results, err := db.GetWarResults(dbx, i.GuildID)
	if err != nil {
		log.Printf("warresults error: %v", err)
		discord.RespondEphemeral(s, i, "Failed to retrieve war results. Please try again.")
		return
	}

	if len(results) == 0 {
		discord.RespondEphemeral(s, i, "No war data found.")
		return
	}

	// Calculate cumulative stats
	var cumulativeKills, cumulativeDeaths int
	for _, result := range results {
		cumulativeKills += result.TotalKills
		cumulativeDeaths += result.TotalDeaths
	}

	// Calculate cumulative K/D ratio
	var cumulativeKD string
	if cumulativeDeaths > 0 {
		kd := float64(cumulativeKills) / float64(cumulativeDeaths)
		cumulativeKD = fmt.Sprintf("%.2f", kd)
	} else if cumulativeKills > 0 {
		cumulativeKD = fmt.Sprintf("%.2f", float64(cumulativeKills))
	} else {
		cumulativeKD = "0.00"
	}

	// Build response message with aligned columns
	var response strings.Builder
	response.WriteString("**War Results**\n```\n")

	// Header
	response.WriteString(fmt.Sprintf("%-15s %8s %10s %10s %10s\n",
		"Date", "Result", "Kills", "Deaths", "K/D"))
	response.WriteString(strings.Repeat("-", 60) + "\n")

	// Data rows
	for _, result := range results {
		dateStr := result.WarDate.Format("02-01-06")
		
		// Format result as W/L or empty
		var resultStr string
		if result.Result == "win" {
			resultStr = "W"
		} else if result.Result == "lose" {
			resultStr = "L"
		} else {
			resultStr = "-"
		}
		
		// Calculate K/D ratio for this war
		var kdStr string
		if result.TotalDeaths > 0 {
			kd := float64(result.TotalKills) / float64(result.TotalDeaths)
			kdStr = fmt.Sprintf("%.2f", kd)
		} else if result.TotalKills > 0 {
			kdStr = fmt.Sprintf("%.2f", float64(result.TotalKills))
		} else {
			kdStr = "0.00"
		}

		response.WriteString(fmt.Sprintf("%-15s %8s %10d %10d %10s\n",
			dateStr, resultStr, result.TotalKills, result.TotalDeaths, kdStr))
	}

	// Add cumulative line
	response.WriteString(strings.Repeat("-", 60) + "\n")
	response.WriteString(fmt.Sprintf("%-15s %8s %10d %10d %10s\n",
		"TOTAL", "", cumulativeKills, cumulativeDeaths, cumulativeKD))

	response.WriteString("```")

	discord.RespondText(s, i, response.String())
}

func handleRemoveWar(s *discordgo.Session, i *discordgo.InteractionCreate, dbx *db.DB, cfg *GuildConfig) {
	if !hasOfficerPermission(s, i, cfg) {
		discord.RespondEphemeral(s, i, "You need officer role or admin permission to use this command.")
		return
	}

	// Get the date parameter
	options := i.ApplicationCommandData().Options
	if len(options) == 0 {
		discord.RespondEphemeral(s, i, "Please provide a date in DD-MM-YY format.")
		return
	}

	dateStr := options[0].StringValue()

	// Parse the date in Eastern timezone
	est := getEasternLocation()
	warDate, err := time.ParseInLocation("02-01-06", dateStr, est)
	if err != nil {
		discord.RespondEphemeral(s, i, "Invalid date format. Please use DD-MM-YY format (e.g., 15-01-25).")
		return
	}

	// Delete the war
	err = db.DeleteWarByDate(dbx, i.GuildID, warDate)
	if err != nil {
		log.Printf("removewar error: %v", err)
		if strings.Contains(err.Error(), "no war found") {
			discord.RespondEphemeral(s, i, fmt.Sprintf("No war found for date %s.", dateStr))
		} else {
			discord.RespondEphemeral(s, i, "Failed to remove war. Please try again.")
		}
		return
	}

	discord.RespondText(s, i, fmt.Sprintf("Successfully removed war data for %s.", dateStr))
}
