package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDetectEntryExists(t *testing.T) {
	dir := t.TempDir()
	wasmDir := filepath.Join(dir, "cmd", "wasm")
	if err := os.MkdirAll(wasmDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wasmDir, "main.go"), []byte("package main"), 0o644); err != nil {
		t.Fatal(err)
	}

	entry := detectEntry(dir)
	if entry != "./cmd/wasm" {
		t.Fatalf("expected ./cmd/wasm, got %s", entry)
	}
}

func TestDetectEntryFallback(t *testing.T) {
	dir := t.TempDir()
	entry := detectEntry(dir)
	if entry != "./cmd/wasm" {
		t.Fatalf("expected ./cmd/wasm fallback, got %s", entry)
	}
}

func TestLatestModTime(t *testing.T) {
	dir := t.TempDir()

	// Create a .go file
	goFile := filepath.Join(dir, "main.go")
	if err := os.WriteFile(goFile, []byte("package main"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a non-go file (should be ignored)
	txtFile := filepath.Join(dir, "readme.txt")
	if err := os.WriteFile(txtFile, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	latest := latestModTime(dir)
	if latest.IsZero() {
		t.Fatal("expected non-zero mod time")
	}

	info, _ := os.Stat(goFile)
	if !latest.Equal(info.ModTime()) {
		t.Fatalf("expected %v, got %v", info.ModTime(), latest)
	}
}

func TestLatestModTimeSkipsHidden(t *testing.T) {
	dir := t.TempDir()

	// Create hidden dir with a .go file
	hiddenDir := filepath.Join(dir, ".hidden")
	if err := os.MkdirAll(hiddenDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hiddenDir, "secret.go"), []byte("package x"), 0o644); err != nil {
		t.Fatal(err)
	}

	latest := latestModTime(dir)
	if !latest.IsZero() {
		t.Fatal("expected zero mod time (hidden dir should be skipped)")
	}
}

func TestLatestModTimeEmpty(t *testing.T) {
	dir := t.TempDir()
	latest := latestModTime(dir)
	if !latest.IsZero() {
		t.Fatal("expected zero mod time for empty dir")
	}
}

func TestParseBuildFlags(t *testing.T) {
	opts := parseBuildFlags([]string{"-dir", "/tmp/proj", "-output", "build", "-entry", "./cmd/app"})
	if opts.Dir != "/tmp/proj" {
		t.Fatalf("expected dir /tmp/proj, got %s", opts.Dir)
	}
	if opts.Output != "build" {
		t.Fatalf("expected output build, got %s", opts.Output)
	}
	if opts.Entry != "./cmd/app" {
		t.Fatalf("expected entry ./cmd/app, got %s", opts.Entry)
	}
}

func TestParseBuildFlagsDefaults(t *testing.T) {
	opts := parseBuildFlags(nil)
	if opts.Dir != "." || opts.Output != "dist" || opts.Entry != "" {
		t.Fatalf("unexpected defaults: %+v", opts)
	}
}

func TestParseServeFlags(t *testing.T) {
	opts := parseServeFlags([]string{"-dir", "/tmp/proj", "-entry", "./cmd/app"})
	if opts.Dir != "/tmp/proj" {
		t.Fatalf("expected dir /tmp/proj, got %s", opts.Dir)
	}
	if opts.Entry != "./cmd/app" {
		t.Fatalf("expected entry ./cmd/app, got %s", opts.Entry)
	}
}

func TestLatestModTimeNewerFile(t *testing.T) {
	dir := t.TempDir()

	old := filepath.Join(dir, "old.go")
	if err := os.WriteFile(old, []byte("package main"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Set old file to the past
	past := time.Now().Add(-1 * time.Hour)
	_ = os.Chtimes(old, past, past)

	newer := filepath.Join(dir, "newer.go")
	if err := os.WriteFile(newer, []byte("package main"), 0o644); err != nil {
		t.Fatal(err)
	}

	latest := latestModTime(dir)
	newerInfo, _ := os.Stat(newer)
	if !latest.Equal(newerInfo.ModTime()) {
		t.Fatalf("expected latest to match newer file")
	}
}
