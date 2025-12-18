package storage

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"ti-server/internal/models"
)

var (
	ErrFeedNotFound    = errors.New("feed not found")
	ErrUnauthorized    = errors.New("unauthorized")
	ErrInvalidIOCType  = errors.New("invalid IOC type")
	ErrInvalidSeverity = errors.New("invalid severity")
)

type Storage struct {
	pool *pgxpool.Pool
}

func NewStorage(ctx context.Context, connString string) (*Storage, error) {
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}

	return &Storage{pool: pool}, nil
}

func (s *Storage) Close() {
	s.pool.Close()
}

func (s *Storage) CleanExpiredData(ctx context.Context, cutoff time.Duration) error {
	cutoffTime := time.Now().Add(-cutoff)

	if _, err := s.pool.Exec(ctx, `DELETE FROM iocs WHERE created_at < $1`, cutoffTime); err != nil {
		return err
	}

	_, err := s.pool.Exec(ctx, `DELETE FROM feeds WHERE updated_at < $1`, cutoffTime)
	return err
}

func (s *Storage) CreateFeed(ctx context.Context, name, description string, isPublic bool) (*models.Feed, error) {
	userUuid, err := uuid.NewUUID()
	if err != nil {
		return nil, err
	}

	var apiKeyParam interface{}
	if !isPublic {
		uuid, err := uuid.NewUUID()
		if err != nil {
			return nil, err
		}
		apiKeyParam = uuid.String()
	}

	query := `
		INSERT INTO feeds (id, name, description, is_public, api_key, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err = s.pool.Exec(ctx, query,
		userUuid.String(),
		name,
		description,
		isPublic,
		apiKeyParam,
		time.Now(),
		time.Now(),
	)

	if err != nil {
		return nil, err
	}

	selectQuery := `SELECT id, name, description, is_public, api_key, created_at, updated_at FROM feeds WHERE id = $1`

	var feed models.Feed
	var apiKeyNullable *string
	err = s.pool.QueryRow(ctx, selectQuery, userUuid.String()).Scan(
		&feed.ID,
		&feed.Name,
		&feed.Description,
		&feed.IsPublic,
		&apiKeyNullable,
		&feed.CreatedAt,
		&feed.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	if apiKeyNullable != nil {
		feed.APIKey = *apiKeyNullable
	}

	return &feed, nil
}

func (s *Storage) GetFeed(ctx context.Context, feedID string, apiKey string) (*models.Feed, error) {
	query := `SELECT id, name, description, is_public, api_key, created_at, updated_at FROM feeds WHERE id = $1`

	var feed models.Feed
	var apiKeyNullable *string
	err := s.pool.QueryRow(ctx, query, feedID).Scan(
		&feed.ID,
		&feed.Name,
		&feed.Description,
		&feed.IsPublic,
		&apiKeyNullable,
		&feed.CreatedAt,
		&feed.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrFeedNotFound
		}
		return nil, err
	}

	if apiKeyNullable != nil {
		feed.APIKey = *apiKeyNullable
	}

	if !feed.IsPublic {
		if apiKey == "" || apiKey != feed.APIKey {
			return nil, ErrUnauthorized
		}
	}

	if apiKey != feed.APIKey {
		feed.APIKey = ""
	}

	return &feed, nil
}

func (s *Storage) GetFeeds(ctx context.Context, visibility string, limit, offset int) ([]*models.Feed, int, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	whereClause := ""
	switch visibility {
	case "public":
		whereClause = " WHERE is_public = true"
	case "private":
		whereClause = " WHERE is_public = false"
	}

	countQuery := "SELECT COUNT(*) FROM feeds" + whereClause
	var total int
	if err := s.pool.QueryRow(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := "SELECT id, name, description, is_public, created_at, updated_at FROM feeds" + whereClause + " ORDER BY created_at DESC LIMIT $1 OFFSET $2"
	rows, err := s.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var feeds []*models.Feed
	for rows.Next() {
		var feed models.Feed
		err := rows.Scan(
			&feed.ID,
			&feed.Name,
			&feed.Description,
			&feed.IsPublic,
			&feed.CreatedAt,
			&feed.UpdatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		feeds = append(feeds, &feed)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return feeds, total, nil
}

func (s *Storage) AddIOC(ctx context.Context, feedID string, iocType, value, severity, description, apiKey string) (*models.IOC, error) {
	feed, err := s.GetFeed(ctx, feedID, apiKey)
	if err != nil {
		return nil, err
	}

	if !feed.IsPublic && (apiKey == "" || apiKey != feed.APIKey) {
		return nil, ErrUnauthorized
	}

	validTypes := map[string]bool{
		"ip": true, "domain": true, "hash": true, "url": true,
		"email": true, "md5": true, "sha1": true, "sha256": true,
	}
	if !validTypes[iocType] {
		return nil, ErrInvalidIOCType
	}

	validSeverities := map[string]bool{
		"low": true, "medium": true, "high": true, "critical": true,
	}
	if !validSeverities[severity] {
		return nil, ErrInvalidSeverity
	}

	ioc := &models.IOC{
		ID:          uuid.New().String(),
		FeedID:      feedID,
		Type:        iocType,
		Value:       value,
		Severity:    severity,
		Description: description,
		CreatedAt:   time.Now(),
	}

	query := `
		INSERT INTO iocs (id, feed_id, type, value, severity, description, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err = s.pool.Exec(ctx, query,
		ioc.ID,
		ioc.FeedID,
		ioc.Type,
		ioc.Value,
		ioc.Severity,
		ioc.Description,
		ioc.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return ioc, nil
}

func (s *Storage) GetIOCs(ctx context.Context, feedID string, apiKey string, limit, offset int) ([]*models.IOC, error) {
	_, err := s.GetFeed(ctx, feedID, apiKey)
	if err != nil {
		return nil, err
	}

	query := `SELECT id, feed_id, type, value, severity, description, created_at
	          FROM iocs
	          WHERE feed_id = $1
	          ORDER BY created_at DESC
	          LIMIT $2 OFFSET $3`

	rows, err := s.pool.Query(ctx, query, feedID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var iocs []*models.IOC
	for rows.Next() {
		var ioc models.IOC
		err := rows.Scan(
			&ioc.ID,
			&ioc.FeedID,
			&ioc.Type,
			&ioc.Value,
			&ioc.Severity,
			&ioc.Description,
			&ioc.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		iocs = append(iocs, &ioc)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return iocs, nil
}
