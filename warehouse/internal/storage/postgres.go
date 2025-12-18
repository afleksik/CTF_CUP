package storage

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"warehouse/internal/models"
)

var (
	ErrRealmNotFound      = errors.New("realm not found")
	ErrAssetNotFound      = errors.New("asset not found")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrInvalidAssetType   = errors.New("invalid asset type")
	ErrInvalidRole        = errors.New("invalid role")
	ErrUserAlreadyInRealm = errors.New("user already in realm")
	ErrUserNotInRealm     = errors.New("user not in realm")
	ErrUserOwnsAssets     = errors.New("user owns assets in realm")
)

var validAssetTypes = map[string]bool{
	"spirits":   true,
	"wine":      true,
	"beer":      true,
	"mixers":    true,
	"garnishes": true,
}

var validRoles = map[string]bool{
	"admin":  true,
	"member": true,
}

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

func (s *Storage) CreateRealm(ctx context.Context, name, description, ownerUserID string) (*models.Realm, error) {
	realm := &models.Realm{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		OwnerUserID: ownerUserID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	query := `INSERT INTO realms (id, name, description, owner_user_id, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6)`
	_, err = tx.Exec(ctx, query, realm.ID, realm.Name, realm.Description, realm.OwnerUserID, realm.CreatedAt, realm.UpdatedAt)
	if err != nil {
		return nil, err
	}

	query = `INSERT INTO realm_users (realm_id, user_id, role, added_at) VALUES ($1, $2, $3, $4)`
	_, err = tx.Exec(ctx, query, realm.ID, ownerUserID, "admin", time.Now())
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return realm, nil
}

func (s *Storage) GetRealmsByUser(ctx context.Context, userID string) ([]*models.RealmWithRole, error) {
	query := `
		SELECT r.id, r.name, r.description, r.owner_user_id, r.gateway_vs_id, r.gateway_vs_slug, r.gateway_protected, r.created_at, r.updated_at, ru.role
		FROM realms r
		INNER JOIN realm_users ru ON r.id = ru.realm_id
		WHERE ru.user_id = $1
		ORDER BY r.created_at DESC
	`

	rows, err := s.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var realms []*models.RealmWithRole
	for rows.Next() {
		var realm models.RealmWithRole
		err := rows.Scan(&realm.ID, &realm.Name, &realm.Description, &realm.OwnerUserID, &realm.GatewayVSID, &realm.GatewayVSSlug, &realm.GatewayProtected, &realm.CreatedAt, &realm.UpdatedAt, &realm.Role)
		if err != nil {
			return nil, err
		}
		realms = append(realms, &realm)
	}

	return realms, rows.Err()
}

func (s *Storage) GetRealm(ctx context.Context, realmID string) (*models.Realm, error) {
	query := `SELECT id, name, description, owner_user_id, gateway_vs_id, gateway_vs_slug, gateway_protected, created_at, updated_at FROM realms WHERE id = $1`

	var realm models.Realm
	err := s.pool.QueryRow(ctx, query, realmID).Scan(
		&realm.ID, &realm.Name, &realm.Description, &realm.OwnerUserID, &realm.GatewayVSID, &realm.GatewayVSSlug, &realm.GatewayProtected, &realm.CreatedAt, &realm.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRealmNotFound
		}
		return nil, err
	}

	return &realm, nil
}

func (s *Storage) UpdateRealm(ctx context.Context, realmID, name, description string) (*models.Realm, error) {
	query := `
		UPDATE realms
		SET name = $2, description = $3, updated_at = $4
		WHERE id = $1
		RETURNING id, name, description, owner_user_id, created_at, updated_at
	`

	var realm models.Realm
	err := s.pool.QueryRow(ctx, query, realmID, name, description, time.Now()).Scan(
		&realm.ID, &realm.Name, &realm.Description, &realm.OwnerUserID, &realm.CreatedAt, &realm.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRealmNotFound
		}
		return nil, err
	}

	return &realm, nil
}

func (s *Storage) DeleteRealm(ctx context.Context, realmID string) error {
	query := `DELETE FROM realms WHERE id = $1`
	result, err := s.pool.Exec(ctx, query, realmID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrRealmNotFound
	}

	return nil
}

func (s *Storage) GetUserRole(ctx context.Context, realmID, userID string) (string, error) {
	query := `SELECT role FROM realm_users WHERE realm_id = $1 AND user_id = $2`

	var role string
	err := s.pool.QueryRow(ctx, query, realmID, userID).Scan(&role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrUserNotInRealm
		}
		return "", err
	}

	return role, nil
}

func (s *Storage) AddUserToRealm(ctx context.Context, realmID, userID, role string) error {
	if !validRoles[role] {
		return ErrInvalidRole
	}

	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM realm_users WHERE realm_id = $1 AND user_id = $2)`
	err := s.pool.QueryRow(ctx, query, realmID, userID).Scan(&exists)
	if err != nil {
		return err
	}
	if exists {
		return ErrUserAlreadyInRealm
	}

	query = `INSERT INTO realm_users (realm_id, user_id, role, added_at) VALUES ($1, $2, $3, $4)`
	_, err = s.pool.Exec(ctx, query, realmID, userID, role, time.Now())
	return err
}

func (s *Storage) GetRealmUsers(ctx context.Context, realmID string, limit, offset int) ([]*models.RealmUser, int, error) {
	var total int
	countQuery := `SELECT COUNT(*) FROM realm_users WHERE realm_id = $1`
	err := s.pool.QueryRow(ctx, countQuery, realmID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := `SELECT realm_id, user_id, role, added_at FROM realm_users WHERE realm_id = $1 ORDER BY added_at DESC LIMIT $2 OFFSET $3`
	rows, err := s.pool.Query(ctx, query, realmID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []*models.RealmUser
	for rows.Next() {
		var user models.RealmUser
		err := rows.Scan(&user.RealmID, &user.UserID, &user.Role, &user.AddedAt)
		if err != nil {
			return nil, 0, err
		}
		users = append(users, &user)
	}

	return users, total, rows.Err()
}

func (s *Storage) RemoveUserFromRealm(ctx context.Context, realmID, userID string) error {
	query := `DELETE FROM realm_users WHERE realm_id = $1 AND user_id = $2`
	result, err := s.pool.Exec(ctx, query, realmID, userID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrUserNotInRealm
	}

	return nil
}

func (s *Storage) CreateAsset(ctx context.Context, realmID, name, assetType, description, ownerUserID string) (*models.Asset, error) {
	if !validAssetTypes[assetType] {
		return nil, ErrInvalidAssetType
	}

	asset := &models.Asset{
		ID:          uuid.New().String(),
		RealmID:     realmID,
		Name:        name,
		AssetType:   assetType,
		Description: description,
		OwnerUserID: ownerUserID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	query := `
		INSERT INTO assets (id, realm_id, name, asset_type, description, owner_user_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := s.pool.Exec(ctx, query, asset.ID, asset.RealmID, asset.Name, asset.AssetType, asset.Description, asset.OwnerUserID, asset.CreatedAt, asset.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return asset, nil
}

func (s *Storage) GetAssetsByRealm(ctx context.Context, realmID string, assetType, search string, limit, offset int) ([]*models.Asset, int, error) {
	countQuery := `SELECT COUNT(*) FROM assets WHERE realm_id = $1`
	query := `SELECT id, realm_id, name, asset_type, description, owner_user_id, created_at, updated_at FROM assets WHERE realm_id = $1`

	args := []interface{}{realmID}
	argIndex := 2

	if assetType != "" {
		query += ` AND asset_type = $` + string(rune('0'+argIndex))
		countQuery += ` AND asset_type = $` + string(rune('0'+argIndex))
		args = append(args, assetType)
		argIndex++
	}

	if search != "" {
		searchPattern := "%" + search + "%"
		query += ` AND (name ILIKE $` + string(rune('0'+argIndex)) + ` OR description ILIKE $` + string(rune('0'+argIndex)) + `)`
		countQuery += ` AND (name ILIKE $` + string(rune('0'+argIndex)) + ` OR description ILIKE $` + string(rune('0'+argIndex)) + `)`
		args = append(args, searchPattern)
		argIndex++
	}

	var total int
	err := s.pool.QueryRow(ctx, countQuery, args[:len(args)]...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query += ` ORDER BY created_at DESC LIMIT $` + string(rune('0'+argIndex)) + ` OFFSET $` + string(rune('0'+argIndex+1))
	args = append(args, limit, offset)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var assets []*models.Asset
	for rows.Next() {
		var asset models.Asset
		err := rows.Scan(&asset.ID, &asset.RealmID, &asset.Name, &asset.AssetType, &asset.Description, &asset.OwnerUserID, &asset.CreatedAt, &asset.UpdatedAt)
		if err != nil {
			return nil, 0, err
		}
		assets = append(assets, &asset)
	}

	return assets, total, rows.Err()
}

func (s *Storage) GetAsset(ctx context.Context, assetID string) (*models.Asset, error) {
	query := `SELECT id, realm_id, name, asset_type, description, owner_user_id, created_at, updated_at FROM assets WHERE id = $1`

	var asset models.Asset
	err := s.pool.QueryRow(ctx, query, assetID).Scan(
		&asset.ID, &asset.RealmID, &asset.Name, &asset.AssetType, &asset.Description, &asset.OwnerUserID, &asset.CreatedAt, &asset.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAssetNotFound
		}
		return nil, err
	}

	return &asset, nil
}

func (s *Storage) UpdateAsset(ctx context.Context, assetID, name, assetType, description, ownerUserID string) (*models.Asset, error) {
	if !validAssetTypes[assetType] {
		return nil, ErrInvalidAssetType
	}

	query := `
		UPDATE assets
		SET name = $2, asset_type = $3, description = $4, owner_user_id = $5, updated_at = $6
		WHERE id = $1
		RETURNING id, realm_id, name, asset_type, description, owner_user_id, created_at, updated_at
	`

	var asset models.Asset
	err := s.pool.QueryRow(ctx, query, assetID, name, assetType, description, ownerUserID, time.Now()).Scan(
		&asset.ID, &asset.RealmID, &asset.Name, &asset.AssetType, &asset.Description, &asset.OwnerUserID, &asset.CreatedAt, &asset.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAssetNotFound
		}
		return nil, err
	}

	return &asset, nil
}

func (s *Storage) DeleteAsset(ctx context.Context, assetID string) error {
	query := `DELETE FROM assets WHERE id = $1`
	result, err := s.pool.Exec(ctx, query, assetID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrAssetNotFound
	}

	return nil
}

func (s *Storage) GetUserAssetsByRealm(ctx context.Context, realmID, userID string) ([]*models.Asset, error) {
	query := `
		SELECT id, realm_id, name, asset_type, description, owner_user_id, created_at, updated_at
		FROM assets
		WHERE realm_id = $1 AND owner_user_id = $2
		ORDER BY created_at DESC
	`

	rows, err := s.pool.Query(ctx, query, realmID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assets []*models.Asset
	for rows.Next() {
		var asset models.Asset
		err := rows.Scan(&asset.ID, &asset.RealmID, &asset.Name, &asset.AssetType, &asset.Description, &asset.OwnerUserID, &asset.CreatedAt, &asset.UpdatedAt)
		if err != nil {
			return nil, err
		}
		assets = append(assets, &asset)
	}

	return assets, rows.Err()
}

func (s *Storage) ReassignUserAssets(ctx context.Context, realmID, fromUserID, toUserID string) error {
	query := `
		UPDATE assets
		SET owner_user_id = $3, updated_at = $4
		WHERE realm_id = $1 AND owner_user_id = $2
	`
	_, err := s.pool.Exec(ctx, query, realmID, fromUserID, toUserID, time.Now())
	return err
}

func (s *Storage) DeleteUserAssets(ctx context.Context, realmID, userID string) error {
	query := `DELETE FROM assets WHERE realm_id = $1 AND owner_user_id = $2`
	_, err := s.pool.Exec(ctx, query, realmID, userID)
	return err
}

func (s *Storage) UpdateRealmGatewayInfo(ctx context.Context, realmID, vsID, vsSlug string) error {
	query := `
		UPDATE realms
		SET gateway_vs_id = $1, gateway_vs_slug = $2, gateway_protected = true, updated_at = NOW()
		WHERE id = $3
	`
	_, err := s.pool.Exec(ctx, query, vsID, vsSlug, realmID)
	return err
}

func (s *Storage) RemoveRealmGatewayInfo(ctx context.Context, realmID string) error {
	query := `
		UPDATE realms
		SET gateway_vs_id = NULL, gateway_vs_slug = NULL, gateway_protected = false, updated_at = NOW()
		WHERE id = $1
	`
	_, err := s.pool.Exec(ctx, query, realmID)
	return err
}

func (s *Storage) CleanExpiredData(ctx context.Context, cutoff time.Duration) error {
	cutoffTime := time.Now().Add(-cutoff)

	if _, err := s.pool.Exec(ctx, `DELETE FROM assets WHERE updated_at < $1`, cutoffTime); err != nil {
		return err
	}

	if _, err := s.pool.Exec(ctx, `DELETE FROM realm_users WHERE added_at < $1`, cutoffTime); err != nil {
		return err
	}

	_, err := s.pool.Exec(ctx, `DELETE FROM realms WHERE updated_at < $1`, cutoffTime)
	return err
}
