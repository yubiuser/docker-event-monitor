package main

import (
	"slices"
	"strings"
	"time"

	"github.com/docker/docker/api/types/events"
	"github.com/rs/zerolog/log"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func checkReporter(event events.Message) {

	// check if notifications are configured and apply event checks
	// if none are configured process event right away
	if len(config.Notifications) > 0 {

		for _, notification := range config.Notifications {
			log.Debug().Str("rule", notification.Name).Msg("Checking event for match")

			if matchEvent(event, notification) {
				log.Debug().Str("rule", notification.Name).Msg("Rule matched")

				// check which reporters should be used
				// if none are configured notification is send to all enabled reporters
				if len(notification.Notify) > 0 {

					// remove disabled reporters
					for _, reporter := range notification.Notify {
						if !slices.Contains(config.EnabledReporter, reporter) {
							log.Error().Str("reporter", reporter).Msg("Reporter not enabled")
							notification.Notify = removeFromSlice(notification.Notify, reporter)
						}
					}

					// check if there are reporters left after removing disabled ones
					if len(notification.Notify) > 0 {
						log.Debug().Str("rule", notification.Name).Interface("using reporters", notification.Notify).Send()
						processEvent(event, notification.Notify)
					} else {
						log.Error().Str("rule", notification.Name).Msg("No enabled reporter for this rule found")
					}

				} else {
					processEvent(event, config.EnabledReporter)
				}
			}
		}

	} else {
		processEvent(event, config.EnabledReporter)
	}
}
func matchEvent(event events.Message, notification notification) bool {
	return true
}

func processEvent(event events.Message, reporters []string) {
	// the Docker Events endpoint will return a struct events.Message
	// https://pkg.go.dev/github.com/docker/docker/api/types/events#Message

	var msg_builder, title_builder strings.Builder
	var ActorID, ActorImage, ActorName, TitleID, ActorImageVersion string

	ActorID = getActorID(event)
	ActorImage = getActorImage(event)
	ActorName = getActorName(event)
	ActorImageVersion = getActorImageVersion(event)

	// Check possible image and container name
	// The order of the checks is important, because we want name rather than ActorID
	// as identifier in the title
	if len(ActorID) > 0 {
		msg_builder.WriteString("ID: " + ActorID + "\n")
		TitleID = ActorID
	}
	if len(ActorImage) > 0 {
		msg_builder.WriteString("Image: " + ActorImage + "\n")
		// Not using ActorImage as possible title, because it's too long
	}
	if len(ActorImageVersion) > 0 {
		msg_builder.WriteString("Image version: " + ActorImageVersion + "\n")
	}

	if len(ActorName) > 0 {
		msg_builder.WriteString("Name: " + ActorName + "\n")
		TitleID = ActorName
	}

	// Build title
	title_builder.WriteString(cases.Title(language.English, cases.Compact).String(string(event.Type)))
	if len(TitleID) > 0 {
		title_builder.WriteString(" " + TitleID)
	}
	title_builder.WriteString(": " + string(event.Action))

	// Get event timestamp
	timestamp := time.Unix(event.Time, 0)
	msg_builder.WriteString("Time: " + timestamp.Format(time.RFC1123Z) + "\n")

	// Append possible docker compose context
	if len(event.Actor.Attributes["com.docker.compose.project.working_dir"]) > 0 {
		msg_builder.WriteString("Docker compose context: " + event.Actor.Attributes["com.docker.compose.project.working_dir"] + "\n")
	}
	if len(event.Actor.Attributes["com.docker.compose.service"]) > 0 {
		msg_builder.WriteString("Docker compose service: " + event.Actor.Attributes["com.docker.compose.service"] + "\n")
	}

	// Build message and title
	title := title_builder.String()
	message := strings.TrimRight(msg_builder.String(), "\n")

	// Log message
	log.Info().
		Str("eventType", string(event.Type)).
		Str("ActorID", ActorID).
		Str("eventAction", string(event.Action)).
		Str("ActorImage", ActorImage).
		Str("ActorImageVersion", ActorImageVersion).
		Str("ActorName", ActorName).
		Str("DockerComposeContext", event.Actor.Attributes["com.docker.compose.project.working_dir"]).
		Str("DockerComposeService", event.Actor.Attributes["com.docker.compose.service"]).
		Msg(title)

	// send notifications to various reporters
	// function will finish when all reporters finished
	sendNotifications(timestamp, message, title, reporters)

}

func getActorID(event events.Message) string {
	var ActorID string

	if len(event.Actor.ID) > 0 {
		if strings.HasPrefix(event.Actor.ID, "sha256:") {
			ActorID = strings.TrimPrefix(event.Actor.ID, "sha256:")[:8] //remove prefix + limit ActorID legth
		} else {
			ActorID = event.Actor.ID[:8] //limit ActorID legth
		}
	}
	return ActorID
}

func getActorImage(event events.Message) string {
	var ActorImage string

	if len(event.Actor.Attributes["image"]) > 0 {
		ActorImage = event.Actor.Attributes["image"]
	} else {
		// try to recover image name from org.opencontainers.image info
		if len(event.Actor.Attributes["org.opencontainers.image.title"]) > 0 && len(event.Actor.Attributes["org.opencontainers.image.version"]) > 0 {
			ActorImage = event.Actor.Attributes["org.opencontainers.image.title"] + ":" + event.Actor.Attributes["org.opencontainers.image.version"]
		}
	}
	return ActorImage
}

func getActorImageVersion(event events.Message) string {
	var ActorImageVersion string

	if len(event.Actor.Attributes["org.opencontainers.image.version"]) > 0 {
		ActorImageVersion = event.Actor.Attributes["org.opencontainers.image.version"]
	}
	return ActorImageVersion

}

func getActorName(event events.Message) string {
	var ActorName string

	if len(event.Actor.Attributes["name"]) > 0 {
		// in case the ActorName is only an hash
		if strings.HasPrefix(event.Actor.Attributes["name"], "sha256:") {
			ActorName = strings.TrimPrefix(event.Actor.Attributes["name"], "sha256:")[:8] //remove prefix + limit ActorName legth
		} else {
			ActorName = event.Actor.Attributes["name"]
		}
	}
	return ActorName

}
