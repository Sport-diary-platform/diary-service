package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"time"

	"diary-service/internal/entities"
	"diary-service/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type workoutRepository struct{ db DB }

func NewWorkoutRepository(db DB) repository.WorkoutRepository { return &workoutRepository{db: db} }

func (r *workoutRepository) Create(ctx context.Context, workout *entities.Workout) error {
	const op = "WorkoutRepository.Create"
	err := runInTransaction(ctx, r.db, func(txCtx context.Context, db queryer) error {
		_, err := db.Exec(txCtx, `
			INSERT INTO workouts
				(id, training_plan_id, coach_id, athlete_id, title, description, sport_type,
				 scheduled_at, estimated_duration_seconds, status, version, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
			workout.ID, workout.TrainingPlanID, workout.CoachID, workout.AthleteID, workout.Title,
			workout.Description, workout.SportType, workout.ScheduledAt, workout.EstimatedDurationSeconds,
			workout.Status, workout.Version, workout.CreatedAt, workout.UpdatedAt)
		if err != nil {
			return err
		}
		return insertWorkoutChildren(txCtx, db, workout)
	})
	return wrap(op, err)
}

func insertWorkoutChildren(ctx context.Context, db queryer, workout *entities.Workout) error {
	for blockIndex := range workout.Blocks {
		block := &workout.Blocks[blockIndex]
		_, err := db.Exec(ctx, `
			INSERT INTO workout_blocks (id, workout_id, name, type, position, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7)`, block.ID, workout.ID, block.Name, block.Type,
			block.Position, block.CreatedAt, block.UpdatedAt)
		if err != nil {
			return err
		}
		for exerciseIndex := range block.Exercises {
			exercise := &block.Exercises[exerciseIndex]
			_, err = db.Exec(ctx, `
				INSERT INTO exercises
					(id, workout_block_id, name, type, position, target, created_at, updated_at)
				VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, exercise.ID, block.ID, exercise.Name,
				exercise.Type, exercise.Position, exercise.Target, exercise.CreatedAt, exercise.UpdatedAt)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

const workoutColumns = `id, training_plan_id::text, coach_id, athlete_id, title, description,
	sport_type, scheduled_at, estimated_duration_seconds, status, version, created_at, updated_at`

func scanWorkout(row pgx.Row) (*entities.Workout, error) {
	workout := new(entities.Workout)
	var planID *string
	err := row.Scan(&workout.ID, &planID, &workout.CoachID, &workout.AthleteID, &workout.Title,
		&workout.Description, &workout.SportType, &workout.ScheduledAt, &workout.EstimatedDurationSeconds,
		&workout.Status, &workout.Version, &workout.CreatedAt, &workout.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if planID != nil {
		id, parseErr := uuid.Parse(*planID)
		if parseErr != nil {
			return nil, parseErr
		}
		workout.TrainingPlanID = &id
	}
	return workout, nil
}

func (r *workoutRepository) GetByID(ctx context.Context, workoutID uuid.UUID) (*entities.Workout, error) {
	const op = "WorkoutRepository.GetByID"
	db := dbFromContext(ctx, r.db)
	workout, err := scanWorkout(db.QueryRow(ctx, "SELECT "+workoutColumns+" FROM workouts WHERE id=$1", workoutID))
	if err != nil {
		return nil, mapNotFound(op, err, entities.ErrWorkoutNotFound)
	}
	if err = loadWorkoutChildren(ctx, db, []*entities.Workout{workout}); err != nil {
		return nil, wrap(op, err)
	}
	return workout, nil
}

func (r *workoutRepository) Update(ctx context.Context, workout *entities.Workout) error {
	const op = "WorkoutRepository.Update"
	err := runInTransaction(ctx, r.db, func(txCtx context.Context, db queryer) error {
		tag, err := db.Exec(txCtx, `
			UPDATE workouts SET training_plan_id=$2, title=$3, description=$4, sport_type=$5,
				scheduled_at=$6, estimated_duration_seconds=$7, status=$8,
				version=version+1, updated_at=$9
			WHERE id=$1 AND version=$10`, workout.ID, workout.TrainingPlanID, workout.Title,
			workout.Description, workout.SportType, workout.ScheduledAt, workout.EstimatedDurationSeconds,
			workout.Status, workout.UpdatedAt, workout.Version)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return entities.ErrVersionConflict
		}
		equal, err := workoutChildrenEqual(txCtx, db, workout)
		if err != nil {
			return err
		}
		if !equal {
			if _, err = db.Exec(txCtx, "DELETE FROM workout_blocks WHERE workout_id=$1", workout.ID); err != nil {
				return err
			}
			return insertWorkoutChildren(txCtx, db, workout)
		}
		return nil
	})
	if err != nil {
		return wrap(op, err)
	}
	workout.Version++
	return nil
}

func workoutChildrenEqual(ctx context.Context, db queryer, desired *entities.Workout) (bool, error) {
	stored := &entities.Workout{ID: desired.ID}
	if err := loadWorkoutChildren(ctx, db, []*entities.Workout{stored}); err != nil {
		return false, err
	}
	if len(stored.Blocks) != len(desired.Blocks) {
		return false, nil
	}

	desiredBlocks := make(map[uuid.UUID]entities.WorkoutBlock, len(desired.Blocks))
	for _, block := range desired.Blocks {
		desiredBlocks[block.ID] = block
	}
	for _, currentBlock := range stored.Blocks {
		desiredBlock, ok := desiredBlocks[currentBlock.ID]
		if !ok || currentBlock.Name != desiredBlock.Name || currentBlock.Type != desiredBlock.Type ||
			currentBlock.Position != desiredBlock.Position || len(currentBlock.Exercises) != len(desiredBlock.Exercises) {
			return false, nil
		}
		desiredExercises := make(map[uuid.UUID]entities.WorkoutExercise, len(desiredBlock.Exercises))
		for _, exercise := range desiredBlock.Exercises {
			desiredExercises[exercise.ID] = exercise
		}
		for _, currentExercise := range currentBlock.Exercises {
			desiredExercise, exists := desiredExercises[currentExercise.ID]
			if !exists || currentExercise.Name != desiredExercise.Name || currentExercise.Type != desiredExercise.Type ||
				currentExercise.Position != desiredExercise.Position || !bytes.Equal(currentExercise.Target, desiredExercise.Target) {
				return false, nil
			}
		}
	}
	return true, nil
}

func (r *workoutRepository) Delete(ctx context.Context, workoutID uuid.UUID, expectedVersion int64) error {
	const op = "WorkoutRepository.Delete"
	tag, err := dbFromContext(ctx, r.db).Exec(ctx, "DELETE FROM workouts WHERE id=$1 AND version=$2", workoutID, expectedVersion)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return wrap(op, entities.ErrWorkoutHasResult)
		}
		return wrap(op, err)
	}
	if tag.RowsAffected() == 0 {
		return versionConflict(op)
	}
	return nil
}

func (r *workoutRepository) ListByAthlete(ctx context.Context, athleteID uuid.UUID, from, to time.Time) ([]entities.Workout, error) {
	return r.list(ctx, "WorkoutRepository.ListByAthlete",
		"SELECT "+workoutColumns+" FROM workouts WHERE athlete_id=$1 AND scheduled_at >= $2 AND scheduled_at <= $3 ORDER BY scheduled_at, id",
		athleteID, from, to)
}

func (r *workoutRepository) ListByCoach(ctx context.Context, coachID uuid.UUID, from, to time.Time) ([]entities.Workout, error) {
	return r.list(ctx, "WorkoutRepository.ListByCoach",
		"SELECT "+workoutColumns+" FROM workouts WHERE coach_id=$1 AND scheduled_at >= $2 AND scheduled_at <= $3 ORDER BY scheduled_at, id",
		coachID, from, to)
}

func (r *workoutRepository) list(ctx context.Context, op, sql string, args ...any) ([]entities.Workout, error) {
	db := dbFromContext(ctx, r.db)
	rows, err := db.Query(ctx, sql, args...)
	if err != nil {
		return nil, wrap(op, err)
	}
	defer rows.Close()

	workouts := make([]entities.Workout, 0)
	workoutPointers := make([]*entities.Workout, 0)
	for rows.Next() {
		workout, scanErr := scanWorkout(rows)
		if scanErr != nil {
			return nil, wrap(op, scanErr)
		}
		workouts = append(workouts, *workout)
	}
	if err = rows.Err(); err != nil {
		return nil, wrap(op, err)
	}
	for i := range workouts {
		workoutPointers = append(workoutPointers, &workouts[i])
	}
	if err = loadWorkoutChildren(ctx, db, workoutPointers); err != nil {
		return nil, wrap(op, err)
	}
	return workouts, nil
}

func loadWorkoutChildren(ctx context.Context, db queryer, workouts []*entities.Workout) error {
	if len(workouts) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, len(workouts))
	byID := make(map[uuid.UUID]*entities.Workout, len(workouts))
	for i, workout := range workouts {
		ids[i] = workout.ID
		byID[workout.ID] = workout
		workout.Blocks = make([]entities.WorkoutBlock, 0)
	}

	rows, err := db.Query(ctx, `
		SELECT b.id, b.workout_id, b.name, b.type, b.position, b.created_at, b.updated_at,
			e.id::text, e.name, e.type, e.position, e.target, e.created_at, e.updated_at
		FROM workout_blocks b
		LEFT JOIN exercises e ON e.workout_block_id = b.id
		WHERE b.workout_id = ANY($1)
		ORDER BY b.workout_id, b.position, b.id, e.position, e.id`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()

	blocks := make(map[uuid.UUID]*entities.WorkoutBlock)
	for rows.Next() {
		var blockID, workoutID uuid.UUID
		var name string
		var blockType entities.WorkoutBlockType
		var position int
		var createdAt, updatedAt time.Time
		var exerciseID, exerciseName, exerciseType *string
		var exercisePosition *int
		var target json.RawMessage
		var exerciseCreatedAt, exerciseUpdatedAt *time.Time
		if err = rows.Scan(&blockID, &workoutID, &name, &blockType, &position, &createdAt, &updatedAt,
			&exerciseID, &exerciseName, &exerciseType, &exercisePosition, &target,
			&exerciseCreatedAt, &exerciseUpdatedAt); err != nil {
			return err
		}
		block := blocks[blockID]
		if block == nil {
			workout := byID[workoutID]
			workout.Blocks = append(workout.Blocks, entities.WorkoutBlock{
				ID: blockID, WorkoutID: workoutID, Name: name, Type: blockType, Position: position,
				Exercises: make([]entities.WorkoutExercise, 0), CreatedAt: createdAt, UpdatedAt: updatedAt,
			})
			block = &workout.Blocks[len(workout.Blocks)-1]
			blocks[blockID] = block
		}
		if exerciseID != nil {
			id, parseErr := uuid.Parse(*exerciseID)
			if parseErr != nil {
				return parseErr
			}
			block.Exercises = append(block.Exercises, entities.WorkoutExercise{
				ID: id, WorkoutBlockID: blockID, Name: *exerciseName, Type: *exerciseType,
				Position: *exercisePosition, Target: target, CreatedAt: *exerciseCreatedAt, UpdatedAt: *exerciseUpdatedAt,
			})
		}
	}
	return rows.Err()
}
