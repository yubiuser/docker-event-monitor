package main

import (
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

func buildStartupMessage(timestamp time.Time) string {
	var startup_message_builder strings.Builder

	startup_message_builder.WriteString("Docker event monitor started at " + timestamp.Format(time.RFC1123Z) + "\n")
	startup_message_builder.WriteString("Docker event monitor version: " + version + "\n")

	if glb_arguments.Pushover {
		startup_message_builder.WriteString("Pushover notification enabled")
	} else {
		startup_message_builder.WriteString("Pushover notification disabled")
	}

	if glb_arguments.Gotify {
		startup_message_builder.WriteString("\nGotify notification enabled")
	} else {
		startup_message_builder.WriteString("\nGotify notification disabled")
	}
	if glb_arguments.Mail {
		startup_message_builder.WriteString("\nE-Mail notification enabled")
	} else {
		startup_message_builder.WriteString("\nE-Mail notification disabled")
	}

	if glb_arguments.Mattermost {
		startup_message_builder.WriteString("\nMattermost notification enabled")
		if glb_arguments.MattermostChannel != "" {
			startup_message_builder.WriteString("\nMattermost channel: " + glb_arguments.MattermostChannel)
		}
		if glb_arguments.MattermostUser != "" {
			startup_message_builder.WriteString("\nMattermost username: " + glb_arguments.MattermostUser)
		}
	} else {
		startup_message_builder.WriteString("\nMattermost notification disabled")
	}

	if glb_arguments.Delay > 0 {
		startup_message_builder.WriteString("\nUsing delay of " + glb_arguments.Delay.String())
	} else {
		startup_message_builder.WriteString("\nDelay disabled")
	}

	startup_message_builder.WriteString("\nLog level: " + glb_arguments.LogLevel)

	if glb_arguments.ServerTag != "" {
		startup_message_builder.WriteString("\nServerTag: " + glb_arguments.ServerTag)
	} else {
		startup_message_builder.WriteString("\nServerTag: none")
	}

	if len(glb_arguments.FilterStrings) > 0 {
		startup_message_builder.WriteString("\nFilterStrings: " + strings.Join(glb_arguments.FilterStrings, " "))
	} else {
		startup_message_builder.WriteString("\nFilterStrings: none")
	}

	if len(glb_arguments.ExcludeStrings) > 0 {
		startup_message_builder.WriteString("\nExcludeStrings: " + strings.Join(glb_arguments.ExcludeStrings, " "))
	} else {
		startup_message_builder.WriteString("\nExcludeStrings: none")
	}

	return startup_message_builder.String()
}

func logArguments() {
	logger.Info().
		Dict("options", zerolog.Dict().
			Dict("reporter", zerolog.Dict().
				Dict("Pushover", zerolog.Dict().
					Bool("enabled", glb_arguments.Pushover).
					Str("PushoverAPIToken", glb_arguments.PushoverAPIToken).
					Str("PushoverUserKey", glb_arguments.PushoverUserKey),
				).
				Dict("Gotify", zerolog.Dict().
					Bool("enabled", glb_arguments.Gotify).
					Str("GotifyURL", glb_arguments.GotifyURL).
					Str("GotifyToken", glb_arguments.GotifyToken),
				).
				Dict("Mail", zerolog.Dict().
					Bool("enabled", glb_arguments.Mail).
					Str("MailFrom", glb_arguments.MailFrom).
					Str("MailTo", glb_arguments.MailTo).
					Str("MailHost", glb_arguments.MailHost).
					Str("MailUser", glb_arguments.MailUser).
					Int("Port", glb_arguments.MailPort),
				).
				Dict("Mattermost", zerolog.Dict().
					Bool("enabled", glb_arguments.Mattermost).
					Str("MattermostURL", glb_arguments.MattermostURL).
					Str("MattermostChannel", glb_arguments.MattermostChannel).
					Str("MattermostUser", glb_arguments.MattermostUser),
				),
			).
			Str("Delay", glb_arguments.Delay.String()).
			Str("Loglevel", glb_arguments.LogLevel).
			Str("ServerTag", glb_arguments.ServerTag).
			Str("Filter", strings.Join(glb_arguments.FilterStrings, " ")).
			Str("Exclude", strings.Join(glb_arguments.ExcludeStrings, " ")),
		).
		Dict("version", zerolog.Dict().
			Str("Version", version).
			Str("Branch", branch).
			Str("Commit", commit).
			Time("Compile_date", stringToUnix(date)).
			Time("Git_date", stringToUnix(gitdate)),
		).
		Msg("Docker event monitor started")
}

func stringToUnix(str string) time.Time {
	i, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		logger.Fatal().Err(err).Msg("String to timestamp conversion failed")
	}
	tm := time.Unix(i, 0)
	return tm
}
