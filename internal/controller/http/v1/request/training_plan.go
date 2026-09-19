package request

import "github.com/google/uuid"

type CreateTrainingPlan struct {
	AthleteID   uuid.UUID `json:"athlete_id" validate:"required"`
	Name        string    `json:"name" validate:"required,max=255"`
	Description *string   `json:"description,omitempty" validate:"omitempty,max=5000"`
	StartDate   string    `json:"start_date" validate:"required,datetime=2006-01-02"`
	EndDate     *string   `json:"end_date,omitempty" validate:"omitempty,datetime=2006-01-02"`
}

type UpdateTrainingPlan struct {
	Name            string  `json:"name" validate:"required,max=255"`
	Description     *string `json:"description,omitempty" validate:"omitempty,max=5000"`
	StartDate       string  `json:"start_date" validate:"required,datetime=2006-01-02"`
	EndDate         *string `json:"end_date,omitempty" validate:"omitempty,datetime=2006-01-02"`
	ExpectedVersion int64   `json:"expected_version" validate:"required,min=1"`
}
