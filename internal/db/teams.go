package db

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	sqlcdb "PanickedBot/internal/db/sqlc"
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
func GetTeamByName(db *DB, guildID, teamName string) (*Team, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	row, err := db.Queries.GetTeamByName(ctx, sqlcdb.GetTeamByNameParams{
		DiscordGuildID: guildID,
		DisplayName:    teamName,
	})

	if err != nil {
		return nil, err
	}

	return &Team{
		ID:       int64(row.ID),
		Code:     row.Code,
		Name:     row.DisplayName,
		IsActive: row.IsActive,
	}, nil
}

// CreateTeam creates a new team or reactivates an existing inactive team
// Returns the team ID and a boolean indicating if it was reactivated
func CreateTeam(db *DB, guildID, teamName string) (int64, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Generate code from name (lowercase, replace spaces with underscores)
	code := strings.ToLower(strings.ReplaceAll(teamName, " ", "_"))

	// Check if team already exists
	existingTeam, err := db.Queries.GetTeamByCodeOrName(ctx, sqlcdb.GetTeamByCodeOrNameParams{
		DiscordGuildID: guildID,
		Code:           code,
		DisplayName:    teamName,
	})

	if err == nil {
		// Team exists - check if it's inactive
		if !existingTeam.IsActive {
			// Reactivate the team
			_, err := db.Queries.ReactivateTeam(ctx, existingTeam.ID)
			if err != nil {
				return 0, false, err
			}

			return int64(existingTeam.ID), true, nil
		}
		// Team is already active - return error
		return int64(existingTeam.ID), false, ErrTeamAlreadyExists
	} else if err != sql.ErrNoRows {
		return 0, false, err
	}

	// Team doesn't exist - create it
	result, err := db.Queries.CreateTeam(ctx, sqlcdb.CreateTeamParams{
		DiscordGuildID: guildID,
		Code:           code,
		DisplayName:    teamName,
	})

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
func DeactivateTeam(db *DB, guildID, teamName string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := db.Queries.DeactivateTeam(ctx, sqlcdb.DeactivateTeamParams{
		DiscordGuildID: guildID,
		DisplayName:    teamName,
	})

	if err != nil {
		return false, err
	}

	rowsAffected, _ := result.RowsAffected()
	return rowsAffected > 0, nil
}
