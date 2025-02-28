// Package config contains configuration data structures and ways to read them.
package config

import (
	"bytes"
	_ "embed"
	"errors"
	"strings"
	"time"

	// TODO consider dropping viper and write it manually
	"github.com/spf13/viper"
)

//go:embed defaults.yaml
var configDefaults []byte

// ServerConfig is the configuration of the server.
type ServerConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	WriteTimeout    time.Duration `mapstructure:"writeTimeout"`
	ReadTimeout     time.Duration `mapstructure:"readTimeout"`
	IdleTimeout     time.Duration `mapstructure:"idleTimeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdownTimeout"`
}

// LogConfig is the configuration of logging.
type LogConfig struct {
	Level       string `mapstructure:"level"`
	JSON        bool   `mapstructure:"jSON"`
	Concise     bool   `mapstructure:"concise"`
	LogRequests bool   `mapstructure:"logRequests"`
}

// CORSConfig is the configuration of CORS.
type CORSConfig struct {
	Enabled          bool     `mapstructure:"enabled"`
	AllowedOrigins   []string `mapstructure:"allowedOrigins"`
	AllowedMethods   []string `mapstructure:"allowedMethods"`
	AllowedHeaders   []string `mapstructure:"allowedHeaders"`
	ExposedHeaders   []string `mapstructure:"exposedHeaders"`
	AllowCredentials bool     `mapstructure:"allowCredentials"`
	MaxAge           int      `mapstructure:"maxAge"`
}

// ClamConfig is the configuration of ClamAV.
type ClamConfig struct {
	Network              string        `mapstructure:"network"`
	Address              string        `mapstructure:"address"`
	MinWorkers           int           `mapstructure:"minWorkers"`
	MaxWorkers           int           `mapstructure:"maxWorkers"`
	ConnectMaxRetries    int           `mapstructure:"connectMaxRetries"`
	ConnectRetryInterval time.Duration `mapstructure:"connectRetryInterval"`
	ConnectTimeout       time.Duration `mapstructure:"connectTimeout"`
	ReadTimeout          time.Duration `mapstructure:"readTimeout"`
	WriteTimeout         time.Duration `mapstructure:"writeTimeout"`
	StreamChunkSize      int           `mapstructure:"streamChunkSize"`
	HeartbeatInterval    time.Duration `mapstructure:"heartbeatInterval"`
}

// FeatureFlags control switchin on/off experimental features
type FeatureFlags struct {
	//nolint:revive,stylecheck
	ApiV0 bool `mapstructure:"apiV0"`
	//nolint:revive,stylecheck
	ApiV1 bool `mapstructure:"apiV1"`
}

// AppConfig is the global application configuration.
type AppConfig struct {
	Environment  string       `mapstructure:"environment"`
	Server       ServerConfig `mapstructure:"server"`
	Log          LogConfig    `mapstructure:"log"`
	Cors         CORSConfig   `mapstructure:"cors"`
	Clam         ClamConfig   `mapstructure:"clam"`
	FeatureFlags FeatureFlags `mapstructure:"featureFlags"`
}

type configReader func(v *viper.Viper) error

func readDefaults(v *viper.Viper) error {
	v.SetConfigType("yaml")
	// read embedded defaults file
	return v.ReadConfig(bytes.NewReader(configDefaults))
}

func readFromFile(v *viper.Viper) error {
	// set file properties
	v.SetConfigType("yaml")
	v.SetConfigName("config")
	// path in which to search config.yaml, the first found wins
	v.AddConfigPath("/etc/restclam/")
	v.AddConfigPath("$HOME/.restclam")
	v.AddConfigPath(".")

	// read config
	var confNotFoundErr viper.ConfigFileNotFoundError
	if err := v.MergeInConfig(); errors.As(err, &confNotFoundErr) {
		// config file not found; ignore error because properties can still be parsed from env
		return nil
	} else if err != nil {
		// config file was found but another error was produced
		return err
	}

	return nil
}

func loadConfig(defaultsReader configReader, configReader configReader) (AppConfig, error) {
	v := viper.NewWithOptions(viper.ExperimentalBindStruct())

	// load defaults
	if err := defaultsReader(v); err != nil {
		return AppConfig{}, err
	}

	// load file config overrides
	if err := configReader(v); err != nil {
		return AppConfig{}, err
	}

	// load environment variables overrides
	v.SetEnvPrefix("RESTCLAM")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// unmarshal config into config struct
	var c AppConfig
	if err := v.Unmarshal(&c); err != nil {
		return AppConfig{}, err
	}

	// TODO validate config! fail fast
	return c, nil
}

// LoadConfig loads the application configuration.
// The configuration properties and the configuration defaults can be seen above.
// Every configuration property can be set by one of:
//   - put the property in the YAML config file, with every dot in the property is
//     a YAML level of object nesting (see the provided DEV config.yaml for an example).
//   - set an environment variable prefixed by "RESTCLAM_" with the name of the config
//     property uppercase with dots replaced with underscores (_).
//     Examples: RESTCLAM_SERVER_PORT=8080, RESTCLAM_CLAM_MINWORKERS=5.
//
// The config file will be searched, in order, in this locations:
//   - /etc/restclam/config.yaml
//   - $HOME/.restclam/config.yaml
//   - ./config.yaml
func LoadConfig() (AppConfig, error) {
	return loadConfig(readDefaults, readFromFile)
}
