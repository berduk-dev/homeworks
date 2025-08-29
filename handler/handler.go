package handler

import (
	"errors"
	errors2 "github.com/berduk-dev/networks/errors"
	"github.com/berduk-dev/networks/manager"
	"github.com/berduk-dev/networks/service"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
)

const (
	HostURL = "127.0.0.1:8080/"
)

type Handler struct {
	linksManager manager.LinksManager
	linksService service.LinksService
}

type CreateLinkRequest struct {
	Link            string  `json:"link"`
	CustomShortLink *string `json:"customshortlink"`
}

func New(linksManager manager.LinksManager, linksService service.LinksService) Handler {
	return Handler{
		linksManager: linksManager,
		linksService: linksService,
	}
}

func (h *Handler) CreateLink(c *gin.Context) {
	// анмаршаллинг
	var req CreateLinkRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, "У вас невалидный запрос")
		return
	}

	shortLink, err := h.linksService.CreateShortLink(c, req.Link, req.CustomShortLink)
	if err != nil {
		if errors.Is(err, errors2.ErrorLinkAlreadyExists) ||
			errors.Is(err, errors2.ErrorLinkTooShort) ||
			errors.Is(err, errors2.ErrorInvalidSymbolInLink) {
			c.JSON(http.StatusBadRequest, err)
			return
		}

		log.Printf("error linksService.CreateShortLink: %v", err)
		c.JSON(http.StatusInternalServerError, "Ошибка! Попробуйте позже!")
	}

	// ответ юзеру
	c.JSON(http.StatusOK, gin.H{
		"short": HostURL + shortLink,
		"long":  req.Link,
	})
}

func (h *Handler) Redirect(c *gin.Context) {
	shortLink := c.Param("path")

	err := h.linksService.Redirect(c, shortLink)
	if err != nil {
		if errors.Is(err, errors2.ErrorLinkNotFound) {
			c.JSON(http.StatusNotFound, "link not found")
			return
		}
		log.Println("GetLongByShort error: ", err)
		c.JSON(http.StatusInternalServerError, "Произошла ошибка, попробуйте позже!")
		return
	}
}

func (h *Handler) GetAnalytics(c *gin.Context) {
	shortLink := c.Param("short_url")

	redirects, err := h.linksService.GetAnalytics(c, shortLink)
	if err != nil {
		if errors.Is(err, errors2.ErrorLinkNotFound) {
			c.JSON(http.StatusNotFound, "link not found")
			return
		}
		log.Println("GetLongByShort error: ", err)
		c.JSON(http.StatusInternalServerError, "Произошла ошибка, попробуйте позже!") // 500
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"redirects":   redirects,
		"total_count": len(redirects),
	})
}
