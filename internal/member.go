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
	ID             int64  `db:"id"`
	DiscordGuildID string `db:"discord_guild_id"`
	DiscordUserID  string `db:"discord_user_id"`
	BDOName        string `db:"bdo_name"`
	FamilyName     string `db:"family_name"`
	Class          string `db:"class"`
	Spec           string `db:"spec"`
	TeamID         *int64 `db:"team_id"`
	
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
	FamilyName *string
	Class      *string
	Spec       *string
	TeamID     *int64
	MeetsCap   *bool
}

// GetMemberByDiscordUserID retrieves a member by Discord user ID
func GetMemberByDiscordUserID(db *sqlx.DB, guildID, userID string) (*Member, error) {
	var m Member
	err := db.Get(&m, `
		SELECT id, discord_guild_id, discord_user_id, bdo_name, family_name, 
		       class, spec, team_id, ap, aap, dp, evasion, dr, drr, 
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
		SELECT id, discord_guild_id, discord_user_id, bdo_name, family_name, 
		       class, spec, team_id, ap, aap, dp, evasion, dr, drr, 
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
		SELECT id, discord_guild_id, discord_user_id, bdo_name, family_name, 
		       class, spec, team_id, ap, aap, dp, evasion, dr, drr, 
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
		SELECT id, discord_guild_id, discord_user_id, bdo_name, family_name, 
		       class, spec, team_id, ap, aap, dp, evasion, dr, drr, 
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
func UpdateMember(db *sqlx.DB, memberID int64, fields UpdateFields) error {
	updates := []string{}
	args := []interface{}{}
	
	if fields.FamilyName != nil {
		updates = append(updates, "family_name = ?")
		args = append(args, *fields.FamilyName)
	}
	if fields.Class != nil {
		updates = append(updates, "class = ?")
		args = append(args, *fields.Class)
	}
	if fields.Spec != nil {
		updates = append(updates, "spec = ?")
		args = append(args, *fields.Spec)
	}
	if fields.TeamID != nil {
		updates = append(updates, "team_id = ?")
		args = append(args, *fields.TeamID)
	}
	if fields.MeetsCap != nil {
		updates = append(updates, "meets_cap = ?")
		args = append(args, *fields.MeetsCap)
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
func CreateMember(db *sqlx.DB, guildID, discordUserID, bdoName string) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	result, err := db.ExecContext(ctx, `
		INSERT INTO roster_members (discord_guild_id, discord_user_id, bdo_name, is_active)
		VALUES (?, ?, ?, 1)
	`, guildID, discordUserID, bdoName)
	
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
