package main

import (
	"os/exec"
	"strings"
	"testing"
)

func TestModulePath(t *testing.T) {
	t.Setenv("GOWORK", "off")
	output, err := exec.Command("go", "list", "-m", "-f", "{{.Path}}").CombinedOutput()
	if err != nil {
		t.Fatalf("go list -m: %v\n%s", err, output)
	}

	const want = "github.com/magarcia/cc2md"
	if got := strings.TrimSpace(string(output)); got != want {
		t.Errorf("module path = %q, want %q for the documented go install command", got, want)
	}
}
