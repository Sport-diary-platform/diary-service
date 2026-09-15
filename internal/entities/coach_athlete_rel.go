package entities

import (
	"time"

	"github.com/google/uuid"
)

type StatusRel string

const (
	StatusRelActive StatusRel = "active"
	StatusRelTerminates StatusRel = "terminated"
)


type CoachAthleteRelationship struct {
	ID uuid.UUID
	CoachID uuid.UUID
	AthleteID uuid.UUID
	Status StatusRel
	CreatedAt time.Time
	UpdatedAt time.Time
}