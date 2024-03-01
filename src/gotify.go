package main

import (
	"io"
	"net/http"
	"net/url"
)

func sendGotify(message string, title string) {

	response, err := http.PostForm(glb_arguments.GotifyURL+"/message?token="+glb_arguments.GotifyToken,
		url.Values{"message": {message}, "title": {title}})
	if err != nil {
		logger.Error().Err(err).Str("reporter", "Gotify").Msg("")
		return
	}

	defer response.Body.Close()

	statusCode := response.StatusCode

	body, err := io.ReadAll(response.Body)
	if err != nil {
		logger.Error().Err(err).Str("reporter", "Gotify").Msg("")
		return
	}

	logger.Debug().Str("reporter", "Gotify").Msgf("Gotify response statusCode: %d", statusCode)
	logger.Debug().Str("reporter", "Gotify").Msgf("Gotify response body: %s", string(body))

	// Log non successfull status codes
	if statusCode == 200 {
		logger.Debug().Str("reporter", "Gotify").Msgf("Gotify message delivered")
	} else {
		logger.Error().Str("reporter", "Gotify").Msgf("Pushing gotify message failed.")
		logger.Error().Str("reporter", "Gotify").Msgf("Gotify response body: %s", string(body))
	}

}
