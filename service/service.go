package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	errors2 "github.com/berduk-dev/networks/errors"
	"github.com/berduk-dev/networks/manager"
	"github.com/berduk-dev/networks/repo"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"log"
	"net/http"
	"unicode"
)

const ShortLinkLength = 6

type LinksService struct {
	linksManager manager.LinksManager
}

func New(linksManager manager.LinksManager) *LinksService {
	return &LinksService{
		linksManager: linksManager,
	}
}

func (s *LinksService) CreateShortLink(ctx context.Context, longLink string, customShortLink *string) (string, error) {
	// проверка на наличие длинной ссылки в БД
	existingShortLink, err := s.linksManager.GetShortByLong(ctx, longLink)
	if err == nil {
		return existingShortLink, nil
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("linksManager.GetShortByLong: %w", err) // 500
	}

	// создание по кастомной ссылке
	if customShortLink != nil {
		err := validateShortLink(*customShortLink)
		if err != nil {
			return "", err // 400
		}

		isExists, err := s.linksManager.IsShortExists(ctx, *customShortLink)
		if err != nil {
			return "", fmt.Errorf("linksManager.IsShortExists: %w", err) // 500
		}

		if isExists {
			return "", errors2.ErrorLinkAlreadyExists // 400
		}

		err = s.linksManager.CreateLink(ctx, longLink, *customShortLink)
		if err != nil {
			return "", fmt.Errorf("linksManager.CreateLink: %w", err) // 500
		}

		return *customShortLink, nil
	}

	// генерация случайной короткой ссылки и проверка на её наличие в БД
	shortLink := ""
	for {
		b := make([]byte, ShortLinkLength)
		_, err = rand.Read(b)
		if err != nil {
			log.Println("Ошибка при генерации короткой ссылки: ", err)
		}
		shortLink = base64.URLEncoding.EncodeToString(b)[:ShortLinkLength]

		isExist, err := s.linksManager.IsShortExists(ctx, shortLink)
		if err != nil {
			return "", fmt.Errorf("linksManager.IsShortExists: %w", err) // 500
		}

		if !isExist {
			break
		}
	}

	// добавляем в БД которкую ссылку
	err = s.linksManager.CreateLink(ctx, longLink, shortLink)
	if err != nil {
		return "", fmt.Errorf("linksManager.CreateLink: %w", err) // 500
	}

	return shortLink, nil
}

func (s *LinksService) Redirect(c *gin.Context, shortLink string) error {

	longLink, err := s.linksManager.GetLongByShort(c, shortLink)
	if err != nil {
		if errors.Is(err, errors.New("error link not found")) {
			return errors2.ErrorLinkNotFound
		}
		return err
	}

	err = s.linksManager.StoreRedirect(c, repo.StoreRedirectParams{
		UserAgent: c.GetHeader("User-Agent"),
		LongLink:  longLink,
		ShortLink: shortLink,
	})
	if err != nil {
		return err
	}

	c.Redirect(http.StatusTemporaryRedirect, longLink)
	return nil
}

func (s *LinksService) GetAnalytics(c *gin.Context, shortLink string) ([]repo.Redirect, error) {

	_, err := s.linksManager.GetLongByShort(c, shortLink)
	if err != nil {
		if errors.Is(err, errors.New("error link not found")) {
			return nil, errors2.ErrorLinkNotFound
		}
		return nil, err
	}

	redirects, err := s.linksManager.GetRedirectsByShortLink(c, shortLink)
	if err != nil {
		log.Println("Ошибка получения аналитики: ", err)
		c.JSON(http.StatusInternalServerError, "Ошибка при получении аналитики")
		return nil, err
	}

	c.JSON(http.StatusOK, gin.H{
		"redirects":   redirects,
		"total_count": len(redirects),
	})
	return redirects, nil
}

func validateShortLink(link string) error {
	if len(link) < ShortLinkLength {
		return errors2.ErrorLinkTooShort
	}

	for _, r := range link {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return errors2.ErrorInvalidSymbolInLink
		}
	}
	return nil
}
