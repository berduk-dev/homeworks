package repo

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"log"
	"net/http"
	"time"
)

type Repository struct {
	db *pgx.Conn
}

func NewRepo(db *pgx.Conn) Repository {
	return Repository{db: db}
}

type Redirect struct {
	ID        int       `json:"id"`
	LongLink  string    `json:"long_link"`
	ShortLink string    `json:"short_link"`
	UserAgent string    `json:"user_agent"`
	CreatedAt time.Time `json:"created_at"`
}

func (r *Repository) CreateLink(c *gin.Context, longLink string, shortLink string) error {
	_, err := r.db.Exec(c, "INSERT INTO links (long_link, short_link) VALUES ($1, $2)", longLink, shortLink)
	if err != nil {
		return err
	}

	return nil
}

func (r *Repository) CreateRedirect(c *gin.Context, longLink, shortLink, userAgent string) error {
	_, err := r.db.Exec(c,
		"INSERT INTO redirects (long_link, short_link, user_agent) VALUES ($1, $2, $3)",
		longLink, shortLink, userAgent)
	if err != nil {
		return err
	}
	return nil
}

func (r *Repository) GetLongByShort(c *gin.Context, shortLink string) (string, error) {
	var longLink string
	err := r.db.QueryRow(c, "SELECT long_link FROM links WHERE short_link = $1", shortLink).Scan(&longLink)
	if err != nil {
		return "", err
	}
	return longLink, nil
}

func (r *Repository) GetShortByLong(c *gin.Context, longLink string) (string, error) {
	var shortLink string
	err := r.db.QueryRow(c, "SELECT short_link FROM links WHERE long_link = $1", longLink).Scan(&shortLink)
	if err != nil {
		return "", err
	}
	return shortLink, nil
}

func (r *Repository) GetRedirects(c *gin.Context) ([]Redirect, error) {
	shortLink := c.Param("short_url")
	rows, err := r.db.Query(c, "SELECT id, long_link, short_link, user_agent, created_at FROM redirects WHERE short_link = $1", shortLink)

	var redirects []Redirect
	for rows.Next() {

		var redirect Redirect
		err = rows.Scan(&redirect.ID, &redirect.LongLink, &redirect.ShortLink, &redirect.UserAgent, &redirect.CreatedAt)

		if err != nil {
			c.JSON(http.StatusInternalServerError, "Ошибка при выводе аналитики!")
			log.Println("Ошибка при выводе аналитики: ", err)
			return nil, err
		}
		redirects = append(redirects, redirect)
	}

	if err = rows.Err(); err != nil {
		log.Println("Ошибка БД Query: ", err)
		c.JSON(http.StatusInternalServerError, "Произошла ошибка! Попробуйте позже!")
		return nil, err
	}
	return redirects, nil
}

func (r *Repository) IsShortExists(c *gin.Context, shortLink string) (bool, error) {
	var existingShortLink string
	err := r.db.QueryRow(c, "SELECT short_link FROM links WHERE short_link = $1", shortLink).Scan(&existingShortLink)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
