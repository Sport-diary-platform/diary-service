package v1

import (
	"errors"
	"testing"

	"diary-service/internal/entities"
	"diary-service/internal/usecase"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/require"
)

func TestMapError(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{name: "wrapped invalid command", err: errors.Join(usecase.ErrInvalidCommand, errors.New("field")), status: fiber.StatusBadRequest, code: "invalid_input"},
		{name: "forbidden", err: usecase.ErrForbidden, status: fiber.StatusForbidden, code: "forbidden"},
		{name: "missing comment", err: usecase.ErrCommentNotFound, status: fiber.StatusNotFound, code: "not_found"},
		{name: "relationship already exists", err: usecase.ErrRelationshipAlreadyExists, status: fiber.StatusConflict, code: "conflict"},
		{name: "version conflict", err: entities.ErrVersionConflict, status: fiber.StatusConflict, code: "conflict"},
		{name: "unexpected", err: errors.New("database unavailable"), status: fiber.StatusInternalServerError, code: "internal_error"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			status, code, _ := mapError(test.err)
			require.Equal(t, test.status, status)
			require.Equal(t, test.code, code)
		})
	}
}
