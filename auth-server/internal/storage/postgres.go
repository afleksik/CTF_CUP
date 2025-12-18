package storage

import (
	"context"
	"errors"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"auth-server/internal/models"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrUserExists         = errors.New("user already exists")
	ErrInvalidEmail       = errors.New("invalid email format")
	ErrInvalidUsername    = errors.New("invalid username")
	ErrInvalidPassword    = errors.New("invalid password")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+=-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

type Storage struct {
	pool *pgxpool.Pool
}

func NewStorage(ctx context.Context, connString string) (*Storage, error) {
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, err
	}

	config.MaxConns = 50
	config.MinConns = 10

	pool, err := pgxpool.NewWithConfig(ctx, config)
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

	if _, err := s.pool.Exec(ctx, `DELETE FROM auth_codes WHERE created_at < $1 OR expires_at < $1`, cutoffTime); err != nil {
		return err
	}

	if _, err := s.pool.Exec(ctx, `DELETE FROM refresh_tokens WHERE created_at < $1 OR expires_at < $1`, cutoffTime); err != nil {
		return err
	}

	_, err := s.pool.Exec(ctx, `DELETE FROM users WHERE updated_at < $1`, cutoffTime)
	return err
}

func IsValidEmail(email string) bool {
	return emailRegex.MatchString(email)
}

func (s *Storage) CreateUser(ctx context.Context, username, email, passwordHash, bio string) (*models.User, error) {
	if username == "" || len(username) < 3 || len(username) > 50 {
		return nil, ErrInvalidUsername
	}
	if !IsValidEmail(email) {
		return nil, ErrInvalidEmail
	}
	if passwordHash == "" {
		return nil, ErrInvalidPassword
	}

	var exists bool
	err := s.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)", username).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrUserExists
	}

	user := &models.User{
		ID:           uuid.New().String(),
		Username:     username,
		Email:        email,
		PasswordHash: passwordHash,
		Bio:          bio,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	query := `
		INSERT INTO users (id, username, email, password_hash, bio, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err = s.pool.Exec(ctx, query,
		user.ID,
		user.Username,
		user.Email,
		user.PasswordHash,
		user.Bio,
		user.CreatedAt,
		user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *Storage) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	query := `SELECT * FROM users WHERE username = $1`

	var user models.User
	err := s.pool.QueryRow(ctx, query, username).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.Bio,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return &user, nil
}

func (s *Storage) GetUserByID(ctx context.Context, userID string) (*models.User, error) {
	query := `SELECT id, username, email, password_hash, bio, created_at, updated_at
	          FROM users WHERE id = $1`

	var user models.User
	err := s.pool.QueryRow(ctx, query, userID).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.Bio,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return &user, nil
}

func (s *Storage) GetPublicUsers(ctx context.Context, limit, offset int) ([]*models.PublicUser, int, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	var total int
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `SELECT id, username, bio, created_at
	          FROM users
	          ORDER BY created_at DESC
	          LIMIT $1 OFFSET $2`

	rows, err := s.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []*models.PublicUser
	for rows.Next() {
		var user models.PublicUser
		err := rows.Scan(&user.ID, &user.Username, &user.Bio, &user.CreatedAt)
		if err != nil {
			return nil, 0, err
		}
		users = append(users, &user)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func (s *Storage) SearchUsers(ctx context.Context, query string, limit int) ([]*models.User, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}

	searchPattern := "%" + query + "%"
	sqlQuery := `SELECT * FROM users
	             WHERE username ILIKE $1 OR CAST(id AS TEXT) ILIKE $1
	             ORDER BY created_at DESC
	             LIMIT $2`

	rows, err := s.pool.Query(ctx, sqlQuery, searchPattern, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(&user.ID,
			&user.Username,
			&user.Email,
			&user.PasswordHash,
			&user.Bio,
			&user.CreatedAt,
			&user.UpdatedAt)
		if err != nil {
			return nil, err
		}
		users = append(users, &user)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

func (s *Storage) GetPublicUserByID(ctx context.Context, userID string) (*models.PublicUser, error) {
	query := `SELECT id, username, bio, created_at FROM users WHERE id = $1`

	var user models.PublicUser
	err := s.pool.QueryRow(ctx, query, userID).Scan(&user.ID, &user.Username, &user.Bio, &user.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return &user, nil
}

func (s *Storage) UpdateUser(ctx context.Context, userID string, email, bio string) (*models.User, error) {
	if email != "" && !IsValidEmail(email) {
		return nil, ErrInvalidEmail
	}

	query := `
		UPDATE users
		SET email = COALESCE(NULLIF($2, ''), email),
		    bio = COALESCE(NULLIF($3, ''), bio),
		    updated_at = $4
		WHERE id = $1
		RETURNING id, username, email, password_hash, bio, created_at, updated_at
	`

	var user models.User
	err := s.pool.QueryRow(ctx, query, userID, email, bio, time.Now()).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.Bio,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return &user, nil
}

func (s *Storage) GetOAuthClient(ctx context.Context, clientID string) (*models.OAuthClient, error) {
	query := `SELECT client_id, client_secret, name, redirect_uris, created_at
	          FROM oauth_clients WHERE client_id = $1`

	var client models.OAuthClient
	err := s.pool.QueryRow(ctx, query, clientID).Scan(
		&client.ClientID,
		&client.ClientSecret,
		&client.Name,
		&client.RedirectURIs,
		&client.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("client not found")
		}
		return nil, err
	}

	return &client, nil
}

func (s *Storage) UpdateOAuthClientSecret(ctx context.Context, clientID, secret string) error {
	query := `UPDATE oauth_clients SET client_secret = $2 WHERE client_id = $1`
	result, err := s.pool.Exec(ctx, query, clientID, secret)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return errors.New("client not found")
	}
	return nil
}

func (s *Storage) CreateAuthCode(ctx context.Context, code *models.AuthCode) error {
	query := `INSERT INTO auth_codes (code, user_id, client_id, redirect_uri, state,
	                                   code_challenge, code_challenge_method, expires_at, created_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := s.pool.Exec(ctx, query,
		code.Code,
		code.UserID,
		code.ClientID,
		code.RedirectURI,
		code.State,
		code.CodeChallenge,
		code.CodeChallengeMethod,
		code.ExpiresAt,
		code.CreatedAt,
	)

	return err
}

func (s *Storage) GetAuthCode(ctx context.Context, code string) (*models.AuthCode, error) {
	query := `SELECT code, user_id, client_id, redirect_uri, state,
	                 code_challenge, code_challenge_method, expires_at, created_at, used, used_at
	          FROM auth_codes WHERE code = $1`

	var authCode models.AuthCode
	err := s.pool.QueryRow(ctx, query, code).Scan(
		&authCode.Code,
		&authCode.UserID,
		&authCode.ClientID,
		&authCode.RedirectURI,
		&authCode.State,
		&authCode.CodeChallenge,
		&authCode.CodeChallengeMethod,
		&authCode.ExpiresAt,
		&authCode.CreatedAt,
		&authCode.Used,
		&authCode.UsedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("code not found")
		}
		return nil, err
	}

	if authCode.Used {
		return nil, errors.New("code already used")
	}

	if time.Now().After(authCode.ExpiresAt) {
		return nil, errors.New("code expired")
	}

	updateQuery := `UPDATE auth_codes SET used = true, used_at = $1 WHERE code = $2`
	_, err = s.pool.Exec(ctx, updateQuery, time.Now(), code)
	if err != nil {
		return nil, err
	}

	return &authCode, nil
}

func (s *Storage) CreateRefreshToken(ctx context.Context, token *models.RefreshToken) error {
	query := `INSERT INTO refresh_tokens (token, user_id, client_id, expires_at, created_at)
	          VALUES ($1, $2, $3, $4, $5)`

	_, err := s.pool.Exec(ctx, query,
		token.Token,
		token.UserID,
		token.ClientID,
		token.ExpiresAt,
		token.CreatedAt,
	)

	return err
}

func (s *Storage) GetRefreshToken(ctx context.Context, token string) (*models.RefreshToken, error) {
	query := `SELECT token, user_id, client_id, expires_at, created_at, revoked, revoked_at
	          FROM refresh_tokens WHERE token = $1`

	var refreshToken models.RefreshToken
	err := s.pool.QueryRow(ctx, query, token).Scan(
		&refreshToken.Token,
		&refreshToken.UserID,
		&refreshToken.ClientID,
		&refreshToken.ExpiresAt,
		&refreshToken.CreatedAt,
		&refreshToken.Revoked,
		&refreshToken.RevokedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("token not found")
		}
		return nil, err
	}

	if refreshToken.Revoked {
		return nil, errors.New("token revoked")
	}

	if time.Now().After(refreshToken.ExpiresAt) {
		return nil, errors.New("token expired")
	}

	return &refreshToken, nil
}

func (s *Storage) RevokeRefreshToken(ctx context.Context, token string) error {
	query := `UPDATE refresh_tokens SET revoked = true, revoked_at = $1 WHERE token = $2`
	_, err := s.pool.Exec(ctx, query, time.Now(), token)
	return err
}

func (s *Storage) RevokeRefreshTokensByUser(ctx context.Context, userID string) error {
	query := `UPDATE refresh_tokens SET revoked = true, revoked_at = $1 WHERE user_id = $2 AND revoked = false`
	_, err := s.pool.Exec(ctx, query, time.Now(), userID)
	return err
}
