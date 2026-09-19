package usecase

import (
	"context"
	"math"
	"time"

	"diary-service/internal/entities"
	"diary-service/internal/repository"

	"github.com/google/uuid"
)

type workoutResultUseCase struct {
	repository    repository.WorkoutResultRepository
	workouts      repository.WorkoutRepository
	relationships repository.RelationshipRepository
	outbox        repository.OutboxRepository
	tx            repository.TransactionManager
	profiles      ProfileProvider
}

func NewWorkoutResultUseCase(
	results repository.WorkoutResultRepository,
	workouts repository.WorkoutRepository,
	relationships repository.RelationshipRepository,
	outbox repository.OutboxRepository,
	tx repository.TransactionManager,
	profiles ProfileProvider,
) WorkoutResultUseCase {
	return &workoutResultUseCase{
		repository: results, workouts: workouts, relationships: relationships,
		outbox: outbox, tx: tx, profiles: profiles,
	}
}

func validateWorkoutResult(
	performedAt time.Time,
	durationSeconds *int,
	distanceMeters *float64,
	avgHeartRate, maxHeartRate, calories *int,
	rpe, feeling int,
	exercises []ExerciseResultCommand,
) error {
	if performedAt.IsZero() {
		return invalid("performed_at", "must be set")
	}
	if durationSeconds != nil && *durationSeconds <= 0 {
		return invalid("duration_seconds", "must be greater than zero")
	}
	if distanceMeters != nil && (math.IsNaN(*distanceMeters) || math.IsInf(*distanceMeters, 0) || *distanceMeters < 0) {
		return invalid("distance_meters", "must be finite and not negative")
	}
	if avgHeartRate != nil && *avgHeartRate <= 0 {
		return invalid("avg_heart_rate", "must be greater than zero")
	}
	if maxHeartRate != nil && *maxHeartRate <= 0 {
		return invalid("max_heart_rate", "must be greater than zero")
	}
	if avgHeartRate != nil && maxHeartRate != nil && *maxHeartRate < *avgHeartRate {
		return invalid("max_heart_rate", "must not be less than avg_heart_rate")
	}
	if calories != nil && *calories < 0 {
		return invalid("calories", "must not be negative")
	}
	if rpe < 1 || rpe > 10 {
		return entities.ErrInvalidRPE
	}
	if feeling < 1 || feeling > 10 {
		return entities.ErrInvalidFeeling
	}

	exerciseIDs := make(map[uuid.UUID]struct{}, len(exercises))
	for _, exercise := range exercises {
		if exercise.ExerciseID == uuid.Nil {
			return invalid("exercises.exercise_id", "must be set")
		}
		if _, exists := exerciseIDs[exercise.ExerciseID]; exists {
			return invalid("exercises.exercise_id", "must be unique")
		}
		exerciseIDs[exercise.ExerciseID] = struct{}{}
		if err := validateJSON("exercises.actual", exercise.Actual); err != nil {
			return err
		}
	}

	return nil
}

func validateExerciseOwnership(workout *entities.Workout, commands []ExerciseResultCommand) error {
	available := make(map[uuid.UUID]struct{})
	for _, block := range workout.Blocks {
		for _, exercise := range block.Exercises {
			available[exercise.ID] = struct{}{}
		}
	}
	for _, command := range commands {
		if _, exists := available[command.ExerciseID]; !exists {
			return invalid("exercises.exercise_id", "exercise does not belong to workout")
		}
	}

	return nil
}

func buildExerciseResults(resultID uuid.UUID, commands []ExerciseResultCommand, now time.Time) []entities.WorkoutExerciseResult {
	results := make([]entities.WorkoutExerciseResult, 0, len(commands))
	for _, command := range commands {
		results = append(results, entities.WorkoutExerciseResult{
			ID: uuid.New(), WorkoutResultID: resultID, ExerciseID: command.ExerciseID,
			Actual: cloneJSON(command.Actual), Comment: command.Comment, Version: 1, CreatedAt: now, UpdatedAt: now,
		})
	}

	return results
}

func (uc *workoutResultUseCase) Create(ctx context.Context, cmd CreateWorkoutResultCommand) (*entities.WorkoutResult, error) {
	if err := requireProfile(ctx, uc.profiles, cmd.ActorID, ProfileTypeAthlete); err != nil {
		return nil, err
	}
	if cmd.WorkoutID == uuid.Nil {
		return nil, invalid("workout_id", "must be set")
	}
	if err := validExpectedVersion(cmd.ExpectedWorkoutVersion); err != nil {
		return nil, err
	}
	if err := validateWorkoutResult(cmd.PerformedAt, cmd.DurationSeconds, cmd.DistanceMeters, cmd.AvgHeartRate, cmd.MaxHeartRate, cmd.Calories, cmd.RPE, cmd.Feeling, cmd.Exercises); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	var result *entities.WorkoutResult
	err := runTransaction(ctx, uc.tx, "WorkoutResultUseCase.Create", func(txCtx context.Context) error {
		workout, err := uc.workouts.GetByID(txCtx, cmd.WorkoutID)
		if err != nil {
			return operationError("WorkoutRepository.GetByID", err)
		}
		if workout == nil {
			return entities.ErrWorkoutNotFound
		}
		if workout.AthleteID != cmd.ActorID {
			return ErrForbidden
		}
		if err := ensureWorkoutPlanned(workout); err != nil {
			return err
		}
		if err := versionMatches(workout.Version, cmd.ExpectedWorkoutVersion); err != nil {
			return err
		}
		exists, err := uc.repository.ExistsByWorkoutID(txCtx, workout.ID)
		if err != nil {
			return operationError("WorkoutResultRepository.ExistsByWorkoutID", err)
		}
		if exists {
			return entities.ErrWorkoutResultAlreadyExists
		}
		if err := validateExerciseOwnership(workout, cmd.Exercises); err != nil {
			return err
		}

		result = &entities.WorkoutResult{
			ID: uuid.New(), WorkoutID: workout.ID, AthleteID: cmd.ActorID, PerformedAt: cmd.PerformedAt,
			DurationSeconds: cmd.DurationSeconds, DistanceMeters: cmd.DistanceMeters,
			AvgHeartRate: cmd.AvgHeartRate, MaxHeartRate: cmd.MaxHeartRate, Calories: cmd.Calories,
			RPE: cmd.RPE, Feeling: cmd.Feeling, Comment: cmd.Comment,
			Version: 1, CreatedAt: now, UpdatedAt: now,
		}
		result.ExerciseResults = buildExerciseResults(result.ID, cmd.Exercises, now)
		if err := uc.repository.Create(txCtx, result); err != nil {
			return operationError("WorkoutResultRepository.Create", err)
		}

		if err := workout.Complete(); err != nil {
			return err
		}
		workout.UpdatedAt = now
		if err := uc.workouts.Update(txCtx, workout); err != nil {
			return operationError("WorkoutRepository.Update", err)
		}
		if err := createEvent(txCtx, uc.outbox, "workout_result.created", "workout_result", result.ID, resultEventPayload(result, workout.CoachID), now); err != nil {
			return err
		}
		if err := createEvent(txCtx, uc.outbox, "workout.completed", "workout", workout.ID, workoutEventPayload(workout), now); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (uc *workoutResultUseCase) GetByWorkoutID(ctx context.Context, actorID, workoutID uuid.UUID) (*entities.WorkoutResult, error) {
	if err := requireActor(actorID); err != nil {
		return nil, err
	}
	if workoutID == uuid.Nil {
		return nil, invalid("workout_id", "must be set")
	}

	workout, err := uc.workouts.GetByID(ctx, workoutID)
	if err != nil {
		return nil, operationError("WorkoutRepository.GetByID", err)
	}
	if workout == nil {
		return nil, entities.ErrWorkoutNotFound
	}
	if err := uc.authorizeRead(ctx, actorID, workout); err != nil {
		return nil, err
	}
	result, err := uc.repository.GetByWorkoutID(ctx, workoutID)
	if err != nil {
		return nil, operationError("WorkoutResultRepository.GetByWorkoutID", err)
	}
	if result == nil {
		return nil, entities.ErrWorkoutResultNotFound
	}

	return result, nil
}

func (uc *workoutResultUseCase) Update(ctx context.Context, cmd UpdateWorkoutResultCommand) (*entities.WorkoutResult, error) {
	if err := requireProfile(ctx, uc.profiles, cmd.ActorID, ProfileTypeAthlete); err != nil {
		return nil, err
	}
	if cmd.WorkoutID == uuid.Nil {
		return nil, invalid("workout_id", "must be set")
	}
	if err := validExpectedVersion(cmd.ExpectedVersion); err != nil {
		return nil, err
	}
	if err := validateWorkoutResult(cmd.PerformedAt, cmd.DurationSeconds, cmd.DistanceMeters, cmd.AvgHeartRate, cmd.MaxHeartRate, cmd.Calories, cmd.RPE, cmd.Feeling, cmd.Exercises); err != nil {
		return nil, err
	}

	var result *entities.WorkoutResult
	now := time.Now().UTC()
	err := runTransaction(ctx, uc.tx, "WorkoutResultUseCase.Update", func(txCtx context.Context) error {
		workout, err := uc.workouts.GetByID(txCtx, cmd.WorkoutID)
		if err != nil {
			return operationError("WorkoutRepository.GetByID", err)
		}
		if workout == nil {
			return entities.ErrWorkoutNotFound
		}
		if workout.AthleteID != cmd.ActorID {
			return ErrForbidden
		}
		if err := validateExerciseOwnership(workout, cmd.Exercises); err != nil {
			return err
		}

		result, err = uc.repository.GetByWorkoutID(txCtx, workout.ID)
		if err != nil {
			return operationError("WorkoutResultRepository.GetByWorkoutID", err)
		}
		if result == nil {
			return entities.ErrWorkoutResultNotFound
		}
		if result.AthleteID != cmd.ActorID {
			return ErrForbidden
		}
		if err := versionMatches(result.Version, cmd.ExpectedVersion); err != nil {
			return err
		}

		result.PerformedAt = cmd.PerformedAt
		result.DurationSeconds = cmd.DurationSeconds
		result.DistanceMeters = cmd.DistanceMeters
		result.AvgHeartRate = cmd.AvgHeartRate
		result.MaxHeartRate = cmd.MaxHeartRate
		result.Calories = cmd.Calories
		result.RPE = cmd.RPE
		result.Feeling = cmd.Feeling
		result.Comment = cmd.Comment
		result.ExerciseResults = buildExerciseResults(result.ID, cmd.Exercises, now)
		result.UpdatedAt = now
		if err := uc.repository.Update(txCtx, result); err != nil {
			return operationError("WorkoutResultRepository.Update", err)
		}
		if err := createEvent(txCtx, uc.outbox, "workout_result.updated", "workout_result", result.ID, resultEventPayload(result, workout.CoachID), now); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (uc *workoutResultUseCase) authorizeRead(ctx context.Context, actorID uuid.UUID, workout *entities.Workout) error {
	if actorID == workout.AthleteID {
		return requireProfile(ctx, uc.profiles, actorID, ProfileTypeAthlete)
	}
	if err := requireProfile(ctx, uc.profiles, actorID, ProfileTypeCoach); err != nil {
		return err
	}

	return requireActiveRelationship(ctx, uc.relationships, actorID, workout.AthleteID)
}

func resultEventPayload(result *entities.WorkoutResult, coachID uuid.UUID) map[string]any {
	return map[string]any{
		"workout_result_id": result.ID,
		"workout_id":        result.WorkoutID,
		"athlete_id":        result.AthleteID,
		"coach_id":          coachID,
	}
}
