package internal

import (
	"testing"
	"time"
)

func TestGetWeekStart(t *testing.T) {
	tests := []struct {
		name     string
		date     time.Time
		expected time.Time
	}{
		{
			name:     "Sunday",
			date:     time.Date(2026, 1, 11, 15, 30, 0, 0, time.UTC),
			expected: time.Date(2026, 1, 11, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "Monday",
			date:     time.Date(2026, 1, 12, 15, 30, 0, 0, time.UTC),
			expected: time.Date(2026, 1, 11, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "Saturday",
			date:     time.Date(2026, 1, 17, 15, 30, 0, 0, time.UTC),
			expected: time.Date(2026, 1, 11, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "Wednesday",
			date:     time.Date(2026, 1, 14, 15, 30, 0, 0, time.UTC),
			expected: time.Date(2026, 1, 11, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetWeekStart(tt.date)
			if !result.Equal(tt.expected) {
				t.Errorf("GetWeekStart(%v) = %v, want %v", tt.date, result, tt.expected)
			}
		})
	}
}

func TestGetWeekEnd(t *testing.T) {
	tests := []struct {
		name      string
		weekStart time.Time
		expected  time.Time
	}{
		{
			name:      "Week of Jan 11, 2026",
			weekStart: time.Date(2026, 1, 11, 0, 0, 0, 0, time.UTC),
			expected:  time.Date(2026, 1, 17, 23, 59, 59, 0, time.UTC),
		},
		{
			name:      "Week of Jan 18, 2026",
			weekStart: time.Date(2026, 1, 18, 0, 0, 0, 0, time.UTC),
			expected:  time.Date(2026, 1, 24, 23, 59, 59, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetWeekEnd(tt.weekStart)
			if !result.Equal(tt.expected) {
				t.Errorf("GetWeekEnd(%v) = %v, want %v", tt.weekStart, result, tt.expected)
			}
		})
	}
}

func TestGetWeekPeriod(t *testing.T) {
	tests := []struct {
		name          string
		date          time.Time
		expectedStart time.Time
		expectedEnd   time.Time
	}{
		{
			name:          "Monday Jan 12, 2026",
			date:          time.Date(2026, 1, 12, 15, 30, 0, 0, time.UTC),
			expectedStart: time.Date(2026, 1, 11, 0, 0, 0, 0, time.UTC),
			expectedEnd:   time.Date(2026, 1, 17, 23, 59, 59, 0, time.UTC),
		},
		{
			name:          "Saturday Jan 17, 2026",
			date:          time.Date(2026, 1, 17, 15, 30, 0, 0, time.UTC),
			expectedStart: time.Date(2026, 1, 11, 0, 0, 0, 0, time.UTC),
			expectedEnd:   time.Date(2026, 1, 17, 23, 59, 59, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetWeekPeriod(tt.date)
			if !result.StartDate.Equal(tt.expectedStart) {
				t.Errorf("GetWeekPeriod(%v).StartDate = %v, want %v", tt.date, result.StartDate, tt.expectedStart)
			}
			if !result.EndDate.Equal(tt.expectedEnd) {
				t.Errorf("GetWeekPeriod(%v).EndDate = %v, want %v", tt.date, result.EndDate, tt.expectedEnd)
			}
		})
	}
}

func TestWeekPeriodString(t *testing.T) {
	week := WeekPeriod{
		StartDate: time.Date(2026, 1, 11, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 1, 17, 23, 59, 59, 0, time.UTC),
	}

	expected := "11-01-26 to 17-01-26"
	result := week.String()

	if result != expected {
		t.Errorf("WeekPeriod.String() = %q, want %q", result, expected)
	}
}

func TestMemberAttendanceHasAttendanceIssue(t *testing.T) {
	tests := []struct {
		name     string
		attendance MemberAttendance
		expected bool
	}{
		{
			name: "No missed weeks",
			attendance: MemberAttendance{
				MissedWeeks: []WeekPeriod{},
			},
			expected: false,
		},
		{
			name: "One missed week",
			attendance: MemberAttendance{
				MissedWeeks: []WeekPeriod{
					{
						StartDate: time.Date(2026, 1, 11, 0, 0, 0, 0, time.UTC),
						EndDate:   time.Date(2026, 1, 17, 23, 59, 59, 0, time.UTC),
					},
				},
			},
			expected: true,
		},
		{
			name: "Multiple missed weeks",
			attendance: MemberAttendance{
				MissedWeeks: []WeekPeriod{
					{
						StartDate: time.Date(2026, 1, 11, 0, 0, 0, 0, time.UTC),
						EndDate:   time.Date(2026, 1, 17, 23, 59, 59, 0, time.UTC),
					},
					{
						StartDate: time.Date(2026, 1, 18, 0, 0, 0, 0, time.UTC),
						EndDate:   time.Date(2026, 1, 24, 23, 59, 59, 0, time.UTC),
					},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.attendance.HasAttendanceIssue()
			if result != tt.expected {
				t.Errorf("HasAttendanceIssue() = %v, want %v", result, tt.expected)
			}
		})
	}
}
