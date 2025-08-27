package manager

import (
	"context"
	"errors"
	"github.com/berduk-dev/networks/cache"
	"github.com/berduk-dev/networks/repo"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"log"
	"net/http"
)

type LinksManager struct {
	LinksRepo  *repo.Repository
	LinksCache *cache.LinksCache
}

func New(linksRepo *repo.Repository, linksCache *cache.LinksCache) LinksManager {
	return LinksManager{
		LinksRepo:  linksRepo,
		LinksCache: linksCache,
	}
}

func (m *LinksManager) CreateLink(c *gin.Context, longLink string, shortLink string) error {
	return m.LinksRepo.CreateLink(c, longLink, shortLink)
}

func (m *LinksManager) CreateRedirect(c *gin.Context, longLink, shortLink, userAgent string) error {
	return m.LinksRepo.CreateRedirect(c, longLink, shortLink, userAgent)
}

func (m *LinksManager) GetLongByShort(c *gin.Context, shortLink string) (string, error) {
	return m.LinksRepo.GetLongByShort(c, shortLink)
}

func (m *LinksManager) GetShortByLong(c *gin.Context, longLink string) (string, error) {
	return m.LinksRepo.GetShortByLong(c, longLink)
}

func (m *LinksManager) GetCacheLongLink(shortLink string) (string, error) {
	return m.LinksCache.GetLink(shortLink)
}

func (m *LinksManager) GetPopularLinks(ctx context.Context, n int) ([]repo.LinkPair, error) {
	return m.LinksRepo.GetPopularLinks(ctx, n)
}

func (m *LinksManager) Redirect(c *gin.Context, shortLink string) (string, error) {

	// сначала посмотреть в кэше
	longLink, err := m.GetCacheLongLink(shortLink)
	if err != nil {
		log.Println("error LinksCache.GetLink: ", err)
	}

	if longLink == "" {
		longLink, err = m.GetLongByShort(c, shortLink)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				c.JSON(http.StatusNotFound, "Ссылка не найдена!")
				return "", pgx.ErrNoRows
			}
			log.Println("GetLongByShort error: ", err)
			c.JSON(http.StatusInternalServerError, "Произошла ошибка, попробуйте позже!")
			return "", err
		}
	}

	err = m.CreateRedirect(c, longLink, shortLink, c.Request.UserAgent())
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Произошла ошибка. Попробуйте позже!")
		log.Println("error CreateAnalytics: ", err)
	}

	c.Redirect(http.StatusTemporaryRedirect, longLink)
	return longLink, nil
}
