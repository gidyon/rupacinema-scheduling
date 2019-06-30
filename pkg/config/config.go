package config

import (
	"fmt"
	"strings"
)

// Config contains configuration variables for service
type Config struct {
	// gRPC server start parameters section
	// gRPC is TCP port to listen by gRPC server
	GRPCPort string

	// Logging section
	// LogLevel id global loge Level: Debug(-1), Info(0), Warn(1), Error(2), DPanic(3), Panic(4), Fatal(5)
	LogLevel int
	// LogTimeFormat id print time format for logger e.g 2006-01-02T15:04:05Z07:00
	LogTimeFormat string

	// Certificates and Key section
	// Path to Certificate
	TLSCertPath string
	TLSKeyPath  string

	// External services section
	// Movie service
	MovieAPIAddress  string
	MovieAPIPort     string
	MovieAPICertPath string

	// Account service
	AccountServiceAddress  string
	AccountServicePort     string
	AccountServiceCertPath string
}

// Parse validates configuration data
func (cfg *Config) Parse() error {
	if strings.Trim(cfg.GRPCPort, " ") == "" {
		return fmt.Errorf("TCP port for gRPC server is required")
	}

	return nil
}
