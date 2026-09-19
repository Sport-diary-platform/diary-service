package entities

import (
	"time"

	"github.com/google/uuid"
)

type DailyCheckIn struct {
	ID               uuid.UUID
	AthleteID        uuid.UUID
	Date             LocalDate
	SleepDurationMin *int
	SleepQuality     *int // 1-10
	Energy           *int // 1-10
	Stress           *int // 1-10
	Soreness         *int // 1-10
	Motivation       *int // 1-10
	WeightKg         *float64
	RestingHeartRate *int
	Comment          *string
	Version          int64
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
