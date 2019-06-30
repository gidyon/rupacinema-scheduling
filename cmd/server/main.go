package main

import (
	"bufio"
	"context"
	"flag"
	"github.com/Sirupsen/logrus"
	http_server "github.com/gidyon/rupacinema/scheduling/internal/protocol/http"
	"os"
	"strconv"

	"github.com/gidyon/rupacinema/scheduling/pkg/config"
)

var (
	defaultLogLevel      = 0
	defaultLogTimeFormat = "2006-01-02T15:04:05Z07:00"
)

func main() {
	var (
		cfg      = &config.Config{}
		useFlags bool
	)

	flag.BoolVar(
		&useFlags,
		"uflag", false,
		"Whether to pass config in flags",
	)

	// gRPC section
	flag.StringVar(
		&cfg.GRPCPort,
		"grpc-port", ":5600",
		"gRPC port to bind",
	)

	// Logging section
	flag.IntVar(
		&cfg.LogLevel,
		"log-level", defaultLogLevel,
		"Global log level",
	)
	flag.StringVar(
		&cfg.LogTimeFormat,
		"log-time-format", defaultLogTimeFormat,
		"Print time format for logger e.g 2006-01-02T15:04:05Z07:00",
	)

	// TLS Certificate and Private key paths for service
	flag.StringVar(
		&cfg.TLSCertPath,
		"tls-cert", "certs/cert.pem",
		"Path to TLS certificate for the service",
	)
	flag.StringVar(
		&cfg.TLSKeyPath,
		"tls-key", "certs/key.pem",
		"Path to Private key for the service",
	)

	// External Services
	// Notification Service
	flag.StringVar(
		&cfg.AccountServiceAddress,
		"account-host", "localhost",
		"Address of the account service",
	)
	flag.StringVar(
		&cfg.AccountServicePort,
		"account-port", ":5540",
		"Port where the account service is running",
	)
	flag.StringVar(
		&cfg.AccountServiceCertPath,
		"account-cert", "certs/cert.pem",
		"Path to TLS certificate for account service",
	)
	// Movie Service
	flag.StringVar(
		&cfg.MovieAPIAddress,
		"movie-host", "localhost",
		"Address of the movie service",
	)
	flag.StringVar(
		&cfg.MovieAPIPort,
		"movie-port", ":5540",
		"Port where the movie service is running",
	)
	flag.StringVar(
		&cfg.MovieAPICertPath,
		"movie-cert", "certs/cert.pem",
		"Path to TLS certificate for movie service",
	)

	flag.Parse()

	if !useFlags {
		// Get from environmnent variables
		cfg = &config.Config{
			// GRPC section
			GRPCPort: os.Getenv("GRPC_PORT"),
			// TLS certificate and private key paths
			TLSCertPath: os.Getenv("TLS_CERT_PATH"),
			TLSKeyPath:  os.Getenv("TLS_KEY_PATH"),
			// Account service
			AccountServiceAddress:  os.Getenv("ACCOUNT_ADDRESS"),
			AccountServicePort:     os.Getenv("ACCOUNT_PORT"),
			AccountServiceCertPath: os.Getenv("ACCOUNT_CERT_PATH"),
			// Movie Service
			MovieAPIAddress:  os.Getenv("MOVIE_ADDRESS"),
			MovieAPIPort:     os.Getenv("MOVIE_PORT"),
			MovieAPICertPath: os.Getenv("MOVIE_CERT_PATH"),
		}
		logLevel := os.Getenv("LOG_LEVEL")
		logTimeFormat := os.Getenv("LOG_TIME_FORMAT")

		// Log Level
		if logLevel == "" {
			cfg.LogLevel = defaultLogLevel
		} else {
			logLevelInt64, err := strconv.ParseInt(logLevel, 10, 64)
			if err != nil {
				panic(err)
			}
			cfg.LogLevel = int(logLevelInt64)
		}

		// Log Time Format
		if logTimeFormat == "" {
			cfg.LogTimeFormat = defaultLogTimeFormat
		} else {
			cfg.LogTimeFormat = logTimeFormat
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	s := bufio.NewScanner(os.Stdin)
	defer cancel()

	logrus.Infof(
		"Type %q or %q or %q or %q to stop the service",
		"kill", "KILL", "quit", "QUIT",
	)

	// Shutdown when user press q or Q
	go func() {
		for s.Scan() {
			if s.Text() == "kill" || s.Text() == "KILL" || s.Text() == "quit" || s.Text() == "QUIT" {
				cancel()
				return
			}
		}
	}()

	if err := http_server.Serve(ctx, cfg); err != nil {
		cancel()
		logrus.Fatalf("%v\n", err)
	}
}
