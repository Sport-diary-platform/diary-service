package v1

import (
	"diary-service/internal/controller/http/v1/request"
	"diary-service/internal/controller/http/v1/response"
	"diary-service/internal/entities"
	"diary-service/internal/usecase"

	"github.com/gofiber/fiber/v3"
)

func (h *V1) CreateCheckIn(c fiber.Ctx) error {
	const function = "CreateCheckIn"
	actorID, err := h.actorID(c, function)
	if err != nil {
		return err
	}
	var body request.CreateCheckIn
	if err = h.bindAndValidate(c, function, &body); err != nil {
		return err
	}
	date, err := h.localDate(c, function, body.Date, "date")
	if err != nil {
		return err
	}
	result, err := h.services.CheckIns.Create(c.Context(), usecase.CreateDailyCheckInCommand{
		ActorID: actorID, DailyCheckInData: checkInData(date, body.CheckInData),
	})
	if err != nil {
		return h.usecaseError(c, function, err)
	}
	if result == nil {
		return h.usecaseError(c, function, nilResultError("daily check-in"))
	}
	return c.Status(fiber.StatusCreated).JSON(response.DailyCheckInFromEntity(result))
}

func (h *V1) GetCheckIn(c fiber.Ctx) error {
	const function = "GetCheckIn"
	actorID, err := h.actorID(c, function)
	if err != nil {
		return err
	}
	date, err := h.localDate(c, function, c.Params("date"), "date")
	if err != nil {
		return err
	}
	result, err := h.services.CheckIns.GetByDate(c.Context(), actorID, date)
	if err != nil {
		return h.usecaseError(c, function, err)
	}
	if result == nil {
		return h.usecaseError(c, function, nilResultError("daily check-in"))
	}
	return c.JSON(response.DailyCheckInFromEntity(result))
}

func (h *V1) UpdateCheckIn(c fiber.Ctx) error {
	const function = "UpdateCheckIn"
	actorID, err := h.actorID(c, function)
	if err != nil {
		return err
	}
	date, err := h.localDate(c, function, c.Params("date"), "date")
	if err != nil {
		return err
	}
	var body request.UpdateCheckIn
	if err = h.bindAndValidate(c, function, &body); err != nil {
		return err
	}
	result, err := h.services.CheckIns.Update(c.Context(), usecase.UpdateDailyCheckInCommand{
		ActorID: actorID, ExpectedVersion: body.ExpectedVersion, DailyCheckInData: checkInData(date, body.CheckInData),
	})
	if err != nil {
		return h.usecaseError(c, function, err)
	}
	if result == nil {
		return h.usecaseError(c, function, nilResultError("daily check-in"))
	}
	return c.JSON(response.DailyCheckInFromEntity(result))
}

func (h *V1) ListCheckIns(c fiber.Ctx) error {
	const function = "ListCheckIns"
	actorID, err := h.actorID(c, function)
	if err != nil {
		return err
	}
	from, to, err := h.dateRange(c, function)
	if err != nil {
		return err
	}
	result, err := h.services.CheckIns.List(c.Context(), actorID, from, to)
	if err != nil {
		return h.usecaseError(c, function, err)
	}
	return c.JSON(response.DailyCheckInsFromEntities(result))
}

func checkInData(date entities.LocalDate, value request.CheckInData) usecase.DailyCheckInData {
	return usecase.DailyCheckInData{
		Date: date, SleepDurationMin: value.SleepDurationMin, SleepQuality: value.SleepQuality, Energy: value.Energy,
		Stress: value.Stress, Soreness: value.Soreness, Motivation: value.Motivation, WeightKg: value.WeightKg,
		RestingHeartRate: value.RestingHeartRate, Comment: value.Comment,
	}
}
