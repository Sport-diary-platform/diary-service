package postgres

import (
	"context"

	"diary-service/internal/entities"
	"diary-service/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type goalRepository struct{ db DB }

func NewGoalRepository(db DB) repository.GoalRepository { return &goalRepository{db: db} }

func (r *goalRepository) Create(ctx context.Context, goal *entities.Goal) error {
	const op = "GoalRepository.Create"
	deadline, err := nullableLocalDateValue(goal.Deadline)
	if err != nil {
		return wrap(op, err)
	}
	_, err = dbFromContext(ctx, r.db).Exec(ctx, `
		INSERT INTO goals
			(id, athlete_id, coach_id, title, description, type, target, deadline, status, version, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		goal.ID, goal.AthleteID, goal.CoachID, goal.Title, goal.Description, goal.Type, goal.Target,
		deadline, goal.Status, goal.Version, goal.CreatedAt, goal.UpdatedAt)
	return wrap(op, err)
}

const goalColumns = `id, athlete_id, coach_id, title, description, type, target, deadline::text,
	status, version, created_at, updated_at`

func scanGoal(row pgx.Row) (*entities.Goal, error) {
	goal := new(entities.Goal)
	var deadline *string
	err := row.Scan(&goal.ID, &goal.AthleteID, &goal.CoachID, &goal.Title, &goal.Description,
		&goal.Type, &goal.Target, &deadline, &goal.Status, &goal.Version, &goal.CreatedAt, &goal.UpdatedAt)
	if deadline != nil {
		value := entities.LocalDate(*deadline)
		goal.Deadline = &value
	}
	return goal, err
}

func (r *goalRepository) GetByID(ctx context.Context, goalID uuid.UUID) (*entities.Goal, error) {
	const op = "GoalRepository.GetByID"
	goal, err := scanGoal(dbFromContext(ctx, r.db).QueryRow(ctx,
		"SELECT "+goalColumns+" FROM goals WHERE id = $1", goalID))
	if err != nil {
		return nil, mapNotFound(op, err, entities.ErrGoalNotFound)
	}
	return goal, nil
}

func (r *goalRepository) Update(ctx context.Context, goal *entities.Goal) error {
	const op = "GoalRepository.Update"
	deadline, err := nullableLocalDateValue(goal.Deadline)
	if err != nil {
		return wrap(op, err)
	}
	tag, err := dbFromContext(ctx, r.db).Exec(ctx, `
		UPDATE goals SET title=$2, description=$3, type=$4, target=$5, deadline=$6,
			status=$7, version=version+1, updated_at=$8
		WHERE id=$1 AND version=$9`, goal.ID, goal.Title, goal.Description, goal.Type,
		goal.Target, deadline, goal.Status, goal.UpdatedAt, goal.Version)
	if err != nil {
		return wrap(op, err)
	}
	if tag.RowsAffected() == 0 {
		return versionConflict(op)
	}
	goal.Version++
	return nil
}

func (r *goalRepository) ListByAthlete(ctx context.Context, athleteID uuid.UUID) ([]entities.Goal, error) {
	const op = "GoalRepository.ListByAthlete"
	rows, err := dbFromContext(ctx, r.db).Query(ctx,
		"SELECT "+goalColumns+" FROM goals WHERE athlete_id=$1 ORDER BY created_at, id", athleteID)
	if err != nil {
		return nil, wrap(op, err)
	}
	defer rows.Close()

	goals := make([]entities.Goal, 0)
	for rows.Next() {
		goal, scanErr := scanGoal(rows)
		if scanErr != nil {
			return nil, wrap(op, scanErr)
		}
		goals = append(goals, *goal)
	}
	if err = rows.Err(); err != nil {
		return nil, wrap(op, err)
	}
	return goals, nil
}
