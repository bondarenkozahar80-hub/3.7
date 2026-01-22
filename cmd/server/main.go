package main

import (
	"fmt"
	"log"
	"os"
	"3.6/internal/database"
	"3.6/internal/handlers"
	"3.6/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Загрузка переменных окружения
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Инициализация БД
	if err := database.Init(); err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer database.Close()

	// Создание маршрутов
	router := gin.Default()

	// Настройка CORS
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Публичные маршруты
	router.POST("/api/auth/login", handlers.Login)
	router.POST("/api/auth/register", handlers.Register)

	// Защищенные маршруты
	api := router.Group("/api")
	api.Use(middleware.AuthMiddleware())
	{
		// Товары
		api.GET("/items", handlers.GetItems)
		api.POST("/items", handlers.CreateItem)
		api.PUT("/items/:id", handlers.UpdateItem)
		api.DELETE("/items/:id", handlers.DeleteItem)
		// История
		api.GET("/items/:id/history", handlers.GetItemHistory)
		api.GET("/history/:history_id/diff", handlers.GetHistoryDiff)
		// Экспорт истории (CSV)
		api.GET("/items/:id/history/export", handlers.ExportHistory)
	}

	// Статические файлы для фронтенда
	router.Static("/static", "./frontend")
	router.GET("/", func(c *gin.Context) {
		c.File("./frontend/index.html")
	})

	// Запуск сервера
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Printf("Server running on http://localhost:%s\n", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
