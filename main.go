package main

import (
	"context"
	"fmt"
	"github.com/berduk-dev/networks/cache"
	"github.com/berduk-dev/networks/handler"
	"github.com/berduk-dev/networks/manager"
	"github.com/berduk-dev/networks/repo"
	"github.com/berduk-dev/networks/service"
	"time"

	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
	"github.com/jackc/pgx/v5"
)

const (
	cacheLinksInterval = time.Hour
	popularLinksCount  = 10
)

func main() {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	// проверяем соединение
	_, err := rdb.Ping().Result()
	if err != nil {
		log.Fatalf("Ошибка подключения к Redis: %v", err)
	}

	connString := "postgres://admin:admin@localhost:5432/links"
	conn, err := pgx.Connect(context.Background(), connString)
	if err != nil {
		log.Fatal("Ошибка при подключении к БД", err)
	}

	r := gin.Default()

	linksRepository := repo.New(conn)
	linksCache := cache.New(rdb)
	linksManager := manager.New(linksCache, &linksRepository)
	linksService := service.New(linksManager)
	linksHandler := handler.New(linksManager, *linksService)

	go func() { // TODO: Вынести из main.go в другое место
		err := cachePopularLinks(&linksRepository, linksCache)
		if err != nil {
			log.Println("error cachePopularLinks:", err)
		}

		c := time.Tick(cacheLinksInterval)
		for range c {
			err := cachePopularLinks(&linksRepository, linksCache)
			if err != nil {
				log.Println("error cachePopularLinks:", err)
			}
		}
	}()

	// --- CORS middleware ----
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

	// API-роуты:
	r.POST("/shorten", linksHandler.CreateLink)
	r.POST("/shorten/:custom", linksHandler.CreateLink) // можно убрать, если перешёл на JSON-поле "custom"
	r.GET("/analytics/:short_url", linksHandler.GetAnalytics)
	r.GET("/:path", linksHandler.Redirect)

	r.Run()
}

func cachePopularLinks(linksRepository *repo.Repository, linksCache *cache.LinksCache) error {
	links, err := linksRepository.GetPopularLinks(context.Background(), popularLinksCount)
	if err != nil {
		return fmt.Errorf("error updateCache GetPopularLinks: %w", err)
	}

	for _, link := range links {
		err := linksCache.StoreLink(link.Short, link.Long)
		if err != nil {
			return fmt.Errorf("error updateCache StoreLink: %w", err)
		}
	}
	return nil
}
