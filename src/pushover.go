package main

import (
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
)

type PushoverMessage struct {
	Token     string `json:"token"`
	User      string `json:"user"`
	Title     string `json:"title"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

func sendPushover(timestamp time.Time, message string, title string, errCh chan ReporterError) {
	// Send a message to Pushover

	m := PushoverMessage{
		Token:     config.Reporter.Pushover.APIToken,
		User:      config.Reporter.Pushover.UserKey,
		Title:     title,
		Message:   message,
		Timestamp: strconv.FormatInt(timestamp.Unix(), 10),
	}

	e := ReporterError{
		Reporter: "pushover",
	}

	messageJSON, err := json.Marshal(m)
	if err != nil {
		log.Error().Err(err).Str("reporter", "Pushover").Msg("Failed to marshal JSON")
		e.Error = errors.New("failed to marshal JSON")
		errCh <- e
		return
	}

	err = sendhttpMessage("Pushover", "https://api.pushover.net/1/messages.json", messageJSON)
	if err != nil {
		e.Error = err
		errCh <- e
		return
	}

}
