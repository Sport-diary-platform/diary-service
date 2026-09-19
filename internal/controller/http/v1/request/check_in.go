package request

type CheckInData struct {
	SleepDurationMin *int     `json:"sleep_duration_min,omitempty" validate:"omitempty,gte=0"`
	SleepQuality     *int     `json:"sleep_quality,omitempty" validate:"omitempty,min=1,max=10"`
	Energy           *int     `json:"energy,omitempty" validate:"omitempty,min=1,max=10"`
	Stress           *int     `json:"stress,omitempty" validate:"omitempty,min=1,max=10"`
	Soreness         *int     `json:"soreness,omitempty" validate:"omitempty,min=1,max=10"`
	Motivation       *int     `json:"motivation,omitempty" validate:"omitempty,min=1,max=10"`
	WeightKg         *float64 `json:"weight_kg,omitempty" validate:"omitempty,gt=0"`
	RestingHeartRate *int     `json:"resting_heart_rate,omitempty" validate:"omitempty,gt=0"`
	Comment          *string  `json:"comment,omitempty" validate:"omitempty,max=5000"`
}

type CreateCheckIn struct {
	Date string `json:"date" validate:"required,datetime=2006-01-02"`
	CheckInData
}

type UpdateCheckIn struct {
	CheckInData
	ExpectedVersion int64 `json:"expected_version" validate:"required,min=1"`
}
