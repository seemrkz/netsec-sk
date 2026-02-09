package repo

import (
	"errors"
	"fmt"
	"os/exec"
)

var ErrGitMissing = errors.New("git executable is not available on PATH")

type LookPathFunc func(file string) (string, error)

func CheckGitAvailable(lookPath LookPathFunc) error {
	if lookPath == nil {
		lookPath = exec.LookPath
	}

	if _, err := lookPath("git"); err != nil {
		return fmt.Errorf("%w: %v", ErrGitMissing, err)
	}

	return nil
}
