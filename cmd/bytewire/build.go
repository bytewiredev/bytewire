package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// BuildOptions configures the WASM build.
type BuildOptions struct {
	Dir    string // project root directory
	Output string // output directory (relative to Dir)
	Entry  string // entry package path (relative to Dir)
}

// Build compiles the Bytewire WASM client module.
func Build(opts BuildOptions) error {
	if opts.Dir == "" {
		opts.Dir = "."
	}
	if opts.Output == "" {
		opts.Output = "dist"
	}
	if opts.Entry == "" {
		entry, err := detectEntry(opts.Dir)
		if err != nil {
			return err
		}
		opts.Entry = entry
	}

	// Validate the entry package exists before building.
	if err := validateEntry(opts.Dir, opts.Entry); err != nil {
		return err
	}

	outDir := filepath.Join(opts.Dir, opts.Output)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	wasmOut := filepath.Join(outDir, "bytewire.wasm")
	start := time.Now()

	// Build WASM binary
	cmd := exec.Command("go", "build", "-o", wasmOut, opts.Entry)
	cmd.Dir = opts.Dir
	cmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Printf("  GOOS=js GOARCH=wasm go build -o %s %s\n", wasmOut, opts.Entry)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go build: %w", err)
	}

	// Copy wasm_exec.js from GOROOT
	if err := copyWasmExecJS(outDir); err != nil {
		return fmt.Errorf("copy wasm_exec.js: %w", err)
	}

	printBuildStats(outDir, time.Since(start))
	return nil
}

// detectEntry finds the WASM entry package in a Bytewire project.
func detectEntry(dir string) (string, error) {
	candidate := filepath.Join(dir, "cmd", "wasm", "main.go")
	if _, err := os.Stat(candidate); err == nil {
		return "./cmd/wasm", nil
	}
	return "", fmt.Errorf("no WASM entry package found (looked for cmd/wasm/main.go); use --entry to specify one")
}

// validateEntry checks that the entry package directory exists.
func validateEntry(dir, entry string) error {
	entryDir := filepath.Join(dir, entry)
	info, err := os.Stat(entryDir)
	if err != nil {
		return fmt.Errorf("entry package %q not found: %w", entry, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("entry package %q is not a directory", entry)
	}
	return nil
}

// printBuildStats prints file sizes and build duration.
func printBuildStats(outDir string, elapsed time.Duration) {
	fmt.Printf("  build completed in %s\n", elapsed.Round(time.Millisecond))
	entries, err := os.ReadDir(outDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		fmt.Printf("  %s  %s\n", formatSize(info.Size()), e.Name())
	}
}

// formatSize formats a byte count as a human-readable string.
func formatSize(bytes int64) string {
	const (
		kb = 1024
		mb = 1024 * kb
	)
	switch {
	case bytes >= mb:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(mb))
	case bytes >= kb:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(kb))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// copyWasmExecJS copies Go's wasm_exec.js to the output directory.
func copyWasmExecJS(outDir string) error {
	goroot := os.Getenv("GOROOT")
	if goroot == "" {
		out, err := exec.Command("go", "env", "GOROOT").Output()
		if err != nil {
			return fmt.Errorf("determine GOROOT: %w", err)
		}
		goroot = string(out)
		// Trim newline
		if len(goroot) > 0 && goroot[len(goroot)-1] == '\n' {
			goroot = goroot[:len(goroot)-1]
		}
	}

	src := filepath.Join(goroot, "lib", "wasm", "wasm_exec.js")
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read %s: %w", src, err)
	}

	dst := filepath.Join(outDir, "wasm_exec.js")
	fmt.Printf("  cp %s %s\n", src, dst)
	return os.WriteFile(dst, data, 0o644)
}
