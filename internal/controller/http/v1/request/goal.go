package request

import (
	"encoding/json"

	"github.com/google/uuid"
)

type CreateGoal struct {
	AthleteID   *uuid.UUID      `json:"athlete_id,omitempty"`
	Title       string          `json:"title" validate:"required,max=255"`
	Description *string         `json:"description,omitempty" validate:"omitempty,max=5000"`
	Type        string          `json:"type" validate:"required,oneof=performance weight strength distance time custom"`
	Target      json.RawMessage `json:"target" validate:"required"`
	Deadline    *string         `json:"deadline,omitempty" validate:"omitempty,datetime=2006-01-02"`
}

type UpdateGoal struct {
	Title           string          `json:"title" validate:"required,max=255"`
	Description     *string         `json:"description,omitempty" validate:"omitempty,max=5000"`
	Type            string          `json:"type" validate:"required,oneof=performance weight strength distance time custom"`
	Target          json.RawMessage `json:"target" validate:"required"`
	Deadline        *string         `json:"deadline,omitempty" validate:"omitempty,datetime=2006-01-02"`
	ExpectedVersion int64           `json:"expected_version" validate:"required,min=1"`
}
