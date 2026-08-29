package main

import (
	"encoding/json"
	"errors"

	"github.com/rs/zerolog/log"
)

type GotifyMessage struct {
	Title   string `json:"title"`
	Message string `json:"message"`
}

func sendGotify(message string, title string, errCh chan ReporterError) {
	// Send a message to Gotify

	m := GotifyMessage{
		Title:   title,
		Message: message,
	}

	e := ReporterError{
		Reporter: "gotify",
	}

	messageJSON, err := json.Marshal(m)
	if err != nil {
		log.Error().Err(err).Str("reporter", "Gotify").Msg("Failed to marshal JSON")
		e.Error = errors.New("failed to marshal JSON")
		errCh <- e
		return
	}

	err = sendhttpMessage("Gotify", config.Reporter.Gotify.URL+"/message?token="+config.Reporter.Gotify.Token, messageJSON)
	if err != nil {
		e.Error = err
		errCh <- e
		return
	}

}
