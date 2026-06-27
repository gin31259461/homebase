package testutil

import (
	"testing"

	"github.com/gin31259461/homebase/internal/run"
)

func TestExitErrorCarriesExitCode(t *testing.T) {
	got, ok := run.ExitCode(ExitError(3))
	if !ok || got != 3 {
		t.Fatalf("ExitCode() = %d, %v; want 3, true", got, ok)
	}
}
