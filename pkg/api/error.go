package api

import (
	"easyflow-backend/pkg/enum"
)

type ApiError struct {
	Code    int            `json:"code"`
	Error   enum.ErrorCode `json:"error"`
	Details interface{}    `json:"details,omitempty"`
}
