package db

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
)

// GuildConfig represents guild configuration
type GuildConfig struct {
	CommandChannelID  string  `db:"command_channel_id"`
	OfficerRoleID     *string `db:"officer_role_id"`
	GuildMemberRoleID *string `db:"guild_member_role_id"`
	MercenaryRoleID   *string `db:"mercenary_role_id"`
}

// UpsertGuildAndConfig creates or updates guild and configuration in a transaction
func UpsertGuildAndConfig(db *sqlx.DB, guildID, guildName, commandChannelID string, officerRoleID, guildMemberRoleID, mercenaryRoleID *string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// Prepare guild name value (nil if empty)
	var guildNameValue interface{}
	if guildName == "" {
		guildNameValue = nil
	} else {
		guildNameValue = guildName
	}

	// Upsert guild row (keeps latest name if provided)
	_, err = tx.ExecContext(ctx, `
		INSERT INTO guilds (discord_guild_id, name)
		VALUES (?, ?)
		ON DUPLICATE KEY UPDATE
			name = COALESCE(VALUES(name), name)
	`, guildID, guildNameValue)
	if err != nil {
		return err
	}

	// Upsert config row
	_, err = tx.ExecContext(ctx, `
		INSERT INTO config (discord_guild_id, command_channel_id,
		                    officer_role_id, guild_member_role_id, mercenary_role_id)
		VALUES (?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			command_channel_id   = VALUES(command_channel_id),
			officer_role_id      = VALUES(officer_role_id),
			guild_member_role_id = VALUES(guild_member_role_id),
			mercenary_role_id    = VALUES(mercenary_role_id)
	`, guildID, commandChannelID, officerRoleID, guildMemberRoleID, mercenaryRoleID)
	if err != nil {
		return err
	}

	return tx.Commit()
}
