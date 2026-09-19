package entities

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type GoalType string
type GoalStatus string

const (
	GoalTypePerformance GoalType = "performance"
	GoalTypeWeight      GoalType = "weight"
	GoalTypeStrength    GoalType = "strength"
	GoalTypeDistance    GoalType = "distance"
	GoalTypeTime        GoalType = "time"
	GoalTypeCustom      GoalType = "custom"
)

const (
	GoalStatusActive    GoalStatus = "active"
	GoalStatusCompleted GoalStatus = "completed"
	GoalStatusCancelled GoalStatus = "cancelled"
)

type Goal struct {
	ID          uuid.UUID
	AthleteID   uuid.UUID
	CoachID     *uuid.UUID
	Title       string
	Description *string
	Type        GoalType
	Target      json.RawMessage
	Deadline    *LocalDate
	Status      GoalStatus
	Version     int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
