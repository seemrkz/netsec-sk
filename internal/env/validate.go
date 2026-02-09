package env

import (
	"errors"
	"regexp"
	"strings"
)

var (
	ErrInvalidEnvID = errors.New("invalid env_id")
	envIDPattern    = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,30}[a-z0-9])?$`)
)

func NormalizeEnvID(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

func ValidateEnvID(envID string) error {
	if !envIDPattern.MatchString(envID) {
		return ErrInvalidEnvID
	}

	return nil
}
