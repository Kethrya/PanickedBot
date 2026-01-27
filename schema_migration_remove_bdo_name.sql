-- Migration script to remove bdo_name and add display_name
-- This script migrates from the old schema to the new schema

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- Step 1: Add display_name column to roster_members if it doesn't exist
ALTER TABLE roster_members 
ADD COLUMN IF NOT EXISTS display_name VARCHAR(128) NULL COMMENT 'Cached Discord display name' 
AFTER family_name;

-- Step 2: Migrate data - copy bdo_name to family_name if family_name is NULL
UPDATE roster_members 
SET family_name = bdo_name 
WHERE family_name IS NULL OR family_name = '';

-- Step 3: Drop the old unique constraint if it exists
ALTER TABLE roster_members DROP INDEX IF EXISTS uq_roster_guild_name;

-- Step 4: Add new unique constraint on (discord_guild_id, family_name)
ALTER TABLE roster_members 
ADD UNIQUE KEY uq_roster_guild_family (discord_guild_id, family_name);

-- Step 5: Remove team_id column from roster_members (replaced by member_teams table)
ALTER TABLE roster_members DROP FOREIGN KEY IF EXISTS fk_roster_team;
ALTER TABLE roster_members DROP INDEX IF EXISTS idx_roster_team;
ALTER TABLE roster_members DROP COLUMN IF EXISTS team_id;

-- Step 6: Remove team_id from war_lines (replaced by member_teams table)
ALTER TABLE war_lines DROP FOREIGN KEY IF EXISTS fk_lines_team;
ALTER TABLE war_lines DROP INDEX IF EXISTS idx_lines_team;
ALTER TABLE war_lines DROP COLUMN IF EXISTS team_id;

-- Step 7: Drop bdo_name column (now using family_name)
ALTER TABLE roster_members DROP COLUMN IF EXISTS bdo_name;

SET FOREIGN_KEY_CHECKS = 1;

-- Note: After running this migration, all roster members will be identified by family_name
-- Make sure to update your application code before running this migration
