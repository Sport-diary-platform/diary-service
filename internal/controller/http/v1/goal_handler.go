package v1

import (
	"context"
	"diary-service/internal/controller/http/v1/request"
	"diary-service/internal/controller/http/v1/response"
	"diary-service/internal/entities"
	"diary-service/internal/usecase"

	"github.com/gofiber/fiber/v3"
)

func (h *V1) CreateGoal(c fiber.Ctx) error {
	const function = "CreateGoal"
	actorID, err := h.actorID(c, function)
	if err != nil {
		return err
	}
	var body request.CreateGoal
	if err = h.bindAndValidate(c, function, &body); err != nil {
		return err
	}
	athleteID := actorID
	if body.AthleteID != nil {
		athleteID = *body.AthleteID
	}
	deadline, err := h.optionalDate(c, function, body.Deadline, "deadline")
	if err != nil {
		return err
	}
	result, err := h.services.Goals.Create(c.Context(), usecase.CreateGoalCommand{
		ActorID: actorID, AthleteID: athleteID, Title: body.Title, Description: body.Description,
		Type: entities.GoalType(body.Type), Target: body.Target, Deadline: deadline,
	})
	if err != nil {
		return h.usecaseError(c, function, err)
	}
	if result == nil {
		return h.usecaseError(c, function, nilResultError("goal"))
	}
	return c.Status(fiber.StatusCreated).JSON(response.GoalFromEntity(result))
}

func (h *V1) GetGoal(c fiber.Ctx) error {
	const function = "GetGoal"
	actorID, err := h.actorID(c, function)
	if err != nil {
		return err
	}
	goalID, err := h.pathUUID(c, function, "id")
	if err != nil {
		return err
	}
	result, err := h.services.Goals.GetByID(c.Context(), actorID, goalID)
	if err != nil {
		return h.usecaseError(c, function, err)
	}
	if result == nil {
		return h.usecaseError(c, function, nilResultError("goal"))
	}
	return c.JSON(response.GoalFromEntity(result))
}

func (h *V1) UpdateGoal(c fiber.Ctx) error {
	const function = "UpdateGoal"
	actorID, err := h.actorID(c, function)
	if err != nil {
		return err
	}
	goalID, err := h.pathUUID(c, function, "id")
	if err != nil {
		return err
	}
	var body request.UpdateGoal
	if err = h.bindAndValidate(c, function, &body); err != nil {
		return err
	}
	deadline, err := h.optionalDate(c, function, body.Deadline, "deadline")
	if err != nil {
		return err
	}
	result, err := h.services.Goals.Update(c.Context(), usecase.UpdateGoalCommand{
		ActorID: actorID, GoalID: goalID, Title: body.Title, Description: body.Description,
		Type: entities.GoalType(body.Type), Target: body.Target, Deadline: deadline, ExpectedVersion: body.ExpectedVersion,
	})
	if err != nil {
		return h.usecaseError(c, function, err)
	}
	if result == nil {
		return h.usecaseError(c, function, nilResultError("goal"))
	}
	return c.JSON(response.GoalFromEntity(result))
}

func (h *V1) CompleteGoal(c fiber.Ctx) error {
	return h.changeGoalState(c, "CompleteGoal", h.services.Goals.Complete)
}

func (h *V1) CancelGoal(c fiber.Ctx) error {
	return h.changeGoalState(c, "CancelGoal", h.services.Goals.Cancel)
}

func (h *V1) changeGoalState(c fiber.Ctx, function string, operation func(context.Context, usecase.ChangeGoalStateCommand) error) error {
	actorID, err := h.actorID(c, function)
	if err != nil {
		return err
	}
	goalID, err := h.pathUUID(c, function, "id")
	if err != nil {
		return err
	}
	var body request.ExpectedVersion
	if err = h.bindAndValidate(c, function, &body); err != nil {
		return err
	}
	if err = operation(c.Context(), usecase.ChangeGoalStateCommand{ActorID: actorID, GoalID: goalID, ExpectedVersion: body.ExpectedVersion}); err != nil {
		return h.usecaseError(c, function, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *V1) ListGoals(c fiber.Ctx) error {
	const function = "ListGoals"
	actorID, err := h.actorID(c, function)
	if err != nil {
		return err
	}
	result, err := h.services.Goals.List(c.Context(), actorID)
	if err != nil {
		return h.usecaseError(c, function, err)
	}
	return c.JSON(response.GoalsFromEntities(result))
}

func (h *V1) optionalDate(c fiber.Ctx, function string, value *string, parameter string) (*entities.LocalDate, error) {
	if value == nil {
		return nil, nil
	}
	date, err := h.localDate(c, function, *value, parameter)
	if err != nil {
		return nil, err
	}
	return &date, nil
}
