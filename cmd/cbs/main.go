// CBS CLI tool for building and running CBS applications.
package main

import (
	"fmt"
	"os"
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
		fmt.Println("cbs build: compiling WASM client...")
		if err := buildWASM(); err != nil {
			fmt.Fprintf(os.Stderr, "build error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("cbs build: done")
	case "serve":
		fmt.Println("cbs serve: starting development server...")
		fmt.Println("TODO: implement dev server with hot reload")
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`CBS - Continuous Binary Synchronization

Usage:
  cbs <command>

Commands:
  build    Compile the WASM client module
  serve    Start the development server
  version  Print version information`)
}

func buildWASM() error {
	// This will invoke: GOOS=js GOARCH=wasm go build -o dist/cbs.wasm ./pkg/wasm
	fmt.Println("  GOOS=js GOARCH=wasm go build -o dist/cbs.wasm ./pkg/wasm")
	return os.MkdirAll("dist", 0o755)
}
