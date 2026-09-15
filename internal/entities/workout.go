package entities

import (
	"time"

	"github.com/google/uuid"
)

type StatusWorkout string
type WorkoutBlockType string

const (
	StatusWorkoutPlanned StatusWorkout = "planned"
	StatusWorkoutCompleted StatusWorkout = "completed"
	StatusWorkoutCancelled StatusWorkout = "canceled"
)

const (
	WorkoutBlockTypeWarmup WorkoutBlockType = "warmup"
	WorkoutBlockTypeMain WorkoutBlockType = "main"
	WorkoutBlockTypeCooldown WorkoutBlockType = "cooldown"
	WorkoutBlockTypeOther WorkoutBlockType = "other"
)

type Workout struct {
	Id uuid.UUID
	TrainingPlanId *uuid.UUID
	CoachId uuid.UUID
	AthleteId uuid.UUID
	Title string
	Description string
	SportType string
	SchedueledAt *string
	EstimatedDuration *string
	Status StatusWorkout
	Block []WorkoutBlock
	CreatedAt time.Time
	UpdatedAt time.Time
}

type WorkoutBlock struct {
	Id uuid.UUID
	WorkoutId uuid.UUID
	Name string
	Type WorkoutBlockType
	Position int
	Exercises []WorkoutExercise
	CreatedAt time.Time
	UpdatedAt time.Time
}

type WorkoutExercise struct {
	id uuid.UUID
	WorkoutBlockId uuid.UUID
	Name string
	Type string
	Position int
	Target string // jsonb
	CreatedAt time.Time
	UpdatedAt time.Time
}

type WorkoutResult struct {
	id uuid.UUID
	WorkoutId uuid.UUID
	AthleteId uuid.UUID
	PerformedAt time.Time
	Duration int
	Distance *float64
	AvgHeartRate *int
	Calories *int
	RPE int
	Feeling int
	Comments *string
	ExerciseResults []WorkoutExerciseResult
	CreatedAt time.Time
	UpdatedAt time.Time
}

type WorkoutExerciseResult struct {
	Id uuid.UUID
	WorkoutResultId uuid.UUID
	WorkoutExerciseId uuid.UUID
	Actual string // jsonb
	Comments *string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type WorkoutComment struct {
	id uuid.UUID
	WorkoutId uuid.UUID
	AuthorId uuid.UUID
	Text string
	CreatedAt time.Time
	UpdatedAt time.Time
}