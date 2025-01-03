package main

import (
	"context"
	_ "embed"
	"fmt"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"log"
	"os"
	"os/signal"
	"scope3apiproxy/api"
	"scope3apiproxy/internal"
	"scope3apiproxy/internal/cache"
	v2 "scope3apiproxy/internal/scope3/v2"
	"strings"
	"syscall"
	"time"
)

//go:embed config.json
var configFile []byte

func main() {
	log.Println("Scope3 API application starting...")
	// Create logger instance
	environment := os.Getenv("ENVIRONMENT")
	if environment == "" {
		environment = "local"
	}
	var (
		logger *zap.Logger
		err    error
	)
	if environment != "local" {
		logger, err = zap.NewProduction()
	} else {
		logger, err = zap.NewDevelopment()
	}

	if err != nil {
		log.Fatalf("can't initialize zap logger: %v", err)
	}
	logger = logger.With(zap.String("environment", environment))
	defer logger.Sync()

	initializeViper(logger, environment)

	scope3APIClient := v2.NewScope3APIClient(v2.Scope3APIClientConfig{
		Host:               viper.GetString("scope3.host"),
		ApiKey:             viper.GetString("scope3.apiKey"),
		Timeout:            time.Duration(viper.GetInt("scope3.timeoutInSeconds")) * time.Second,
		MaxIdleConnections: viper.GetInt("scope3.maxIdleConnections"),
		IdleConnTimeout:    time.Duration(viper.GetInt("scope3.idleConnTimeoutInSeconds")) * time.Second,
	})

	appCache := cache.NewCache(viper.GetInt("cache.capacity"))

	emissionService := internal.NewEmissionService(
		logger,
		scope3APIClient,
		appCache,
		time.Duration(viper.GetInt("cache.emissionTtlInMinutes"))*time.Minute,
	)

	server := api.NewAPIServer(viper.GetInt("port"), logger, emissionService)
	// Rune the server in a goroutine so that it won't block the graceful shutdown handling below
	go func() {
		server.Run()
	}()

	// Graceful shutdown of the server on kill or CTRL+C
	gracefulStop := make(chan os.Signal, 1)
	// kill -9 is syscall.SIGKILL but can't be caught, so it is not added
	signal.Notify(gracefulStop,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	gracefulShutdownTimeout := time.Duration(viper.GetInt("gracefulShutdownTimeoutInSeconds")) * time.Second
	sig := <-gracefulStop
	logger.Debug(fmt.Sprintf("Caught sig: %+v", sig))

	apiServerShutdownDown := make(chan bool, 1)
	ctx, cancel := context.WithTimeout(context.Background(), gracefulShutdownTimeout)
	defer cancel()

	server.Shutdown(ctx, apiServerShutdownDown)

	select {
	case <-apiServerShutdownDown:
	case <-ctx.Done():
		logger.Warn("HTTP APIServer did not shutdown after " + gracefulShutdownTimeout.String())
	}

	log.Println("Scope3 API application exited.")
}

func initializeViper(logger *zap.Logger, environment string) {
	jsonConfigFile := "config.json"
	if _, err := os.Stat("config." + environment + ".json"); err == nil {
		jsonConfigFile = "config." + environment + ".json"
	}
	logger.Debug("Loading application config from " + jsonConfigFile)
	viper.SetConfigFile(jsonConfigFile)
	viper.SetConfigType("json")
	if err := viper.ReadInConfig(); err != nil {
		logger.Warn("Error application config from "+jsonConfigFile, zap.Error(err))
	}
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)
	viper.AutomaticEnv()
}
