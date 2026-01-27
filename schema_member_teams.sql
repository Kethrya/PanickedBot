-- Migration to support multiple team assignments per member
-- This creates a junction table to replace the single team_id in roster_members

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- Create the junction table for member-team assignments
CREATE TABLE IF NOT EXISTS member_teams (
  id                BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  roster_member_id  BIGINT UNSIGNED NOT NULL,
  team_id           BIGINT UNSIGNED NOT NULL,
  assigned_at       DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  
  PRIMARY KEY (id),
  UNIQUE KEY uq_member_teams (roster_member_id, team_id),
  KEY idx_member_teams_member (roster_member_id),
  KEY idx_member_teams_team (team_id),
  CONSTRAINT fk_member_teams_member
    FOREIGN KEY (roster_member_id) REFERENCES roster_members(id)
    ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT fk_member_teams_team
    FOREIGN KEY (team_id) REFERENCES teams(id)
    ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

SET FOREIGN_KEY_CHECKS = 1;
