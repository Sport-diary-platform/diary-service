package v1

import (
	"diary-service/internal/controller/http/v1/request"
	"diary-service/internal/controller/http/v1/response"
	"diary-service/internal/usecase"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func (h *V1) CreateRelationship(c fiber.Ctx) error {
	const function = "CreateRelationship"
	actorID, err := h.actorID(c, function)
	if err != nil {
		return err
	}
	var body request.CreateRelationship
	if err = h.bindAndValidate(c, function, &body); err != nil {
		return err
	}
	if err = requireDifferentIDs(actorID, body.AthleteID); err != nil {
		return h.fail(c, function, "validate relationship participants", err, fiber.StatusBadRequest, "validation_error", "coach and athlete must differ")
	}
	result, err := h.services.Relationships.Create(c.Context(), usecase.CreateRelationshipCommand{ActorID: actorID, AthleteID: body.AthleteID})
	if err != nil {
		return h.usecaseError(c, function, err)
	}
	if result == nil {
		return h.usecaseError(c, function, nilResultError("relationship"))
	}
	return c.Status(fiber.StatusCreated).JSON(response.RelationshipFromEntity(result))
}

func (h *V1) TerminateRelationship(c fiber.Ctx) error {
	const function = "TerminateRelationship"
	actorID, err := h.actorID(c, function)
	if err != nil {
		return err
	}
	relationshipID, err := h.pathUUID(c, function, "id")
	if err != nil {
		return err
	}
	var body request.ExpectedVersion
	if err = h.bindAndValidate(c, function, &body); err != nil {
		return err
	}
	if err = h.services.Relationships.Terminate(c.Context(), usecase.TerminateRelationshipCommand{ActorID: actorID, RelationshipID: relationshipID, ExpectedVersion: body.ExpectedVersion}); err != nil {
		return h.usecaseError(c, function, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *V1) ListRelationshipAthletes(c fiber.Ctx) error {
	const function = "ListRelationshipAthletes"
	actorID, err := h.actorID(c, function)
	if err != nil {
		return err
	}
	result, err := h.services.Relationships.ListAthletes(c.Context(), actorID)
	if err != nil {
		return h.usecaseError(c, function, err)
	}
	if result == nil {
		result = []uuid.UUID{}
	}
	return c.JSON(response.UserIDs{UserIDs: result})
}

func (h *V1) ListRelationshipCoaches(c fiber.Ctx) error {
	const function = "ListRelationshipCoaches"
	actorID, err := h.actorID(c, function)
	if err != nil {
		return err
	}
	result, err := h.services.Relationships.ListCoaches(c.Context(), actorID)
	if err != nil {
		return h.usecaseError(c, function, err)
	}
	if result == nil {
		result = []uuid.UUID{}
	}
	return c.JSON(response.UserIDs{UserIDs: result})
}
