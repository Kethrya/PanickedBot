package internal

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

// Member represents a roster member
type Member struct {
	ID             int64   `db:"id"`
	DiscordGuildID string  `db:"discord_guild_id"`
	DiscordUserID  *string `db:"discord_user_id"`
	FamilyName     string  `db:"family_name"`
	DisplayName    *string `db:"display_name"`
	Class          *string `db:"class"`
	Spec           *string `db:"spec"`
	
	// Combat stats
	AP       *int     `db:"ap"`
	AAP      *int     `db:"aap"`
	DP       *int     `db:"dp"`
	Evasion  *int     `db:"evasion"`
	DR       *int     `db:"dr"`
	DRR      *float64 `db:"drr"`
	Accuracy *int     `db:"accuracy"`
	HP       *int     `db:"hp"`
	TotalAP  *int     `db:"total_ap"`
	TotalAAP *int     `db:"total_aap"`
	
	// Status flags
	MeetsCap    bool `db:"meets_cap"`
	IsException bool `db:"is_exception"`
	IsActive    bool `db:"is_active"`
}

// UpdateFields represents fields that can be updated
type UpdateFields struct {
	FamilyName  *string
	DisplayName *string
	Class       *string
	Spec        *string
	TeamIDs     []int64 // For multiple team assignments
	MeetsCap    *bool
	AP          *int
	AAP         *int
	DP          *int
}

// GetMemberByDiscordUserID retrieves a member by Discord user ID
func GetMemberByDiscordUserID(db *sqlx.DB, guildID, userID string) (*Member, error) {
	var m Member
	err := db.Get(&m, `
		SELECT id, discord_guild_id, discord_user_id, family_name, display_name,
		       class, spec, ap, aap, dp, evasion, dr, drr, 
		       accuracy, hp, total_ap, total_aap, meets_cap, is_exception, is_active
		FROM roster_members 
		WHERE discord_guild_id = ? AND discord_user_id = ? AND is_active = 1
	`, guildID, userID)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// GetMemberByFamilyName retrieves a member by BDO family name
func GetMemberByFamilyName(db *sqlx.DB, guildID, familyName string) (*Member, error) {
	var m Member
	err := db.Get(&m, `
		SELECT id, discord_guild_id, discord_user_id, family_name, display_name,
		       class, spec, ap, aap, dp, evasion, dr, drr, 
		       accuracy, hp, total_ap, total_aap, meets_cap, is_exception, is_active
		FROM roster_members 
		WHERE discord_guild_id = ? AND family_name = ? AND is_active = 1
	`, guildID, familyName)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// GetMemberByDiscordUserIDIncludingInactive retrieves a member by Discord user ID, including inactive members
func GetMemberByDiscordUserIDIncludingInactive(db *sqlx.DB, guildID, userID string) (*Member, error) {
	var m Member
	err := db.Get(&m, `
		SELECT id, discord_guild_id, discord_user_id, family_name, display_name,
		       class, spec, ap, aap, dp, evasion, dr, drr, 
		       accuracy, hp, total_ap, total_aap, meets_cap, is_exception, is_active
		FROM roster_members 
		WHERE discord_guild_id = ? AND discord_user_id = ?
	`, guildID, userID)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// GetMemberByFamilyNameIncludingInactive retrieves a member by BDO family name, including inactive members
func GetMemberByFamilyNameIncludingInactive(db *sqlx.DB, guildID, familyName string) (*Member, error) {
	var m Member
	err := db.Get(&m, `
		SELECT id, discord_guild_id, discord_user_id, family_name, display_name,
		       class, spec, ap, aap, dp, evasion, dr, drr, 
		       accuracy, hp, total_ap, total_aap, meets_cap, is_exception, is_active
		FROM roster_members 
		WHERE discord_guild_id = ? AND family_name = ?
	`, guildID, familyName)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// UpdateMember updates member fields
// Note: TeamIDs is handled separately via AssignMemberToTeams and is not processed here
func UpdateMember(db *sqlx.DB, memberID int64, fields UpdateFields) error {
	updates := []string{}
	args := []interface{}{}
	
	if fields.FamilyName != nil {
		updates = append(updates, "family_name = ?")
		args = append(args, *fields.FamilyName)
	}
	if fields.DisplayName != nil {
		updates = append(updates, "display_name = ?")
		args = append(args, *fields.DisplayName)
	}
	if fields.Class != nil {
		updates = append(updates, "class = ?")
		args = append(args, *fields.Class)
	}
	if fields.Spec != nil {
		updates = append(updates, "spec = ?")
		args = append(args, *fields.Spec)
	}
	if fields.MeetsCap != nil {
		updates = append(updates, "meets_cap = ?")
		args = append(args, *fields.MeetsCap)
	}
	if fields.AP != nil {
		updates = append(updates, "ap = ?")
		args = append(args, *fields.AP)
	}
	if fields.AAP != nil {
		updates = append(updates, "aap = ?")
		args = append(args, *fields.AAP)
	}
	if fields.DP != nil {
		updates = append(updates, "dp = ?")
		args = append(args, *fields.DP)
	}
	
	if len(updates) == 0 {
		return fmt.Errorf("no fields to update")
	}
	
	args = append(args, memberID)
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	_, err := db.ExecContext(ctx, `
		UPDATE roster_members 
		SET `+strings.Join(updates, ", ")+`
		WHERE id = ?
	`, args...)
	
	return err
}

// CreateMember creates a new roster member
func CreateMember(db *sqlx.DB, guildID, discordUserID, familyName string) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	result, err := db.ExecContext(ctx, `
		INSERT INTO roster_members (discord_guild_id, discord_user_id, family_name, is_active)
		VALUES (?, ?, ?, 1)
	`, guildID, discordUserID, familyName)
	
	if err != nil {
		return 0, err
	}
	
	return result.LastInsertId()
}

// SetMemberActive sets the is_active flag for a member
func SetMemberActive(db *sqlx.DB, memberID int64, active bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	_, err := db.ExecContext(ctx, `
		UPDATE roster_members 
		SET is_active = ?
		WHERE id = ?
	`, active, memberID)
	
	return err
}

// GetAllRosterMembers retrieves all active roster members for a guild
func GetAllRosterMembers(db *sqlx.DB, guildID string) ([]Member, error) {
	var members []Member
	err := db.Select(&members, `
		SELECT id, discord_guild_id, discord_user_id, family_name, display_name,
		       class, spec, ap, aap, dp, evasion, dr, drr, 
		       accuracy, hp, total_ap, total_aap, meets_cap, is_exception, is_active
		FROM roster_members 
		WHERE discord_guild_id = ? AND is_active = 1
		ORDER BY family_name
	`, guildID)
	if err != nil {
		return nil, err
	}
	return members, nil
}

// AssignMemberToTeams assigns a member to multiple teams (replaces all existing team assignments)
func AssignMemberToTeams(db *sqlx.DB, memberID int64, teamIDs []int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// Remove all existing team assignments
	_, err = tx.ExecContext(ctx, `
		DELETE FROM member_teams
		WHERE roster_member_id = ?
	`, memberID)
	if err != nil {
		return err
	}

	// Deduplicate team IDs
	if len(teamIDs) > 0 {
		seen := make(map[int64]bool)
		uniqueTeamIDs := []int64{}
		for _, teamID := range teamIDs {
			if !seen[teamID] {
				seen[teamID] = true
				uniqueTeamIDs = append(uniqueTeamIDs, teamID)
			}
		}
		
		// Add new team assignments
		for _, teamID := range uniqueTeamIDs {
			_, err = tx.ExecContext(ctx, `
				INSERT INTO member_teams (roster_member_id, team_id)
				VALUES (?, ?)
			`, memberID, teamID)
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

// GetMemberTeamIDs retrieves all team IDs for a member
func GetMemberTeamIDs(db *sqlx.DB, memberID int64) ([]int64, error) {
	var teamIDs []int64
	err := db.Select(&teamIDs, `
		SELECT team_id
		FROM member_teams
		WHERE roster_member_id = ?
		ORDER BY assigned_at
	`, memberID)
	if err != nil {
		return nil, err
	}
	return teamIDs, nil
}

// GetMemberTeamNames retrieves team names for a member as a comma-separated string
func GetMemberTeamNames(db *sqlx.DB, guildID string, memberID int64) (string, error) {
	var teamNames []string
	err := db.Select(&teamNames, `
		SELECT t.display_name
		FROM member_teams mt
		JOIN teams t ON mt.team_id = t.id
		WHERE mt.roster_member_id = ? 
		  AND t.discord_guild_id = ?
		  AND t.is_active = 1
		ORDER BY mt.assigned_at
	`, memberID, guildID)
	if err != nil {
		return "", err
	}
	if len(teamNames) == 0 {
		return "", nil
	}
	return strings.Join(teamNames, ", "), nil
}
