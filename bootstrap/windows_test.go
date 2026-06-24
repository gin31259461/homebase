package bootstrap_test

import (
	"bytes"
	"os"
	"testing"
)

func TestWindowsBootstrapHasNoUTF8BOM(t *testing.T) {
	data, err := os.ReadFile("windows.ps1")
	if err != nil {
		t.Fatal(err)
	}

	if bytes.HasPrefix(data, []byte{0xEF, 0xBB, 0xBF}) {
		t.Fatal("windows.ps1 must not start with a UTF-8 BOM; irm | iex preserves it and breaks the param block")
	}
	if !bytes.HasPrefix(data, []byte("#Requires")) {
		t.Fatalf("windows.ps1 must start with #Requires, got %q", data[:min(len(data), 16)])
	}
}
