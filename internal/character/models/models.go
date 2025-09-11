package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Character represents a character document in the database
type Character struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	CharacterID    int                `bson:"character_id" json:"character_id"`
	Name           string             `bson:"name" json:"name"`
	CorporationID  int                `bson:"corporation_id" json:"corporation_id"`
	AllianceID     int                `bson:"alliance_id,omitempty" json:"alliance_id,omitempty"`
	Birthday       time.Time          `bson:"birthday" json:"birthday"`
	SecurityStatus float64            `bson:"security_status" json:"security_status"`
	Description    string             `bson:"description,omitempty" json:"description,omitempty"`
	Gender         string             `bson:"gender" json:"gender"`
	RaceID         int                `bson:"race_id" json:"race_id"`
	BloodlineID    int                `bson:"bloodline_id" json:"bloodline_id"`
	AncestryID     int                `bson:"ancestry_id,omitempty" json:"ancestry_id,omitempty"`
	FactionID      int                `bson:"faction_id,omitempty" json:"faction_id,omitempty"`
	CreatedAt      time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt      time.Time          `bson:"updated_at" json:"updated_at"`
}

// CollectionName returns the MongoDB collection name for characters
func (c *Character) CollectionName() string {
	return "characters"
}

// CharacterAttributes represents character attributes in the database
type CharacterAttributes struct {
	ID                       primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	CharacterID              int                `bson:"character_id" json:"character_id"`
	Charisma                 int                `bson:"charisma" json:"charisma"`
	Intelligence             int                `bson:"intelligence" json:"intelligence"`
	Memory                   int                `bson:"memory" json:"memory"`
	Perception               int                `bson:"perception" json:"perception"`
	Willpower                int                `bson:"willpower" json:"willpower"`
	AccruedRemapCooldownDate *time.Time         `bson:"accrued_remap_cooldown_date,omitempty" json:"accrued_remap_cooldown_date,omitempty"`
	BonusRemaps              *int               `bson:"bonus_remaps,omitempty" json:"bonus_remaps,omitempty"`
	LastRemapDate            *time.Time         `bson:"last_remap_date,omitempty" json:"last_remap_date,omitempty"`
	CreatedAt                time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt                time.Time          `bson:"updated_at" json:"updated_at"`
}

// CollectionName returns the MongoDB collection name for character attributes
func (ca *CharacterAttributes) CollectionName() string {
	return "character_attributes"
}

// CharacterSkillQueue represents the skill queue for a character in the database
type CharacterSkillQueue struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	CharacterID int                `bson:"character_id" json:"character_id"`
	Skills      []SkillQueueItem   `bson:"skills" json:"skills"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}

// SkillQueueItem represents a single skill in the queue
type SkillQueueItem struct {
	SkillID         int        `bson:"skill_id" json:"skill_id"`
	FinishedLevel   int        `bson:"finished_level" json:"finished_level"`
	QueuePosition   int        `bson:"queue_position" json:"queue_position"`
	StartDate       *time.Time `bson:"start_date,omitempty" json:"start_date,omitempty"`
	FinishDate      *time.Time `bson:"finish_date,omitempty" json:"finish_date,omitempty"`
	TrainingStartSP *int       `bson:"training_start_sp,omitempty" json:"training_start_sp,omitempty"`
	LevelEndSP      *int       `bson:"level_end_sp,omitempty" json:"level_end_sp,omitempty"`
	LevelStartSP    *int       `bson:"level_start_sp,omitempty" json:"level_start_sp,omitempty"`
}

// CollectionName returns the MongoDB collection name for character skill queues
func (csq *CharacterSkillQueue) CollectionName() string {
	return "character_skill_queues"
}

// CharacterSkills represents the character's trained skills in the database
type CharacterSkills struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	CharacterID   int                `bson:"character_id" json:"character_id"`
	Skills        []Skill            `bson:"skills" json:"skills"`
	TotalSP       int64              `bson:"total_sp" json:"total_sp"`
	UnallocatedSP *int               `bson:"unallocated_sp,omitempty" json:"unallocated_sp,omitempty"`
	CreatedAt     time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt     time.Time          `bson:"updated_at" json:"updated_at"`
}

// Skill represents a single trained skill
type Skill struct {
	SkillID            int `bson:"skill_id" json:"skill_id"`
	SkillpointsInSkill int `bson:"skillpoints_in_skill" json:"skillpoints_in_skill"`
	TrainedSkillLevel  int `bson:"trained_skill_level" json:"trained_skill_level"`
	ActiveSkillLevel   int `bson:"active_skill_level" json:"active_skill_level"`
}

// CollectionName returns the MongoDB collection name for character skills
func (cs *CharacterSkills) CollectionName() string {
	return "character_skills"
}

// CorporationHistoryEntry represents a single corporation history entry in the database
type CorporationHistoryEntry struct {
	CorporationID int       `bson:"corporation_id" json:"corporation_id"`
	IsDeleted     bool      `bson:"is_deleted,omitempty" json:"is_deleted,omitempty"`
	RecordID      int       `bson:"record_id" json:"record_id"`
	StartDate     time.Time `bson:"start_date" json:"start_date"`
}

// CharacterCorporationHistory represents the corporation history for a character in the database
type CharacterCorporationHistory struct {
	ID          primitive.ObjectID        `bson:"_id,omitempty" json:"id,omitempty"`
	CharacterID int                       `bson:"character_id" json:"character_id"`
	History     []CorporationHistoryEntry `bson:"history" json:"history"`
	CreatedAt   time.Time                 `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time                 `bson:"updated_at" json:"updated_at"`
}

// CollectionName returns the MongoDB collection name for character corporation history
func (cch *CharacterCorporationHistory) CollectionName() string {
	return "character_corporation_history"
}

// HomeLocation represents the character's home location in the database
type HomeLocation struct {
	LocationID     int64  `bson:"location_id" json:"location_id"`
	LocationType   string `bson:"location_type" json:"location_type"`
	LocationName   string `bson:"location_name,omitempty" json:"location_name,omitempty"`
	LocationTypeID int32  `bson:"location_type_id,omitempty" json:"location_type_id,omitempty"`
}

// JumpClone represents a single jump clone in the database
type JumpClone struct {
	Implants       []int  `bson:"implants" json:"implants"`
	JumpCloneID    int    `bson:"jump_clone_id" json:"jump_clone_id"`
	LocationID     int64  `bson:"location_id" json:"location_id"`
	LocationType   string `bson:"location_type" json:"location_type"`
	LocationName   string `bson:"location_name,omitempty" json:"location_name,omitempty"`
	LocationTypeID int32  `bson:"location_type_id,omitempty" json:"location_type_id,omitempty"`
	Name           string `bson:"name,omitempty" json:"name,omitempty"`
}

// CharacterClones represents the character's clone information in the database
type CharacterClones struct {
	ID                    primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	CharacterID           int                `bson:"character_id" json:"character_id"`
	HomeLocation          *HomeLocation      `bson:"home_location,omitempty" json:"home_location,omitempty"`
	JumpClones            []JumpClone        `bson:"jump_clones" json:"jump_clones"`
	ActiveImplants        []int              `bson:"active_implants" json:"active_implants"`
	LastCloneJumpDate     *time.Time         `bson:"last_clone_jump_date,omitempty" json:"last_clone_jump_date,omitempty"`
	LastStationChangeDate *time.Time         `bson:"last_station_change_date,omitempty" json:"last_station_change_date,omitempty"`
	CreatedAt             time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt             time.Time          `bson:"updated_at" json:"updated_at"`
}

// CollectionName returns the MongoDB collection name for character clones
func (cc *CharacterClones) CollectionName() string {
	return "character_clones"
}

// CharacterImplants represents the character's active implants in the database
type CharacterImplants struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	CharacterID int                `bson:"character_id" json:"character_id"`
	Implants    []int              `bson:"implants" json:"implants"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}

// CollectionName returns the MongoDB collection name for character implants
func (ci *CharacterImplants) CollectionName() string {
	return "character_implants"
}
