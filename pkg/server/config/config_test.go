package config

import (
	"bytes"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestLoadConfigEnv(t *testing.T) {
	// prepare env
	t.Setenv("RESTCLAM_CLAM_HEARTBEATINTERVAL", "12s")
	t.Setenv("RESTCLAM_LOG_LEVEL", "warn")
	t.Setenv("RESTCLAM_SERVER_PORT", "9090")

	// execute
	config, err := loadConfig(readNop, readNop)
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}

	// assert
	assert.Equal(t, 12*time.Second, config.Clam.HeartbeatInterval, "Clam heartbeat interval")
	assert.Equal(t, "warn", config.Log.Level, "Log level from env")
	assert.Equal(t, 9090, config.Server.Port, "Server port from env")
}

func TestLoadConfigDefaults(t *testing.T) {
	// no preparation, load embedded defaults

	// execute
	config, err := loadConfig(readDefaults, readNop)
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}

	// assert
	assert.Equal(t, "unix", config.Clam.Network, "Clam network")
	assert.Equal(t, "debug", config.Log.Level, "Log level from default")
	assert.Equal(t, 8080, config.Server.Port, "Server port from default")
}

func TestLoadConfigFile(t *testing.T) {
	// prepare
	configContent := `
clam:
  network: unix
  address: /run/clamav/clamd.sock
log:
  level: error
server:
  port: 7070
`
	readFunc := func(v *viper.Viper) error {
		return readFromFileMock(configContent, v)
	}

	// execute
	config, err := loadConfig(readNop, readFunc)
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}

	// assert
	assert.Equal(t, "/run/clamav/clamd.sock", config.Clam.Address, "Clamd address")
	assert.Equal(t, "error", config.Log.Level, "Log level from default")
	assert.Equal(t, 7070, config.Server.Port, "Server port from default")
}

func TestUnmarshal(t *testing.T) {
	// prepare
	t.Setenv("RESTCLAM_CLAM_HEARTBEATINTERVAL", "12s")
	t.Setenv("RESTCLAM_LOG_LEVEL", "warn")
	t.Setenv("RESTCLAM_SERVER_HOST", "127.0.0.1")
	t.Setenv("RESTCLAM_ENVIRONMENT", "test")

	readFunc := func(v *viper.Viper) error {
		return readFromFileMock(`
clam:
  network: unix
  address: /run/clamav/clamd.sock
log:
  level: error
server:
  port: 7070
`, v)
	}
	// execute
	config, err := loadConfig(readNop, readFunc)
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}

	// assert
	assert.Equal(t, "test", config.Environment, "Environment")
	assert.Equal(t, "unix", config.Clam.Network, "Clamd network")
	assert.Equal(t, "/run/clamav/clamd.sock", config.Clam.Address, "Clamd address")
	assert.Equal(t, "warn", config.Log.Level, "Log level")
	assert.Equal(t, "127.0.0.1", config.Server.Host, "Server host")
	assert.Equal(t, 7070, config.Server.Port, "Server port")
	assert.Equal(t, 12 * time.Second, config.Clam.HeartbeatInterval, "Heartbeat interval")
}

// helpers

func readNop(v *viper.Viper) error {
	// do nothing
	return nil
}

func readFromFileMock(mockFile string, v *viper.Viper) error {
	v.SetConfigType("yaml")
	return v.ReadConfig(bytes.NewBuffer([]byte(mockFile)))
}
