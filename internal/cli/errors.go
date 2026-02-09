package cli

import (
	"errors"
	"fmt"
)

type ErrorCode string

const (
	ErrUsage      ErrorCode = "E_USAGE"
	ErrGitMissing ErrorCode = "E_GIT_MISSING"
	ErrRepoUnsafe ErrorCode = "E_REPO_UNSAFE"
	ErrLockHeld   ErrorCode = "E_LOCK_HELD"
	ErrParseFatal ErrorCode = "E_PARSE_FATAL"
	ErrParsePart  ErrorCode = "E_PARSE_PARTIAL"
	ErrIO         ErrorCode = "E_IO"
	ErrInternal   ErrorCode = "E_INTERNAL"
)

type AppError struct {
	Code    ErrorCode
	Message string
}

func (e *AppError) Error() string {
	return e.Message
}

func NewAppError(code ErrorCode, message string) error {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

func ExitCodeFor(err error) int {
	if err == nil {
		return 0
	}

	var appErr *AppError
	if !errors.As(err, &appErr) {
		return 9
	}

	switch appErr.Code {
	case ErrUsage:
		return 2
	case ErrGitMissing:
		return 3
	case ErrRepoUnsafe:
		return 4
	case ErrLockHeld:
		return 5
	case ErrParseFatal, ErrIO:
		return 6
	case ErrParsePart:
		return 7
	default:
		return 9
	}
}

func FormatErrorLine(err error) string {
	if err == nil {
		return ""
	}

	var appErr *AppError
	if errors.As(err, &appErr) {
		return fmt.Sprintf("ERROR %s %s", appErr.Code, appErr.Message)
	}

	return fmt.Sprintf("ERROR %s %s", ErrInternal, err.Error())
}
