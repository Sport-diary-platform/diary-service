package profile

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"diary-service/internal/entities"
	"diary-service/internal/usecase"

	"github.com/google/uuid"
)

type Client struct {
	baseURL string
	client  *http.Client
}

func New(baseURL string, timeout time.Duration) (*Client, error) {
	parsed, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, errors.Join(errors.New("profile/client - New - invalid base URL"), err)
	}
	return &Client{baseURL: parsed.String(), client: &http.Client{Timeout: timeout}}, nil
}

func (c *Client) GetProfileType(ctx context.Context, userID uuid.UUID) (usecase.ProfileType, error) {
	endpoint := fmt.Sprintf("%s/api/v1/internal/profiles/%s", c.baseURL, userID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", errors.Join(errors.New("profile/client - GetProfileType - NewRequestWithContext"), err)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return "", errors.Join(errors.New("profile/client - GetProfileType - Do"), err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return "", entities.ErrForbidden
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("profile/client - GetProfileType - unexpected status: %d", resp.StatusCode)
	}
	var body struct {
		ProfileType usecase.ProfileType `json:"profile_type"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", errors.Join(errors.New("profile/client - GetProfileType - Decode"), err)
	}
	if body.ProfileType != usecase.ProfileTypeAthlete && body.ProfileType != usecase.ProfileTypeCoach {
		return "", entities.ErrInvalidProfileType
	}
	return body.ProfileType, nil
}
