SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

CREATE TABLE IF NOT EXISTS specs (
  id           TINYINT UNSIGNED NOT NULL AUTO_INCREMENT,
  code         VARCHAR(16) NOT NULL,    -- 'succession', 'awakening', 'ascension'
  display_name VARCHAR(32) NOT NULL,    -- 'Succession', 'Awakening', 'Ascension'
  is_active    TINYINT(1) NOT NULL DEFAULT 1,
  PRIMARY KEY (id),
  UNIQUE KEY uq_specs_code (code),
  UNIQUE KEY uq_specs_display_name (display_name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

ALTER TABLE roster_members
  ADD COLUMN spec_id TINYINT UNSIGNED NULL,
  ADD KEY idx_roster_spec (spec_id),
  ADD CONSTRAINT fk_roster_spec
    FOREIGN KEY (spec_id) REFERENCES specs(id)
    ON DELETE SET NULL ON UPDATE CASCADE;

ALTER TABLE war_lines
  ADD COLUMN spec_id TINYINT UNSIGNED NULL,
  ADD KEY idx_lines_spec (spec_id),
  ADD CONSTRAINT fk_lines_spec
    FOREIGN KEY (spec_id) REFERENCES specs(id)
    ON DELETE SET NULL ON UPDATE CASCADE;


INSERT IGNORE INTO specs (code, display_name, is_active) VALUES
  ('succession', 'Succession', 1),
  ('awakening',  'Awakening',  1),
  ('ascension',  'Ascension',  1);

SET FOREIGN_KEY_CHECKS = 1;
