package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strconv"
	"sync"
	"time"
)

type ReporterError struct {
	Reporter string
	Error    error
}

func sendNotifications(timestamp time.Time, message string, title string, reporters []string) {
	// Sending messages to different services as goroutines concurrently
	// Adding a wait group here to delay execution until all functions return,
	// otherwise delaying in processEvent() would not make any sense

	var wg sync.WaitGroup
	var ReporterErrors []ReporterError

	// Buffered error channel with a buffer size of the number of enabled reporters
	errCh := make(chan ReporterError, len(reporters))

	// If there is a server tag, add it to the title
	if len(glb_arguments.ServerTag) > 0 {
		title = "[" + glb_arguments.ServerTag + "] " + title
	}

	if slices.Contains(reporters, "Pushover") {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sendPushover(timestamp, message, title, errCh)
		}()
	}

	if slices.Contains(reporters, "Gotify") {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sendGotify(message, title, errCh)
		}()
	}

	if slices.Contains(reporters, "Mail") {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sendMail(timestamp, message, title, errCh)
		}()
	}

	if slices.Contains(reporters, "Mattermost") {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sendMattermost(message, title, errCh)
		}()
	}
	wg.Wait()

	// all reporters finished, closing the error channel
	close(errCh)

	// iterate over the items in the error channel
	for err := range errCh {
		ReporterErrors = append(ReporterErrors, err)
	}

	// if some reporters failed, send notifications to all working reporters
	if len(ReporterErrors) > 0 {

		// Error if all failed
		if len(ReporterErrors) == len(reporters) {
			logger.Error().Msg("All reporters failed!")
			return
		}

		// iterate over the failed reportes and remove them from all enabled reports
		// send error notifications to remaining (working) reporters only to
		// prevent looping error notifications to non-working reporters
		for _, item := range ReporterErrors {
			reporters = removeFromSlice(reporters, item.Reporter)
		}

		for _, item := range ReporterErrors {
			err := fmt.Sprint(item.Error)
			sendNotifications(time.Now(), "Error: "+err+"\nCheck log for details", "Error: Reporter "+item.Reporter+" failed", reporters)
		}

	}
}

func removeFromSlice(slice []string, element string) []string {
	var result []string
	for _, item := range slice {
		if item != element {
			result = append(result, item)
		}
	}
	return result
}

func sendhttpMessage(reporter string, address string, messageJSON []byte) error {

	// Create request
	req, err := http.NewRequest("POST", address, bytes.NewBuffer(messageJSON))
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	if err != nil {
		logger.Error().Err(err).Str("reporter", reporter).Msg("Failed to build request")
		return errors.New("failed to build request")
	}

	// define custom httpClient with a default timeout
	var netClient = &http.Client{
		Timeout: time.Second * 10,
	}

	// Send request
	resp, err := netClient.Do(req)
	if err != nil {
		logger.Error().Err(err).Str("reporter", reporter).Msg("Failed to send request")
		return errors.New("failed to send request")
	}
	defer resp.Body.Close()

	statusCode := resp.StatusCode

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error().Err(err).Str("reporter", reporter).Msg("")
		return errors.New("reading response body failed")
	}

	// Log non successfull status codes
	if statusCode != 200 {
		logger.Error().
			Str("reporter", reporter).
			Int("status code", statusCode).
			Str("response body", string(respBody)).
			Msg("Pushing message failed")
		return errors.New("pushing message failed\nhttp status code: " + strconv.Itoa(statusCode))
	}
	logger.Debug().
		Str("reporter", reporter).
		Int("statusCode", statusCode).
		Str("responseBody", string(respBody)).
		Msg("Message delivered")
	return nil
}
