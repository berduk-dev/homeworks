package main

import (
	"context"
	"github.com/berduk-dev/networks/handler"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"log"
	"net/http"
)

func main() {
	connString := "postgres://admin:admin@localhost:5432/links"
	conn, err := pgx.Connect(context.Background(), connString)
	if err != nil {
		log.Fatal("Ошибка при подключении к БД", err)
	}

	r := gin.Default()

	linksHandler := handler.NewHandler(conn)

	// --- CORS middleware ---
	r.Use(func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin == "" || origin == "null" {
			// Разрешаем для file:// и случаев без Origin
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		} else {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Vary", "Origin")
		}
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	// Твои API-роуты:
	r.POST("/shorten", linksHandler.CreateLink)
	r.POST("/shorten/:custom", linksHandler.CreateLink) // можно убрать, если перешёл на JSON-поле "custom"
	r.GET("/:path", linksHandler.Redirect)
	r.GET("/analytics/:short_url", linksHandler.Analytics)

	r.Run("127.0.0.1:8080")
}
