package main

import (
	"fmt"
	"strconv"
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

	if len(config.Options.ExcludeStrings) > 0 {
		startup_message_builder.WriteString("\nExcludeStrings: " + strings.Join(config.Options.ExcludeStrings, " "))
	} else {
		startup_message_builder.WriteString("\nExcludeStrings: none")
	}

	return startup_message_builder.String()
}

func logArguments() {
	log.Info().
		Interface("options", config.Options).
		Interface("reporter", config.Reporter).
		Dict("version", zerolog.Dict().
			Str("Version", version).
			Str("Branch", branch).
			Str("Commit", commit).
			Time("Compile_date", stringToUnixTime(date)).
			Time("Git_date", stringToUnixTime(gitdate)),
		).
		Msg("Docker event monitor started")
}

// converts a string to a time.Time in unix fortmat
func stringToUnixTime(str string) time.Time {
	i, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		log.Fatal().Err(err).Msg("String to timestamp conversion failed")
	}
	tm := time.Unix(i, 0)
	return tm
}

// walks through a nested map and returns a unified string as in
// string = [key:value,value] [key:value] []key:value,value
func mapToString(m map[string][]string) string {
	var builder strings.Builder

	// Iterate over the map
	for key, values := range m {
		// Append key to the string
		builder.WriteString(fmt.Sprintf("[%s: ", key))

		lenght := len(values)
		i := 0
		// Append values to the string
		for _, value := range values {
			i++
			builder.WriteString(value)
			// Add comma except for the last value
			if i < lenght {
				builder.WriteString(", ")
			}
		}
		// Add a space to separate keys
		builder.WriteString("] ")
	}

	return builder.String()
}
