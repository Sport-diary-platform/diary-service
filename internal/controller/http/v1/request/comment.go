package request

type CreateComment struct {
	Text string `json:"text" validate:"required,max=5000"`
}

type UpdateComment struct {
	Text            string `json:"text" validate:"required,max=5000"`
	ExpectedVersion int64  `json:"expected_version" validate:"required,min=1"`
}
