package response

import (
	"encoding/json"
	"time"

	"diary-service/internal/entities"

	"github.com/google/uuid"
)

type Error struct {
	Error ErrorDetails `json:"error"`
}

type ErrorDetails struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type WorkoutExercise struct {
	ID             uuid.UUID       `json:"id"`
	WorkoutBlockID uuid.UUID       `json:"workout_block_id"`
	Name           string          `json:"name"`
	Type           string          `json:"type"`
	Position       int             `json:"position"`
	Target         json.RawMessage `json:"target"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type WorkoutBlock struct {
	ID        uuid.UUID         `json:"id"`
	WorkoutID uuid.UUID         `json:"workout_id"`
	Name      string            `json:"name"`
	Type      string            `json:"type"`
	Position  int               `json:"position"`
	Exercises []WorkoutExercise `json:"exercises"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

type Workout struct {
	ID                       uuid.UUID      `json:"id"`
	TrainingPlanID           *uuid.UUID     `json:"training_plan_id,omitempty"`
	CoachID                  uuid.UUID      `json:"coach_id"`
	AthleteID                uuid.UUID      `json:"athlete_id"`
	Title                    string         `json:"title"`
	Description              *string        `json:"description,omitempty"`
	SportType                string         `json:"sport_type"`
	ScheduledAt              time.Time      `json:"scheduled_at"`
	EstimatedDurationSeconds *int           `json:"estimated_duration_seconds,omitempty"`
	Status                   string         `json:"status"`
	Blocks                   []WorkoutBlock `json:"blocks"`
	Version                  int64          `json:"version"`
	CreatedAt                time.Time      `json:"created_at"`
	UpdatedAt                time.Time      `json:"updated_at"`
}

func WorkoutFromEntity(value *entities.Workout) Workout {
	blocks := make([]WorkoutBlock, 0, len(value.Blocks))
	for _, block := range value.Blocks {
		exercises := make([]WorkoutExercise, 0, len(block.Exercises))
		for _, exercise := range block.Exercises {
			exercises = append(exercises, WorkoutExercise{
				ID: exercise.ID, WorkoutBlockID: exercise.WorkoutBlockID, Name: exercise.Name,
				Type: exercise.Type, Position: exercise.Position, Target: exercise.Target,
				CreatedAt: exercise.CreatedAt, UpdatedAt: exercise.UpdatedAt,
			})
		}
		blocks = append(blocks, WorkoutBlock{
			ID: block.ID, WorkoutID: block.WorkoutID, Name: block.Name, Type: string(block.Type),
			Position: block.Position, Exercises: exercises, CreatedAt: block.CreatedAt, UpdatedAt: block.UpdatedAt,
		})
	}
	return Workout{
		ID: value.ID, TrainingPlanID: value.TrainingPlanID, CoachID: value.CoachID, AthleteID: value.AthleteID,
		Title: value.Title, Description: value.Description, SportType: value.SportType, ScheduledAt: value.ScheduledAt,
		EstimatedDurationSeconds: value.EstimatedDurationSeconds, Status: string(value.Status), Blocks: blocks,
		Version: value.Version, CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt,
	}
}

func WorkoutsFromEntities(values []entities.Workout) []Workout {
	result := make([]Workout, 0, len(values))
	for i := range values {
		result = append(result, WorkoutFromEntity(&values[i]))
	}
	return result
}

type ExerciseResult struct {
	ID              uuid.UUID       `json:"id"`
	WorkoutResultID uuid.UUID       `json:"workout_result_id"`
	ExerciseID      uuid.UUID       `json:"exercise_id"`
	Actual          json.RawMessage `json:"actual"`
	Comment         *string         `json:"comment,omitempty"`
	Version         int64           `json:"version"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type WorkoutResult struct {
	ID              uuid.UUID        `json:"id"`
	WorkoutID       uuid.UUID        `json:"workout_id"`
	AthleteID       uuid.UUID        `json:"athlete_id"`
	PerformedAt     time.Time        `json:"performed_at"`
	DurationSeconds *int             `json:"duration_seconds,omitempty"`
	DistanceMeters  *float64         `json:"distance_meters,omitempty"`
	AvgHeartRate    *int             `json:"avg_heart_rate,omitempty"`
	MaxHeartRate    *int             `json:"max_heart_rate,omitempty"`
	Calories        *int             `json:"calories,omitempty"`
	RPE             int              `json:"rpe"`
	Feeling         int              `json:"feeling"`
	Comment         *string          `json:"comment,omitempty"`
	Exercises       []ExerciseResult `json:"exercises"`
	Version         int64            `json:"version"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
}

func WorkoutResultFromEntity(value *entities.WorkoutResult) WorkoutResult {
	exercises := make([]ExerciseResult, 0, len(value.ExerciseResults))
	for _, exercise := range value.ExerciseResults {
		exercises = append(exercises, ExerciseResult{
			ID: exercise.ID, WorkoutResultID: exercise.WorkoutResultID, ExerciseID: exercise.ExerciseID,
			Actual: exercise.Actual, Comment: exercise.Comment, Version: exercise.Version,
			CreatedAt: exercise.CreatedAt, UpdatedAt: exercise.UpdatedAt,
		})
	}
	return WorkoutResult{
		ID: value.ID, WorkoutID: value.WorkoutID, AthleteID: value.AthleteID, PerformedAt: value.PerformedAt,
		DurationSeconds: value.DurationSeconds, DistanceMeters: value.DistanceMeters, AvgHeartRate: value.AvgHeartRate,
		MaxHeartRate: value.MaxHeartRate, Calories: value.Calories, RPE: value.RPE, Feeling: value.Feeling,
		Comment: value.Comment, Exercises: exercises, Version: value.Version, CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt,
	}
}

type TrainingPlan struct {
	ID          uuid.UUID `json:"id"`
	CoachID     uuid.UUID `json:"coach_id"`
	AthleteID   uuid.UUID `json:"athlete_id"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	StartDate   string    `json:"start_date"`
	EndDate     *string   `json:"end_date,omitempty"`
	Status      string    `json:"status"`
	Version     int64     `json:"version"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func TrainingPlanFromEntity(value *entities.TrainingPlan) TrainingPlan {
	var endDate *string
	if value.EndDate != nil {
		date := string(*value.EndDate)
		endDate = &date
	}
	return TrainingPlan{
		ID: value.ID, CoachID: value.CoachID, AthleteID: value.AthleteID, Name: value.Name,
		Description: value.Description, StartDate: string(value.StartDate), EndDate: endDate,
		Status: string(value.Status), Version: value.Version, CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt,
	}
}

type Relationship struct {
	ID        uuid.UUID `json:"id"`
	CoachID   uuid.UUID `json:"coach_id"`
	AthleteID uuid.UUID `json:"athlete_id"`
	Status    string    `json:"status"`
	Version   int64     `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func RelationshipFromEntity(value *entities.CoachAthleteRelationship) Relationship {
	return Relationship{ID: value.ID, CoachID: value.CoachID, AthleteID: value.AthleteID, Status: string(value.Status), Version: value.Version, CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt}
}

type UserIDs struct {
	UserIDs []uuid.UUID `json:"user_ids"`
}

type DailyCheckIn struct {
	ID               uuid.UUID `json:"id"`
	AthleteID        uuid.UUID `json:"athlete_id"`
	Date             string    `json:"date"`
	SleepDurationMin *int      `json:"sleep_duration_min,omitempty"`
	SleepQuality     *int      `json:"sleep_quality,omitempty"`
	Energy           *int      `json:"energy,omitempty"`
	Stress           *int      `json:"stress,omitempty"`
	Soreness         *int      `json:"soreness,omitempty"`
	Motivation       *int      `json:"motivation,omitempty"`
	WeightKg         *float64  `json:"weight_kg,omitempty"`
	RestingHeartRate *int      `json:"resting_heart_rate,omitempty"`
	Comment          *string   `json:"comment,omitempty"`
	Version          int64     `json:"version"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func DailyCheckInFromEntity(value *entities.DailyCheckIn) DailyCheckIn {
	return DailyCheckIn{
		ID: value.ID, AthleteID: value.AthleteID, Date: string(value.Date), SleepDurationMin: value.SleepDurationMin,
		SleepQuality: value.SleepQuality, Energy: value.Energy, Stress: value.Stress, Soreness: value.Soreness,
		Motivation: value.Motivation, WeightKg: value.WeightKg, RestingHeartRate: value.RestingHeartRate,
		Comment: value.Comment, Version: value.Version, CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt,
	}
}

func DailyCheckInsFromEntities(values []entities.DailyCheckIn) []DailyCheckIn {
	result := make([]DailyCheckIn, 0, len(values))
	for i := range values {
		result = append(result, DailyCheckInFromEntity(&values[i]))
	}
	return result
}

type Goal struct {
	ID          uuid.UUID       `json:"id"`
	AthleteID   uuid.UUID       `json:"athlete_id"`
	CoachID     *uuid.UUID      `json:"coach_id,omitempty"`
	Title       string          `json:"title"`
	Description *string         `json:"description,omitempty"`
	Type        string          `json:"type"`
	Target      json.RawMessage `json:"target"`
	Deadline    *string         `json:"deadline,omitempty"`
	Status      string          `json:"status"`
	Version     int64           `json:"version"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

func GoalFromEntity(value *entities.Goal) Goal {
	var deadline *string
	if value.Deadline != nil {
		date := string(*value.Deadline)
		deadline = &date
	}
	return Goal{
		ID: value.ID, AthleteID: value.AthleteID, CoachID: value.CoachID, Title: value.Title,
		Description: value.Description, Type: string(value.Type), Target: value.Target, Deadline: deadline,
		Status: string(value.Status), Version: value.Version, CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt,
	}
}

func GoalsFromEntities(values []entities.Goal) []Goal {
	result := make([]Goal, 0, len(values))
	for i := range values {
		result = append(result, GoalFromEntity(&values[i]))
	}
	return result
}

type Comment struct {
	ID        uuid.UUID `json:"id"`
	WorkoutID uuid.UUID `json:"workout_id"`
	AuthorID  uuid.UUID `json:"author_id"`
	Text      string    `json:"text"`
	Version   int64     `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func CommentFromEntity(value *entities.WorkoutComment) Comment {
	return Comment{ID: value.ID, WorkoutID: value.WorkoutID, AuthorID: value.AuthorID, Text: value.Text, Version: value.Version, CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt}
}

func CommentsFromEntities(values []entities.WorkoutComment) []Comment {
	result := make([]Comment, 0, len(values))
	for i := range values {
		result = append(result, CommentFromEntity(&values[i]))
	}
	return result
}
