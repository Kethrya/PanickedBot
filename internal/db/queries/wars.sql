-- name: GetWarStats :many
SELECT 
    rm.family_name,
    CAST(COUNT(DISTINCT CASE WHEN w.id IS NOT NULL THEN w.id END) AS UNSIGNED) as total_wars,
    MAX(CASE WHEN w.id IS NOT NULL THEN w.war_date END) as most_recent_war,
    CAST(COALESCE(SUM(CASE WHEN w.id IS NOT NULL THEN wl.kills ELSE 0 END), 0) AS SIGNED) as total_kills,
    CAST(COALESCE(SUM(CASE WHEN w.id IS NOT NULL THEN wl.deaths ELSE 0 END), 0) AS SIGNED) as total_deaths
FROM roster_members rm
LEFT JOIN war_lines wl ON rm.id = wl.roster_member_id
LEFT JOIN wars w ON wl.war_id = w.id AND w.is_excluded = 0
WHERE rm.discord_guild_id = ? 
  AND rm.is_active = 1
GROUP BY rm.id, rm.family_name
ORDER BY rm.family_name;

-- name: CreateWarJob :execresult
INSERT INTO war_jobs (discord_guild_id, request_channel_id, request_message_id, 
                      requested_by_user_id, status, started_at, finished_at)
VALUES (?, ?, ?, ?, 'done', NOW(), NOW());

-- name: CreateWar :execresult
INSERT INTO wars (discord_guild_id, job_id, war_date, label)
VALUES (?, ?, ?, ?);

-- name: GetRosterMemberByFamilyName :one
SELECT id FROM roster_members
WHERE discord_guild_id = ? AND LOWER(family_name) = LOWER(?)
LIMIT 1;

-- name: CreateRosterMember :execresult
INSERT INTO roster_members (discord_guild_id, family_name, is_active)
VALUES (?, ?, 1);

-- name: CreateWarLine :exec
INSERT INTO war_lines (war_id, roster_member_id, ocr_name, kills, deaths, matched_name)
VALUES (?, ?, ?, ?, ?, ?);
