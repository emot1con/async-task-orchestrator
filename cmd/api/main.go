package main

import (
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"task_handler/internal/cache"
	"task_handler/internal/config"
	"task_handler/internal/db"
	"task_handler/internal/handler"
	"task_handler/internal/middleware"
	"task_handler/internal/observability"
	"task_handler/internal/queue"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

func main() {
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	config := config.Load()
	db := db.Init(&config.DB)

	defer func() {
		if err := db.Close(); err != nil {
			logrus.WithError(err).Fatal("Failed to close database connection")
		}
	}()

	rdb := cache.SetupRedis(&config.Redis)
	defer func() {
		if err := rdb.Close(); err != nil {
			logrus.WithError(err).Fatal("Failed to close redis connection")
		}
	}()

	conn := queue.SetupRabbitMQ(&config.RabbitMQ)
	defer func() {
		if err := conn.Close(); err != nil {
			logrus.WithError(err).Fatal("Failed to close RabbitMQ connection")
		}
	}()

	// Initialize Prometheus metrics
	observability.InitMetrics()
	logrus.Info("Metrics initialized")

	r := handler.SetupHandler(db, conn, rdb, config)

	// Add Prometheus middleware
	r.Use(middleware.PrometheusMiddleware(observability.GlobalMetrics))

	// Expose /metrics endpoint for Prometheus to scrape
	r.GET("/metrics", func(c *gin.Context) {
		promhttp.Handler().ServeHTTP(c.Writer, c.Request)
	})
	logrus.Info("Metrics endpoint exposed at /metrics")

	srv := &http.Server{
		Addr:    ":8087",
		Handler: r,
	}

	go func() {
		logrus.Info("Starting server on :8087")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.WithError(err).Fatal("Failed to start server")
		}
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logrus.Info("Shutting down server...")
}
