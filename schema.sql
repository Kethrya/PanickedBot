-- PanickedBot Database Schema
-- Complete schema for creating and configuring the database

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- ============================================================================
-- Core Tables
-- ============================================================================

CREATE TABLE IF NOT EXISTS guilds (
  id                 BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  discord_guild_id   VARCHAR(32) NOT NULL,
  name               VARCHAR(255) NULL,
  created_at         DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  PRIMARY KEY (id),
  UNIQUE KEY uq_guilds_discord_guild_id (discord_guild_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS config (
  discord_guild_id      VARCHAR(32) NOT NULL,
  officer_role_id       VARCHAR(32) NULL COMMENT 'Role required to manage other members information, add/update wars, etc.',
  guild_member_role_id  VARCHAR(32) NULL COMMENT 'Role required for a member to update their own information',
  mercenary_role_id     VARCHAR(32) NULL COMMENT 'Role for mercenary members',
  command_channel_id    VARCHAR(32) NULL COMMENT 'Channel where commands and results are posted',
  timezone              VARCHAR(64) NOT NULL DEFAULT 'America/New_York',
  updated_at            DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (discord_guild_id),
  CONSTRAINT fk_config_guild
    FOREIGN KEY (discord_guild_id) REFERENCES guilds(discord_guild_id)
    ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================================================
-- Teams
-- ============================================================================

CREATE TABLE IF NOT EXISTS teams (
  id               BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  discord_guild_id VARCHAR(32) NOT NULL,
  code             VARCHAR(32) NOT NULL,
  display_name     VARCHAR(64) NOT NULL,
  is_active        TINYINT(1) NOT NULL DEFAULT 1,
  created_at       DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  PRIMARY KEY (id),
  UNIQUE KEY uq_teams_guild_code (discord_guild_id, code),
  UNIQUE KEY uq_teams_guild_display (discord_guild_id, display_name),
  KEY idx_teams_guild_active (discord_guild_id, is_active),
  CONSTRAINT fk_teams_guild
    FOREIGN KEY (discord_guild_id) REFERENCES guilds(discord_guild_id)
    ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================================================
-- Roster Members
-- ============================================================================

CREATE TABLE IF NOT EXISTS roster_members (
  id                BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  discord_guild_id  VARCHAR(32) NOT NULL,
  discord_user_id   VARCHAR(32) NULL COMMENT 'Discord user ID for this member',
  family_name       VARCHAR(128) NOT NULL COMMENT 'BDO family name',
  display_name      VARCHAR(128) NULL COMMENT 'Cached Discord display name',
  class             VARCHAR(64) NULL COMMENT 'BDO class name',
  spec              VARCHAR(32) NULL COMMENT 'Class specialization: succession, awakening, or ascension',
  
  -- Combat stats
  ap                INT UNSIGNED NULL COMMENT 'Attack Power',
  aap               INT UNSIGNED NULL COMMENT 'Awakening Attack Power',
  dp                INT UNSIGNED NULL COMMENT 'Defense Power',
  evasion           INT UNSIGNED NULL COMMENT 'Evasion stat',
  dr                INT UNSIGNED NULL COMMENT 'Damage Reduction',
  drr               DECIMAL(5,2) NULL COMMENT 'Damage Reduction Rate percentage',
  accuracy          INT UNSIGNED NULL COMMENT 'Accuracy stat',
  hp                INT UNSIGNED NULL COMMENT 'HP stat',
  total_ap          INT UNSIGNED NULL COMMENT 'Total Attack Power',
  total_aap         INT UNSIGNED NULL COMMENT 'Total Awakening Attack Power',
  
  -- Status flags
  meets_cap         TINYINT(1) NOT NULL DEFAULT 0 COMMENT 'Officer verified member meets required stat caps',
  is_exception      TINYINT(1) NOT NULL DEFAULT 0 COMMENT 'Member K/D stats excluded from guild overall K/D calculations',
  is_mercenary      TINYINT(1) NOT NULL DEFAULT 0 COMMENT 'Member is a mercenary and excluded from roster',
  is_active         TINYINT(1) NOT NULL DEFAULT 1,
  created_at        DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  updated_at        DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  
  PRIMARY KEY (id),
  UNIQUE KEY uq_roster_guild_family (discord_guild_id, family_name),
  KEY idx_roster_guild_active (discord_guild_id, is_active),
  KEY idx_roster_discord_user (discord_user_id),
  CONSTRAINT fk_roster_guild
    FOREIGN KEY (discord_guild_id) REFERENCES guilds(discord_guild_id)
    ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================================================
-- War Processing
-- ============================================================================

CREATE TABLE IF NOT EXISTS war_jobs (
  id                   BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  discord_guild_id     VARCHAR(32) NOT NULL,
  request_channel_id   VARCHAR(32) NOT NULL,
  request_message_id   VARCHAR(32) NOT NULL,
  requested_by_user_id VARCHAR(32) NOT NULL,
  status               ENUM('queued','processing','done','canceled','error') NOT NULL DEFAULT 'queued',
  error                TEXT NULL,
  created_at           DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  started_at           DATETIME(6) NULL,
  finished_at          DATETIME(6) NULL,
  PRIMARY KEY (id),
  KEY idx_jobs_guild_status_created (discord_guild_id, status, created_at),
  CONSTRAINT fk_jobs_guild
    FOREIGN KEY (discord_guild_id) REFERENCES guilds(discord_guild_id)
    ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS war_job_attachments (
  id                    BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  job_id                BIGINT UNSIGNED NOT NULL,
  idx                   INT NOT NULL,
  discord_attachment_id  VARCHAR(32) NOT NULL,
  filename              VARCHAR(255) NOT NULL,
  content_type          VARCHAR(128) NULL,
  size_bytes            BIGINT UNSIGNED NULL,
  url                   TEXT NOT NULL,
  local_path            VARCHAR(512) NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_job_attachment_order (job_id, idx),
  KEY idx_attach_job (job_id),
  CONSTRAINT fk_attach_job
    FOREIGN KEY (job_id) REFERENCES war_jobs(id)
    ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS wars (
  id               BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  discord_guild_id VARCHAR(32) NOT NULL,
  job_id           BIGINT UNSIGNED NOT NULL,
  war_date         DATE NOT NULL,
  label            VARCHAR(255) NULL,
  result           ENUM('win','lose') NULL COMMENT 'War result: win or lose',
  is_excluded      TINYINT(1) NOT NULL DEFAULT 0,
  created_at       DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  PRIMARY KEY (id),
  UNIQUE KEY uq_wars_job (job_id),
  KEY idx_wars_guild_date (discord_guild_id, war_date),
  KEY idx_wars_guild_excl_date (discord_guild_id, is_excluded, war_date),
  CONSTRAINT fk_wars_guild
    FOREIGN KEY (discord_guild_id) REFERENCES guilds(discord_guild_id)
    ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT fk_wars_job
    FOREIGN KEY (job_id) REFERENCES war_jobs(id)
    ON DELETE RESTRICT ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS war_lines (
  id               BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  war_id           BIGINT UNSIGNED NOT NULL,
  roster_member_id BIGINT UNSIGNED NULL,
  ocr_name         VARCHAR(255) NOT NULL,
  kills            INT NOT NULL,
  deaths           INT NOT NULL,
  matched_name     VARCHAR(128) NULL,
  match_confidence DECIMAL(5,4) NULL,
  class            VARCHAR(64) NULL COMMENT 'BDO class name',
  spec             VARCHAR(32) NULL COMMENT 'Class specialization',
  created_at       DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  PRIMARY KEY (id),
  KEY idx_lines_war (war_id),
  KEY idx_lines_member (roster_member_id),
  CONSTRAINT fk_lines_war
    FOREIGN KEY (war_id) REFERENCES wars(id)
    ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT fk_lines_member
    FOREIGN KEY (roster_member_id) REFERENCES roster_members(id)
    ON DELETE SET NULL ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS member_exceptions (
  id                 BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  discord_guild_id   VARCHAR(32) NOT NULL,
  roster_member_id   BIGINT UNSIGNED NOT NULL,
  type               ENUM('vacation','exclude') NOT NULL,
  start_date         DATE NOT NULL,
  end_date           DATE NOT NULL,
  reason             VARCHAR(255) NULL,
  created_by_user_id VARCHAR(32) NOT NULL,
  created_at         DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  PRIMARY KEY (id),
  KEY idx_exceptions_member_dates (roster_member_id, start_date, end_date),
  KEY idx_exceptions_guild_type_dates (discord_guild_id, type, start_date, end_date),
  CONSTRAINT fk_exceptions_guild
    FOREIGN KEY (discord_guild_id) REFERENCES guilds(discord_guild_id)
    ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT fk_exceptions_member
    FOREIGN KEY (roster_member_id) REFERENCES roster_members(id)
    ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================================================
-- Member-Team Assignments (Many-to-Many)
-- ============================================================================

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
