package entities

import (
	"time"

	"github.com/google/uuid"
)

type GoalType string
type GoalStatus string

const (
	GoalTypePerformance GoalType = "performance"
	GoalTypeWeight GoalType = "weight"
	GoalTypeStrength GoalType = "strength"
	GoalTypeDistance GoalType = "distance"
	GoalTypeTime GoalType = "time"
	GolaTypeCustom GoalType = "custom"
)

const (
	GoalStatusActive GoalStatus = "active"
	GoalStatusCompleted GoalStatus = "completed"
	GoalStatusCancelled GoalStatus = "cancelled"
)

type Goal struct {
	id uuid.UUID
	AthleteID uuid.UUID
	CoachID *uuid.UUID
	Title string
	Description string
	Type GoalType
	Target string // jsonb
	Deadline *string
	Status GoalStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}