package postgres

import (
	"context"

	"diary-service/internal/entities"
	"diary-service/internal/repository"

	"github.com/google/uuid"
)

type relationshipRepository struct{ db DB }

func NewRelationshipRepository(db DB) repository.RelationshipRepository {
	return &relationshipRepository{db: db}
}

func (r *relationshipRepository) Create(ctx context.Context, relationship *entities.CoachAthleteRelationship) error {
	const op = "RelationshipRepository.Create"
	_, err := dbFromContext(ctx, r.db).Exec(ctx, `
		INSERT INTO coach_athlete_relationships
			(id, coach_id, athlete_id, status, version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		relationship.ID, relationship.CoachID, relationship.AthleteID, relationship.Status,
		relationship.Version, relationship.CreatedAt, relationship.UpdatedAt)
	if err != nil {
		return mapUnique(op, err, entities.ErrRelationshipAlreadyExists)
	}
	return nil
}

func (r *relationshipRepository) GetByID(ctx context.Context, relationshipID uuid.UUID) (*entities.CoachAthleteRelationship, error) {
	const op = "RelationshipRepository.GetByID"
	rel := new(entities.CoachAthleteRelationship)
	err := dbFromContext(ctx, r.db).QueryRow(ctx, `
		SELECT id, coach_id, athlete_id, status, version, created_at, updated_at
		FROM coach_athlete_relationships WHERE id = $1`, relationshipID).
		Scan(&rel.ID, &rel.CoachID, &rel.AthleteID, &rel.Status, &rel.Version, &rel.CreatedAt, &rel.UpdatedAt)
	if err != nil {
		return nil, mapNotFound(op, err, entities.ErrRelationshipNotFound)
	}
	return rel, nil
}

func (r *relationshipRepository) GetActive(ctx context.Context, coachID, athleteID uuid.UUID) (*entities.CoachAthleteRelationship, error) {
	const op = "RelationshipRepository.GetActive"
	rel := new(entities.CoachAthleteRelationship)
	err := dbFromContext(ctx, r.db).QueryRow(ctx, `
		SELECT id, coach_id, athlete_id, status, version, created_at, updated_at
		FROM coach_athlete_relationships
		WHERE coach_id = $1 AND athlete_id = $2 AND status = 'active'`, coachID, athleteID).
		Scan(&rel.ID, &rel.CoachID, &rel.AthleteID, &rel.Status, &rel.Version, &rel.CreatedAt, &rel.UpdatedAt)
	if err != nil {
		return nil, mapNotFound(op, err, entities.ErrRelationshipNotFound)
	}
	return rel, nil
}

func (r *relationshipRepository) IsActive(ctx context.Context, coachID, athleteID uuid.UUID) (bool, error) {
	const op = "RelationshipRepository.IsActive"
	var active bool
	err := dbFromContext(ctx, r.db).QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM coach_athlete_relationships
			WHERE coach_id = $1 AND athlete_id = $2 AND status = 'active'
		)`, coachID, athleteID).Scan(&active)
	return active, wrap(op, err)
}

func (r *relationshipRepository) Terminate(ctx context.Context, relationshipID uuid.UUID, expectedVersion int64) error {
	const op = "RelationshipRepository.Terminate"
	tag, err := dbFromContext(ctx, r.db).Exec(ctx, `
		UPDATE coach_athlete_relationships
		SET status = 'terminated', version = version + 1, updated_at = NOW()
		WHERE id = $1 AND version = $2`, relationshipID, expectedVersion)
	if err != nil {
		return wrap(op, err)
	}
	if tag.RowsAffected() == 0 {
		return versionConflict(op)
	}
	return nil
}

func (r *relationshipRepository) ListAthletes(ctx context.Context, coachID uuid.UUID) ([]uuid.UUID, error) {
	return r.listIDs(ctx, "RelationshipRepository.ListAthletes", `
		SELECT athlete_id FROM coach_athlete_relationships
		WHERE coach_id = $1 AND status = 'active'
		ORDER BY created_at, athlete_id`, coachID)
}

func (r *relationshipRepository) ListCoaches(ctx context.Context, athleteID uuid.UUID) ([]uuid.UUID, error) {
	return r.listIDs(ctx, "RelationshipRepository.ListCoaches", `
		SELECT coach_id FROM coach_athlete_relationships
		WHERE athlete_id = $1 AND status = 'active'
		ORDER BY created_at, coach_id`, athleteID)
}

func (r *relationshipRepository) listIDs(ctx context.Context, op, sql string, id uuid.UUID) ([]uuid.UUID, error) {
	rows, err := dbFromContext(ctx, r.db).Query(ctx, sql, id)
	if err != nil {
		return nil, wrap(op, err)
	}
	defer rows.Close()

	ids := make([]uuid.UUID, 0)
	for rows.Next() {
		var value uuid.UUID
		if err = rows.Scan(&value); err != nil {
			return nil, wrap(op, err)
		}
		ids = append(ids, value)
	}
	if err = rows.Err(); err != nil {
		return nil, wrap(op, err)
	}
	return ids, nil
}
