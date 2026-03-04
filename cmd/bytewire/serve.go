package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ServeOptions configures the dev server.
type ServeOptions struct {
	Dir   string // project root directory
	Entry string // entry package for WASM build
}

// Serve runs the development server with auto-rebuild on file changes.
func Serve(ctx context.Context, opts ServeOptions) error {
	if opts.Dir == "" {
		opts.Dir = "."
	}

	// Initial build
	fmt.Println("bytewire serve: initial build...")
	buildOpts := BuildOptions{Dir: opts.Dir, Entry: opts.Entry}
	if err := Build(buildOpts); err != nil {
		return fmt.Errorf("initial build: %w", err)
	}

	// Start the app process
	proc, err := startProcess(opts.Dir)
	if err != nil {
		return fmt.Errorf("start process: %w", err)
	}

	fmt.Println("bytewire serve: ready")

	lastMod := latestModTime(opts.Dir)
	w := &watcher{
		pollInterval: 500 * time.Millisecond,
		debounce:     300 * time.Millisecond,
	}

	return w.run(ctx, func() bool {
		current := latestModTime(opts.Dir)
		if current.After(lastMod) {
			lastMod = current
			return true
		}
		return false
	}, func() {
		fmt.Println("\nbytewire serve: rebuilding...")

		stopProcess(proc)

		if err := Build(buildOpts); err != nil {
			fmt.Fprintf(os.Stderr, "rebuild error: %v\n", err)
			return
		}

		proc, err = startProcess(opts.Dir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "restart error: %v\n", err)
			return
		}
		fmt.Println("bytewire serve: ready")
	}, func() {
		stopProcess(proc)
	})
}

// watcher polls for changes and debounces rebuild triggers.
type watcher struct {
	pollInterval time.Duration // how often to poll for changes
	debounce     time.Duration // how long to wait after a change before rebuilding
}

// run polls using changed() and calls rebuild() after debouncing.
// cleanup is called on context cancellation.
func (w *watcher) run(ctx context.Context, changed func() bool, rebuild func(), cleanup func()) error {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	var debounceTimer *time.Timer
	var debounceCh <-chan time.Time

	for {
		select {
		case <-ctx.Done():
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			cleanup()
			return nil
		case <-ticker.C:
			if changed() {
				// Reset debounce timer on each change detection
				if debounceTimer != nil {
					debounceTimer.Stop()
				}
				debounceTimer = time.NewTimer(w.debounce)
				debounceCh = debounceTimer.C
			}
		case <-debounceCh:
			debounceCh = nil
			debounceTimer = nil
			rebuild()
		}
	}
}

// startProcess starts `go run .` in the given directory.
func startProcess(dir string) (*exec.Cmd, error) {
	cmd := exec.Command("go", "run", ".")
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return cmd, nil
}

// stopProcess sends interrupt then waits, falling back to kill.
func stopProcess(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = cmd.Process.Signal(os.Interrupt)
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		_ = cmd.Process.Kill()
		<-done
	}
}

// latestModTime walks dir for .go files and returns the most recent mod time.
func latestModTime(dir string) time.Time {
	var latest time.Time
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		name := d.Name()
		// Skip hidden dirs and common non-source dirs
		if d.IsDir() && (strings.HasPrefix(name, ".") || name == "vendor" || name == "node_modules" || name == "dist") {
			return filepath.SkipDir
		}
		if !d.IsDir() && strings.HasSuffix(name, ".go") {
			info, err := d.Info()
			if err == nil && info.ModTime().After(latest) {
				latest = info.ModTime()
			}
		}
		return nil
	})
	return latest
}
