package v1

import (
	"context"
	"errors"

	"diary-service/internal/controller/http/v1/request"
	"diary-service/internal/controller/http/v1/response"
	"diary-service/internal/entities"
	"diary-service/internal/usecase"

	"github.com/gofiber/fiber/v3"
)

func (h *V1) CreateTrainingPlan(c fiber.Ctx) error {
	const function = "CreateTrainingPlan"
	actorID, err := h.actorID(c, function)
	if err != nil {
		return err
	}
	var body request.CreateTrainingPlan
	if err = h.bindAndValidate(c, function, &body); err != nil {
		return err
	}
	startDate, endDate, err := h.planDates(c, function, body.StartDate, body.EndDate)
	if err != nil {
		return err
	}
	result, err := h.services.TrainingPlans.Create(c.Context(), usecase.CreateTrainingPlanCommand{
		ActorID: actorID, AthleteID: body.AthleteID, Name: body.Name, Description: body.Description,
		StartDate: startDate, EndDate: endDate,
	})
	if err != nil {
		return h.usecaseError(c, function, err)
	}
	if result == nil {
		return h.usecaseError(c, function, nilResultError("training plan"))
	}
	return c.Status(fiber.StatusCreated).JSON(response.TrainingPlanFromEntity(result))
}

func (h *V1) GetTrainingPlan(c fiber.Ctx) error {
	const function = "GetTrainingPlan"
	actorID, err := h.actorID(c, function)
	if err != nil {
		return err
	}
	planID, err := h.pathUUID(c, function, "id")
	if err != nil {
		return err
	}
	result, err := h.services.TrainingPlans.GetByID(c.Context(), actorID, planID)
	if err != nil {
		return h.usecaseError(c, function, err)
	}
	if result == nil {
		return h.usecaseError(c, function, nilResultError("training plan"))
	}
	return c.JSON(response.TrainingPlanFromEntity(result))
}

func (h *V1) UpdateTrainingPlan(c fiber.Ctx) error {
	const function = "UpdateTrainingPlan"
	actorID, err := h.actorID(c, function)
	if err != nil {
		return err
	}
	planID, err := h.pathUUID(c, function, "id")
	if err != nil {
		return err
	}
	var body request.UpdateTrainingPlan
	if err = h.bindAndValidate(c, function, &body); err != nil {
		return err
	}
	startDate, endDate, err := h.planDates(c, function, body.StartDate, body.EndDate)
	if err != nil {
		return err
	}
	result, err := h.services.TrainingPlans.Update(c.Context(), usecase.UpdateTrainingPlanCommand{
		ActorID: actorID, PlanID: planID, Name: body.Name, Description: body.Description,
		StartDate: startDate, EndDate: endDate, ExpectedVersion: body.ExpectedVersion,
	})
	if err != nil {
		return h.usecaseError(c, function, err)
	}
	if result == nil {
		return h.usecaseError(c, function, nilResultError("training plan"))
	}
	return c.JSON(response.TrainingPlanFromEntity(result))
}

func (h *V1) ActivateTrainingPlan(c fiber.Ctx) error {
	return h.changeTrainingPlanState(c, "ActivateTrainingPlan", h.services.TrainingPlans.Activate)
}

func (h *V1) CompleteTrainingPlan(c fiber.Ctx) error {
	return h.changeTrainingPlanState(c, "CompleteTrainingPlan", h.services.TrainingPlans.Complete)
}

func (h *V1) CancelTrainingPlan(c fiber.Ctx) error {
	return h.changeTrainingPlanState(c, "CancelTrainingPlan", h.services.TrainingPlans.Cancel)
}

func (h *V1) changeTrainingPlanState(c fiber.Ctx, function string, operation func(context.Context, usecase.ChangeTrainingPlanStateCommand) error) error {
	actorID, err := h.actorID(c, function)
	if err != nil {
		return err
	}
	planID, err := h.pathUUID(c, function, "id")
	if err != nil {
		return err
	}
	var body request.ExpectedVersion
	if err = h.bindAndValidate(c, function, &body); err != nil {
		return err
	}
	if err = operation(c.Context(), usecase.ChangeTrainingPlanStateCommand{ActorID: actorID, PlanID: planID, ExpectedVersion: body.ExpectedVersion}); err != nil {
		return h.usecaseError(c, function, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *V1) planDates(c fiber.Ctx, function, start string, end *string) (entities.LocalDate, *entities.LocalDate, error) {
	startDate, err := h.localDate(c, function, start, "start_date")
	if err != nil {
		return "", nil, err
	}
	if end == nil {
		return startDate, nil, nil
	}
	endDate, err := h.localDate(c, function, *end, "end_date")
	if err != nil {
		return "", nil, err
	}
	if string(endDate) < string(startDate) {
		err = errors.New("end_date precedes start_date")
		return "", nil, h.fail(c, function, "validate training plan dates", err, fiber.StatusBadRequest, "validation_error", "end_date must not be before start_date")
	}
	return startDate, &endDate, nil
}
