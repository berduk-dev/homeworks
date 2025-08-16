package handler

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"log"
	"net/http"
	"time"
)

const HostURL = "127.0.0.1:8080/"

type CreateLinkRequest struct {
	Link string `json:"link"`
}
type Handler struct {
	db *pgx.Conn
}

type Analytics struct {
	ID        int       `json:"id"`
	LongLink  string    `json:"long_link"`
	ShortLink string    `json:"short_link"`
	UserAgent string    `json:"user_agent"`
	CreatedAt time.Time `json:"created_at"`
}

func NewHandler(db *pgx.Conn) Handler {
	return Handler{
		db,
	}
}

func (h *Handler) CreateLink(c *gin.Context) {
	var req CreateLinkRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, "У вас невалидный запрос")
		return
	}

	// Проверка на наличие длинной ссылки в БД
	var shortLink string
	row := h.db.QueryRow(c, "SELECT short_link FROM links WHERE long_link = $1", req.Link)
	err = row.Scan(&shortLink)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		log.Println("Ошибка при проверке long_link в БД: ", err)
		c.JSON(http.StatusInternalServerError, "Что-то пошло не так. Попробуйте позже!")
		return
	} else if err == nil {
		c.JSON(http.StatusOK, gin.H{
			"short": HostURL + shortLink,
			"long":  req.Link,
		})
		return
	}

	// Кастомная ссылка
	customURL := c.Param("custom")
	var shortLinkCheck string
	if customURL != "" {
		for _, r := range []rune(customURL) {
			if !((r >= 'a' && r <= 'z') ||
				(r >= 'A' && r <= 'Z') ||
				(r >= '0' && r <= '9') ||
				r == '-') {
				c.JSON(http.StatusBadRequest, "Кастомная ссылка содержит недопустимые символы.")
				return
			}
			row = h.db.QueryRow(c, "SELECT short_link FROM links WHERE short_link = $1", shortLink)
			err = row.Scan(&shortLinkCheck)
			if errors.Is(err, pgx.ErrNoRows) { // если короткая ссылка не занята
				// Добавляем в БД
				_, err = h.db.Exec(c, "INSERT INTO links (long_link, short_link) VALUES ($1, $2)", req.Link, customURL)
				if err != nil {
					c.JSON(http.StatusInternalServerError, "Произошла ошибка, попробуйте позже")
					return
				}

				c.JSON(http.StatusOK, gin.H{
					"short": HostURL + customURL,
					"long":  req.Link,
				})
				return
			} else if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"Ошибка в БД": err})
				return
			} else { // если короткая ссылка уже занята
				c.JSON(http.StatusConflict, "Такая ссылка уже существует! Попробуйте другую")
				return
			}
		}
	}

	// Генерация короткой ссылки и проверка на её наличие в БД
	for {
		b := make([]byte, 6)
		_, err := rand.Read(b)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"Внутрення ошибка": err})
		}
		shortLink = base64.URLEncoding.EncodeToString(b)[:6]

		row = h.db.QueryRow(c, "SELECT short_link FROM links WHERE short_link = $1", shortLink)
		err = row.Scan(&shortLinkCheck)
		if errors.Is(err, pgx.ErrNoRows) {
			break
		} else if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"Ошибка в БД": err})
			return
		}
	}

	// Добавляем в БД
	_, err = h.db.Exec(c, "INSERT INTO links (long_link, short_link) VALUES ($1, $2)", req.Link, shortLink)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Произошла ошибка, попробуйте позже")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"short": HostURL + shortLink,
		"long":  req.Link,
	})
}

func (h *Handler) Redirect(c *gin.Context) {
	shortLink := c.Param("path")
	var longLink string
	row := h.db.QueryRow(c, "SELECT long_link FROM links WHERE short_link = $1", shortLink)
	err := row.Scan(&longLink)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, "Ссылка не найдена!")
			return
		}
		c.JSON(http.StatusInternalServerError, "Произошла ошибка, попробуйте позже!")
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, longLink)

	_, err = h.db.Exec(c,
		"INSERT INTO redirects (long_link, short_link, user_agent) VALUES ($1, $2, $3)",
		longLink, shortLink, c.Request.UserAgent())
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Произошла ошибка. Попробуйте позже!")
		log.Println("Ошибка во время добавления в БД: ", err)
		return
	}
}

func (h *Handler) Analytics(c *gin.Context) {
	shortLink := c.Param("short_url")
	rows, err := h.db.Query(c, "SELECT id, long_link, short_link, user_agent,created_at FROM redirects WHERE short_link = $1", shortLink)

	analytics := Analytics{}
	for rows.Next() {
		err := rows.Scan(&analytics.ID, &analytics.LongLink, &analytics.ShortLink, &analytics.UserAgent, &analytics.CreatedAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, "Ошибка при выводе аналитики!")
			log.Println(err)
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"id":         analytics.ID,
			"long_link":  analytics.LongLink,
			"short_link": analytics.ShortLink,
			"user_agent": analytics.UserAgent,
			"created_at": analytics.CreatedAt,
		})
	}
	if err = rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, "Ошибка БД Query.")
		return
	}
	var count int
	err = h.db.QueryRow(c, "SELECT COUNT(*) FROM redirects WHERE short_link = $1", shortLink).Scan(&count)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(500, gin.H{"error": "database error"})
		log.Println("database error: ", err)
		return
	} else if err != nil {
		log.Println("Ошибка в выводе total_redirects: ", err)
		return
	}
	c.JSON(200, gin.H{"total_redirects": count})
}
