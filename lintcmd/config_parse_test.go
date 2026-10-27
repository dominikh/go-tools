package lintcmd

import (
	"strings"
	"testing"
)

func TestParseBuildConfigsUnterminatedFinalLine(t *testing.T) {
	got, err := parseBuildConfigs(strings.NewReader("linux:\nwindows:"))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Name != "linux" || got[1].Name != "windows" {
		t.Fatalf("parseBuildConfigs() = %#v, want linux and windows", got)
	}
}

func TestParseBuildConfigsErrorUsesPhysicalLine(t *testing.T) {
	_, err := parseBuildConfigs(strings.NewReader("linux:\n\nnot a config"))
	if err == nil {
		t.Fatal("parseBuildConfigs() succeeded, want error")
	}
	parseErr, ok := err.(parseBuildConfigError)
	if !ok {
		t.Fatalf("error type = %T, want parseBuildConfigError", err)
	}
	if parseErr.line != 3 {
		t.Fatalf("error line = %d, want 3", parseErr.line)
	}
}
