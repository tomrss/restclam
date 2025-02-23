// Package config contains configuration data structures and ways to read them.
package config

import (
	"errors"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// ServerConfig is the configuration of the server.
type ServerConfig struct {
	Host            string
	Port            int
	WriteTimeout    time.Duration
	ReadTimeout     time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

func loadServerConfig(v *viper.Viper) ServerConfig {
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.writeTimeout", 15*time.Second)
	v.SetDefault("server.readTimeout", 15*time.Second)
	v.SetDefault("server.idleTimeout", 60*time.Second)
	v.SetDefault("server.shutdownTimeout", 30*time.Second)

	return ServerConfig{
		Host:            v.GetString("server.host"),
		Port:            v.GetInt("server.port"),
		WriteTimeout:    v.GetDuration("server.writeTimeout"),
		ReadTimeout:     v.GetDuration("server.readTimeout"),
		IdleTimeout:     v.GetDuration("server.idleTimeout"),
		ShutdownTimeout: v.GetDuration("server.shutdownTimeout"),
	}
}

// LogConfig is the configuration of logging.
type LogConfig struct {
	Level       string
	JSON        bool
	Concise     bool
	LogRequests bool
}

func loadLogConfig(v *viper.Viper) LogConfig {
	v.SetDefault("log.level", "info")
	v.SetDefault("log.json", false)
	v.SetDefault("log.concise", true)
	v.SetDefault("log.logRequests", false)

	return LogConfig{
		Level:       v.GetString("log.level"),
		JSON:        v.GetBool("log.json"),
		Concise:     v.GetBool("log.concise"),
		LogRequests: v.GetBool("log.logRequests"),
	}
}

// CORSConfig is the configuration of CORS.
type CORSConfig struct {
	Enabled          bool
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

func loadCORSConfig(v *viper.Viper) CORSConfig {
	// by default disable cors as it will be probably taken care of by some network component
	// es api gateway, reverse proxy...
	v.SetDefault("cors.enabled", false)
	// defaults reflect the fact that cors will be probably enabled only in local dev
	v.SetDefault("cors.allowedOrigins", []string{"https://*", "http://*"})
	v.SetDefault("cors.allowedMethods", []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"})
	v.SetDefault("cors.allowedHeaders", []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"})
	v.SetDefault("cors.exposedHeaders", []string{"Link"})
	v.SetDefault("cors.allowCredentials", false)
	v.SetDefault("cors.maxAge", 300)

	return CORSConfig{
		Enabled:          v.GetBool("cors.enabled"),
		AllowedOrigins:   v.GetStringSlice("cors.allowedOrigins"),
		AllowedMethods:   v.GetStringSlice("cors.allowedMethods"),
		AllowedHeaders:   v.GetStringSlice("cors.allowedHeaders"),
		ExposedHeaders:   v.GetStringSlice("cors.exposedHeaders"),
		AllowCredentials: v.GetBool("cors.allowCredentials"),
		MaxAge:           v.GetInt("cors.maxAge"),
	}
}

// ClamConfig is the configuration of ClamAV.
type ClamConfig struct {
	Network              string
	Address              string
	MinWorkers           int
	MaxWorkers           int
	ConnectMaxRetries    int
	ConnectRetryInterval time.Duration
	ConnectTimeout       time.Duration
	ReadTimeout          time.Duration
	WriteTimeout         time.Duration
	StreamChunkSize      int
	HeartbeatInterval    time.Duration
}

func loadClamConfig(v *viper.Viper) ClamConfig {
	v.SetDefault("clam.network", "unix")
	v.SetDefault("clam.address", "/tmp/clamd.sock")
	v.SetDefault("clam.minWorkers", 10)
	v.SetDefault("clam.maxWorkers", 50)
	v.SetDefault("clam.connectMaxRetries", 10)
	v.SetDefault("clam.connectRetryInterval", 2*time.Second)
	v.SetDefault("clam.connectTimeout", 10*time.Second)
	v.SetDefault("clam.readTimeout", 60*time.Second)
	v.SetDefault("clam.writeTimeout", 60*time.Second)
	v.SetDefault("clam.streamChunkSize", 2048)
	v.SetDefault("clam.heartbeatInterval", 10*time.Second)

	return ClamConfig{
		Network:              v.GetString("clam.network"),
		Address:              v.GetString("clam.address"),
		MinWorkers:           v.GetInt("clam.minWorkers"),
		MaxWorkers:           v.GetInt("clam.maxWorkers"),
		ConnectMaxRetries:    v.GetInt("clam.connectMaxRetries"),
		ConnectRetryInterval: v.GetDuration("clam.connectRetryInterval"),
		ConnectTimeout:       v.GetDuration("clam.connectTimeout"),
		ReadTimeout:          v.GetDuration("clam.readTimeout"),
		WriteTimeout:         v.GetDuration("clam.writeTimeout"),
		StreamChunkSize:      v.GetInt("clam.streamChunkSize"),
		HeartbeatInterval:    v.GetDuration("clam.heartbeatInterval"),
	}
}

// FeatureFlags control switchin on/off experimental features
type FeatureFlags struct {
	//nolint:revive,stylecheck
	ApiV0 bool
	//nolint:revive,stylecheck
	ApiV1 bool
}

func loadFeatureFlags(v *viper.Viper) FeatureFlags {
	v.SetDefault("features.apiV0", "false")
	v.SetDefault("features.apiV1", "true")

	return FeatureFlags{
		ApiV0: v.GetBool("features.apiV0"),
		ApiV1: v.GetBool("features.apiV1"),
	}
}

// AppConfig is the global application configuration.
type AppConfig struct {
	Environment  string
	Server       ServerConfig
	Log          LogConfig
	Cors         CORSConfig
	Clam         ClamConfig
	FeatureFlags FeatureFlags
}

func loadAppConfig(v *viper.Viper) AppConfig {
	return AppConfig{
		Environment:  v.GetString("environment"),
		Server:       loadServerConfig(v),
		Log:          loadLogConfig(v),
		Cors:         loadCORSConfig(v),
		Clam:         loadClamConfig(v),
		FeatureFlags: loadFeatureFlags(v),
	}
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
	if err := v.ReadInConfig(); errors.As(err, &confNotFoundErr) {
		// config file not found; ignore error because properties can still be parsed from env
		return nil
	} else if err != nil {
		// config file was found but another error was produced
		return err
	}

	return nil
}

func loadConfig(configReader func(v *viper.Viper) error) (AppConfig, error) {
	v := viper.New()
	// set parsing from environment variables
	v.SetEnvPrefix("RESTCLAM")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// read config
	if err := configReader(v); err != nil {
		return AppConfig{}, err
	}

	// TODO validate config! fail fast
	return loadAppConfig(v), nil
}

// LoadConfig loads the application configuration.
// The configuration properties and the configuration defaults can be seen above.
// Every configuration property can be set by one of:
//   - put the property in the YAML config file, with every dot in the property is
//     a YAML level of object nesting (see the provided DEV config.yaml for an example).
//   - set an environment variable prefixed by "RESTCLAM_" with the name of the config
//     property uppercase with dots replaced with underscores (_).
//     Examples: RESTCLAM_DATABASE_USERNAME=admin, RESTCLAM_HELM_CHART_CACHE_ENABLED=true.
//
// The config file will be searched, in order, in this locations:
//   - /etc/restclam/config.yaml
//   - $HOME/.restclam/config.yaml
//   - ./config.yaml
func LoadConfig() (AppConfig, error) {
	return loadConfig(readFromFile)
}
