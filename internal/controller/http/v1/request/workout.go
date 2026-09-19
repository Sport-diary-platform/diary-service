package request

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type WorkoutExercise struct {
	ID       *uuid.UUID      `json:"id,omitempty"`
	Name     string          `json:"name" validate:"required,max=255"`
	Type     string          `json:"type" validate:"required,max=100"`
	Position int             `json:"position" validate:"min=0"`
	Target   json.RawMessage `json:"target" validate:"required"`
}

type WorkoutBlock struct {
	ID        *uuid.UUID        `json:"id,omitempty"`
	Name      string            `json:"name" validate:"required,max=255"`
	Type      string            `json:"type" validate:"required,oneof=warmup main cooldown other"`
	Position  int               `json:"position" validate:"min=0"`
	Exercises []WorkoutExercise `json:"exercises" validate:"dive"`
}

type CreateWorkout struct {
	AthleteID                uuid.UUID      `json:"athlete_id" validate:"required"`
	TrainingPlanID           *uuid.UUID     `json:"training_plan_id,omitempty"`
	Title                    string         `json:"title" validate:"required,max=255"`
	Description              *string        `json:"description,omitempty" validate:"omitempty,max=5000"`
	SportType                string         `json:"sport_type" validate:"required,max=100"`
	ScheduledAt              time.Time      `json:"scheduled_at" validate:"required"`
	EstimatedDurationSeconds *int           `json:"estimated_duration_seconds,omitempty" validate:"omitempty,gt=0"`
	Blocks                   []WorkoutBlock `json:"blocks" validate:"dive"`
}

type UpdateWorkout struct {
	Title                    string         `json:"title" validate:"required,max=255"`
	Description              *string        `json:"description,omitempty" validate:"omitempty,max=5000"`
	SportType                string         `json:"sport_type" validate:"required,max=100"`
	ScheduledAt              time.Time      `json:"scheduled_at" validate:"required"`
	EstimatedDurationSeconds *int           `json:"estimated_duration_seconds,omitempty" validate:"omitempty,gt=0"`
	Blocks                   []WorkoutBlock `json:"blocks" validate:"dive"`
	ExpectedVersion          int64          `json:"expected_version" validate:"required,min=1"`
}

type ExerciseResult struct {
	ExerciseID uuid.UUID       `json:"exercise_id" validate:"required"`
	Actual     json.RawMessage `json:"actual" validate:"required"`
	Comment    *string         `json:"comment,omitempty" validate:"omitempty,max=5000"`
}

type CreateWorkoutResult struct {
	PerformedAt            time.Time        `json:"performed_at" validate:"required"`
	DurationSeconds        *int             `json:"duration_seconds,omitempty" validate:"omitempty,gt=0"`
	DistanceMeters         *float64         `json:"distance_meters,omitempty" validate:"omitempty,gte=0"`
	AvgHeartRate           *int             `json:"avg_heart_rate,omitempty" validate:"omitempty,gt=0"`
	MaxHeartRate           *int             `json:"max_heart_rate,omitempty" validate:"omitempty,gt=0"`
	Calories               *int             `json:"calories,omitempty" validate:"omitempty,gte=0"`
	RPE                    int              `json:"rpe" validate:"required,min=1,max=10"`
	Feeling                int              `json:"feeling" validate:"required,min=1,max=10"`
	Comment                *string          `json:"comment,omitempty" validate:"omitempty,max=5000"`
	Exercises              []ExerciseResult `json:"exercises" validate:"dive"`
	ExpectedWorkoutVersion int64            `json:"expected_workout_version" validate:"required,min=1"`
}

type UpdateWorkoutResult struct {
	PerformedAt     time.Time        `json:"performed_at" validate:"required"`
	DurationSeconds *int             `json:"duration_seconds,omitempty" validate:"omitempty,gt=0"`
	DistanceMeters  *float64         `json:"distance_meters,omitempty" validate:"omitempty,gte=0"`
	AvgHeartRate    *int             `json:"avg_heart_rate,omitempty" validate:"omitempty,gt=0"`
	MaxHeartRate    *int             `json:"max_heart_rate,omitempty" validate:"omitempty,gt=0"`
	Calories        *int             `json:"calories,omitempty" validate:"omitempty,gte=0"`
	RPE             int              `json:"rpe" validate:"required,min=1,max=10"`
	Feeling         int              `json:"feeling" validate:"required,min=1,max=10"`
	Comment         *string          `json:"comment,omitempty" validate:"omitempty,max=5000"`
	Exercises       []ExerciseResult `json:"exercises" validate:"dive"`
	ExpectedVersion int64            `json:"expected_version" validate:"required,min=1"`
}
