package main

import (
	"bytes"
	"io"
	"net/http"
	"sync"
	"time"
)

func sendNotifications(timestamp time.Time, message string, title string) {
	// Sending messages to different services as goroutines concurrently
	// Adding a wait group here to delay execution until all functions return,
	// otherwise delaying in processEvent() would not make any sense

	var wg sync.WaitGroup

	// If there is a server tag, add it to the title
	if len(glb_arguments.ServerTag) > 0 {
		title = "[" + glb_arguments.ServerTag + "] " + title
	}

	if glb_arguments.Pushover {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sendPushover(timestamp, message, title)
		}()
	}

	if glb_arguments.Gotify {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sendGotify(message, title)
		}()
	}

	if glb_arguments.Mail {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sendMail(timestamp, message, title)
		}()
	}

	if glb_arguments.Mattermost {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sendMattermost(message, title)
		}()
	}
	wg.Wait()

}

func sendhttpMessage(reporter string, address string, messageJSON []byte) {

	// Create request
	req, err := http.NewRequest("POST", address, bytes.NewBuffer(messageJSON))
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	if err != nil {
		logger.Error().Err(err).Str("reporter", reporter).Msg("Faild to build request")
		return
	}

	// define custom httpClient with a default timeout
	var netClient = &http.Client{
		Timeout: time.Second * 10,
	}

	// Send request
	resp, err := netClient.Do(req)
	if err != nil {
		logger.Error().Err(err).Str("reporter", reporter).Msg("Faild to send request")
		return
	}
	defer resp.Body.Close()

	statusCode := resp.StatusCode

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error().Err(err).Str("reporter", reporter).Msg("")
		return
	}

	// Log non successfull status codes
	if statusCode == 200 {
		logger.Debug().
			Str("reporter", reporter).
			Int("statusCode", statusCode).
			Str("responseBody", string(respBody)).
			Msg("Message delivered")
	} else {
		logger.Error().
			Str("reporter", reporter).
			Int("statusCode", statusCode).
			Str("responseBody", string(respBody)).
			Msg("Pushing message failed.")
	}
}
