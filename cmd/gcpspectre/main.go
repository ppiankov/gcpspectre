package main

import (
	"errors"
	"log/slog"
	"os"

	"github.com/ppiankov/gcpspectre/internal/commands"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if err := commands.Execute(version, commit, date); err != nil {
		var exitErr commands.ExitCodeError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.Code)
		}
		slog.Warn("Command failed", "error", err)
		os.Exit(1)
	}
}
