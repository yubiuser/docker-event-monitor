package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"gopkg.in/yaml.v3"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// hold config options and settings globally
var config Config

// should we only print version information and exit
var showVersion bool

// config file path
var configFilePath string

var PrintUsage = func() {
	fmt.Printf("Usage of %s", os.Args[0])
	fmt.Print(
		`
-v, --version		prints version information
-c, --config [path]	config file path (default "config.yml")
-h, --help		prints help information
`)
}

func init() {
	flag.BoolVar(&showVersion, "v", false, "print version information")
	flag.BoolVar(&showVersion, "version", false, "print version information")
	flag.StringVar(&configFilePath, "c", "config.yml", "config file path")
	flag.StringVar(&configFilePath, "config", "config.yml", "config file path")
	flag.Usage = PrintUsage
	flag.Parse()

	configureLogger()
	loadConfig()

	// after loading the config, we we migth increase log level
	if config.Options.LogLevel == "debug" {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	parseArgs()

	if config.Reporter.Pushover.Enabled {
		if len(config.Reporter.Pushover.APIToken) == 0 {
			log.Fatal().Msg("Pushover Enabled. Pushover API token required!")
		}
		if len(config.Reporter.Pushover.UserKey) == 0 {
			log.Fatal().Msg("Pushover Enabled. Pushover user key required!")
		}
	}
	if config.Reporter.Gotify.Enabled {
		if len(config.Reporter.Gotify.URL) == 0 {
			log.Fatal().Msg("Gotify Enabled. Gotify URL required!")
		}
		if len(config.Reporter.Gotify.Token) == 0 {
			log.Fatal().Msg("Gotify Enabled. Gotify APP token required!")
		}
	}
	if config.Reporter.Mail.Enabled {
		if len(config.Reporter.Mail.User) == 0 {
			log.Fatal().Msg("E-Mail notification Enabled. SMTP username required!")
		}
		if len(config.Reporter.Mail.To) == 0 {
			log.Fatal().Msg("E-Mail notification Enabled. Recipient address required!")
		}
		if len(config.Reporter.Mail.From) == 0 {
			config.Reporter.Mail.From = config.Reporter.Mail.User
		}
		if len(config.Reporter.Mail.Password) == 0 {
			log.Fatal().Msg("E-Mail notification Enabled. SMTP Password required!")
		}
		if len(config.Reporter.Mail.Host) == 0 {
			log.Fatal().Msg("E-Mail notification Enabled. SMTP host address required!")
		}
	}
	if config.Reporter.Mattermost.Enabled {
		if len(config.Reporter.Mattermost.URL) == 0 {
			log.Fatal().Msg("Mattermost Enabled. Mattermost URL required!")
		}
	}
}

func loadConfig() {
	configFile, err := filepath.Abs(configFilePath)

	if err != nil {
		log.Fatal().Err(err).Msg("Failed to set config file path")
	}

	buf, err := os.ReadFile(configFile)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to read config file")
	}

	err = yaml.Unmarshal(buf, &config)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to parse config file")
	}
}

func parseArgs() {

	// Parse (include) filters
	config.Filter = make(map[string][]string)

	for _, filter := range config.Options.FilterStrings {
		pos := strings.Index(filter, "=")
		if pos == -1 {
			log.Fatal().Msg("each filter should be of the form key=value")
		}
		key := filter[:pos]
		val := filter[pos+1:]
		config.Filter[key] = append(config.Filter[key], val)
	}

	// Parse exclude filters
	config.Exclude = make(map[string][]string)

	for _, exclude := range config.Options.ExcludeStrings {
		pos := strings.Index(exclude, "=")
		if pos == -1 {
			log.Fatal().Msg("each filter should be of the form key=value")
		}
		//trim whitespaces
		key := strings.TrimSpace(exclude[:pos])
		val := exclude[pos+1:]
		config.Exclude[key] = append(config.Exclude[key], val)
	}

	//Parse Enabled reportes

	if config.Reporter.Gotify.Enabled {
		config.EnabledReporter = append(config.EnabledReporter, "Gotify")
	}
	if config.Reporter.Mattermost.Enabled {
		config.EnabledReporter = append(config.EnabledReporter, "Mattermost")
	}
	if config.Reporter.Pushover.Enabled {
		config.EnabledReporter = append(config.EnabledReporter, "Pushover")
	}
	if config.Reporter.Mail.Enabled {
		config.EnabledReporter = append(config.EnabledReporter, "Mail")
	}

}

func configureLogger() {

	// Configure time/timestamp format
	zerolog.TimeFieldFormat = time.RFC1123Z

	// Default level is info
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	// Add timestamp and service string
	log.Logger = log.With().Timestamp().Str("service", "docker event monitor").Logger()
}

func main() {
	// if the -v flag was set, print version information and exit
	if showVersion {
		printVersion()
	}

	// log all supplied arguments
	logArguments()

	timestamp := time.Now()
	startup_message := buildStartupMessage(timestamp)
	sendNotifications(timestamp, startup_message, "Starting docker event monitor", config.EnabledReporter)

	filterArgs := filters.NewArgs()
	for key, values := range config.Filter {
		for _, value := range values {
			filterArgs.Add(key, value)
		}
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create new docker client")
	}
	defer cli.Close()

	// receives events from the channel
	event_chan, errs := cli.Events(context.Background(), events.ListOptions{Filters: filterArgs})

	for {
		select {
		case err := <-errs:
			log.Fatal().Err(err).Msg("")
		case event := <-event_chan:
			// if logging level is debug, log the event
			log.Debug().
				Interface("event", event).Msg("")

			// Check if event should be exlcuded from reporting
			if len(config.Exclude) > 0 {
				log.Debug().Msg("Performing check for event exclusion")
				if excludeEvent(event) {
					break //breaks out of the select and waits for the next event to arrive
				}
			}
			// Check if event is from a crash_only container
			if isCrashOnlyEvent(event) {
				break
			}
			processEvent(event)
		}
	}
}
