
package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"bakeorder/config"
	"bakeorder/database"
	"bakeorder/route"

	"github.com/gofiber/fiber/v2"
)

func main() {
	if err := run(); err != nil {
		slog.Error("Aplikasi gagal dijalankan", "error", err)
		os.Exit(1)
	}
}

func run() error {
	roleFlag := flag.String("role", "", "Mode aplikasi: api, worker, atau all")
	flag.Parse()

	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	if *roleFlag != "" {
		cfg.AppRole = *roleFlag
	}

	if cfg.AppRole != "api" && cfg.AppRole != "worker" && cfg.AppRole != "all" {
		return fiber.NewError(fiber.StatusBadRequest, "Role harus api, worker, atau all")
	}

	logger := config.NewLogger(cfg.LogLevel)
	slog.SetDefault(logger)

	logger.Info("Memulai aplikasi BakeOrder", "role", cfg.AppRole)

	if cfg.AppRole == "worker" {
		logger.Warn("Worker belum diaktifkan; implementasi worker akan ditambahkan pada tahap berikutnya")
		return nil
	}

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	dbCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	pgPool, err := database.NewPostgresPool(dbCtx, cfg)
	if err != nil {
		return err
	}
	defer pgPool.Close()

	mongoClient, err := database.NewMongoClient(dbCtx, cfg)
	if err != nil {
		return err
	}
	defer func() {
		disconnectCtx, disconnectCancel := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		defer disconnectCancel()

		if err := mongoClient.Disconnect(disconnectCtx); err != nil {
			logger.Error("Gagal memutus koneksi MongoDB", "error", err)
		}
	}()

	app := fiber.New(fiber.Config{
		AppName: "BakeOrder API",
	})

	route.SetupRoutes(app, pgPool, mongoClient)

	go func() {
		<-ctx.Done()
		logger.Info("Mematikan server BakeOrder")

		shutdownCtx, shutdownCancel := context.WithTimeout(
			context.Background(),
			10*time.Second,
		)
		defer shutdownCancel()

		if err := app.ShutdownWithContext(shutdownCtx); err != nil {
			logger.Error("Gagal mematikan server dengan baik", "error", err)
		}
	}()

	logger.Info("Server berjalan", "port", cfg.AppPort)

	if err := app.Listen(":" + cfg.AppPort); err != nil {
		if ctx.Err() != nil {
			return nil
		}
		return err
	}

	return nil
}