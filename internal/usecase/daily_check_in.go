package usecase

import (
	"context"
	"errors"
	"math"
	"time"

	"diary-service/internal/entities"
	"diary-service/internal/repository"

	"github.com/google/uuid"
)

type dailyCheckInUseCase struct {
	repository repository.DailyCheckInRepository
	outbox     repository.OutboxRepository
	tx         repository.TransactionManager
	profiles   ProfileProvider
}

func NewDailyCheckInUseCase(
	checkIns repository.DailyCheckInRepository,
	outbox repository.OutboxRepository,
	tx repository.TransactionManager,
	profiles ProfileProvider,
) DailyCheckInUseCase {
	return &dailyCheckInUseCase{repository: checkIns, outbox: outbox, tx: tx, profiles: profiles}
}

func validateDailyCheckIn(data DailyCheckInData) error {
	if err := validateLocalDate("date", data.Date); err != nil {
		return err
	}
	if data.SleepDurationMin != nil && *data.SleepDurationMin < 0 {
		return invalid("sleep_duration_min", "must not be negative")
	}
	for name, value := range map[string]*int{
		"sleep_quality": data.SleepQuality,
		"energy":        data.Energy,
		"stress":        data.Stress,
		"soreness":      data.Soreness,
		"motivation":    data.Motivation,
	} {
		if value != nil && (*value < 1 || *value > 10) {
			return invalid(name, "must be between 1 and 10")
		}
	}
	if data.WeightKg != nil && (math.IsNaN(*data.WeightKg) || math.IsInf(*data.WeightKg, 0) || *data.WeightKg <= 0) {
		return invalid("weight_kg", "must be finite and greater than zero")
	}
	if data.RestingHeartRate != nil && *data.RestingHeartRate <= 0 {
		return invalid("resting_heart_rate", "must be greater than zero")
	}

	return nil
}

func (uc *dailyCheckInUseCase) Create(ctx context.Context, cmd CreateDailyCheckInCommand) (*entities.DailyCheckIn, error) {
	if err := requireProfile(ctx, uc.profiles, cmd.ActorID, ProfileTypeAthlete); err != nil {
		return nil, err
	}
	if err := validateDailyCheckIn(cmd.DailyCheckInData); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	checkIn := &entities.DailyCheckIn{
		ID: uuid.New(), AthleteID: cmd.ActorID, Date: cmd.Date,
		SleepDurationMin: cmd.SleepDurationMin, SleepQuality: cmd.SleepQuality,
		Energy: cmd.Energy, Stress: cmd.Stress, Soreness: cmd.Soreness, Motivation: cmd.Motivation,
		WeightKg: cmd.WeightKg, RestingHeartRate: cmd.RestingHeartRate, Comment: cmd.Comment,
		Version: 1, CreatedAt: now, UpdatedAt: now,
	}

	err := runTransaction(ctx, uc.tx, "DailyCheckInUseCase.Create", func(txCtx context.Context) error {
		existing, err := uc.repository.GetByDate(txCtx, cmd.ActorID, cmd.Date)
		if err != nil && !errors.Is(err, entities.ErrDailyCheckInNotFound) {
			return operationError("DailyCheckInRepository.GetByDate", err)
		}
		if err == nil && existing != nil {
			return entities.ErrDailyCheckInAlreadyExists
		}
		if err := uc.repository.Create(txCtx, checkIn); err != nil {
			return operationError("DailyCheckInRepository.Create", err)
		}

		return createEvent(txCtx, uc.outbox, "daily_checkin.created", "daily_check_in", checkIn.ID, checkInEventPayload(checkIn), now)
	})
	if err != nil {
		return nil, err
	}

	return checkIn, nil
}

func (uc *dailyCheckInUseCase) GetByDate(ctx context.Context, actorID uuid.UUID, date entities.LocalDate) (*entities.DailyCheckIn, error) {
	if err := requireProfile(ctx, uc.profiles, actorID, ProfileTypeAthlete); err != nil {
		return nil, err
	}
	if err := validateLocalDate("date", date); err != nil {
		return nil, err
	}

	checkIn, err := uc.repository.GetByDate(ctx, actorID, date)
	if err != nil {
		return nil, operationError("DailyCheckInRepository.GetByDate", err)
	}
	if checkIn == nil {
		return nil, entities.ErrDailyCheckInNotFound
	}

	return checkIn, nil
}

func (uc *dailyCheckInUseCase) Update(ctx context.Context, cmd UpdateDailyCheckInCommand) (*entities.DailyCheckIn, error) {
	if err := requireProfile(ctx, uc.profiles, cmd.ActorID, ProfileTypeAthlete); err != nil {
		return nil, err
	}
	if err := validateDailyCheckIn(cmd.DailyCheckInData); err != nil {
		return nil, err
	}
	if err := validExpectedVersion(cmd.ExpectedVersion); err != nil {
		return nil, err
	}

	var checkIn *entities.DailyCheckIn
	now := time.Now().UTC()
	err := runTransaction(ctx, uc.tx, "DailyCheckInUseCase.Update", func(txCtx context.Context) error {
		var err error
		checkIn, err = uc.repository.GetByDate(txCtx, cmd.ActorID, cmd.Date)
		if err != nil {
			return operationError("DailyCheckInRepository.GetByDate", err)
		}
		if checkIn == nil {
			return entities.ErrDailyCheckInNotFound
		}
		if checkIn.AthleteID != cmd.ActorID {
			return ErrForbidden
		}
		if err := versionMatches(checkIn.Version, cmd.ExpectedVersion); err != nil {
			return err
		}

		checkIn.SleepDurationMin = cmd.SleepDurationMin
		checkIn.SleepQuality = cmd.SleepQuality
		checkIn.Energy = cmd.Energy
		checkIn.Stress = cmd.Stress
		checkIn.Soreness = cmd.Soreness
		checkIn.Motivation = cmd.Motivation
		checkIn.WeightKg = cmd.WeightKg
		checkIn.RestingHeartRate = cmd.RestingHeartRate
		checkIn.Comment = cmd.Comment
		checkIn.UpdatedAt = now
		if err := uc.repository.Update(txCtx, checkIn); err != nil {
			return operationError("DailyCheckInRepository.Update", err)
		}
		if err := createEvent(txCtx, uc.outbox, "daily_checkin.updated", "daily_check_in", checkIn.ID, checkInEventPayload(checkIn), now); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return checkIn, nil
}

func (uc *dailyCheckInUseCase) List(ctx context.Context, actorID uuid.UUID, from, to entities.LocalDate) ([]entities.DailyCheckIn, error) {
	if err := requireProfile(ctx, uc.profiles, actorID, ProfileTypeAthlete); err != nil {
		return nil, err
	}
	if err := validateDateRange(from, to); err != nil {
		return nil, err
	}

	checkIns, err := uc.repository.List(ctx, actorID, from, to)
	if err != nil {
		return nil, operationError("DailyCheckInRepository.List", err)
	}

	return checkIns, nil
}

func checkInEventPayload(checkIn *entities.DailyCheckIn) map[string]any {
	return map[string]any{
		"check_in_id": checkIn.ID,
		"athlete_id":  checkIn.AthleteID,
		"date":        checkIn.Date,
	}
}
