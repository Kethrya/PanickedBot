package internal

import "time"

// GetEasternLocation returns the Eastern timezone location (America/New_York)
// This handles both EST (Eastern Standard Time) and EDT (Eastern Daylight Time) automatically
func GetEasternLocation() *time.Location {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		// Fallback to UTC if Eastern timezone is not available
		// This should not happen in production but provides a safe fallback
		return time.UTC
	}
	return loc
}
