package repo

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"time"
)

type Repository struct {
	db *pgx.Conn
}

func New(db *pgx.Conn) Repository {
	return Repository{
		db: db,
	}
}

type StoreRedirectParams struct {
	UserAgent string
	LongLink  string
	ShortLink string
}

type Redirect struct {
	ID        int       `json:"id"`
	LongLink  string    `json:"long_link"`
	ShortLink string    `json:"short_link"`
	UserAgent string    `json:"user_agent"`
	CreatedAt time.Time `json:"created_at"`
}

type LinkPair struct {
	Short string
	Long  string
}

func (r *Repository) IsShortExists(ctx context.Context, shortLink string) (bool, error) {
	var existingShortLink string
	err := r.db.QueryRow(ctx, "SELECT short_link FROM links WHERE short_link = $1", shortLink).Scan(&existingShortLink)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *Repository) CreateLink(ctx context.Context, longLink string, shortLink string) error {
	_, err := r.db.Exec(ctx, "INSERT INTO links (long_link, short_link) VALUES ($1, $2)", longLink, shortLink)
	if err != nil {
		return err
	}

	return nil
}

func (r *Repository) StoreRedirect(ctx context.Context, params StoreRedirectParams) error {
	_, err := r.db.Exec(ctx,
		"INSERT INTO redirects (long_link, short_link, user_agent) VALUES ($1, $2, $3)",
		params.LongLink,
		params.ShortLink,
		params.UserAgent,
	)
	return err
}

func (r *Repository) GetRedirectsByShortLink(ctx context.Context, shortLink string) ([]Redirect, error) {
	rows, err := r.db.Query(ctx, "SELECT id, long_link, short_link, user_agent, created_at FROM redirects WHERE short_link = $1", shortLink)

	var redirects []Redirect
	for rows.Next() {
		var redirect Redirect
		err = rows.Scan(&redirect.ID, &redirect.LongLink, &redirect.ShortLink, &redirect.UserAgent, &redirect.CreatedAt)

		if err != nil {
			return nil, err
		}
		redirects = append(redirects, redirect)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return redirects, nil
}

func (r *Repository) GetLongByShort(ctx context.Context, shortLink string) (string, error) {
	var longLink string
	err := r.db.QueryRow(ctx, "SELECT long_link FROM links WHERE short_link = $1", shortLink).Scan(&longLink)
	if err != nil {
		return "", err
	}
	return longLink, nil
}

func (r *Repository) GetShortByLong(ctx context.Context, longLink string) (string, error) {
	var shortLink string
	err := r.db.QueryRow(ctx, "SELECT short_link FROM links WHERE long_link = $1", longLink).Scan(&shortLink)
	if err != nil {
		return "", err
	}
	return shortLink, nil
}

func (r *Repository) GetPopularLinks(ctx context.Context, n int) ([]LinkPair, error) {
	rows, err := r.db.Query(
		ctx,
		`select
				short_link,
				long_link
		   from redirects
	   group by short_link, long_link
	   order by count(id) desc
		  limit $1;`,
		n,
	)
	if err != nil {
		return nil, fmt.Errorf("error GetPopularLinks: %w", err)
	}

	res := make([]LinkPair, 0, n)

	for rows.Next() {
		linkPair := LinkPair{}
		err := rows.Scan(&linkPair.Short, &linkPair.Long)
		if err != nil {
			return nil, fmt.Errorf("error GetPopularLinks Scan: %w", err)
		}
		res = append(res, linkPair)
	}

	return res, nil
}
