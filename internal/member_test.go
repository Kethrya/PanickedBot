package internal

import (
	"testing"
)

func TestUpdateFields_Validation(t *testing.T) {
	// Test that UpdateFields can hold various field types
	t.Run("all fields nil", func(t *testing.T) {
		fields := UpdateFields{}
		
		if fields.FamilyName != nil {
			t.Error("expected FamilyName to be nil")
		}
		if fields.DisplayName != nil {
			t.Error("expected DisplayName to be nil")
		}
		if fields.Class != nil {
			t.Error("expected Class to be nil")
		}
		if fields.Spec != nil {
			t.Error("expected Spec to be nil")
		}
		if fields.MeetsCap != nil {
			t.Error("expected MeetsCap to be nil")
		}
		if fields.AP != nil {
			t.Error("expected AP to be nil")
		}
		if fields.AAP != nil {
			t.Error("expected AAP to be nil")
		}
		if fields.DP != nil {
			t.Error("expected DP to be nil")
		}
		if len(fields.TeamIDs) != 0 {
			t.Error("expected TeamIDs to be empty")
		}
	})

	t.Run("set string fields", func(t *testing.T) {
		familyName := "TestFamily"
		displayName := "TestDisplay"
		class := "Warrior"
		spec := "Awakening"

		fields := UpdateFields{
			FamilyName:  &familyName,
			DisplayName: &displayName,
			Class:       &class,
			Spec:        &spec,
		}

		if fields.FamilyName == nil || *fields.FamilyName != familyName {
			t.Errorf("expected FamilyName %q, got %v", familyName, fields.FamilyName)
		}
		if fields.DisplayName == nil || *fields.DisplayName != displayName {
			t.Errorf("expected DisplayName %q, got %v", displayName, fields.DisplayName)
		}
		if fields.Class == nil || *fields.Class != class {
			t.Errorf("expected Class %q, got %v", class, fields.Class)
		}
		if fields.Spec == nil || *fields.Spec != spec {
			t.Errorf("expected Spec %q, got %v", spec, fields.Spec)
		}
	})

	t.Run("set boolean fields", func(t *testing.T) {
		meetsCap := true

		fields := UpdateFields{
			MeetsCap: &meetsCap,
		}

		if fields.MeetsCap == nil || *fields.MeetsCap != meetsCap {
			t.Errorf("expected MeetsCap %v, got %v", meetsCap, fields.MeetsCap)
		}
	})

	t.Run("set numeric fields", func(t *testing.T) {
		ap := 300
		aap := 320
		dp := 400

		fields := UpdateFields{
			AP:  &ap,
			AAP: &aap,
			DP:  &dp,
		}

		if fields.AP == nil || *fields.AP != ap {
			t.Errorf("expected AP %d, got %v", ap, fields.AP)
		}
		if fields.AAP == nil || *fields.AAP != aap {
			t.Errorf("expected AAP %d, got %v", aap, fields.AAP)
		}
		if fields.DP == nil || *fields.DP != dp {
			t.Errorf("expected DP %d, got %v", dp, fields.DP)
		}
	})

	t.Run("set team IDs", func(t *testing.T) {
		teamIDs := []int64{1, 2, 3}

		fields := UpdateFields{
			TeamIDs: teamIDs,
		}

		if len(fields.TeamIDs) != len(teamIDs) {
			t.Errorf("expected %d team IDs, got %d", len(teamIDs), len(fields.TeamIDs))
		}
		for i, id := range teamIDs {
			if fields.TeamIDs[i] != id {
				t.Errorf("expected team ID %d at index %d, got %d", id, i, fields.TeamIDs[i])
			}
		}
	})
}
