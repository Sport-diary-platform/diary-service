package postgres

import (
	"context"

	"diary-service/internal/entities"
	"diary-service/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type workoutResultRepository struct{ db DB }

func NewWorkoutResultRepository(db DB) repository.WorkoutResultRepository {
	return &workoutResultRepository{db: db}
}

func (r *workoutResultRepository) Create(ctx context.Context, result *entities.WorkoutResult) error {
	const op = "WorkoutResultRepository.Create"
	err := runInTransaction(ctx, r.db, func(txCtx context.Context, db queryer) error {
		_, err := db.Exec(txCtx, `
			INSERT INTO workout_results
				(id, workout_id, athlete_id, performed_at, duration_seconds, distance_meters,
				 avg_heart_rate, max_heart_rate, calories, rpe, feeling, comment, version, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`, result.ID,
			result.WorkoutID, result.AthleteID, result.PerformedAt, result.DurationSeconds,
			result.DistanceMeters, result.AvgHeartRate, result.MaxHeartRate, result.Calories,
			result.RPE, result.Feeling, result.Comment, result.Version, result.CreatedAt, result.UpdatedAt)
		if err != nil {
			return err
		}
		return insertExerciseResults(txCtx, db, result)
	})
	if err != nil {
		return mapUnique(op, err, entities.ErrWorkoutResultAlreadyExists)
	}
	return nil
}

func insertExerciseResults(ctx context.Context, db queryer, result *entities.WorkoutResult) error {
	for i := range result.ExerciseResults {
		exercise := &result.ExerciseResults[i]
		_, err := db.Exec(ctx, `
			INSERT INTO exercise_results
				(id, workout_result_id, exercise_id, actual, comment, version, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, exercise.ID, result.ID, exercise.ExerciseID,
			exercise.Actual, exercise.Comment, exercise.Version, exercise.CreatedAt, exercise.UpdatedAt)
		if err != nil {
			return err
		}
	}
	return nil
}

const workoutResultColumns = `id, workout_id, athlete_id, performed_at, duration_seconds,
	distance_meters, avg_heart_rate, max_heart_rate, calories, rpe, feeling, comment,
	version, created_at, updated_at`

func scanWorkoutResult(row pgx.Row) (*entities.WorkoutResult, error) {
	result := new(entities.WorkoutResult)
	err := row.Scan(&result.ID, &result.WorkoutID, &result.AthleteID, &result.PerformedAt,
		&result.DurationSeconds, &result.DistanceMeters, &result.AvgHeartRate, &result.MaxHeartRate,
		&result.Calories, &result.RPE, &result.Feeling, &result.Comment, &result.Version,
		&result.CreatedAt, &result.UpdatedAt)
	return result, err
}

func (r *workoutResultRepository) GetByWorkoutID(ctx context.Context, workoutID uuid.UUID) (*entities.WorkoutResult, error) {
	const op = "WorkoutResultRepository.GetByWorkoutID"
	db := dbFromContext(ctx, r.db)
	result, err := scanWorkoutResult(db.QueryRow(ctx,
		"SELECT "+workoutResultColumns+" FROM workout_results WHERE workout_id=$1", workoutID))
	if err != nil {
		return nil, mapNotFound(op, err, entities.ErrWorkoutResultNotFound)
	}
	rows, err := db.Query(ctx, `
		SELECT id, workout_result_id, exercise_id, actual, comment, version, created_at, updated_at
		FROM exercise_results WHERE workout_result_id=$1 ORDER BY created_at, id`, result.ID)
	if err != nil {
		return nil, wrap(op, err)
	}
	defer rows.Close()
	result.ExerciseResults = make([]entities.WorkoutExerciseResult, 0)
	for rows.Next() {
		var exercise entities.WorkoutExerciseResult
		if err = rows.Scan(&exercise.ID, &exercise.WorkoutResultID, &exercise.ExerciseID,
			&exercise.Actual, &exercise.Comment, &exercise.Version, &exercise.CreatedAt, &exercise.UpdatedAt); err != nil {
			return nil, wrap(op, err)
		}
		result.ExerciseResults = append(result.ExerciseResults, exercise)
	}
	if err = rows.Err(); err != nil {
		return nil, wrap(op, err)
	}
	return result, nil
}

func (r *workoutResultRepository) Update(ctx context.Context, result *entities.WorkoutResult) error {
	const op = "WorkoutResultRepository.Update"
	err := runInTransaction(ctx, r.db, func(txCtx context.Context, db queryer) error {
		tag, err := db.Exec(txCtx, `
			UPDATE workout_results SET performed_at=$2, duration_seconds=$3, distance_meters=$4,
				avg_heart_rate=$5, max_heart_rate=$6, calories=$7, rpe=$8, feeling=$9,
				comment=$10, version=version+1, updated_at=$11
			WHERE id=$1 AND version=$12`, result.ID, result.PerformedAt, result.DurationSeconds,
			result.DistanceMeters, result.AvgHeartRate, result.MaxHeartRate, result.Calories,
			result.RPE, result.Feeling, result.Comment, result.UpdatedAt, result.Version)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return entities.ErrVersionConflict
		}
		if _, err = db.Exec(txCtx, "DELETE FROM exercise_results WHERE workout_result_id=$1", result.ID); err != nil {
			return err
		}
		return insertExerciseResults(txCtx, db, result)
	})
	if err != nil {
		return wrap(op, err)
	}
	result.Version++
	return nil
}

func (r *workoutResultRepository) ExistsByWorkoutID(ctx context.Context, workoutID uuid.UUID) (bool, error) {
	const op = "WorkoutResultRepository.ExistsByWorkoutID"
	var exists bool
	err := dbFromContext(ctx, r.db).QueryRow(ctx,
		"SELECT EXISTS (SELECT 1 FROM workout_results WHERE workout_id=$1)", workoutID).Scan(&exists)
	return exists, wrap(op, err)
}
