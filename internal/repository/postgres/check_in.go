package postgres

import (
	"context"

	"diary-service/internal/entities"
	"diary-service/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type dailyCheckInRepository struct{ db DB }

func NewDailyCheckInRepository(db DB) repository.DailyCheckInRepository {
	return &dailyCheckInRepository{db: db}
}

func (r *dailyCheckInRepository) Create(ctx context.Context, checkIn *entities.DailyCheckIn) error {
	const op = "DailyCheckInRepository.Create"
	date, err := localDateValue(checkIn.Date)
	if err != nil {
		return wrap(op, err)
	}
	_, err = dbFromContext(ctx, r.db).Exec(ctx, `
		INSERT INTO daily_check_ins
			(id, athlete_id, date, sleep_duration_min, sleep_quality, energy, stress, soreness,
			 motivation, weight_kg, resting_heart_rate, comment, version, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		checkIn.ID, checkIn.AthleteID, date, checkIn.SleepDurationMin, checkIn.SleepQuality,
		checkIn.Energy, checkIn.Stress, checkIn.Soreness, checkIn.Motivation, checkIn.WeightKg,
		checkIn.RestingHeartRate, checkIn.Comment, checkIn.Version, checkIn.CreatedAt, checkIn.UpdatedAt)
	if err != nil {
		return mapUnique(op, err, entities.ErrDailyCheckInAlreadyExists)
	}
	return nil
}

const checkInColumns = `id, athlete_id, date::text, sleep_duration_min, sleep_quality, energy,
	stress, soreness, motivation, weight_kg, resting_heart_rate, comment, version, created_at, updated_at`

func scanCheckIn(row pgx.Row) (*entities.DailyCheckIn, error) {
	checkIn := new(entities.DailyCheckIn)
	var date string
	err := row.Scan(&checkIn.ID, &checkIn.AthleteID, &date, &checkIn.SleepDurationMin,
		&checkIn.SleepQuality, &checkIn.Energy, &checkIn.Stress, &checkIn.Soreness,
		&checkIn.Motivation, &checkIn.WeightKg, &checkIn.RestingHeartRate, &checkIn.Comment,
		&checkIn.Version, &checkIn.CreatedAt, &checkIn.UpdatedAt)
	checkIn.Date = entities.LocalDate(date)
	return checkIn, err
}

func (r *dailyCheckInRepository) GetByDate(ctx context.Context, athleteID uuid.UUID, date entities.LocalDate) (*entities.DailyCheckIn, error) {
	const op = "DailyCheckInRepository.GetByDate"
	dateValue, err := localDateValue(date)
	if err != nil {
		return nil, wrap(op, err)
	}
	checkIn, err := scanCheckIn(dbFromContext(ctx, r.db).QueryRow(ctx,
		"SELECT "+checkInColumns+" FROM daily_check_ins WHERE athlete_id=$1 AND date=$2", athleteID, dateValue))
	if err != nil {
		return nil, mapNotFound(op, err, entities.ErrDailyCheckInNotFound)
	}
	return checkIn, nil
}

func (r *dailyCheckInRepository) Update(ctx context.Context, checkIn *entities.DailyCheckIn) error {
	const op = "DailyCheckInRepository.Update"
	date, err := localDateValue(checkIn.Date)
	if err != nil {
		return wrap(op, err)
	}
	tag, err := dbFromContext(ctx, r.db).Exec(ctx, `
		UPDATE daily_check_ins SET date=$2, sleep_duration_min=$3, sleep_quality=$4, energy=$5,
			stress=$6, soreness=$7, motivation=$8, weight_kg=$9, resting_heart_rate=$10,
			comment=$11, version=version+1, updated_at=$12
		WHERE id=$1 AND version=$13`, checkIn.ID, date, checkIn.SleepDurationMin,
		checkIn.SleepQuality, checkIn.Energy, checkIn.Stress, checkIn.Soreness, checkIn.Motivation,
		checkIn.WeightKg, checkIn.RestingHeartRate, checkIn.Comment, checkIn.UpdatedAt, checkIn.Version)
	if err != nil {
		return mapUnique(op, err, entities.ErrDailyCheckInAlreadyExists)
	}
	if tag.RowsAffected() == 0 {
		return versionConflict(op)
	}
	checkIn.Version++
	return nil
}

func (r *dailyCheckInRepository) List(ctx context.Context, athleteID uuid.UUID, from, to entities.LocalDate) ([]entities.DailyCheckIn, error) {
	const op = "DailyCheckInRepository.List"
	fromValue, err := localDateValue(from)
	if err != nil {
		return nil, wrap(op, err)
	}
	toValue, err := localDateValue(to)
	if err != nil {
		return nil, wrap(op, err)
	}
	rows, err := dbFromContext(ctx, r.db).Query(ctx,
		"SELECT "+checkInColumns+" FROM daily_check_ins WHERE athlete_id=$1 AND date >= $2 AND date <= $3 ORDER BY date, id",
		athleteID, fromValue, toValue)
	if err != nil {
		return nil, wrap(op, err)
	}
	defer rows.Close()

	checkIns := make([]entities.DailyCheckIn, 0)
	for rows.Next() {
		checkIn, scanErr := scanCheckIn(rows)
		if scanErr != nil {
			return nil, wrap(op, scanErr)
		}
		checkIns = append(checkIns, *checkIn)
	}
	if err = rows.Err(); err != nil {
		return nil, wrap(op, err)
	}
	return checkIns, nil
}
