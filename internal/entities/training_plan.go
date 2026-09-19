package entities

import (
	"time"

	"github.com/google/uuid"
)

type StatusPlan string

const (
	StatusPlanDraft     StatusPlan = "draft"
	StatusPlanActive    StatusPlan = "active"
	StatusPlanCompleted StatusPlan = "completed"
	StatusPlanCancelled StatusPlan = "cancelled"
)

type TrainingPlan struct {
	ID          uuid.UUID
	CoachID     uuid.UUID
	AthleteID   uuid.UUID
	Name        string
	Description *string
	StartDate   LocalDate
	EndDate     *LocalDate
	Status      StatusPlan
	Version     int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
