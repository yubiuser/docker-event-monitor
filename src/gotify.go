package main

import (
	"encoding/json"
)

type GotifyMessage struct {
	Title   string `json:"title"`
	Message string `json:"message"`
}

func sendGotify(message string, title string) {
	// Send a message to Gotify

	m := GotifyMessage{
		Title:   title,
		Message: message,
	}

	messageJSON, err := json.Marshal(m)
	if err != nil {
		logger.Error().Err(err).Str("reporter", "Gotify").Msg("Faild to marshal JSON")
		return
	}

	sendhttpMessage("Gotify", glb_arguments.GotifyURL+"/message?token="+glb_arguments.GotifyToken, messageJSON)

}
