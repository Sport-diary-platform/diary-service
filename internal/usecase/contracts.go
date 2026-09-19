package usecase

import (
	"context"
	"encoding/json"
	"time"

	"diary-service/internal/entities"

	"github.com/google/uuid"
)

type CreateRelationshipCommand struct {
	ActorID   uuid.UUID
	AthleteID uuid.UUID
}

type TerminateRelationshipCommand struct {
	ActorID         uuid.UUID
	RelationshipID  uuid.UUID
	ExpectedVersion int64
}

type RelationshipUseCase interface {
	Create(ctx context.Context, cmd CreateRelationshipCommand) (*entities.CoachAthleteRelationship, error)
	Terminate(ctx context.Context, cmd TerminateRelationshipCommand) error
	ListAthletes(ctx context.Context, actorID uuid.UUID) ([]uuid.UUID, error)
	ListCoaches(ctx context.Context, actorID uuid.UUID) ([]uuid.UUID, error)
}

type WorkoutExerciseCommand struct {
	ID       *uuid.UUID
	Name     string
	Type     string
	Position int
	Target   json.RawMessage
}

type WorkoutBlockCommand struct {
	ID        *uuid.UUID
	Name      string
	Type      entities.WorkoutBlockType
	Position  int
	Exercises []WorkoutExerciseCommand
}

type CreateWorkoutCommand struct {
	ActorID                  uuid.UUID
	AthleteID                uuid.UUID
	TrainingPlanID           *uuid.UUID
	Title                    string
	Description              *string
	SportType                string
	ScheduledAt              time.Time
	EstimatedDurationSeconds *int
	Blocks                   []WorkoutBlockCommand
}

type UpdateWorkoutCommand struct {
	ActorID                  uuid.UUID
	WorkoutID                uuid.UUID
	Title                    string
	Description              *string
	SportType                string
	ScheduledAt              time.Time
	EstimatedDurationSeconds *int
	Blocks                   []WorkoutBlockCommand
	ExpectedVersion          int64
}

type ChangeWorkoutStateCommand struct {
	ActorID         uuid.UUID
	WorkoutID       uuid.UUID
	ExpectedVersion int64
}

type WorkoutUseCase interface {
	Create(ctx context.Context, cmd CreateWorkoutCommand) (*entities.Workout, error)
	GetByID(ctx context.Context, actorID, workoutID uuid.UUID) (*entities.Workout, error)
	Update(ctx context.Context, cmd UpdateWorkoutCommand) (*entities.Workout, error)
	Cancel(ctx context.Context, cmd ChangeWorkoutStateCommand) error
	Delete(ctx context.Context, cmd ChangeWorkoutStateCommand) error
	ListByAthlete(ctx context.Context, actorID, athleteID uuid.UUID, from, to time.Time) ([]entities.Workout, error)
	ListByCoach(ctx context.Context, actorID uuid.UUID, from, to time.Time) ([]entities.Workout, error)
}

type ExerciseResultCommand struct {
	ExerciseID uuid.UUID
	Actual     json.RawMessage
	Comment    *string
}

type CreateWorkoutResultCommand struct {
	ActorID                uuid.UUID
	WorkoutID              uuid.UUID
	PerformedAt            time.Time
	DurationSeconds        *int
	DistanceMeters         *float64
	AvgHeartRate           *int
	MaxHeartRate           *int
	Calories               *int
	RPE                    int
	Feeling                int
	Comment                *string
	Exercises              []ExerciseResultCommand
	ExpectedWorkoutVersion int64
}

type UpdateWorkoutResultCommand struct {
	ActorID         uuid.UUID
	WorkoutID       uuid.UUID
	PerformedAt     time.Time
	DurationSeconds *int
	DistanceMeters  *float64
	AvgHeartRate    *int
	MaxHeartRate    *int
	Calories        *int
	RPE             int
	Feeling         int
	Comment         *string
	Exercises       []ExerciseResultCommand
	ExpectedVersion int64
}

type WorkoutResultUseCase interface {
	Create(ctx context.Context, cmd CreateWorkoutResultCommand) (*entities.WorkoutResult, error)
	GetByWorkoutID(ctx context.Context, actorID, workoutID uuid.UUID) (*entities.WorkoutResult, error)
	Update(ctx context.Context, cmd UpdateWorkoutResultCommand) (*entities.WorkoutResult, error)
}

type CreateTrainingPlanCommand struct {
	ActorID     uuid.UUID
	AthleteID   uuid.UUID
	Name        string
	Description *string
	StartDate   entities.LocalDate
	EndDate     *entities.LocalDate
}

type UpdateTrainingPlanCommand struct {
	ActorID         uuid.UUID
	PlanID          uuid.UUID
	Name            string
	Description     *string
	StartDate       entities.LocalDate
	EndDate         *entities.LocalDate
	ExpectedVersion int64
}

type ChangeTrainingPlanStateCommand struct {
	ActorID         uuid.UUID
	PlanID          uuid.UUID
	ExpectedVersion int64
}

type TrainingPlanUseCase interface {
	Create(ctx context.Context, cmd CreateTrainingPlanCommand) (*entities.TrainingPlan, error)
	GetByID(ctx context.Context, actorID, planID uuid.UUID) (*entities.TrainingPlan, error)
	Update(ctx context.Context, cmd UpdateTrainingPlanCommand) (*entities.TrainingPlan, error)
	Activate(ctx context.Context, cmd ChangeTrainingPlanStateCommand) error
	Complete(ctx context.Context, cmd ChangeTrainingPlanStateCommand) error
	Cancel(ctx context.Context, cmd ChangeTrainingPlanStateCommand) error
}

type DailyCheckInData struct {
	Date             entities.LocalDate
	SleepDurationMin *int
	SleepQuality     *int
	Energy           *int
	Stress           *int
	Soreness         *int
	Motivation       *int
	WeightKg         *float64
	RestingHeartRate *int
	Comment          *string
}

type CreateDailyCheckInCommand struct {
	ActorID uuid.UUID
	DailyCheckInData
}

type UpdateDailyCheckInCommand struct {
	ActorID         uuid.UUID
	ExpectedVersion int64
	DailyCheckInData
}

type DailyCheckInUseCase interface {
	Create(ctx context.Context, cmd CreateDailyCheckInCommand) (*entities.DailyCheckIn, error)
	GetByDate(ctx context.Context, actorID uuid.UUID, date entities.LocalDate) (*entities.DailyCheckIn, error)
	Update(ctx context.Context, cmd UpdateDailyCheckInCommand) (*entities.DailyCheckIn, error)
	List(ctx context.Context, actorID uuid.UUID, from, to entities.LocalDate) ([]entities.DailyCheckIn, error)
}

type CreateGoalCommand struct {
	ActorID     uuid.UUID
	AthleteID   uuid.UUID
	Title       string
	Description *string
	Type        entities.GoalType
	Target      json.RawMessage
	Deadline    *entities.LocalDate
}

type UpdateGoalCommand struct {
	ActorID         uuid.UUID
	GoalID          uuid.UUID
	Title           string
	Description     *string
	Type            entities.GoalType
	Target          json.RawMessage
	Deadline        *entities.LocalDate
	ExpectedVersion int64
}

type ChangeGoalStateCommand struct {
	ActorID         uuid.UUID
	GoalID          uuid.UUID
	ExpectedVersion int64
}

type GoalUseCase interface {
	Create(ctx context.Context, cmd CreateGoalCommand) (*entities.Goal, error)
	GetByID(ctx context.Context, actorID, goalID uuid.UUID) (*entities.Goal, error)
	Update(ctx context.Context, cmd UpdateGoalCommand) (*entities.Goal, error)
	Complete(ctx context.Context, cmd ChangeGoalStateCommand) error
	Cancel(ctx context.Context, cmd ChangeGoalStateCommand) error
	List(ctx context.Context, actorID uuid.UUID) ([]entities.Goal, error)
}

type CreateCommentCommand struct {
	ActorID   uuid.UUID
	WorkoutID uuid.UUID
	Text      string
}

type UpdateCommentCommand struct {
	ActorID         uuid.UUID
	WorkoutID       uuid.UUID
	CommentID       uuid.UUID
	Text            string
	ExpectedVersion int64
}

type CommentUseCase interface {
	Create(ctx context.Context, cmd CreateCommentCommand) (*entities.WorkoutComment, error)
	ListByWorkout(ctx context.Context, actorID, workoutID uuid.UUID) ([]entities.WorkoutComment, error)
	Update(ctx context.Context, cmd UpdateCommentCommand) (*entities.WorkoutComment, error)
}
