package internal

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"PanickedBot/internal/db"
)

// AttendanceChecker handles attendance checking logic
type AttendanceChecker struct {
	db *db.DB
}

// NewAttendanceChecker creates a new attendance checker
func NewAttendanceChecker(database *db.DB) *AttendanceChecker {
	return &AttendanceChecker{db: database}
}

// WeekPeriod represents a week starting on Sunday
type WeekPeriod struct {
	StartDate time.Time
	EndDate   time.Time
}

// String returns a formatted string representation of the week
func (w WeekPeriod) String() string {
	return fmt.Sprintf("%s to %s", w.StartDate.Format("2006-01-02"), w.EndDate.Format("2006-01-02"))
}

// GetWeekStart returns the start of the week (Sunday) for a given date
func GetWeekStart(date time.Time) time.Time {
	// Get the day of week (0 = Sunday, 1 = Monday, etc.)
	weekday := int(date.Weekday())
	
	// Calculate days to subtract to get to Sunday
	daysToSubtract := weekday
	
	// Subtract to get to the start of the week (Sunday)
	weekStart := date.AddDate(0, 0, -daysToSubtract)
	
	// Zero out the time component
	return time.Date(weekStart.Year(), weekStart.Month(), weekStart.Day(), 0, 0, 0, 0, weekStart.Location())
}

// GetWeekEnd returns the end of the week (Saturday) for a given date
func GetWeekEnd(weekStart time.Time) time.Time {
	// Add 6 days to Sunday to get Saturday
	weekEnd := weekStart.AddDate(0, 0, 6)
	return time.Date(weekEnd.Year(), weekEnd.Month(), weekEnd.Day(), 23, 59, 59, 0, weekEnd.Location())
}

// GetWeekPeriod returns the week period for a given date
func GetWeekPeriod(date time.Time) WeekPeriod {
	start := GetWeekStart(date)
	end := GetWeekEnd(start)
	return WeekPeriod{StartDate: start, EndDate: end}
}

// GetWeekPeriodsBack returns a list of week periods going back N weeks from today
func GetWeekPeriodsBack(weeksBack int) []WeekPeriod {
	weeks := make([]WeekPeriod, 0, weeksBack)
	now := time.Now()
	
	for i := 0; i < weeksBack; i++ {
		// Calculate the date for this week
		weekDate := now.AddDate(0, 0, -7*i)
		week := GetWeekPeriod(weekDate)
		weeks = append(weeks, week)
	}
	
	return weeks
}

// MemberAttendance represents attendance information for a member
type MemberAttendance struct {
	MemberID      int64
	FamilyName    string
	CreatedAt     time.Time
	MissedWeeks   []WeekPeriod
	TotalWeeks    int
	AttendedWeeks int
}

// HasAttendanceIssue returns true if the member has missed any weeks
func (ma MemberAttendance) HasAttendanceIssue() bool {
	return len(ma.MissedWeeks) > 0
}

// CheckMemberAttendance checks attendance for a specific member
func (ac *AttendanceChecker) CheckMemberAttendance(guildID string, memberID int64, weeksBack int) (*MemberAttendance, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get member information
	var member Member
	err := ac.db.Get(&member, `
		SELECT id, discord_guild_id, discord_user_id, family_name, display_name,
		       class, spec, ap, aap, dp, created_at
		FROM roster_members 
		WHERE id = ? AND discord_guild_id = ? AND is_active = 1
	`, memberID, guildID)
	if err != nil {
		return nil, fmt.Errorf("failed to get member: %w", err)
	}

	// Get member's vacations
	vacations, err := ac.getMemberVacations(ctx, memberID)
	if err != nil {
		return nil, fmt.Errorf("failed to get vacations: %w", err)
	}

	// Get member's war participation dates
	warDates, err := ac.getMemberWarDates(ctx, guildID, memberID)
	if err != nil {
		return nil, fmt.Errorf("failed to get war dates: %w", err)
	}

	// Build set of war dates for quick lookup
	warDateSet := make(map[string]bool)
	for _, date := range warDates {
		warDateSet[date.Format("2006-01-02")] = true
	}

	// Calculate which weeks to check
	weeks := GetWeekPeriodsBack(weeksBack)
	missedWeeks := []WeekPeriod{}
	totalWeeks := 0

	for _, week := range weeks {
		// Skip weeks before member was created
		if week.StartDate.Before(member.CreatedAt) {
			continue
		}

		totalWeeks++

		// Check if member is on vacation for the entire week
		if ac.isOnVacationForEntireWeek(week, vacations) {
			continue // Week is excused
		}

		// Check if member participated in any war during this week
		participated := false
		for date := week.StartDate; !date.After(week.EndDate); date = date.AddDate(0, 0, 1) {
			if warDateSet[date.Format("2006-01-02")] {
				participated = true
				break
			}
		}

		if !participated {
			missedWeeks = append(missedWeeks, week)
		}
	}

	return &MemberAttendance{
		MemberID:      memberID,
		FamilyName:    member.FamilyName,
		CreatedAt:     member.CreatedAt,
		MissedWeeks:   missedWeeks,
		TotalWeeks:    totalWeeks,
		AttendedWeeks: totalWeeks - len(missedWeeks),
	}, nil
}

// CheckAllMembersAttendance checks attendance for all active members
func (ac *AttendanceChecker) CheckAllMembersAttendance(guildID string, weeksBack int) ([]MemberAttendance, error) {
	// Get all active members (excluding mercenaries)
	var members []Member
	err := ac.db.Select(&members, `
		SELECT id, discord_guild_id, discord_user_id, family_name, display_name,
		       class, spec, ap, aap, dp, created_at
		FROM roster_members 
		WHERE discord_guild_id = ? AND is_active = 1 AND is_mercenary = 0
		ORDER BY family_name
	`, guildID)
	if err != nil {
		return nil, fmt.Errorf("failed to get members: %w", err)
	}

	results := make([]MemberAttendance, 0)

	for _, member := range members {
		attendance, err := ac.CheckMemberAttendance(guildID, member.ID, weeksBack)
		if err != nil {
			// Log error but continue with other members
			continue
		}

		results = append(results, *attendance)
	}

	return results, nil
}

// getMemberVacations retrieves all vacations for a member
func (ac *AttendanceChecker) getMemberVacations(ctx context.Context, memberID int64) ([]Vacation, error) {
	var vacations []Vacation
	err := ac.db.SelectContext(ctx, &vacations, `
		SELECT id, discord_guild_id, roster_member_id, start_date, end_date, reason, created_by_user_id, created_at
		FROM member_exceptions
		WHERE roster_member_id = ? AND type = 'vacation'
		ORDER BY start_date
	`, memberID)
	if err != nil {
		return nil, err
	}
	return vacations, nil
}

// Vacation represents a vacation period
type Vacation struct {
	ID              int64
	DiscordGuildID  string
	RosterMemberID  int64
	StartDate       time.Time
	EndDate         time.Time
	Reason          sql.NullString
	CreatedByUserID string
	CreatedAt       time.Time
}

// getMemberWarDates retrieves dates when a member participated in wars
func (ac *AttendanceChecker) getMemberWarDates(ctx context.Context, guildID string, memberID int64) ([]time.Time, error) {
	var dates []time.Time
	err := ac.db.SelectContext(ctx, &dates, `
		SELECT DISTINCT w.war_date
		FROM wars w
		JOIN war_lines wl ON w.id = wl.war_id
		WHERE w.discord_guild_id = ? 
		  AND wl.roster_member_id = ?
		  AND w.is_excluded = 0
		ORDER BY w.war_date
	`, guildID, memberID)
	if err != nil {
		return nil, err
	}
	return dates, nil
}

// isOnVacationForEntireWeek checks if a member is on vacation for the entire week
func (ac *AttendanceChecker) isOnVacationForEntireWeek(week WeekPeriod, vacations []Vacation) bool {
	for _, vacation := range vacations {
		// Check if vacation covers the entire week
		// Vacation must start on or before the week start
		// and end on or after the week end
		if !vacation.StartDate.After(week.StartDate) && !vacation.EndDate.Before(week.EndDate) {
			return true
		}
	}
	return false
}
