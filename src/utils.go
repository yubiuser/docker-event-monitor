package main

//various helper functions
import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// converts a string to a time.Time in unix fortmat
func stringToUnixTime(str string) time.Time {
	i, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		log.Fatal().Err(err).Msg("String to timestamp conversion failed")
	}
	tm := time.Unix(i, 0)
	return tm
}

// walks through a nested map and returns a unified string as in
// string = [key:value,value] [key:value] []key:value,value
func mapToString(m map[string][]string) string {
	var builder strings.Builder

	// Iterate over the map
	for key, values := range m {
		// Append key to the string
		builder.WriteString(fmt.Sprintf("[%s: ", key))

		lenght := len(values)
		i := 0
		// Append values to the string
		for _, value := range values {
			i++
			builder.WriteString(value)
			// Add comma except for the last value
			if i < lenght {
				builder.WriteString(", ")
			}
		}
		// Add a space to separate keys
		builder.WriteString("] ")
	}

	return builder.String()
}

// removes a string from a []string (case insensitive)
func removeStringFromSliceInsensitive(slice []string, element string) []string {
	var result []string
	for _, item := range slice {
		// comparing case-insensitive here
		if !strings.EqualFold(item, element) {
			result = append(result, item)
		}
	}
	return result
}
