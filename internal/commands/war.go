package commands

import (
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jmoiron/sqlx"

	"PanickedBot/internal/discord"
)

// parseWarCSV parses a CSV file with war data
// First line: date in YYYY-mm-dd format
// Remaining lines: family_name, kills, deaths
func parseWarCSV(content io.Reader) (warDate time.Time, warLines []WarLineData, err error) {
	reader := csv.NewReader(content)
	reader.TrimLeadingSpace = true
	
	// Read first line (date)
	dateRecord, err := reader.Read()
	if err != nil {
		return time.Time{}, nil, fmt.Errorf("failed to read date line: %w", err)
	}
	
	if len(dateRecord) == 0 {
		return time.Time{}, nil, fmt.Errorf("date line is empty")
	}
	
	// Parse date
	warDate, err = time.Parse("2006-01-02", strings.TrimSpace(dateRecord[0]))
	if err != nil {
		return time.Time{}, nil, fmt.Errorf("invalid date format (expected YYYY-mm-dd): %w", err)
	}
	
	// Read remaining lines (war data)
	lineNum := 1
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return time.Time{}, nil, fmt.Errorf("failed to read line %d: %w", lineNum+1, err)
		}
		
		lineNum++
		
		if len(record) < 3 {
			return time.Time{}, nil, fmt.Errorf("line %d: expected 3 fields (family_name, kills, deaths), got %d", lineNum, len(record))
		}
		
		familyName := strings.TrimSpace(record[0])
		if familyName == "" {
			return time.Time{}, nil, fmt.Errorf("line %d: family_name cannot be empty", lineNum)
		}
		
		kills, err := strconv.Atoi(strings.TrimSpace(record[1]))
		if err != nil {
			return time.Time{}, nil, fmt.Errorf("line %d: invalid kills value '%s': %w", lineNum, record[1], err)
		}
		if kills < 0 {
			return time.Time{}, nil, fmt.Errorf("line %d: kills cannot be negative (got %d)", lineNum, kills)
		}
		
		deaths, err := strconv.Atoi(strings.TrimSpace(record[2]))
		if err != nil {
			return time.Time{}, nil, fmt.Errorf("line %d: invalid deaths value '%s': %w", lineNum, record[2], err)
		}
		if deaths < 0 {
			return time.Time{}, nil, fmt.Errorf("line %d: deaths cannot be negative (got %d)", lineNum, deaths)
		}
		
		warLines = append(warLines, WarLineData{
			FamilyName: familyName,
			Kills:      kills,
			Deaths:     deaths,
		})
	}
	
	if len(warLines) == 0 {
		return time.Time{}, nil, fmt.Errorf("no war data found in CSV")
	}
	
	return warDate, warLines, nil
}

// WarLineData represents a single war line entry
type WarLineData struct {
	FamilyName string
	Kills      int
	Deaths     int
}

// createWarFromCSV creates a war entry and associated war lines from CSV data
func createWarFromCSV(db *sqlx.DB, guildID string, requestChannelID string, requestMessageID string, requestedByUserID string, warDate time.Time, warLines []WarLineData) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	
	// Create war_job entry
	jobResult, err := tx.ExecContext(ctx, `
		INSERT INTO war_jobs (discord_guild_id, request_channel_id, request_message_id, 
		                      requested_by_user_id, status, started_at, finished_at)
		VALUES (?, ?, ?, ?, 'done', NOW(), NOW())
	`, guildID, requestChannelID, requestMessageID, requestedByUserID)
	if err != nil {
		return fmt.Errorf("failed to create war job: %w", err)
	}
	
	jobID, err := jobResult.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get job ID: %w", err)
	}
	
	// Create war entry
	warResult, err := tx.ExecContext(ctx, `
		INSERT INTO wars (discord_guild_id, job_id, war_date, label)
		VALUES (?, ?, ?, ?)
	`, guildID, jobID, warDate.Format("2006-01-02"), fmt.Sprintf("CSV Import - %s", warDate.Format("2006-01-02")))
	if err != nil {
		return fmt.Errorf("failed to create war: %w", err)
	}
	
	warID, err := warResult.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get war ID: %w", err)
	}
	
	// Create war_lines entries
	for _, line := range warLines {
		// Try to match the family name to a roster member
		var rosterMemberID sql.NullInt64
		err := tx.GetContext(ctx, &rosterMemberID, `
			SELECT id FROM roster_members
			WHERE discord_guild_id = ? AND family_name = ?
			LIMIT 1
		`, guildID, line.FamilyName)
		
		if err == sql.ErrNoRows {
			// Roster member doesn't exist - create one
			result, err := tx.ExecContext(ctx, `
				INSERT INTO roster_members (discord_guild_id, family_name, is_active)
				VALUES (?, ?, 1)
			`, guildID, line.FamilyName)
			if err != nil {
				return fmt.Errorf("failed to create roster member for '%s': %w", line.FamilyName, err)
			}
			
			newID, err := result.LastInsertId()
			if err != nil {
				return fmt.Errorf("failed to get new roster member ID for '%s': %w", line.FamilyName, err)
			}
			
			rosterMemberID.Int64 = newID
			rosterMemberID.Valid = true
		} else if err != nil {
			return fmt.Errorf("failed to lookup roster member for '%s': %w", line.FamilyName, err)
		}
		
		// Insert war_line
		_, err = tx.ExecContext(ctx, `
			INSERT INTO war_lines (war_id, roster_member_id, ocr_name, kills, deaths, matched_name)
			VALUES (?, ?, ?, ?, ?, ?)
		`, warID, rosterMemberID, line.FamilyName, line.Kills, line.Deaths, line.FamilyName)
		if err != nil {
			return fmt.Errorf("failed to create war line for '%s': %w", line.FamilyName, err)
		}
	}
	
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	return nil
}

func handleAddWar(s *discordgo.Session, i *discordgo.InteractionCreate, dbx *sqlx.DB, cfg *GuildConfig) {
	if !hasOfficerPermission(s, i, cfg) {
		discord.RespondEphemeral(s, i, "You need officer role or admin permission to use this command.")
		return
	}
	
	// Get the attachment
	if len(i.ApplicationCommandData().Resolved.Attachments) == 0 {
		discord.RespondEphemeral(s, i, "Please attach a CSV file with war data.")
		return
	}
	
	// Get the first attachment
	var attachment *discordgo.MessageAttachment
	for _, att := range i.ApplicationCommandData().Resolved.Attachments {
		attachment = att
		break
	}
	
	if attachment == nil {
		discord.RespondEphemeral(s, i, "No attachment found.")
		return
	}
	
	// Check if it's a CSV file
	if !strings.HasSuffix(strings.ToLower(attachment.Filename), ".csv") {
		discord.RespondEphemeral(s, i, "File must be a CSV file (.csv extension).")
		return
	}
	
	// Check file size (limit to 10MB)
	const maxFileSize = 10 * 1024 * 1024 // 10 MB
	if attachment.Size > maxFileSize {
		discord.RespondEphemeral(s, i, "File size exceeds 10MB limit.")
		return
	}
	
	// Validate that the URL is from Discord's CDN
	if !strings.HasPrefix(attachment.URL, "https://cdn.discordapp.com/") && 
	   !strings.HasPrefix(attachment.URL, "https://media.discordapp.net/") {
		log.Printf("addwar: suspicious attachment URL: %s", attachment.URL)
		discord.RespondEphemeral(s, i, "Invalid attachment source.")
		return
	}
	
	// Create HTTP client with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	req, err := http.NewRequestWithContext(ctx, "GET", attachment.URL, nil)
	if err != nil {
		log.Printf("addwar request creation error: %v", err)
		discord.RespondEphemeral(s, i, "Failed to download the CSV file. Please try again.")
		return
	}
	
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	
	// Download the file
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("addwar download error: %v", err)
		discord.RespondEphemeral(s, i, "Failed to download the CSV file. Please try again.")
		return
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		log.Printf("addwar download failed with status: %d", resp.StatusCode)
		discord.RespondEphemeral(s, i, "Failed to download the CSV file. Please try again.")
		return
	}
	
	// Limit the response body size as an additional safety measure
	limitedReader := io.LimitReader(resp.Body, maxFileSize)
	
	// Parse the CSV
	warDate, warLines, err := parseWarCSV(limitedReader)
	if err != nil {
		log.Printf("addwar parse error: %v", err)
		// Truncate error message to avoid exposing too much detail
		errMsg := err.Error()
		if len(errMsg) > 200 {
			errMsg = errMsg[:200] + "..."
		}
		discord.RespondEphemeral(s, i, fmt.Sprintf("Failed to parse CSV file: %s", errMsg))
		return
	}
	
	// Create the war entry
	err = createWarFromCSV(dbx, i.GuildID, i.ChannelID, i.ID, i.Member.User.ID, warDate, warLines)
	if err != nil {
		log.Printf("addwar create error: %v", err)
		discord.RespondEphemeral(s, i, "Failed to create war entry. Please try again.")
		return
	}
	
	discord.RespondText(s, i, fmt.Sprintf("War data imported successfully!\nDate: %s\nEntries: %d", warDate.Format("2006-01-02"), len(warLines)))
}
