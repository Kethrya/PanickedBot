# PanickedBot

A Discord bot **explicitly designed for tracking Black Desert Online (BDO) guilds and wars**. It helps guilds manage wars, roster members, team assignments, and track war statistics.

## Table of Contents

- [Features](#features)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Building](#building)
- [Configuration](#configuration)
- [Running](#running)
- [Bot Commands](#bot-commands)
- [Development](#development)
- [Contributing](#contributing)
- [License](#license)

## Features

- **Guild War Management**: Import and track war statistics from CSV files or images (using OpenAI vision API)
- **Roster Management**: Manage guild member information, gear stats, and activity status
- **Team Management**: Create and manage teams for organized play
- **War Statistics**: View detailed K/D ratios and participation stats
- **Role-based Permissions**: Configure officer and member roles for different access levels

## Prerequisites

- Go 1.25 or later
- [sqlc](https://sqlc.dev/) for generating type-safe database code
- MySQL/MariaDB database
- Discord Bot Token (from [Discord Developer Portal](https://discord.com/developers/applications))

## Installation

### 1. Clone the Repository

```bash
git clone https://github.com/Kethrya/PanickedBot.git
cd PanickedBot
```

### 2. Install Required Tools

```bash
# Install sqlc
make install-tools

# Or manually:
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
```

### 3. Set Up Database

Create a MySQL/MariaDB database and run the schema:

```bash
mysql -u user -p database < schema.sql
```

### 4. Configure Environment

Set the required environment variables:

```bash
export DISCORD_BOT_TOKEN="your-bot-token"
export DATABASE_DSN="user:password@tcp(localhost:3306)/database?parseTime=true"
export OPENAI_API_KEY="your-openai-api-key"  # Optional, required for image upload in /addwar
```

Or create a `.env` file (not committed to git):

```bash
DISCORD_BOT_TOKEN=your-bot-token
DATABASE_DSN=user:password@tcp(localhost:3306)/database?parseTime=true
OPENAI_API_KEY=your-openai-api-key  # Optional, required for image upload in /addwar
```

## Building

The project uses sqlc to generate type-safe database code from SQL queries. **The generated code is not committed** to the repository and must be generated before building.

### Quick Start (Recommended)

```bash
# Generate sqlc code and build in one step
make

# Or use make all
make all
```

### Step by Step

```bash
# 1. Generate sqlc code from SQL queries
make generate

# 2. Build the binary
make build
```

### Manual Build

```bash
# Generate sqlc code
sqlc generate

# Build the project
go build -o PanickedBot .
```

## Configuration

### Environment Variables

- `DISCORD_BOT_TOKEN` (required) - Your Discord bot token
- `DATABASE_DSN` (required) - MySQL connection string format: `user:password@tcp(host:port)/database?parseTime=true`
- `OPENAI_API_KEY` (optional) - OpenAI API key for image processing in `/addwar` command

### Database Connection String Format

```
username:password@tcp(host:port)/database?parseTime=true
```

Example:
```
root:mypassword@tcp(localhost:3306)/panickedbot?parseTime=true
```

## Running

```bash
./PanickedBot
```

The bot will:
1. Connect to the database
2. Connect to Discord
3. Register all slash commands globally
4. Start listening for interactions

### Command-Line Flags

#### `-deregister`
Deregister all Discord commands and exit. This is useful for cleaning up commands during development or before uninstalling the bot.

**Usage:**
```bash
./PanickedBot -deregister
```

**Note:** This flag only requires the `DISCORD_BOT_TOKEN` environment variable. It does not connect to the database.

## Bot Commands

### Initial Setup

#### `/setup`
**Description:** Configure bot channels and permissions for this server  
**Required Role:** Server Administrator  
**Parameters:**
- `command_channel` (required) - Channel where commands and results will be posted
- `officer_role` (optional) - Role allowed to manage members, wars, etc.
- `guild_member_role` (optional) - Role required for members to update their own information
- `mercenary_role` (optional) - Role for mercenary members

### General Commands

#### `/ping`
**Description:** Health check to verify bot is responsive  
**Required Role:** None

### Member Management

#### `/updateself`
**Description:** Update your own member information  
**Required Role:** Guild Member Role (configured in setup)  
**Parameters:**
- `family_name` (optional) - Your family name in BDO
- `class` (optional) - Your BDO class
- `spec` (optional) - Your class specialization (Succession/Awakening/Ascension)

#### `/gear`
**Description:** Update gear stats (your own or another member's if you're an officer)  
**Required Role:** Guild Member Role (or Officer Role to update others)  
**Parameters:**
- `ap` (required) - Attack Power
- `aap` (required) - Awakening Attack Power
- `dp` (required) - Defense Power
- `member` (optional) - Discord member to update (officers only)

#### `/updatemember`
**Description:** Update another member's information  
**Required Role:** Officer Role  
**Parameters:**
- `member` (required) - Discord member to update
- `family_name` (optional) - Member's family name in BDO
- `class` (optional) - Member's BDO class
- `spec` (optional) - Member's class specialization (Succession/Awakening/Ascension)
- `teams` (optional) - Comma-separated team names to assign
- `meets_cap` (optional) - Whether member meets required stat caps

#### `/active`
**Description:** Mark a member as active  
**Required Role:** Officer Role  
**Parameters:**
- `member` (optional) - Discord member to mark as active
- `family_name` (optional) - Family name of member to mark as active

#### `/inactive`
**Description:** Mark a member as inactive  
**Required Role:** Officer Role  
**Parameters:**
- `member` (optional) - Discord member to mark as inactive
- `family_name` (optional) - Family name of member to mark as inactive

#### `/vacation`
**Description:** Add a vacation period for a member  
**Required Role:** Officer Role  
**Parameters:**
- `member` (required) - Discord member going on vacation
- `start_date` (required) - Vacation start date in DD-MM-YY format (e.g., 25-12-24) in Eastern Time
- `end_date` (required) - Vacation end date in DD-MM-YY format (e.g., 31-12-24) in Eastern Time
- `reason` (optional) - Optional reason for vacation

**Note:** End date must be on or after start date. This helps track member availability during guild wars. All dates are in Eastern Time Zone (America/New_York) to match typical guild war schedules.

#### `/roster`
**Description:** Get all roster member information  
**Required Role:** Officer Role

#### `/link`
**Description:** Link a Discord member to a family name  
**Required Role:** Officer Role  
**Parameters:**
- `member` (required) - Discord member to link
- `family_name` (required) - Family name in BDO to link to the member

**Note:** This command will create a new roster entry if the Discord member doesn't exist in the roster, or update the family name if they already exist. This is useful for quickly associating Discord members with their BDO family names.

#### `/merc`
**Description:** Mark a member as mercenary or not  
**Required Role:** Officer Role  
**Parameters:**
- `member` (required) - Discord member to update
- `is_mercenary` (required) - Whether the member is a mercenary (true/false)

**Note:** Mercenary members are excluded from roster reports and certain statistics.

#### `/attendance`
**Description:** Get all members with attendance problems  
**Required Role:** Officer Role  
**Parameters:**
- `weeks` (optional) - Number of weeks to check (default: 4, max: 52)

**Output:** Displays members who have missed at least one war in the specified time period, excluding weeks covered by vacation. For each member with issues:
- Number of weeks missed
- Number of weeks attended
- List of missed weeks (if 5 or fewer)

**Note:** Attendance tracking only considers weeks after the member was added to the roster. Inactive members are excluded from checks. Week calculations are done in Eastern Time Zone (America/New_York).

#### `/checkattendance`
**Description:** Check attendance for a specific member  
**Required Role:** Officer Role  
**Parameters:**
- `member` (optional) - Discord member to check
- `family_name` (optional) - Family name of member to check
- `weeks` (optional) - Number of weeks to check (default: 4, max: 52)

**Output:** Displays detailed attendance information for the specified member:
- Member creation date
- Total weeks considered
- Number of weeks attended
- Number of weeks missed
- List of all missed weeks

**Note:** Either `member` or `family_name` must be provided. Weeks covered by vacation are not counted as missed. Week calculations are done in Eastern Time Zone (America/New_York).

### Team Management

#### `/addteam`
**Description:** Add a new team  
**Required Role:** Officer Role  
**Parameters:**
- `name` (required) - Team name

#### `/deleteteam`
**Description:** Delete an existing team (soft delete/deactivate)  
**Required Role:** Officer Role  
**Parameters:**
- `name` (required) - Team name to delete

### War Management

#### `/addwar`
**Description:** Import war data from a CSV or image file  
**Required Role:** Officer Role  
**Parameters:**
- `file` (required) - CSV or image file (<5MB for images, <10MB for CSV) with war data
- `result` (required) - War result (Win or Lose)
- `war_type` (required) - Type of war (Node War or Siege)
- `tier` (required) - War tier (Tier 1, Tier 2, or Uncapped)

**CSV Format:**
```
15-01-24
FamilyName1,10,5
FamilyName2,15,8
...
```
- First line: Date in DD-MM-YY format (Eastern Time)
- Following lines: family_name,kills,deaths

**Image Format:**
- Supported formats: PNG, JPG, JPEG, WEBP
- Maximum size: 5MB
- Screenshot should contain:
  - War date at the top in DD-MM-YY format
  - Family names in the leftmost column
  - Kills and deaths in the two rightmost columns
- Requires `OPENAI_API_KEY` environment variable to be set
- Images are automatically saved to the `uploads/` directory with Discord user ID and timestamp

**Note:** All dates are in Eastern Time Zone (America/New_York) to match typical guild war schedules.

#### `/warstats`
**Description:** Get war statistics for all roster members or a specific war date  
**Required Role:** Officer Role  
**Parameters:**
- `date` (optional) - War date in DD-MM-YY format to show stats for that specific war

**Output:** 
- When no date is provided: Displays total wars, most recent war date, kills, deaths, and K/D ratio for each active member across all wars
- When date is provided: Displays kills, deaths, and K/D ratio for each member who participated in that specific war, along with overall totals for the war

**Note:** All dates are in Eastern Time Zone (America/New_York).

#### `/warresults`
**Description:** Get results of all wars from most recent to oldest  
**Required Role:** Officer Role  
**Output:** Displays for each war:
- Date (DD-MM-YY format)
- Result (W for Win, L for Lose)
- Total kills for the guild
- Total deaths for the guild
- K/D ratio for the war
- Cumulative totals (kills, deaths, K/D) at the bottom

#### `/removewar`
**Description:** Remove war data for a specific date  
**Required Role:** Officer Role  
**Parameters:**
- `date` (required) - War date in DD-MM-YY format (e.g., 15-01-25) in Eastern Time

**Note:** This command will remove all war data for the specified date, including all individual member statistics. The operation cannot be undone.

## Development

### Common Tasks

```bash
# Generate sqlc code from SQL queries
make generate

# Build the binary
make build

# Run tests
make test

# Run static analysis
make vet

# Clean generated files and binaries
make clean

# See all available commands
make help
```

### Making Database Changes

When making database changes:

1. Update SQL queries in `internal/db/queries/`
2. Run `make generate` to regenerate Go code
3. **Do not commit** the generated code in `internal/db/sqlc/`
4. The CI will automatically generate the code during builds
5. Test your changes locally before committing

### Continuous Integration

The GitHub Actions CI workflow automatically:
- Installs sqlc
- Generates database code
- Downloads dependencies
- Builds the project
- Runs `go vet` for static analysis
- Runs tests with race detection
- Uploads coverage reports

**For Pull Requests:** The CI tests the merge commit (the result of merging your PR into the base branch) to ensure compatibility and catch any merge conflicts or integration issues before merging.

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Run `make generate` if you modified SQL queries
5. Test your changes (`make test`)
6. Commit your changes (`git commit -m 'Add amazing feature'`)
7. Push to the branch (`git push origin feature/amazing-feature`)
8. Open a Pull Request

**Note:** Do not commit generated files in `internal/db/sqlc/` - they are automatically generated during the build process.

## License

See [LICENSE](LICENSE) file for details.