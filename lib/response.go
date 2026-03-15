package lib

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type APIResponse struct {
	Success bool        `json:"success" default:"true"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

type HTTPError struct {
	Success bool          `json:"success" default:"false"`
	Message string        `json:"message,omitempty"`
	Error   ErrorResponse `json:"error"`
}

type ErrorResponse struct {
	Code    string      `json:"code,omitempty"`
	Details interface{} `json:"details,omitempty"`
}

type HTTPErrorResp struct {
	Success bool   `json:"success" default:"false"`
	Message string `json:"message,omitempty"`
	Error   error  `json:"error"`
}

func RespondSuccess(ctx *gin.Context, code int, message string, data interface{}) {
	if message == "" {
		message = MessageSuccess
	}
	ctx.JSON(code, APIResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func RespondError(ctx *gin.Context, code int, message string, err error) {
	switch err {
	case ErrorMessageUnauthorized:
		code = http.StatusUnauthorized
	case ErrorMessageInvalidInput:
		code = http.StatusBadRequest
	case ErrorMessageDataNotFound, ErrorMessageUserNotFound:
		code = http.StatusNotFound
	}

	ctx.JSON(code, HTTPErrorResp{
		Success: false,
		Message: message,
		Error:   err,
	})
}

func RespondValidationError(ctx *gin.Context, code int, message string, details interface{}) {
	ctx.JSON(code, HTTPError{
		Success: false,
		Message: message,
		Error: ErrorResponse{
			Code:    "VALIDATION_ERROR",
			Details: details,
		},
	})
}

func RespondErrorWithCode(ctx *gin.Context, code int, message string, errorCode string) {
	ctx.JSON(code, HTTPError{
		Success: false,
		Message: message,
		Error: ErrorResponse{
			Code: errorCode,
		},
	})
}
