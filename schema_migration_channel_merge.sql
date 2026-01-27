-- Migration to merge command_channel and results_channel
-- This removes the results_channel_id column from config table

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- Remove results_channel_id column from config table
ALTER TABLE config DROP COLUMN IF EXISTS results_channel_id;

SET FOREIGN_KEY_CHECKS = 1;

-- Note: The team_id column in roster_members table is kept for backward compatibility
-- but is no longer used. Team assignments are now managed through the member_teams junction table.
