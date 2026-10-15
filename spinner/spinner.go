package spinner

import (
	"Builder/output"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/theckman/yacspin"
)

var cfg = yacspin.Config{
	Frequency:     100 * time.Millisecond,
	CharSet:       []string{"⣧", "⣇", "⡇", "⠇", "⠃", "⠁", "⠃", "⠇", "⡇", "⣇", "⣧", "⣧"},
	StopCharacter: "",
}
var Spinner, _ = yacspin.New(spinnerConfig())

// spinnerConfig silences the spinner when output isn't a terminal (agents, CI,
// pipes, --json), where its frames would only add noise to captured output.
func spinnerConfig() yacspin.Config {
	c := cfg
	if !output.TTY() {
		c.Writer = io.Discard
	}
	return c
}

var Caller string

func debugEnabled() bool {
	for _, a := range os.Args[1:] {
		if a == "-d" || a == "--debug" {
			return true
		}
	}
	return false
}

// LogMessage prints msg when --debug is set. A "fatal" message is always
// printed to stderr and exits with status 1, with or without --debug.
func LogMessage(msg string, level string) {
	_, file, no, ok := runtime.Caller(1)
	if ok {
		Caller = filepath.Base(file) + ":" + fmt.Sprint(no)
	}

	Spinner.Stop()

	var label string
	switch level {
	case "info":
		label = "INFO"
	case "warn":
		label = "WARN"
	case "error":
		label = "ERROR"
	default:
		label = "FATAL"
	}
	line := time.Now().Local().String() + "   " + label + "   " + Caller + ":   " + msg

	if label == "FATAL" {
		fmt.Fprintln(os.Stderr, line)
		output.Failure(msg, Caller)
		os.Exit(1)
	}
	if debugEnabled() {
		fmt.Println(line)
	}

	Spinner.Start()
}
