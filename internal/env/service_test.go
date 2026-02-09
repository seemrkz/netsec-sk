package env

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestEnvIDValidation(t *testing.T) {
	valid := []string{
		"default",
		"a",
		"prod-01",
		"abc123",
		"a123456789012345678901234567890z",
	}
	for _, raw := range valid {
		normalized := NormalizeEnvID(raw)
		if err := ValidateEnvID(normalized); err != nil {
			t.Fatalf("expected valid env_id %q, got %v", raw, err)
		}
	}

	invalid := []string{
		"",
		"-abc",
		"abc-",
		"a_b",
		"A B",
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}
	for _, raw := range invalid {
		normalized := NormalizeEnvID(raw)
		if err := ValidateEnvID(normalized); !errors.Is(err, ErrInvalidEnvID) {
			t.Fatalf("expected ErrInvalidEnvID for %q, got %v", raw, err)
		}
	}
}

func TestServiceCreateAndList(t *testing.T) {
	repoPath := t.TempDir()
	svc := NewService(repoPath)

	envID, created, err := svc.Create(" Dev ")
	if err != nil {
		t.Fatalf("Create() unexpected error: %v", err)
	}
	if envID != "dev" {
		t.Fatalf("Create() envID = %q, want %q", envID, "dev")
	}
	if !created {
		t.Fatalf("Create() created = false, want true")
	}

	if _, createdAgain, err := svc.Create("dev"); err != nil {
		t.Fatalf("Create() idempotent unexpected error: %v", err)
	} else if createdAgain {
		t.Fatalf("Create() idempotent created = true, want false")
	}

	if _, _, err := svc.Create("BAD_NAME"); !errors.Is(err, ErrInvalidEnvID) {
		t.Fatalf("Create() invalid name error = %v, want ErrInvalidEnvID", err)
	}

	_, _, err = svc.Create("prod")
	if err != nil {
		t.Fatalf("Create(prod) unexpected error: %v", err)
	}

	got, err := svc.List()
	if err != nil {
		t.Fatalf("List() unexpected error: %v", err)
	}
	want := []string{"dev", "prod"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("List() = %#v, want %#v", got, want)
	}

	mustExist := []string{
		filepath.Join(repoPath, "envs", "dev", "state"),
		filepath.Join(repoPath, "envs", "dev", "exports"),
		filepath.Join(repoPath, "envs", "dev", "overrides"),
	}
	for _, path := range mustExist {
		if info, err := os.Stat(path); err != nil || !info.IsDir() {
			t.Fatalf("expected directory %s to exist", path)
		}
	}
}
