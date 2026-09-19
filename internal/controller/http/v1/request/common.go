package request

type ExpectedVersion struct {
	ExpectedVersion int64 `json:"expected_version" validate:"required,min=1"`
}
