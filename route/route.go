
package route

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func SetupRoutes(
	app *fiber.App,
	pgPool *pgxpool.Pool,
	mongoClient *mongo.Client,
) {
	api := app.Group("/api/v1")

	api.Get("/health", func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		postgresStatus := "connected"
		mongoStatus := "connected"
		httpStatus := fiber.StatusOK

		if pgPool == nil || pgPool.Ping(ctx) != nil {
			postgresStatus = "disconnected"
			httpStatus = fiber.StatusServiceUnavailable
		}

		if mongoClient == nil || mongoClient.Ping(ctx, nil) != nil {
			mongoStatus = "disconnected"
			httpStatus = fiber.StatusServiceUnavailable
		}

		return c.Status(httpStatus).JSON(fiber.Map{
			"success": httpStatus == fiber.StatusOK,
			"message": "BakeOrder API health check",
			"data": fiber.Map{
				"postgres": postgresStatus,
				"mongodb":  mongoStatus,
			},
		})
	})
}