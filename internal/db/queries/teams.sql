-- name: GetTeamByName :one
SELECT id, code, display_name, is_active
FROM teams
WHERE discord_guild_id = ? AND display_name = ?;

-- name: GetTeamByCodeOrName :one
SELECT id, is_active 
FROM teams
WHERE discord_guild_id = ? AND (code = ? OR display_name = ?);

-- name: CreateTeam :execresult
INSERT INTO teams (discord_guild_id, code, display_name, is_active)
VALUES (?, ?, ?, 1);

-- name: ReactivateTeam :execresult
UPDATE teams
SET is_active = 1
WHERE id = ?;

-- name: DeactivateTeam :execresult
UPDATE teams
SET is_active = 0
WHERE discord_guild_id = ? AND display_name = ? AND is_active = 1;
