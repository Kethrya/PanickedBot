-- name: GetMemberByDiscordUserID :one
SELECT id, discord_guild_id, discord_user_id, family_name, display_name,
       class, spec, ap, aap, dp, evasion, dr, drr, 
       accuracy, hp, total_ap, total_aap, meets_cap, is_exception, is_mercenary, is_active, created_at
FROM roster_members 
WHERE discord_guild_id = ? AND discord_user_id = ? AND is_active = 1
LIMIT 1;

-- name: GetMemberByFamilyName :one
SELECT id, discord_guild_id, discord_user_id, family_name, display_name,
       class, spec, ap, aap, dp, evasion, dr, drr, 
       accuracy, hp, total_ap, total_aap, meets_cap, is_exception, is_mercenary, is_active, created_at
FROM roster_members 
WHERE discord_guild_id = ? AND family_name = ? AND is_active = 1
LIMIT 1;

-- name: GetMemberByDiscordUserIDIncludingInactive :one
SELECT id, discord_guild_id, discord_user_id, family_name, display_name,
       class, spec, ap, aap, dp, evasion, dr, drr, 
       accuracy, hp, total_ap, total_aap, meets_cap, is_exception, is_mercenary, is_active, created_at
FROM roster_members 
WHERE discord_guild_id = ? AND discord_user_id = ?
LIMIT 1;

-- name: GetMemberByFamilyNameIncludingInactive :one
SELECT id, discord_guild_id, discord_user_id, family_name, display_name,
       class, spec, ap, aap, dp, evasion, dr, drr, 
       accuracy, hp, total_ap, total_aap, meets_cap, is_exception, is_mercenary, is_active, created_at
FROM roster_members 
WHERE discord_guild_id = ? AND family_name = ?
LIMIT 1;

-- name: GetMemberByID :one
SELECT id, discord_guild_id, discord_user_id, family_name, display_name,
       class, spec, ap, aap, dp, created_at
FROM roster_members 
WHERE id = ? AND discord_guild_id = ? AND is_active = 1
LIMIT 1;

-- name: GetAllActiveMembers :many
SELECT id, discord_guild_id, discord_user_id, family_name, display_name,
       class, spec, ap, aap, dp, evasion, dr, drr, 
       accuracy, hp, total_ap, total_aap, meets_cap, is_exception, is_mercenary, is_active, created_at
FROM roster_members 
WHERE discord_guild_id = ? AND is_active = 1 AND is_mercenary = 0
ORDER BY family_name;

-- name: GetAllActiveMembersForAttendance :many
SELECT id, discord_guild_id, discord_user_id, family_name, display_name,
       class, spec, ap, aap, dp, created_at
FROM roster_members 
WHERE discord_guild_id = ? AND is_active = 1 AND is_mercenary = 0
ORDER BY family_name;

-- name: GetMemberVacationsForAttendance :many
SELECT id, discord_guild_id, roster_member_id, start_date, end_date, reason, created_by_user_id, created_at
FROM member_exceptions
WHERE roster_member_id = ? AND type = 'vacation'
ORDER BY start_date;

-- name: GetMemberWarDates :many
SELECT DISTINCT w.war_date
FROM wars w
JOIN war_lines wl ON w.id = wl.war_id
WHERE w.discord_guild_id = ? 
  AND wl.roster_member_id = ?
  AND w.is_excluded = 0
ORDER BY w.war_date;

-- name: GetMemberTeamIDs :many
SELECT team_id
FROM member_teams
WHERE roster_member_id = ?
ORDER BY assigned_at;

-- name: GetMemberTeamNames :many
SELECT t.display_name
FROM member_teams mt
JOIN teams t ON mt.team_id = t.id
WHERE mt.roster_member_id = ? 
  AND t.discord_guild_id = ?
  AND t.is_active = 1
ORDER BY mt.assigned_at;

-- name: DeleteMemberTeams :exec
DELETE FROM member_teams
WHERE roster_member_id = ?;

-- name: InsertMemberTeam :exec
INSERT INTO member_teams (roster_member_id, team_id)
VALUES (?, ?);

-- name: SetMemberActive :exec
UPDATE roster_members 
SET is_active = ?
WHERE id = ?;

-- name: SetMemberMercenary :exec
UPDATE roster_members 
SET is_mercenary = ?
WHERE id = ?;
