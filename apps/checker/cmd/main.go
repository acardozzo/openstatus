package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/openstatushq/openstatus/apps/checker/handlers"

	"github.com/openstatushq/openstatus/apps/checker/pkg/tinybird"
	"github.com/rs/zerolog/log"
)

func main() {
	log.Info().Msg("Starting OpenStatus Checker")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-done
		log.Info().Msg("Received shutdown signal")
		cancel()
	}()

	// environment variables.
	log.Info().Msg("Loading environment variables")
	flyRegion := env("FLY_REGION", env("REGION", "local"))
	cronSecret := env("CRON_SECRET", "")
	tinyBirdToken := env("TINYBIRD_TOKEN", "")
	logLevel := env("LOG_LEVEL", "warn")
	cloudProvider := env("CLOUD_PROVIDER", "fly")

	log.Info().Msgf("Environment variables loaded: FLY_REGION=%s, CRON_SECRET=%s, TINYBIRD_TOKEN=%s, LOG_LEVEL=%s, CLOUD_PROVIDER=%s", flyRegion, cronSecret, tinyBirdToken, logLevel, cloudProvider)

	// logger.Configure(logLevel)

	// packages.
	log.Info().Msg("Configuring HTTP client")
	httpClient := &http.Client{
		Timeout: 45 * time.Second,
	}

	defer httpClient.CloseIdleConnections()

	log.Info().Msg("Creating Tinybird client")
	tinybirdClient := tinybird.NewClient(httpClient, tinyBirdToken)

	h := &handlers.Handler{
		Secret:        cronSecret,
		CloudProvider: cloudProvider,
		Region:        flyRegion,
		TbClient:      tinybirdClient,
	}

	log.Info().Msg("Setting up router")
	router := gin.New()
	router.POST("/checker", h.HTTPCheckerHandler)
	router.POST("/checker/http", h.HTTPCheckerHandler)
	router.POST("/checker/tcp", h.TCPHandler)
	router.POST("/ping/:region", h.PingRegionHandler)
	router.POST("/tcp/:region", h.TCPHandlerRegion)

	router.GET("/health", func(c *gin.Context) {
		log.Info().Msg("Health check")
		c.JSON(http.StatusOK, gin.H{"message": "pong", "fly_region": flyRegion})
	})

	httpServer := &http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%s", env("PORT", "8080")),
		Handler: router,
	}

	go func() {
		log.Ctx(ctx).Info().Msgf("Server is running on %s", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Ctx(ctx).Error().Err(err).Msg("Failed to start HTTP server")
			cancel()
		}
	}()

	<-ctx.Done()
	log.Info().Msg("Context done, shutting down server")
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Failed to shutdown HTTP server")
		return
	}
	log.Info().Msg("Server gracefully stopped")
}

func env(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
