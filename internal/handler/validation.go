package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

func parseValidationErrors(err error) []gin.H {
	var errors []gin.H

	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, e := range validationErrors {
			errors = append(errors, gin.H{
				"field":   e.Field(),
				"message": getValidationMessage(e),
			})
		}
	} else {
		errors = append(errors, gin.H{
			"field":   "unknown",
			"message": err.Error(),
		})
	}

	return errors
}

func getValidationMessage(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return e.Field() + " is required"
	case "email":
		return "Invalid email format"
	case "min":
		return e.Field() + " must be at least " + e.Param() + " characters"
	default:
		return e.Field() + " is invalid"
	}
}
