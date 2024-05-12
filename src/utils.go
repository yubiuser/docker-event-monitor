package main

//various helper functions
import (
	"encoding/json"
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
