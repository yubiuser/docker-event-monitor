package main

import (
	"encoding/json"
	"errors"
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
		Reporter: "Gotify",
	}

	messageJSON, err := json.Marshal(m)
	if err != nil {
		logger.Error().Err(err).Str("reporter", "Gotify").Msg("Failed to marshal JSON")
		e.Error = errors.New("failed to marshal JSON")
		errCh <- e
		return
	}

	err = sendhttpMessage("Gotify", glb_arguments.GotifyURL+"/message?token="+glb_arguments.GotifyToken, messageJSON)
	if err != nil {
		e.Error = err
		errCh <- e
		return
	}

}
