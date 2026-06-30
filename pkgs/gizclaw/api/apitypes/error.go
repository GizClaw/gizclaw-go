package apitypes

// NewErrorResponse builds the shared API error envelope.
func NewErrorResponse(code, message string) ErrorResponse {
	return ErrorResponse{
		Error: ErrorPayload{
			Code:    code,
			Message: message,
		},
	}
}
