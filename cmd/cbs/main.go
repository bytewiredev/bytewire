// CBS CLI tool for building and running CBS applications.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
)

const version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "version":
		fmt.Printf("cbs v%s\n", version)
	case "build":
		opts := parseBuildFlags(os.Args[2:])
		fmt.Println("cbs build: compiling WASM client...")
		if err := Build(opts); err != nil {
			fmt.Fprintf(os.Stderr, "build error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("cbs build: done")
	case "serve":
		opts := parseServeFlags(os.Args[2:])
		fmt.Println("cbs serve: starting development server...")

		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
		defer cancel()

		if err := Serve(ctx, opts); err != nil {
			fmt.Fprintf(os.Stderr, "serve error: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`CBS - Continuous Binary Synchronization

Usage:
  cbs <command> [flags]

Commands:
  build    Compile the WASM client module
  serve    Start the development server with auto-rebuild
  version  Print version information

Run 'cbs <command> -help' for command-specific flags.`)
}

func parseBuildFlags(args []string) BuildOptions {
	fs := flag.NewFlagSet("build", flag.ExitOnError)
	dir := fs.String("dir", ".", "Project root directory")
	output := fs.String("output", "dist", "Output directory for build artifacts")
	entry := fs.String("entry", "", "WASM entry package (default: auto-detect)")
	_ = fs.Parse(args)
	return BuildOptions{Dir: *dir, Output: *output, Entry: *entry}
}

func parseServeFlags(args []string) ServeOptions {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	dir := fs.String("dir", ".", "Project root directory")
	entry := fs.String("entry", "", "WASM entry package (default: auto-detect)")
	_ = fs.Parse(args)
	return ServeOptions{Dir: *dir, Entry: *entry}
}
