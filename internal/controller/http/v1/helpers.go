package v1

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"diary-service/internal/controller/http/middleware"
	"diary-service/internal/controller/http/v1/response"
	"diary-service/internal/entities"
	"diary-service/internal/usecase"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

const dateLayout = "2006-01-02"

func (h *V1) bindAndValidate(c fiber.Ctx, function string, destination any) error {
	if err := c.Bind().Body(destination); err != nil {
		return h.fail(c, function, "parse request body", err, fiber.StatusBadRequest, "invalid_request", "invalid request body")
	}
	if err := h.validator.Struct(destination); err != nil {
		return h.fail(c, function, "validate request body", err, fiber.StatusBadRequest, "validation_error", "request validation failed")
	}
	return nil
}

func (h *V1) actorID(c fiber.Ctx, function string) (uuid.UUID, error) {
	actorID, err := middleware.ActorID(c)
	if err != nil {
		return uuid.Nil, h.fail(c, function, "extract authenticated actor", err, fiber.StatusUnauthorized, "unauthorized", "authentication required")
	}
	return actorID, nil
}

func (h *V1) pathUUID(c fiber.Ctx, function, parameter string) (uuid.UUID, error) {
	value, err := uuid.Parse(c.Params(parameter))
	if err != nil || value == uuid.Nil {
		if err == nil {
			err = errors.New("UUID is empty")
		}
		return uuid.Nil, h.fail(c, function, "parse path parameter "+parameter, err, fiber.StatusBadRequest, "invalid_parameter", parameter+" must be a valid UUID")
	}
	return value, nil
}

func (h *V1) localDate(c fiber.Ctx, function, value, parameter string) (entities.LocalDate, error) {
	parsed, err := time.Parse(dateLayout, value)
	if err != nil || parsed.Format(dateLayout) != value {
		if err == nil {
			err = errors.New("invalid calendar date")
		}
		return "", h.fail(c, function, "parse date "+parameter, err, fiber.StatusBadRequest, "invalid_parameter", parameter+" must use YYYY-MM-DD format")
	}
	return entities.LocalDate(value), nil
}

func (h *V1) timeRange(c fiber.Ctx, function string) (time.Time, time.Time, error) {
	fromValue, toValue := c.Query("from"), c.Query("to")
	if fromValue == "" || toValue == "" {
		err := errors.New("from and to are required")
		return time.Time{}, time.Time{}, h.fail(c, function, "parse date range", err, fiber.StatusBadRequest, "invalid_query", "from and to are required")
	}
	from, err := time.Parse(time.RFC3339, fromValue)
	if err != nil {
		return time.Time{}, time.Time{}, h.fail(c, function, "parse from query", err, fiber.StatusBadRequest, "invalid_query", "from must be an RFC3339 timestamp")
	}
	to, err := time.Parse(time.RFC3339, toValue)
	if err != nil {
		return time.Time{}, time.Time{}, h.fail(c, function, "parse to query", err, fiber.StatusBadRequest, "invalid_query", "to must be an RFC3339 timestamp")
	}
	if to.Before(from) {
		err = errors.New("to precedes from")
		return time.Time{}, time.Time{}, h.fail(c, function, "validate date range", err, fiber.StatusBadRequest, "invalid_query", "to must not be before from")
	}
	return from, to, nil
}

func (h *V1) dateRange(c fiber.Ctx, function string) (entities.LocalDate, entities.LocalDate, error) {
	from, err := h.localDate(c, function, c.Query("from"), "from")
	if err != nil {
		return "", "", err
	}
	to, err := h.localDate(c, function, c.Query("to"), "to")
	if err != nil {
		return "", "", err
	}
	if string(to) < string(from) {
		err = errors.New("to precedes from")
		return "", "", h.fail(c, function, "validate date range", err, fiber.StatusBadRequest, "invalid_query", "to must not be before from")
	}
	return from, to, nil
}

func (h *V1) ifMatch(c fiber.Ctx, function string) (int64, error) {
	value := strings.TrimSpace(c.Get(fiber.HeaderIfMatch))
	value = strings.TrimPrefix(value, "W/")
	value = strings.Trim(value, "\"")
	version, err := strconv.ParseInt(value, 10, 64)
	if err != nil || version < 1 {
		if err == nil {
			err = errors.New("version must be positive")
		}
		return 0, h.fail(c, function, "parse If-Match header", err, fiber.StatusBadRequest, "invalid_if_match", "If-Match must contain a positive version")
	}
	return version, nil
}

func (h *V1) usecaseError(c fiber.Ctx, function string, err error) error {
	status, code, message := mapError(err)
	return h.fail(c, function, "execute usecase", err, status, code, message)
}

func (h *V1) fail(c fiber.Ctx, function, cause string, err error, status int, code, message string) error {
	if h.logger != nil {
		h.logger.ErrorDetails("http.controller.v1", function, cause, err)
	}
	return c.Status(status).JSON(response.Error{Error: response.ErrorDetails{Code: code, Message: message}})
}

func mapError(err error) (int, string, string) {
	switch {
	case errors.Is(err, usecase.ErrInvalidCommand), errors.Is(err, entities.ErrInvalidInput), errors.Is(err, entities.ErrInvalidRPE), errors.Is(err, entities.ErrInvalidFeeling):
		return fiber.StatusBadRequest, "invalid_input", err.Error()
	case errors.Is(err, usecase.ErrForbidden), errors.Is(err, usecase.ErrInvalidProfileType), errors.Is(err, entities.ErrForbidden), errors.Is(err, entities.ErrInvalidProfileType), errors.Is(err, entities.ErrRelationshipNotActive):
		return fiber.StatusForbidden, "forbidden", "operation is forbidden"
	case errors.Is(err, entities.ErrWorkoutNotFound), errors.Is(err, entities.ErrWorkoutResultNotFound),
		errors.Is(err, entities.ErrTrainingPlanNotFound), errors.Is(err, entities.ErrRelationshipNotFound),
		errors.Is(err, entities.ErrDailyCheckInNotFound), errors.Is(err, entities.ErrGoalNotFound),
		errors.Is(err, usecase.ErrCommentNotFound), errors.Is(err, entities.ErrCommentNotFound):
		return fiber.StatusNotFound, "not_found", err.Error()
	case errors.Is(err, entities.ErrVersionConflict), errors.Is(err, entities.ErrWorkoutCancelled),
		errors.Is(err, entities.ErrWorkoutCompleted), errors.Is(err, entities.ErrWorkoutHasResult),
		errors.Is(err, entities.ErrWorkoutResultAlreadyExists), errors.Is(err, entities.ErrInvalidTrainingPlanState),
		errors.Is(err, entities.ErrDailyCheckInAlreadyExists), errors.Is(err, usecase.ErrRelationshipAlreadyExists):
		return fiber.StatusConflict, "conflict", err.Error()
	default:
		return fiber.StatusInternalServerError, "internal_error", "internal server error"
	}
}

func nilResultError(kind string) error {
	return fmt.Errorf("%s usecase returned a nil result", kind)
}
