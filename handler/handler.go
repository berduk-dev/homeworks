package handler

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"github.com/berduk-dev/networks/cache"
	"github.com/berduk-dev/networks/repo"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"log"
	"net/http"
)

const (
	HostURL         = "127.0.0.1:8080/"
	ShortLinkLength = 6
)

type Handler struct {
	LinksRepository *repo.Repository
	LinksCache      *cache.LinksCache
}

type CreateLinkRequest struct {
	Link   string `json:"link"`
	Custom string `json:"custom"`
}

func New(linksRepo *repo.Repository, linksCache *cache.LinksCache) Handler {
	return Handler{
		LinksRepository: linksRepo,
		LinksCache:      linksCache,
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

	shortLink, err := h.LinksRepository.GetShortByLong(c, req.Link)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		log.Println("Ошибка при проверке long_link в БД: ", err)
		c.JSON(http.StatusInternalServerError, "Что-то пошло не так. Попробуйте позже!")
		return
	}
	if err == nil {
		c.JSON(http.StatusOK, gin.H{
			"short": HostURL + shortLink,
			"long":  req.Link,
		})
		return
	}

	// Кастомная ссылка

	if IsLinkValid(req.Custom) {
		for _, r := range []rune(req.Custom) {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
				c.JSON(http.StatusBadRequest, "Кастомная ссылка содержит недопустимые символы.")
				return
			}
		}

		isExists, err := h.LinksRepository.IsShortExists(c, req.Custom)
		if !isExists {
			if err != nil {
				log.Println("Произошла ошибка: ", err)
				c.JSON(http.StatusInternalServerError, "Произошла ошибка, попробуйте позже!")
				return
			}

			err = h.LinksRepository.CreateLink(c, req.Link, req.Custom)
			if err != nil {
				log.Println("Ошибка при занесении ссылок в БД: ", err)
				c.JSON(http.StatusInternalServerError, "Произошла ошибка, попробуйте позже!")
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"short": HostURL + req.Custom,
				"long":  req.Link,
			})
			return
		}
		c.JSON(http.StatusConflict, "Такая короткая ссылка уже существует! Попробуйте другую.")
		return
	}

	// Генерация короткой ссылки и проверка на её наличие в БД
	for {
		b := make([]byte, ShortLinkLength)
		_, err = rand.Read(b)
		if err != nil {
			log.Println("Ошибка при генерации короткой ссылки: ", err)
			c.JSON(http.StatusInternalServerError, "Ошибка во время генерации ссылки")
			return
		}
		shortLink = base64.URLEncoding.EncodeToString(b)[:ShortLinkLength]

		isExist, err := h.LinksRepository.IsShortExists(c, shortLink)
		if err != nil {
			c.JSON(http.StatusInternalServerError, "Произошла ошибка БД, попробуйте позже!")
			return
		}
		if !isExist {
			break
		}
	}

	// Добавляем в БД
	err = h.LinksRepository.CreateLink(c, req.Link, shortLink)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Произошла ошибка, попробуйте позже")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"short": HostURL + shortLink,
		"long":  req.Link,
	})
	return
}

func (h *Handler) Redirect(c *gin.Context) {
	shortLink := c.Param("path")

	// сначала посмотреть в кэше
	longLink, err := h.LinksCache.GetLink(shortLink)
	if err != nil {
		log.Println("error LinksCache.GetLink: ", err)
	}

	if longLink == "" {
		longLink, err = h.LinksRepository.GetLongByShort(c, shortLink)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				c.JSON(http.StatusNotFound, "Ссылка не найдена!")
				return
			}
			log.Println("GetLongByShort error: ", err)
			c.JSON(http.StatusInternalServerError, "Произошла ошибка, попробуйте позже!")
			return
		}
	}

	err = h.LinksRepository.CreateRedirect(c, longLink, shortLink, c.Request.UserAgent())
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Произошла ошибка. Попробуйте позже!")
		log.Println("error CreateAnalytics: ", err)
	}

	c.Redirect(http.StatusTemporaryRedirect, longLink)
	return
}

func (h *Handler) Analytics(c *gin.Context) {
	redirects, err := h.LinksRepository.GetRedirects(c)
	if err != nil {
		log.Println("Ошибка получения аналитики: ", err)
		c.JSON(http.StatusInternalServerError, "Ошибка при получении аналитики")
		return
	}
	c.JSON(http.StatusOK, gin.H{"redirects": redirects, "total_count": len(redirects)})
}

func IsLinkValid(customLink string) bool {
	if customLink != "" && len(customLink) == ShortLinkLength {
		return true
	}
	return false
}
