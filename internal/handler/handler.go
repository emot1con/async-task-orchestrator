package handler

import (
	"database/sql"
	"task_handler/internal/config"
	"task_handler/internal/user"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/rabbitmq/amqp091-go"
)

func SetupHandler(db *sql.DB, conn *amqp091.Connection, redisClient *redis.Client, cfg *config.Config) *gin.Engine {

	r := gin.Default()

	// Setup user service
	userRepo := user.NewUserRepository()
	userService := user.NewUserService(userRepo, db)
	userController := user.NewUserController(userService, cfg.JWT.Secret)

	userController.SetupRoutes(r, userService, cfg.JWT.Secret)

	return r
}
