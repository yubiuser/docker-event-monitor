package main

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/docker/docker/api/types/events"
	"github.com/rs/zerolog/log"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func processEvent(event events.Message) {
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
	sendNotifications(timestamp, message, title, config.EnabledReporter)

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

func shouldSuppressForCrashOnly(event events.Message) bool {
	// For containers matching crash_only patterns, suppress all events
	// except "die" with a non-zero exit code that isn't a signal-based stop.
	// Exit codes 137 (SIGKILL) and 143 (SIGTERM) are normal Docker stop signals.
	if len(config.CrashOnlyRegexps) == 0 {
		return false
	}

	// crash_only is a per-container filter; ignore non-container events
	if event.Type != "container" {
		return false
	}

	name := getActorName(event)
	image := getActorImage(event)

	matched := false
	for _, re := range config.CrashOnlyRegexps {
		if re.MatchString(name) || re.MatchString(image) {
			matched = true
			break
		}
	}
	if !matched {
		return false
	}

	// Container matches a crash_only pattern — only allow "die" with a real crash exit code
	if event.Action != "die" {
		log.Debug().
			Str("ActorName", name).
			Str("Action", string(event.Action)).
			Msg("Suppressed (crash_only: not a die event)")
		return true
	}

	exitCode, ok := event.Actor.Attributes["exitCode"]
	if !ok || exitCode == "" {
		log.Debug().
			Str("ActorName", name).
			Msg("Suppressed (crash_only: missing exit code)")
		return true
	}
	if exitCode == "0" || exitCode == "137" || exitCode == "143" {
		log.Debug().
			Str("ActorName", name).
			Str("exitCode", exitCode).
			Msg("Suppressed (crash_only: clean/signal exit)")
		return true
	}

	log.Debug().
		Str("ActorName", name).
		Str("exitCode", exitCode).
		Msg("Crash detected (crash_only: non-zero exit)")
	return false
}

func excludeEvent(event events.Message) bool {
	// Checks if any of the exclusion criteria matches the event

	ActorID := getActorID(event)

	// Convert the event (struct of type event.Message) to a flattend map
	eventMap := structToFlatMap(event)

	// Check for all exclude key -> value combinations if they match the event
	for key, values := range config.Exclude {
		eventValue, keyExist := eventMap[key]

		// Check if the exclusion key exists in the eventMap
		if !keyExist {
			log.Debug().
				Str("ActorID", ActorID).
				Msgf("Exclusion key \"%s\" did not match", key)
			return false
		}

		log.Debug().
			Str("ActorID", ActorID).
			Msgf("Exclusion key \"%s\" matched, checking values", key)

		log.Debug().
			Str("ActorID", ActorID).
			Msgf("Event's value for key \"%s\" is \"%s\"", key, eventValue)

		for _, value := range values {
			// comparing the prefix to be able to filter actions like "exec_XXX: YYYY" which use a
			// special, dynamic, syntax
			// see https://github.com/moby/moby/blob/bf053be997f87af233919a76e6ecbd7d17390e62/api/types/events/events.go#L74-L81

			if strings.HasPrefix(eventValue, value) {
				log.Debug().
					Str("ActorID", ActorID).
					Msgf("Event excluded based on exclusion setting \"%s=%s\"", key, value)
				return true
			}
		}
		log.Debug().
			Str("ActorID", ActorID).
			Msgf("Exclusion key \"%s\" matched, but values did not match", key)
	}

	return false
}

// flatten a nested map, separating nested keys by dots
func flattenMap(prefix string, m map[string]interface{}) map[string]string {
	flatMap := make(map[string]string)
	for k, v := range m {
		newKey := k
		// separate nested keys by dot
		if prefix != "" {
			newKey = prefix + "." + k
		}
		// if the value is a map/struct itself, transverse it recursivly
		switch k {
		case "Actor", "Attributes":
			nestedMap := v.(map[string]interface{})
			for nk, nv := range flattenMap(newKey, nestedMap) {
				flatMap[nk] = nv
			}
		case "time", "timeNano":
			flatMap[newKey] = string(v.(json.Number))
		default:
			flatMap[newKey] = v.(string)
		}
	}
	return flatMap
}

// Convert struct to flat map by first converting it to a map (via JSON) and flatten it afterwards
func structToFlatMap(s interface{}) map[string]string {
	m := make(map[string]interface{})
	b, err := json.Marshal(s)
	if err != nil {
		log.Fatal().Err(err).Msg("Marshaling JSON failed")
	}

	// Using a custom decoder to set 'UseNumber' which will preserver a string representation of
	// time and timeNano instead of converting it to float64
	decoder := json.NewDecoder(strings.NewReader(string(b)))
	decoder.UseNumber()
	if err := decoder.Decode(&m); err != nil {
		log.Fatal().Err(err).Msg("Unmarshaling JSON failed")
	}
	return flattenMap("", m)
}
