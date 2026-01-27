package db

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

// ErrTeamAlreadyExists is returned when trying to create a team that already exists and is active
var ErrTeamAlreadyExists = errors.New("team already exists and is active")

// Team represents a team in the database
type Team struct {
	ID       int64  `db:"id"`
	Code     string `db:"code"`
	Name     string `db:"display_name"`
	IsActive bool   `db:"is_active"`
}

// GetTeamByName retrieves a team by its display name
func GetTeamByName(db *sqlx.DB, guildID, teamName string) (*Team, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var team Team
	err := db.GetContext(ctx, &team, `
		SELECT id, code, display_name, is_active
		FROM teams
		WHERE discord_guild_id = ? AND display_name = ?
	`, guildID, teamName)

	if err != nil {
		return nil, err
	}

	return &team, nil
}

// CreateTeam creates a new team or reactivates an existing inactive team
// Returns the team ID and a boolean indicating if it was reactivated
func CreateTeam(db *sqlx.DB, guildID, teamName string) (int64, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Generate code from name (lowercase, replace spaces with underscores)
	code := strings.ToLower(strings.ReplaceAll(teamName, " ", "_"))

	// Check if team already exists
	var existingTeam struct {
		ID       int64 `db:"id"`
		IsActive bool  `db:"is_active"`
	}
	err := db.GetContext(ctx, &existingTeam, `
		SELECT id, is_active FROM teams
		WHERE discord_guild_id = ? AND (code = ? OR display_name = ?)
	`, guildID, code, teamName)

	if err == nil {
		// Team exists - check if it's inactive
		if !existingTeam.IsActive {
			// Reactivate the team
			_, err := db.ExecContext(ctx, `
				UPDATE teams
				SET is_active = 1
				WHERE id = ?
			`, existingTeam.ID)

			if err != nil {
				return 0, false, err
			}

			return existingTeam.ID, true, nil
		}
		// Team is already active - return error
		return existingTeam.ID, false, ErrTeamAlreadyExists
	} else if err != sql.ErrNoRows {
		return 0, false, err
	}

	// Team doesn't exist - create it
	result, err := db.ExecContext(ctx, `
		INSERT INTO teams (discord_guild_id, code, display_name, is_active)
		VALUES (?, ?, ?, 1)
	`, guildID, code, teamName)

	if err != nil {
		return 0, false, err
	}

	teamID, err := result.LastInsertId()
	if err != nil {
		return 0, false, err
	}

	return teamID, false, nil
}

// DeactivateTeam marks a team as inactive (soft delete)
func DeactivateTeam(db *sqlx.DB, guildID, teamName string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := db.ExecContext(ctx, `
		UPDATE teams
		SET is_active = 0
		WHERE discord_guild_id = ? AND display_name = ? AND is_active = 1
	`, guildID, teamName)

	if err != nil {
		return false, err
	}

	rowsAffected, _ := result.RowsAffected()
	return rowsAffected > 0, nil
}
