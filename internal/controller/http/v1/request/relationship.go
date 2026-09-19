package request

import "github.com/google/uuid"

type CreateRelationship struct {
	AthleteID uuid.UUID `json:"athlete_id" validate:"required"`
}
