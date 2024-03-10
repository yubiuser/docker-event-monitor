package main

import "os"

func printVersion() {
	logger.Info().
		Str("Version", version).
		Str("Branch", branch).
		Str("Commit", commit).
		Time("Compile_date", stringToUnix(date)).
		Time("Git_date", stringToUnix(gitdate)).
		Msg("Version Information")
	os.Exit(0)
}
