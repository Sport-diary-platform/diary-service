package entities

import (
	"time"

	"github.com/google/uuid"
)

type DailyCheckIn struct {
	Id uuid.UUID
	AthleteId uuid.UUID
	Date time.Time
	SleepDuration int // minutes
	SleepQuality int // 1-10
	Energy int // 1-10
	Stress int // 1-10
	Soreness int // 1-10
	Motivation int // 1-10
	WeightKg float64
	RestingHeartRate *int
	Comments *string
	CreatedAt time.Time
	UpdatedAt time.Time
}