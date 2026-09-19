package postgres

import (
	"context"

	"diary-service/internal/entities"
	"diary-service/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type trainingPlanRepository struct{ db DB }

func NewTrainingPlanRepository(db DB) repository.TrainingPlanRepository {
	return &trainingPlanRepository{db: db}
}

func (r *trainingPlanRepository) Create(ctx context.Context, plan *entities.TrainingPlan) error {
	const op = "TrainingPlanRepository.Create"
	startDate, err := localDateValue(plan.StartDate)
	if err != nil {
		return wrap(op, err)
	}
	endDate, err := nullableLocalDateValue(plan.EndDate)
	if err != nil {
		return wrap(op, err)
	}
	_, err = dbFromContext(ctx, r.db).Exec(ctx, `
		INSERT INTO training_plans
			(id, coach_id, athlete_id, name, description, start_date, end_date, status, version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		plan.ID, plan.CoachID, plan.AthleteID, plan.Name, plan.Description, startDate,
		endDate, plan.Status, plan.Version, plan.CreatedAt, plan.UpdatedAt)
	return wrap(op, err)
}

const trainingPlanColumns = `id, coach_id, athlete_id, name, description, start_date::text,
	end_date::text, status, version, created_at, updated_at`

func scanTrainingPlan(row pgx.Row) (*entities.TrainingPlan, error) {
	plan := new(entities.TrainingPlan)
	var startDate string
	var endDate *string
	err := row.Scan(&plan.ID, &plan.CoachID, &plan.AthleteID, &plan.Name, &plan.Description,
		&startDate, &endDate, &plan.Status, &plan.Version, &plan.CreatedAt, &plan.UpdatedAt)
	plan.StartDate = entities.LocalDate(startDate)
	if endDate != nil {
		value := entities.LocalDate(*endDate)
		plan.EndDate = &value
	}
	return plan, err
}

func (r *trainingPlanRepository) GetByID(ctx context.Context, planID uuid.UUID) (*entities.TrainingPlan, error) {
	const op = "TrainingPlanRepository.GetByID"
	plan, err := scanTrainingPlan(dbFromContext(ctx, r.db).QueryRow(ctx,
		"SELECT "+trainingPlanColumns+" FROM training_plans WHERE id = $1", planID))
	if err != nil {
		return nil, mapNotFound(op, err, entities.ErrTrainingPlanNotFound)
	}
	return plan, nil
}

func (r *trainingPlanRepository) Update(ctx context.Context, plan *entities.TrainingPlan) error {
	const op = "TrainingPlanRepository.Update"
	startDate, err := localDateValue(plan.StartDate)
	if err != nil {
		return wrap(op, err)
	}
	endDate, err := nullableLocalDateValue(plan.EndDate)
	if err != nil {
		return wrap(op, err)
	}
	tag, err := dbFromContext(ctx, r.db).Exec(ctx, `
		UPDATE training_plans
		SET name = $2, description = $3, start_date = $4, end_date = $5, status = $6,
			version = version + 1, updated_at = $7
		WHERE id = $1 AND version = $8`, plan.ID, plan.Name, plan.Description, startDate,
		endDate, plan.Status, plan.UpdatedAt, plan.Version)
	if err != nil {
		return wrap(op, err)
	}
	if tag.RowsAffected() == 0 {
		return versionConflict(op)
	}
	plan.Version++
	return nil
}

func (r *trainingPlanRepository) ListByAthlete(ctx context.Context, athleteID uuid.UUID) ([]entities.TrainingPlan, error) {
	return r.list(ctx, "TrainingPlanRepository.ListByAthlete",
		"SELECT "+trainingPlanColumns+" FROM training_plans WHERE athlete_id = $1 ORDER BY start_date, created_at, id", athleteID)
}

func (r *trainingPlanRepository) ListByCoach(ctx context.Context, coachID uuid.UUID) ([]entities.TrainingPlan, error) {
	return r.list(ctx, "TrainingPlanRepository.ListByCoach",
		"SELECT "+trainingPlanColumns+" FROM training_plans WHERE coach_id = $1 ORDER BY start_date, created_at, id", coachID)
}

func (r *trainingPlanRepository) list(ctx context.Context, op, sql string, id uuid.UUID) ([]entities.TrainingPlan, error) {
	rows, err := dbFromContext(ctx, r.db).Query(ctx, sql, id)
	if err != nil {
		return nil, wrap(op, err)
	}
	defer rows.Close()

	plans := make([]entities.TrainingPlan, 0)
	for rows.Next() {
		plan, scanErr := scanTrainingPlan(rows)
		if scanErr != nil {
			return nil, wrap(op, scanErr)
		}
		plans = append(plans, *plan)
	}
	if err = rows.Err(); err != nil {
		return nil, wrap(op, err)
	}
	return plans, nil
}
