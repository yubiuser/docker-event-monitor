package main

import (
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func buildStartupMessage(timestamp time.Time) string {
	var startup_message_builder strings.Builder

	startup_message_builder.WriteString("Docker event monitor started at " + timestamp.Format(time.RFC1123Z) + "\n")
	startup_message_builder.WriteString("Docker event monitor version: " + version + "\n")

	if config.Reporter.Pushover.Enabled {
		startup_message_builder.WriteString("Pushover notification Enabled")
	} else {
		startup_message_builder.WriteString("Pushover notification disabled")
	}

	if config.Reporter.Gotify.Enabled {
		startup_message_builder.WriteString("\nGotify notification Enabled")
	} else {
		startup_message_builder.WriteString("\nGotify notification disabled")
	}
	if config.Reporter.Mail.Enabled {
		startup_message_builder.WriteString("\nE-Mail notification Enabled")
	} else {
		startup_message_builder.WriteString("\nE-Mail notification disabled")
	}

	if config.Reporter.Mattermost.Enabled {
		startup_message_builder.WriteString("\nMattermost notification Enabled")
		if config.Reporter.Mattermost.Channel != "" {
			startup_message_builder.WriteString("\nMattermost channel: " + config.Reporter.Mattermost.Channel)
		}
		if config.Reporter.Mattermost.User != "" {
			startup_message_builder.WriteString("\nMattermost username: " + config.Reporter.Mattermost.User)
		}
	} else {
		startup_message_builder.WriteString("\nMattermost notification disabled")
	}

	startup_message_builder.WriteString("\nLog level: " + config.Options.LogLevel)

	if config.Options.ServerTag != "" {
		startup_message_builder.WriteString("\nServerTag: " + config.Options.ServerTag)
	} else {
		startup_message_builder.WriteString("\nServerTag: none")
	}

	if len(config.Options.Filter) > 0 {
		startup_message_builder.WriteString("\nGlobal filter: " + mapToString(config.Options.Filter))
	} else {
		startup_message_builder.WriteString("\nGlobal filter: none")
	}

	return startup_message_builder.String()
}

func logArguments() {
	log.Info().
		Interface("options", config.Options).
		Interface("reporter", config.Reporter).
		Interface("notifications", config.Notifications).
		Dict("version", zerolog.Dict().
			Str("Version", version).
			Str("Branch", branch).
			Str("Commit", commit).
			Time("Compile_date", stringToUnixTime(date)).
			Time("Git_date", stringToUnixTime(gitdate)),
		).
		Msg("Docker event monitor started")
}

