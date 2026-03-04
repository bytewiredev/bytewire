package main

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
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

	entry, err := detectEntry(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry != "./cmd/wasm" {
		t.Fatalf("expected ./cmd/wasm, got %s", entry)
	}
}

func TestDetectEntryNotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := detectEntry(dir)
	if err == nil {
		t.Fatal("expected error when no entry package exists")
	}
}

func TestValidateEntry(t *testing.T) {
	dir := t.TempDir()
	wasmDir := filepath.Join(dir, "cmd", "wasm")
	if err := os.MkdirAll(wasmDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := validateEntry(dir, "./cmd/wasm"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateEntryMissing(t *testing.T) {
	dir := t.TempDir()
	if err := validateEntry(dir, "./cmd/wasm"); err == nil {
		t.Fatal("expected error for missing entry package")
	}
}

func TestValidateEntryNotDir(t *testing.T) {
	dir := t.TempDir()
	// Create a file where a directory is expected
	if err := os.WriteFile(filepath.Join(dir, "notadir"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := validateEntry(dir, "notadir"); err == nil {
		t.Fatal("expected error when entry is not a directory")
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{2621440, "2.5 MB"},
	}
	for _, tt := range tests {
		got := formatSize(tt.bytes)
		if got != tt.want {
			t.Errorf("formatSize(%d) = %q, want %q", tt.bytes, got, tt.want)
		}
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

func TestLatestModTimeSkipsDist(t *testing.T) {
	dir := t.TempDir()

	distDir := filepath.Join(dir, "dist")
	if err := os.MkdirAll(distDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(distDir, "generated.go"), []byte("package x"), 0o644); err != nil {
		t.Fatal(err)
	}

	latest := latestModTime(dir)
	if !latest.IsZero() {
		t.Fatal("expected zero mod time (dist dir should be skipped)")
	}
}

func TestParsServeFlagsDefaults(t *testing.T) {
	opts := parseServeFlags(nil)
	if opts.Dir != "." || opts.Entry != "" {
		t.Fatalf("unexpected defaults: %+v", opts)
	}
}

func TestWatcherDebounce(t *testing.T) {
	// The watcher should debounce multiple rapid changes into a single rebuild.
	var rebuildCount atomic.Int32
	changeCount := 0

	w := &watcher{
		pollInterval: 10 * time.Millisecond,
		debounce:     50 * time.Millisecond,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Simulate 5 rapid changes, then stop changing.
	// Should result in exactly 1 rebuild after debounce settles.
	w.run(ctx, func() bool {
		changeCount++
		// Report changes for the first 5 polls
		return changeCount <= 5
	}, func() {
		rebuildCount.Add(1)
	}, func() {})

	count := rebuildCount.Load()
	if count != 1 {
		t.Fatalf("expected 1 rebuild after debounce, got %d", count)
	}
}

func TestWatcherNoChangeNoRebuild(t *testing.T) {
	var rebuildCount atomic.Int32

	w := &watcher{
		pollInterval: 10 * time.Millisecond,
		debounce:     20 * time.Millisecond,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	w.run(ctx, func() bool {
		return false // never changed
	}, func() {
		rebuildCount.Add(1)
	}, func() {})

	if rebuildCount.Load() != 0 {
		t.Fatal("expected no rebuilds when nothing changes")
	}
}

func TestWatcherCleanupOnCancel(t *testing.T) {
	var cleaned atomic.Bool

	w := &watcher{
		pollInterval: 10 * time.Millisecond,
		debounce:     20 * time.Millisecond,
	}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		w.run(ctx, func() bool { return false }, func() {}, func() {
			cleaned.Store(true)
		})
		close(done)
	}()

	cancel()
	<-done

	if !cleaned.Load() {
		t.Fatal("expected cleanup to be called on context cancellation")
	}
}
