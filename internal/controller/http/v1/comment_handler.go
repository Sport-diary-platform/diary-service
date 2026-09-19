package v1

import (
	"diary-service/internal/controller/http/v1/request"
	"diary-service/internal/controller/http/v1/response"
	"diary-service/internal/usecase"

	"github.com/gofiber/fiber/v3"
)

func (h *V1) CreateComment(c fiber.Ctx) error {
	const function = "CreateComment"
	actorID, err := h.actorID(c, function)
	if err != nil {
		return err
	}
	workoutID, err := h.pathUUID(c, function, "workoutID")
	if err != nil {
		return err
	}
	var body request.CreateComment
	if err = h.bindAndValidate(c, function, &body); err != nil {
		return err
	}
	result, err := h.services.Comments.Create(c.Context(), usecase.CreateCommentCommand{ActorID: actorID, WorkoutID: workoutID, Text: body.Text})
	if err != nil {
		return h.usecaseError(c, function, err)
	}
	if result == nil {
		return h.usecaseError(c, function, nilResultError("comment"))
	}
	return c.Status(fiber.StatusCreated).JSON(response.CommentFromEntity(result))
}

func (h *V1) ListComments(c fiber.Ctx) error {
	const function = "ListComments"
	actorID, err := h.actorID(c, function)
	if err != nil {
		return err
	}
	workoutID, err := h.pathUUID(c, function, "workoutID")
	if err != nil {
		return err
	}
	result, err := h.services.Comments.ListByWorkout(c.Context(), actorID, workoutID)
	if err != nil {
		return h.usecaseError(c, function, err)
	}
	return c.JSON(response.CommentsFromEntities(result))
}

func (h *V1) UpdateComment(c fiber.Ctx) error {
	const function = "UpdateComment"
	actorID, err := h.actorID(c, function)
	if err != nil {
		return err
	}
	workoutID, err := h.pathUUID(c, function, "workoutID")
	if err != nil {
		return err
	}
	commentID, err := h.pathUUID(c, function, "commentID")
	if err != nil {
		return err
	}
	var body request.UpdateComment
	if err = h.bindAndValidate(c, function, &body); err != nil {
		return err
	}
	result, err := h.services.Comments.Update(c.Context(), usecase.UpdateCommentCommand{
		ActorID: actorID, WorkoutID: workoutID, CommentID: commentID, Text: body.Text, ExpectedVersion: body.ExpectedVersion,
	})
	if err != nil {
		return h.usecaseError(c, function, err)
	}
	if result == nil {
		return h.usecaseError(c, function, nilResultError("comment"))
	}
	return c.JSON(response.CommentFromEntity(result))
}
