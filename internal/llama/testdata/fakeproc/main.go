// Package main provides a fake process for testing llama.Process.
// This binary simulates various process behaviors for integration testing.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	mode := flag.String("mode", "run", "Process mode: run, exit, sigterm, sleep, crash")
	exitCode := flag.Int("exit-code", 0, "Exit code for exit mode")
	sleepDuration := flag.Duration("sleep", 5*time.Second, "Sleep duration for sleep mode")
	flag.Parse()

	switch *mode {
	case "run":
		// Run forever until killed
		fmt.Fprintln(os.Stdout, "ready")
		select {}

	case "exit":
		// Exit immediately with specified code
		fmt.Fprintln(os.Stderr, "exiting")
		os.Exit(*exitCode)

	case "sigterm":
		// Wait for SIGTERM and exit gracefully
		fmt.Fprintln(os.Stdout, "waiting for signal")
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, syscall.SIGTERM)
		<-sigc
		fmt.Fprintln(os.Stdout, "received SIGTERM, shutting down")
		time.Sleep(50 * time.Millisecond) // Brief cleanup
		os.Exit(0)

	case "sleep":
		// Sleep for specified duration then exit
		fmt.Fprintln(os.Stdout, "sleeping")
		time.Sleep(*sleepDuration)
		fmt.Fprintln(os.Stdout, "done sleeping")
		os.Exit(0)

	case "crash":
		// Simulate a crash
		fmt.Fprintln(os.Stderr, "crashing")
		os.Exit(1)

	default:
		fmt.Fprintf(os.Stderr, "unknown mode: %s\n", *mode)
		os.Exit(2)
	}
}
