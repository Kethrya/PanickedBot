# Code Review Implementation Summary

This document summarizes all changes made to address the code review comments.

## Critical Issues Fixed

### 1. Database Schema Refactoring
**Problem**: The `bdo_name` column was redundant and caused confusion. The system should use only `family_name` as the unique identifier.

**Solution**:
- Removed `bdo_name` column from `roster_members` table
- Updated unique constraint from `(discord_guild_id, bdo_name)` to `(discord_guild_id, family_name)`
- Updated all SQL queries and Go code to use `family_name` instead of `bdo_name`
- Updated `Member` struct to remove `BDOName` field
- Updated `CreateMember` function to accept `familyName` parameter instead of `bdoName`

**Files Changed**:
- `schema.sql`
- `internal/member.go`
- `internal/commands/war.go`
- `schema_migration_remove_bdo_name.sql` (new migration script)

### 2. War Statistics Aggregation Bug
**Problem**: `GetWarStats` query was counting excluded wars in aggregates. The join to `wars` table with `is_excluded = 0` condition was applied but war_lines data was still counted regardless of whether the joined war was excluded.

**Solution**:
- Changed aggregation to use conditional expressions:
  - `COUNT(DISTINCT CASE WHEN w.id IS NOT NULL THEN w.id END)` for total wars
  - `COALESCE(SUM(CASE WHEN w.id IS NOT NULL THEN wl.kills ELSE 0 END), 0)` for kills
  - `COALESCE(SUM(CASE WHEN w.id IS NOT NULL THEN wl.deaths ELSE 0 END), 0)` for deaths
- This ensures that only non-excluded wars contribute to the statistics

**Files Changed**:
- `internal/commands/warstats.go`

### 3. HTTP Timeout and File Size Validation
**Problem**: `handleAddWar` used `http.Get` without timeout, which could hang indefinitely. No file size validation could lead to memory exhaustion.

**Solution**:
- Created HTTP client with 30-second timeout
- Used `http.NewRequestWithContext` with context timeout
- Added file size check (10MB limit) before download
- Added `io.LimitReader` as additional safety measure
- Truncated error messages to prevent exposing sensitive data

**Files Changed**:
- `internal/commands/war.go`

### 4. Input Validation for Gear Stats
**Problem**: The `/gear` command accepted negative values for ap/aap/dp, which would fail at database insert time with unclear error messages.

**Solution**:
- Added validation in `handleGear` to check for negative values
- Added `MinValue: 0` constraint to Discord command options
- Added `float64Ptr` helper function for option constraints
- Returns clear error message before attempting database operation

**Files Changed**:
- `internal/commands/member.go`
- `internal/commands/commands.go`

### 5. Team ID Deduplication
**Problem**: `AssignMemberToTeams` would fail if duplicate team IDs were provided due to unique constraint on `(roster_member_id, team_id)`.

**Solution**:
- Added deduplication logic using a map before inserting team assignments
- Only inserts unique team IDs

**Files Changed**:
- `internal/member.go`

## Performance Improvements

### 1. Quadratic String Operations in Truncation Loops
**Problem**: Both `/roster` and `/warstats` commands called `truncatedResponse.String()` in each loop iteration to check length, causing O(n²) behavior.

**Solution**:
- Maintain a running `currentLen` counter
- Use `len(line)` to calculate new length
- Only call `String()` once at the end
- Reduced from O(n²) to O(n) complexity

**Files Changed**:
- `internal/commands/roster.go`
- `internal/commands/warstats.go`

### 2. Display Name Caching
**Problem**: `handleGetRoster` made N+1 Discord API calls (one per roster member) to fetch display names, risking rate limits.

**Solution**:
- Added `display_name` column to `roster_members` table
- Update display name whenever member data is updated (gear, updateself, updatemember)
- Use cached display name in roster display
- Fallback to API call only if cached value is missing
- Created `getDisplayNameForRoster` function that prioritizes cached values

**Files Changed**:
- `schema.sql`
- `internal/member.go`
- `internal/commands/member.go`
- `internal/commands/roster.go`

### 3. UTF-8 Safe String Truncation
**Problem**: `truncateString` counted bytes instead of runes, potentially splitting multi-byte UTF-8 characters.

**Solution**:
- Convert string to rune slice
- Count and truncate by runes
- Properly handle multi-byte characters in Discord names

**Files Changed**:
- `internal/commands/roster.go`

## Schema Changes Summary

### Removed Columns:
1. `roster_members.bdo_name` - replaced by `family_name`
2. `roster_members.team_id` - replaced by `member_teams` table
3. `war_lines.team_id` - replaced by `member_teams` table

### Added Columns:
1. `roster_members.display_name VARCHAR(128)` - caches Discord display names

### Modified Constraints:
1. Changed `roster_members` unique key from `uq_roster_guild_name (discord_guild_id, bdo_name)` to `uq_roster_guild_family (discord_guild_id, family_name)`

### Removed Foreign Keys:
1. `fk_roster_team` from `roster_members`
2. `fk_lines_team` from `war_lines`

## Migration Path

For existing databases, run the migration script:
```bash
mysql -u username -p database_name < schema_migration_remove_bdo_name.sql
```

The migration script:
1. Adds `display_name` column
2. Copies `bdo_name` to `family_name` if null
3. Drops old unique constraint
4. Adds new unique constraint
5. Removes `team_id` columns and foreign keys
6. Removes `bdo_name` column

## Code Quality

### Security Scan Results:
- **CodeQL**: 0 alerts found
- **Status**: ✅ PASSED

### Code Review Results:
- **Comments**: 0 issues found
- **Status**: ✅ PASSED

## Testing Recommendations

Before deploying to production:

1. **Test member creation**: Verify family name is required and unique per guild
2. **Test gear updates**: Verify negative values are rejected
3. **Test roster display**: Verify display names are cached and shown correctly
4. **Test war statistics**: Verify excluded wars don't affect totals
5. **Test CSV import**: Verify large files are rejected and timeout works
6. **Test team assignments**: Verify duplicate team IDs are handled
7. **Test migration**: Run on copy of production database first

## MariaDB Compatibility

All SQL statements use MariaDB-compatible syntax:
- `IF NOT EXISTS` / `IF EXISTS` clauses
- `DATETIME(6)` for microsecond precision
- `CURRENT_TIMESTAMP(6)` for defaults
- `COALESCE` and `CASE` expressions
- `BIGINT UNSIGNED` for IDs
- Proper `ENGINE=InnoDB` and charset settings

## Breaking Changes

**WARNING**: This is a breaking change that requires:
1. Database migration
2. All instances of the application to be updated simultaneously
3. No rollback without restoring database backup

After migration, the `bdo_name` column will be permanently removed. Make sure to backup your database before running the migration.
