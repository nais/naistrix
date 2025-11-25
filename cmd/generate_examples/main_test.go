package main

import (
	"strings"
	"testing"
)

func TestTrimSnapshotPadding(t *testing.T) {
	rule := strings.Repeat("─", 80) + "\n"
	before := "> deploy plan\n\nService name: gateway\n\n\n" + rule + "Plans: 1\n\n" + rule
	want := "> deploy plan\n\nService name: gateway\n" + rule + "Plans: 1\n" + rule
	if got := string(trimSnapshotPadding([]byte(before))); got != want {
		t.Fatalf("trimmed snapshot = %q, want %q", got, want)
	}
}
