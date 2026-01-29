-- name: UpsertGuild :exec
INSERT INTO guilds (discord_guild_id, name)
VALUES (?, ?)
ON DUPLICATE KEY UPDATE
    name = COALESCE(VALUES(name), name);

-- name: UpsertConfig :exec
INSERT INTO config (discord_guild_id, command_channel_id,
                    officer_role_id, guild_member_role_id, mercenary_role_id)
VALUES (?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    command_channel_id   = VALUES(command_channel_id),
    officer_role_id      = VALUES(officer_role_id),
    guild_member_role_id = VALUES(guild_member_role_id),
    mercenary_role_id    = VALUES(mercenary_role_id);
