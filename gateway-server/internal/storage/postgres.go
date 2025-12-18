package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"gateway-server/internal/models"
)

type Storage struct {
	pool *pgxpool.Pool
}

func NewStorage(ctx context.Context, connString string) (*Storage, error) {
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("unable to parse config: %w", err)
	}

	config.MaxConns = 50
	config.MinConns = 10

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	return &Storage{pool: pool}, nil
}

func (s *Storage) Close() {
	s.pool.Close()
}

func (s *Storage) CreateVirtualService(ctx context.Context, req *models.CreateVSRequest, ownerUserID string) (*models.VirtualService, error) {
	vs := &models.VirtualService{
		ID:                  uuid.New().String(),
		OwnerUserID:         ownerUserID,
		Name:                req.Name,
		Slug:                req.Slug,
		BackendURL:          req.BackendURL,
		IsActive:            true,
		RequireAuth:         req.RequireAuth || req.RateLimitEnabled,
		TIMode:              req.TIMode,
		RateLimitEnabled:    req.RateLimitEnabled,
		RateLimitRequests:   req.RateLimitRequests,
		RateLimitWindowSec:  req.RateLimitWindowSec,
		LogRetentionMinutes: req.LogRetentionMinutes,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	_, err := s.pool.Exec(ctx, `
		INSERT INTO virtual_services (id, owner_user_id, name, slug, backend_url, is_active, require_auth,
		                              ti_mode, rate_limit_enabled, rate_limit_requests, rate_limit_window_sec,
		                              log_retention_minutes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`, vs.ID, vs.OwnerUserID, vs.Name, vs.Slug, vs.BackendURL, vs.IsActive, vs.RequireAuth,
		vs.TIMode, vs.RateLimitEnabled, vs.RateLimitRequests, vs.RateLimitWindowSec,
		vs.LogRetentionMinutes, vs.CreatedAt, vs.UpdatedAt)

	if err != nil {
		return nil, err
	}

	_, err = s.pool.Exec(ctx, `
		INSERT INTO virtual_service_users (vs_id, user_id, granted_by, granted_at)
		VALUES ($1, $2, $3, $4)
	`, vs.ID, ownerUserID, ownerUserID, time.Now())

	if err != nil {
		return nil, err
	}

	return vs, nil
}

func (s *Storage) GetVirtualService(ctx context.Context, vsID string) (*models.VirtualService, error) {
	var vs models.VirtualService
	err := s.pool.QueryRow(ctx, `
		SELECT id, owner_user_id, name, slug, backend_url, is_active, require_auth, ti_mode,
		       rate_limit_enabled, rate_limit_requests, rate_limit_window_sec, log_retention_minutes,
		       created_at, updated_at
		FROM virtual_services
		WHERE id = $1
	`, vsID).Scan(&vs.ID, &vs.OwnerUserID, &vs.Name, &vs.Slug, &vs.BackendURL, &vs.IsActive,
		&vs.RequireAuth, &vs.TIMode, &vs.RateLimitEnabled, &vs.RateLimitRequests,
		&vs.RateLimitWindowSec, &vs.LogRetentionMinutes, &vs.CreatedAt, &vs.UpdatedAt)

	if err != nil {
		return nil, err
	}

	return &vs, nil
}

func (s *Storage) GetVirtualServiceBySlug(ctx context.Context, slug string) (*models.VirtualService, error) {
	var vs models.VirtualService
	err := s.pool.QueryRow(ctx, `
		SELECT id, owner_user_id, name, slug, backend_url, is_active, require_auth, ti_mode,
		       rate_limit_enabled, rate_limit_requests, rate_limit_window_sec, log_retention_minutes,
		       created_at, updated_at
		FROM virtual_services
		WHERE slug = $1
	`, slug).Scan(&vs.ID, &vs.OwnerUserID, &vs.Name, &vs.Slug, &vs.BackendURL, &vs.IsActive,
		&vs.RequireAuth, &vs.TIMode, &vs.RateLimitEnabled, &vs.RateLimitRequests,
		&vs.RateLimitWindowSec, &vs.LogRetentionMinutes, &vs.CreatedAt, &vs.UpdatedAt)

	if err != nil {
		return nil, err
	}

	return &vs, nil
}

func (s *Storage) GetVirtualServicesByUser(ctx context.Context, userID string) ([]*models.VirtualService, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT DISTINCT vs.id, vs.owner_user_id, vs.name, vs.slug, vs.backend_url, vs.is_active, vs.require_auth,
		       vs.ti_mode, vs.rate_limit_enabled, vs.rate_limit_requests, vs.rate_limit_window_sec,
		       vs.log_retention_minutes, vs.created_at, vs.updated_at
		FROM virtual_services vs
		JOIN virtual_service_users vsu ON vs.id = vsu.vs_id
		WHERE vsu.user_id = $1
		ORDER BY vs.created_at DESC
	`, userID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var services []*models.VirtualService
	for rows.Next() {
		var vs models.VirtualService
		err := rows.Scan(&vs.ID, &vs.OwnerUserID, &vs.Name, &vs.Slug, &vs.BackendURL, &vs.IsActive,
			&vs.RequireAuth, &vs.TIMode, &vs.RateLimitEnabled, &vs.RateLimitRequests,
			&vs.RateLimitWindowSec, &vs.LogRetentionMinutes, &vs.CreatedAt, &vs.UpdatedAt)
		if err != nil {
			return nil, err
		}
		services = append(services, &vs)
	}

	return services, nil
}

func (s *Storage) UpdateVirtualService(ctx context.Context, vsID string, req *models.UpdateVSRequest) error {
	query := "UPDATE virtual_services SET updated_at = NOW()"
	args := []interface{}{vsID}
	argIdx := 2

	requireAuthSet := false
	finalRequireAuth := false

	if req.RequireAuth != nil {
		finalRequireAuth = *req.RequireAuth
		requireAuthSet = true
	}

	if req.RateLimitEnabled != nil && *req.RateLimitEnabled {
		finalRequireAuth = true
		requireAuthSet = true
	}

	if req.Name != nil {
		query += fmt.Sprintf(", name = $%d", argIdx)
		args = append(args, *req.Name)
		argIdx++
	}
	if req.BackendURL != nil {
		query += fmt.Sprintf(", backend_url = $%d", argIdx)
		args = append(args, *req.BackendURL)
		argIdx++
	}
	if req.IsActive != nil {
		query += fmt.Sprintf(", is_active = $%d", argIdx)
		args = append(args, *req.IsActive)
		argIdx++
	}
	if requireAuthSet {
		query += fmt.Sprintf(", require_auth = $%d", argIdx)
		args = append(args, finalRequireAuth)
		argIdx++
	}
	if req.TIMode != nil {
		query += fmt.Sprintf(", ti_mode = $%d", argIdx)
		args = append(args, *req.TIMode)
		argIdx++
	}
	if req.RateLimitEnabled != nil {
		query += fmt.Sprintf(", rate_limit_enabled = $%d", argIdx)
		args = append(args, *req.RateLimitEnabled)
		argIdx++
	}
	if req.RateLimitRequests != nil {
		query += fmt.Sprintf(", rate_limit_requests = $%d", argIdx)
		args = append(args, *req.RateLimitRequests)
		argIdx++
	}
	if req.RateLimitWindowSec != nil {
		query += fmt.Sprintf(", rate_limit_window_sec = $%d", argIdx)
		args = append(args, *req.RateLimitWindowSec)
		argIdx++
	}
	if req.LogRetentionMinutes != nil {
		query += fmt.Sprintf(", log_retention_minutes = $%d", argIdx)
		args = append(args, *req.LogRetentionMinutes)
		argIdx++
	}

	query += " WHERE id = $1"

	_, err := s.pool.Exec(ctx, query, args...)
	return err
}

func (s *Storage) DeleteVirtualService(ctx context.Context, vsID string) error {
	_, err := s.pool.Exec(ctx, "DELETE FROM virtual_services WHERE id = $1", vsID)
	return err
}

func (s *Storage) AddVSUser(ctx context.Context, vsID, userID, grantedBy string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO virtual_service_users (vs_id, user_id, granted_by, granted_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (vs_id, user_id) DO NOTHING
	`, vsID, userID, grantedBy, time.Now())
	return err
}

func (s *Storage) RemoveVSUser(ctx context.Context, vsID, userID string) error {
	_, err := s.pool.Exec(ctx, `
		DELETE FROM virtual_service_users
		WHERE vs_id = $1 AND user_id = $2
	`, vsID, userID)
	return err
}

func (s *Storage) GetVSUsers(ctx context.Context, vsID string) ([]*models.VirtualServiceUser, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT vs_id, user_id, granted_by, granted_at
		FROM virtual_service_users
		WHERE vs_id = $1
		ORDER BY granted_at DESC
	`, vsID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*models.VirtualServiceUser
	for rows.Next() {
		var u models.VirtualServiceUser
		err := rows.Scan(&u.VSID, &u.UserID, &u.GrantedBy, &u.GrantedAt)
		if err != nil {
			return nil, err
		}
		users = append(users, &u)
	}

	return users, nil
}

func (s *Storage) UserHasAccessToVS(ctx context.Context, vsID, userID string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM virtual_service_users WHERE vs_id = $1 AND user_id = $2)
	`, vsID, userID).Scan(&exists)
	return exists, err
}

func (s *Storage) AttachTIFeed(ctx context.Context, vsID, feedID string, apiKey *string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO virtual_service_ti_feeds (vs_id, feed_id, api_key, is_active, added_at)
		VALUES ($1, $2, $3, true, $4)
		ON CONFLICT (vs_id, feed_id) DO UPDATE SET api_key = EXCLUDED.api_key
	`, vsID, feedID, apiKey, time.Now())
	return err
}

func (s *Storage) DetachTIFeed(ctx context.Context, vsID, feedID string) error {
	_, err := s.pool.Exec(ctx, `
		DELETE FROM virtual_service_ti_feeds
		WHERE vs_id = $1 AND feed_id = $2
	`, vsID, feedID)
	return err
}

func (s *Storage) ToggleTIFeed(ctx context.Context, vsID, feedID string, isActive bool) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE virtual_service_ti_feeds
		SET is_active = $3
		WHERE vs_id = $1 AND feed_id = $2
	`, vsID, feedID, isActive)
	return err
}

func (s *Storage) GetVSTIFeeds(ctx context.Context, vsID string) ([]*models.VirtualServiceTIFeed, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT vs_id, feed_id, api_key, is_active, added_at
		FROM virtual_service_ti_feeds
		WHERE vs_id = $1
		ORDER BY added_at DESC
	`, vsID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var feeds []*models.VirtualServiceTIFeed
	for rows.Next() {
		var f models.VirtualServiceTIFeed
		err := rows.Scan(&f.VSID, &f.FeedID, &f.APIKey, &f.IsActive, &f.AddedAt)
		if err != nil {
			return nil, err
		}
		feeds = append(feeds, &f)
	}

	return feeds, nil
}

func (s *Storage) GetActiveVSTIFeeds(ctx context.Context, vsID string) ([]string, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT feed_id
		FROM virtual_service_ti_feeds
		WHERE vs_id = $1 AND is_active = true
	`, vsID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var feedIDs []string
	for rows.Next() {
		var feedID string
		err := rows.Scan(&feedID)
		if err != nil {
			return nil, err
		}
		feedIDs = append(feedIDs, feedID)
	}

	return feedIDs, nil
}

func (s *Storage) GetActiveVSTIFeedsWithKeys(ctx context.Context, vsID string) ([]*models.VirtualServiceTIFeed, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT vs_id, feed_id, api_key, is_active, added_at
		FROM virtual_service_ti_feeds
		WHERE vs_id = $1 AND is_active = true
		ORDER BY added_at DESC
	`, vsID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var feeds []*models.VirtualServiceTIFeed
	for rows.Next() {
		var f models.VirtualServiceTIFeed
		err := rows.Scan(&f.VSID, &f.FeedID, &f.APIKey, &f.IsActive, &f.AddedAt)
		if err != nil {
			return nil, err
		}
		feeds = append(feeds, &f)
	}

	return feeds, nil
}

func (s *Storage) GetAllActiveVSTIFeeds(ctx context.Context) ([]*models.VirtualServiceTIFeed, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT DISTINCT ON (feed_id) feed_id, api_key
		FROM virtual_service_ti_feeds
		WHERE is_active = true
		ORDER BY feed_id
	`)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var feeds []*models.VirtualServiceTIFeed
	for rows.Next() {
		var f models.VirtualServiceTIFeed
		err := rows.Scan(&f.FeedID, &f.APIKey)
		if err != nil {
			return nil, err
		}
		feeds = append(feeds, &f)
	}

	return feeds, nil
}

func (s *Storage) LogTraffic(ctx context.Context, log *models.TrafficLog) error {
	log.ID = uuid.New().String()
	log.Timestamp = time.Now()

	_, err := s.pool.Exec(ctx, `
		INSERT INTO traffic_logs (id, vs_id, user_id, client_ip, method, path, request_headers, request_body,
		                         status_code, response_headers, response_body, ioc_matches, blocked,
		                         response_time_ms, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`, log.ID, log.VSID, log.UserID, log.ClientIP, log.Method, log.Path, log.RequestHeaders, log.RequestBody,
		log.StatusCode, log.ResponseHeaders, log.ResponseBody, log.IOCMatches, log.Blocked,
		log.ResponseTimeMs, log.Timestamp)

	return err
}

func (s *Storage) GetTrafficLogs(ctx context.Context, vsID string, limit, offset int, blocked *bool) ([]*models.TrafficLog, int, error) {
	baseQuery := `
		FROM traffic_logs
		WHERE vs_id = $1
	`
	args := []interface{}{vsID}
	argIdx := 2

	if blocked != nil {
		baseQuery += fmt.Sprintf(" AND blocked = $%d", argIdx)
		args = append(args, *blocked)
		argIdx++
	}

	var total int
	err := s.pool.QueryRow(ctx, "SELECT COUNT(*) "+baseQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, vs_id, user_id, client_ip, method, path, request_headers, request_body,
		       status_code, response_headers, response_body, ioc_matches, blocked, response_time_ms, timestamp
	` + baseQuery + fmt.Sprintf(" ORDER BY timestamp DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []*models.TrafficLog
	for rows.Next() {
		var log models.TrafficLog
		err := rows.Scan(&log.ID, &log.VSID, &log.UserID, &log.ClientIP, &log.Method, &log.Path,
			&log.RequestHeaders, &log.RequestBody, &log.StatusCode, &log.ResponseHeaders,
			&log.ResponseBody, &log.IOCMatches, &log.Blocked, &log.ResponseTimeMs, &log.Timestamp)
		if err != nil {
			return nil, 0, err
		}
		logs = append(logs, &log)
	}

	return logs, total, nil
}

func (s *Storage) CleanOldLogs(ctx context.Context) error {
	rows, err := s.pool.Query(ctx, `
		SELECT id, log_retention_minutes FROM virtual_services
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var vsID string
		var retentionMinutes int
		err := rows.Scan(&vsID, &retentionMinutes)
		if err != nil {
			continue
		}

		_, err = s.pool.Exec(ctx, `
			DELETE FROM traffic_logs
			WHERE vs_id = $1 AND timestamp < NOW() - INTERVAL '1 minute' * $2
		`, vsID, retentionMinutes)
		if err != nil {
			continue
		}
	}

	return nil
}

func (s *Storage) CleanExpiredData(ctx context.Context, cutoff time.Duration) error {
	cutoffTime := time.Now().Add(-cutoff)

	if _, err := s.pool.Exec(ctx, `DELETE FROM traffic_logs WHERE timestamp < $1`, cutoffTime); err != nil {
		return err
	}

	if _, err := s.pool.Exec(ctx, `DELETE FROM virtual_service_ti_feeds WHERE added_at < $1`, cutoffTime); err != nil {
		return err
	}

	if _, err := s.pool.Exec(ctx, `DELETE FROM virtual_service_users WHERE granted_at < $1`, cutoffTime); err != nil {
		return err
	}

	_, err := s.pool.Exec(ctx, `DELETE FROM virtual_services WHERE updated_at < $1`, cutoffTime)
	return err
}

func MarshalIOCMatches(matches []models.IOCMatch) (json.RawMessage, error) {
	if len(matches) == 0 {
		return json.RawMessage("[]"), nil
	}
	return json.Marshal(matches)
}
