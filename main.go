package main

import (
	"context"
	"github.com/berduk-dev/networks/handler"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

func main() {
	connString := "postgres://admin:admin@localhost:5432/links"
	conn, err := pgx.Connect(context.Background(), connString)
	if err != nil {
		log.Fatal("Ошибка при подключении к БД", err)
	}

	r := gin.Default()

	linksHandler := handler.NewHandler(conn)

	r.POST("/shorten", linksHandler.CreateLink)
	r.POST("/shorten/:custom", linksHandler.CreateLink)
	r.GET("/:path", linksHandler.Redirect)
	r.GET("/analytics/:short_url", linksHandler.Analytics)

	r.Run()
}
