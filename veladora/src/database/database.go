package database

import (
	"context"
	"fmt"
	"time"
	"veladora/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

var DB *pgxpool.Pool
var defaultQueryTimeout time.Duration

func InitDatabase(cfg *config.Config) error {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.DBName,
		cfg.Database.SSLMode,
	)

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return fmt.Errorf("failed to parse database config: %w", err)
	}

	poolConfig.MaxConns = 100
	poolConfig.MinConns = 10
	poolConfig.MaxConnLifetime = time.Hour * 8
	poolConfig.MaxConnIdleTime = time.Minute * 30

	if cfg.Database.QueryTimeout > 0 {
		defaultQueryTimeout = time.Duration(cfg.Database.QueryTimeout) * time.Second
		poolConfig.ConnConfig.Config.ConnectTimeout = defaultQueryTimeout
	} else {
		defaultQueryTimeout = 30 * time.Second
	}

	DB, err = pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := DB.Ping(context.Background()); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	if err := createTables(); err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	return nil
}

func createTables() error {
	ctx, cancel := DefaultTimeout(context.Background())
	defer cancel()

	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			username VARCHAR(255) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			balance INTEGER NOT NULL DEFAULT 2500,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS bills (
			id SERIAL PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			amount INTEGER NOT NULL DEFAULT 0,
			comment TEXT,
			status VARCHAR(50) NOT NULL DEFAULT 'active',
			payment_id VARCHAR(255) UNIQUE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`DO $$ 
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='users' AND column_name='payment_links') THEN
				ALTER TABLE users ADD COLUMN payment_links TEXT[];
			END IF;
		END $$`,
		`CREATE TABLE IF NOT EXISTS orders (
			id SERIAL PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			bill_id INTEGER REFERENCES bills(id) ON DELETE CASCADE,
			drink_name VARCHAR(255) NOT NULL,
			amount INTEGER NOT NULL,
			status VARCHAR(50) NOT NULL DEFAULT 'active',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS conversations (
			id SERIAL PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			content TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status)`,
		`CREATE INDEX IF NOT EXISTS idx_bills_user_id ON bills(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_conversations_user_id ON conversations(user_id)`,
		`DO $$ 
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='orders' AND column_name='bill_id') THEN
				ALTER TABLE orders ADD COLUMN bill_id INTEGER;
				ALTER TABLE orders ADD CONSTRAINT orders_bill_id_fkey FOREIGN KEY (bill_id) REFERENCES bills(id) ON DELETE CASCADE;
			END IF;
		END $$`,
		`DO $$ 
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='bills' AND column_name='status') THEN
				ALTER TABLE bills ADD COLUMN status VARCHAR(50) NOT NULL DEFAULT 'active';
			END IF;
		END $$`,
		`CREATE INDEX IF NOT EXISTS idx_orders_bill_id ON orders(bill_id)`,
		`CREATE INDEX IF NOT EXISTS idx_bills_status ON bills(status)`,
		`CREATE INDEX IF NOT EXISTS idx_bills_payment_id ON bills(payment_id)`,

		`CREATE OR REPLACE FUNCTION generate_payment_id(p_u VARCHAR, p_b INTEGER) RETURNS VARCHAR AS $$
		DECLARE
			vu VARCHAR;
			vp VARCHAR;
			vt INTEGER;
			vux VARCHAR;
			vbx INTEGER;
			vbl INTEGER;
			i INTEGER;
			hb VARCHAR(2);
			bv INTEGER;
			xl INTEGER;
		BEGIN
			vu := encode(convert_to(LOWER(p_u), 'UTF8'), 'hex');
			vt := FLOOR(RANDOM() * 16)::INTEGER;
			vbl := LENGTH(vu) * 4;
			vux := '';
			FOR i IN 1..LENGTH(vu) BY 2 LOOP
				hb := SUBSTRING(vu FROM i FOR 2);
				bv := ('x' || hb)::bit(8)::integer;
				xl := bv # vt;
				vux := vux || LPAD(TO_HEX(xl), 2, '0');
			END LOOP;
			vbx := p_b # vt;
			vp := vux || '_' || LPAD(TO_HEX(vbx), 4, '0') || '_' || LPAD(TO_HEX(vt), 2, '0');
			RETURN vp;
		END;
		$$ LANGUAGE plpgsql;`,
		`DO $$ 
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='bills' AND column_name='payment_id') THEN
				ALTER TABLE bills ADD COLUMN payment_id VARCHAR(255);
				CREATE UNIQUE INDEX IF NOT EXISTS idx_bills_payment_id ON bills(payment_id) WHERE payment_id IS NOT NULL;
			END IF;
		END $$`,
		`ALTER TABLE conversations DROP COLUMN IF EXISTS context_token`,
		`DROP INDEX IF EXISTS idx_conversations_token`,
		`ALTER TABLE bills DROP COLUMN IF EXISTS paid_by`,
		`DROP INDEX IF EXISTS idx_bills_paid_by`,
		`DROP TABLE IF EXISTS backups`,
	}

	for _, query := range queries {
		if _, err := DB.Exec(ctx, query); err != nil {
			return fmt.Errorf("failed to execute query: %w", err)
		}
	}

	return nil
}

func WithTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining < timeout {
			timeout = remaining
		}
	}
	return context.WithTimeout(ctx, timeout)
}

func DefaultTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return WithTimeout(ctx, defaultQueryTimeout)
}

func QueryContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return DefaultTimeout(ctx)
}

func Close() {
	if DB != nil {
		DB.Close()
	}
}
