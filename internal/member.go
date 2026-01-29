package internal

import (
	"time"

	"PanickedBot/internal/db"
)

// Member represents a roster member
type Member struct {
	ID             int64
	DiscordGuildID string
	DiscordUserID  *string
	FamilyName     string
	DisplayName    *string
	Class          *string
	Spec           *string

	// Combat stats
	AP       *int
	AAP      *int
	DP       *int
	Evasion  *int
	DR       *int
	DRR      *float64
	Accuracy *int
	HP       *int
	TotalAP  *int
	TotalAAP *int

	// Status flags
	MeetsCap    bool
	IsException bool
	IsMercenary bool
	IsActive    bool
	CreatedAt   time.Time
}

// UpdateFields represents fields that can be updated
type UpdateFields = db.UpdateFields

// GetMemberByDiscordUserID retrieves a member by Discord user ID
func GetMemberByDiscordUserID(database *db.DB, guildID, userID string) (*Member, error) {
	m, err := db.GetMemberByDiscordUserID(database, guildID, userID)
	if err != nil {
		return nil, err
	}
	return convertFromDBMember(m), nil
}

// GetMemberByFamilyName retrieves a member by BDO family name
func GetMemberByFamilyName(database *db.DB, guildID, familyName string) (*Member, error) {
	m, err := db.GetMemberByFamilyName(database, guildID, familyName)
	if err != nil {
		return nil, err
	}
	return convertFromDBMember(m), nil
}

// GetMemberByDiscordUserIDIncludingInactive retrieves a member by Discord user ID, including inactive members
func GetMemberByDiscordUserIDIncludingInactive(database *db.DB, guildID, userID string) (*Member, error) {
	m, err := db.GetMemberByDiscordUserIDIncludingInactive(database, guildID, userID)
	if err != nil {
		return nil, err
	}
	return convertFromDBMember(m), nil
}

// GetMemberByFamilyNameIncludingInactive retrieves a member by BDO family name, including inactive members
func GetMemberByFamilyNameIncludingInactive(database *db.DB, guildID, familyName string) (*Member, error) {
	m, err := db.GetMemberByFamilyNameIncludingInactive(database, guildID, familyName)
	if err != nil {
		return nil, err
	}
	return convertFromDBMember(m), nil
}

// UpdateMember updates member fields
func UpdateMember(database *db.DB, memberID int64, fields UpdateFields) error {
	return db.UpdateMember(database, memberID, fields)
}

// CreateMember creates a new roster member
func CreateMember(database *db.DB, guildID, discordUserID, familyName string) (int64, error) {
	return db.CreateMember(database, guildID, discordUserID, familyName)
}

// SetMemberActive sets the is_active flag for a member
func SetMemberActive(database *db.DB, memberID int64, active bool) error {
	return db.SetMemberActive(database, memberID, active)
}

// SetMemberMercenary sets the is_mercenary flag for a member
func SetMemberMercenary(database *db.DB, memberID int64, mercenary bool) error {
	return db.SetMemberMercenary(database, memberID, mercenary)
}

// GetAllRosterMembers retrieves all active roster members for a guild, excluding mercenaries
func GetAllRosterMembers(database *db.DB, guildID string) ([]Member, error) {
	members, err := db.GetAllActiveMembers(database, guildID)
	if err != nil {
		return nil, err
	}

	result := make([]Member, len(members))
	for i, m := range members {
		result[i] = *convertFromDBMember(&m)
	}

	return result, nil
}

// AssignMemberToTeams assigns a member to multiple teams (replaces all existing team assignments)
func AssignMemberToTeams(database *db.DB, memberID int64, teamIDs []int64) error {
	return db.AssignMemberToTeams(database, memberID, teamIDs)
}

// GetMemberTeamIDs retrieves all team IDs for a member
func GetMemberTeamIDs(database *db.DB, memberID int64) ([]int64, error) {
	return db.GetMemberTeamIDs(database, memberID)
}

// GetMemberTeamNames retrieves team names for a member as a comma-separated string
func GetMemberTeamNames(database *db.DB, guildID string, memberID int64) (string, error) {
	return db.GetMemberTeamNames(database, guildID, memberID)
}

// Helper function to convert from db.Member to internal.Member
func convertFromDBMember(m *db.Member) *Member {
	var discordUserID *string
	if m.DiscordUserID.Valid {
		discordUserID = &m.DiscordUserID.String
	}

	var displayName *string
	if m.DisplayName.Valid {
		displayName = &m.DisplayName.String
	}

	var class *string
	if m.Class.Valid {
		class = &m.Class.String
	}

	var spec *string
	if m.Spec.Valid {
		spec = &m.Spec.String
	}

	var ap, aap, dp, evasion, dr, accuracy, hp, totalAP, totalAAP *int
	if m.AP.Valid {
		val := int(m.AP.Int32)
		ap = &val
	}
	if m.AAP.Valid {
		val := int(m.AAP.Int32)
		aap = &val
	}
	if m.DP.Valid {
		val := int(m.DP.Int32)
		dp = &val
	}
	if m.Evasion.Valid {
		val := int(m.Evasion.Int32)
		evasion = &val
	}
	if m.DR.Valid {
		val := int(m.DR.Int32)
		dr = &val
	}
	if m.Accuracy.Valid {
		val := int(m.Accuracy.Int32)
		accuracy = &val
	}
	if m.HP.Valid {
		val := int(m.HP.Int32)
		hp = &val
	}
	if m.TotalAP.Valid {
		val := int(m.TotalAP.Int32)
		totalAP = &val
	}
	if m.TotalAAP.Valid {
		val := int(m.TotalAAP.Int32)
		totalAAP = &val
	}

	var drr *float64
	if m.DRR.Valid {
		drr = &m.DRR.Float64
	}

	return &Member{
		ID:             m.ID,
		DiscordGuildID: m.DiscordGuildID,
		DiscordUserID:  discordUserID,
		FamilyName:     m.FamilyName,
		DisplayName:    displayName,
		Class:          class,
		Spec:           spec,
		AP:             ap,
		AAP:            aap,
		DP:             dp,
		Evasion:        evasion,
		DR:             dr,
		DRR:            drr,
		Accuracy:       accuracy,
		HP:             hp,
		TotalAP:        totalAP,
		TotalAAP:       totalAAP,
		MeetsCap:       m.MeetsCap,
		IsException:    m.IsException,
		IsMercenary:    m.IsMercenary,
		IsActive:       m.IsActive,
		CreatedAt:      m.CreatedAt,
	}
}
