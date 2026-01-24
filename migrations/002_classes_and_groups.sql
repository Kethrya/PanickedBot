SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

CREATE TABLE IF NOT EXISTS classes (
  id           SMALLINT UNSIGNED NOT NULL AUTO_INCREMENT,
  code         VARCHAR(32) NOT NULL,
  display_name VARCHAR(64) NOT NULL,
  is_active    TINYINT(1) NOT NULL DEFAULT 1,
  PRIMARY KEY (id),
  UNIQUE KEY uq_classes_code (code),
  UNIQUE KEY uq_classes_display_name (display_name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `groups` (
  id               BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  discord_guild_id VARCHAR(32) NOT NULL,
  code             VARCHAR(32) NOT NULL,
  display_name     VARCHAR(64) NOT NULL,
  is_active        TINYINT(1) NOT NULL DEFAULT 1,
  created_at       DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  PRIMARY KEY (id),
  UNIQUE KEY uq_groups_guild_code (discord_guild_id, code),
  UNIQUE KEY uq_groups_guild_display (discord_guild_id, display_name),
  KEY idx_groups_guild_active (discord_guild_id, is_active),
  CONSTRAINT fk_groups_guild
    FOREIGN KEY (discord_guild_id) REFERENCES guilds(discord_guild_id)
    ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

ALTER TABLE roster_members
  ADD COLUMN class_id SMALLINT UNSIGNED NULL,
  ADD COLUMN group_id BIGINT UNSIGNED NULL,
  ADD KEY idx_roster_class (class_id),
  ADD KEY idx_roster_group (group_id),
  ADD CONSTRAINT fk_roster_class
    FOREIGN KEY (class_id) REFERENCES classes(id)
    ON DELETE SET NULL ON UPDATE CASCADE,
  ADD CONSTRAINT fk_roster_group
    FOREIGN KEY (group_id) REFERENCES `groups`(id)
    ON DELETE SET NULL ON UPDATE CASCADE;

ALTER TABLE war_lines
  ADD COLUMN class_id SMALLINT UNSIGNED NULL,
  ADD COLUMN group_id BIGINT UNSIGNED NULL,
  ADD KEY idx_lines_class (class_id),
  ADD KEY idx_lines_group (group_id),
  ADD CONSTRAINT fk_lines_class
    FOREIGN KEY (class_id) REFERENCES classes(id)
    ON DELETE SET NULL ON UPDATE CASCADE,
  ADD CONSTRAINT fk_lines_group
    FOREIGN KEY (group_id) REFERENCES `groups`(id)
    ON DELETE SET NULL ON UPDATE CASCADE;

INSERT IGNORE INTO classes (code, display_name, is_active) VALUES
  ('archer', 'Archer', 1),
  ('berserker', 'Berserker', 1),
  ('corsair', 'Corsair', 1),
  ('dark_knight', 'Dark Knight', 1),
  ('drakania', 'Drakania', 1),
  ('guardian', 'Guardian', 1),
  ('hashashin', 'Hashashin', 1),
  ('kunoichi', 'Kunoichi', 1),
  ('lahn', 'Lahn', 1),
  ('maegu', 'Maegu', 1),
  ('maehwa', 'Maehwa', 1),
  ('musa', 'Musa', 1),
  ('mystic', 'Mystic', 1),
  ('ninja', 'Ninja', 1),
  ('nova', 'Nova', 1),
  ('ranger', 'Ranger', 1),
  ('sage', 'Sage', 1),
  ('scholar', 'Scholar', 1),
  ('seraph', 'Seraph', 1),
  ('shai', 'Shai', 1),
  ('sorceress', 'Sorceress', 1),
  ('striker', 'Striker', 1),
  ('tamer', 'Tamer', 1),
  ('valkyrie', 'Valkyrie', 1),
  ('warrior', 'Warrior', 1),
  ('witch', 'Witch', 1),
  ('wizard', 'Wizard', 1),
  ('wukong', 'Wukong', 1),
  ('woosa', 'Woosa', 1);


SET FOREIGN_KEY_CHECKS = 1;

