package main

type pushover struct {
	Enabled  bool
	APIToken string `yaml:"api_token"`
	UserKey  string `yaml:"user_key"`
}
type gotify struct {
	Enabled bool
	URL     string `yaml:"url"`
	Token   string `yaml:"token"`
}
type mail struct {
	Enabled  bool
	From     string `yaml:"from"`
	To       string `yaml:"to"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Port     int    `yaml:"port"`
	Host     string `yaml:"host"`
}
type mattermost struct {
	Enabled bool
	URL     string `yaml:"url"`
	Channel string `yaml:"channel"`
	User    string `yaml:"user"`
}

type reporter struct {
	Pushover   pushover
	Gotify     gotify
	Mail       mail
	Mattermost mattermost
}

type options struct {
	Filter         map[string][]string `yaml:"filter"`
	LogLevel       string              `yaml:"log_level"`
	ServerTag      string              `yaml:"server_tag"`
}

type notification struct {
	Name   string              `yaml:"name"`
	Event  map[string][]string `yaml:"event"`
	Notify []string            `yaml:"notify"`
}

type Config struct {
	Reporter        reporter
	Options         options
	EnabledReporter []string            `yaml:"-"`
	Notifications   []notification      `yaml:"notifications"`
}
