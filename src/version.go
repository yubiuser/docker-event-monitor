package main

import (
	"os"

	"github.com/rs/zerolog/log"
)

// version information, are injected during build process
var (
	version string = "n/a"
	commit  string = "n/a"
	date    string = "0"
	gitdate string = "0"
	branch  string = "n/a"
)

func printVersion() {
	log.Info().
		Str("Version", version).
		Str("Branch", branch).
		Str("Commit", commit).
		Time("Compile_date", stringToUnixTime(date)).
		Time("Git_date", stringToUnixTime(gitdate)).
		Msg("Version Information")
	os.Exit(0)
}
