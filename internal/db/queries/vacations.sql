-- name: CreateVacation :execresult
INSERT INTO member_exceptions (
    discord_guild_id,
    roster_member_id,
    type,
    start_date,
    end_date,
    reason,
    created_by_user_id
) VALUES (?, ?, 'vacation', ?, ?, ?, ?);

-- name: GetMemberVacations :many
SELECT id, discord_guild_id, roster_member_id, type, start_date, end_date, reason, created_by_user_id, created_at
FROM member_exceptions
WHERE roster_member_id = ? AND type = 'vacation'
ORDER BY start_date DESC;

-- name: DeleteVacation :exec
DELETE FROM member_exceptions
WHERE id = ? AND type = 'vacation';

-- name: GetActiveVacationsForGuild :many
SELECT me.id, me.roster_member_id, me.start_date, me.end_date, me.reason, rm.family_name
FROM member_exceptions me
JOIN roster_members rm ON me.roster_member_id = rm.id
WHERE me.discord_guild_id = ? 
  AND me.type = 'vacation'
  AND me.end_date >= CURDATE()
ORDER BY me.start_date;
