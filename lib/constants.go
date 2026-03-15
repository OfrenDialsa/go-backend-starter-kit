package lib

import "errors"

// Success messages
const (
	MessageSuccess           = "Success"
	MsgOk                    = "OK"
	MsgCreated               = "Created successfully"
	MsgUpdated               = "Updated successfully"
	MsgDeleted               = "Deleted successfully"
	MsgRegistrationSuccess   = "Registration successful"
	MsgLoginSuccess          = "Login successful"
	MsgPasswordResetSuccess  = "Password reset successful. Please login with your new password."
	MsgPasswordForgotSuccess = "If the email exists, a password reset link has been sent"
)

// Error messages
const (
	ErrBadPayload              = "please check your payload"
	ErrMissingRequired         = "missing data required"
	ErrUnauthorized            = "Unauthorized"
	ErrForbidden               = "Forbidden"
	ErrUserNotFound            = "user not found"
	ErrRecordNotFound          = "record not found"
	ErrWrongUsernameOrPassword = "wrong username or password"
	ErrInvalidInput            = "invalid input"
	ErrInvalidCredential       = "invalid credential"
	ErrEmailAlreadyExists      = "Email already registered"
	ErrUsernameNotAvailable    = "Username not available"
	ErrInvalidResetToken       = "Invalid or expired reset token"
	ErrSamePassword            = "New Password Must be different"
	ErrWeakPassword            = "password must be at least 8 characters"
	ErrInvalidPassword         = "Password incorrect"
	ErrInvalidCurrentPassword  = "Current password is incorrect"
	ErrUserHasNoPassword       = "user has no password yet"
	ErrorMessageInvalidRequest = "Invalid request"
	ErrInternalServerError     = "Internal Server Error"
)

// Error codes
const (
	CodeUserNotFound           = "USER_NOT_FOUND"
	CodeMustTransferOwnership  = "MUST_TRANSFER_OWNERSHIP"
	CodeInvalidResetToken      = "INVALID_RESET_TOKEN"
	CodeInvalidPassword        = "INVALID_PASSWORD"
	CodeInvalidCurrentPassword = "INVALID_CURRENT_PASSWORD"
	CodeUserHasNoPassword      = "USER_HAS_NO_PASSWORD"
	CodeSamePassword           = "SAME_AS_OLD_PASSWORD"
	CodeWeakPassword           = "WEAK_PASSWORD"
)

// Error types
var (
	ErrorMessageDataNotFound = errors.New("data not found")
	ErrorMessageInvalidInput = errors.New("invalid input")
	ErrorMessageBadPayload   = errors.New(ErrBadPayload)

	//Auth
	ErrorMessageUnauthorized           = errors.New("unauthorized")
	ErrorMessageUserNotFound           = errors.New(ErrUserNotFound)
	ErrorMessageEmailExists            = errors.New(ErrEmailAlreadyExists)
	ErrorMessageUsernameNotAvailable   = errors.New(ErrUsernameNotAvailable)
	ErrorMessageInvalidCredentials     = errors.New(ErrWrongUsernameOrPassword)
	ErrorMessageInvalidResetToken      = errors.New(ErrInvalidResetToken)
	ErrorMessageInvalidPassword        = errors.New(ErrInvalidPassword)
	ErrorMessageInvalidCurrentPassword = errors.New(ErrInvalidCurrentPassword)
	ErrorMessageUserHasNoPassword      = errors.New(ErrUserHasNoPassword)
	ErrorMessageSamePassword           = errors.New(ErrSamePassword)
	ErrorMessageWeakPassword           = errors.New(ErrWeakPassword)
)
