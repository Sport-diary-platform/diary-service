package entities

import (
	"time"

	"github.com/google/uuid"
)

type StatusPlan string

const (
	StatusPlanDraft StatusPlan = "draft"
	StatusPlanActive StatusPlan = "active"
	StatusPlanCompleted StatusPlan = "completed"
	StatusPlanCancelled StatusPlan = "canceled"
)


type TrainingPlan struct {
	id uuid.UUID
	CoachID uuid.UUID
	AthleteID uuid.UUID
	Name string
	Description string
	StartDate time.Time
	EndDate *time.Time
	Status StatusPlan
	CreatedAt time.Time
	UpdatedAt time.Time
}