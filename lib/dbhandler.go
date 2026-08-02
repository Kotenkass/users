package dbhandler

import (
	"context"
	"crypto/tls"
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
	migrate "github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

const (
	DefaultPGHost     = "localhost"
	DefaultPGPort     = 5432
	DefaultPGUser     = "postgres"
	DefaultPGPassword = "postgres"
	DefaultPGDatabase = "users"
	DefaultPGSSLMode  = "disable"

	DefaultMigrationPath   = "migrations"
	DefaultMigrationsTable = "schema_migrations"
)

// DBHandler owns the go-pg connection and exposes common database operations.
//
// Domain-specific methods such as GetUserByID can be added to this type later.
type DBHandler struct {
	Conn *pg.DB
}

// Config contains database connection and migration settings.
type Config struct {
	// DSN is an optional PostgreSQL URL, for example:
	// postgres://user:password@localhost:5432/users?sslmode=disable.
	// When set, it takes precedence over Host, Port, User, Password, Database,
	// SSLMode, and ApplicationName.
	DSN string

	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	SSLMode         string
	ApplicationName string

	MigrationPath   string
	MigrationsTable string
}

// DefaultConfig returns sane local-development defaults.
func DefaultConfig() Config {
	return Config{
		Host:            DefaultPGHost,
		Port:            DefaultPGPort,
		User:            DefaultPGUser,
		Password:        DefaultPGPassword,
		Database:        DefaultPGDatabase,
		SSLMode:         DefaultPGSSLMode,
		ApplicationName: "users",
		MigrationPath:   DefaultMigrationPath,
		MigrationsTable: DefaultMigrationsTable,
	}
}

// ConfigFromEnv returns DefaultConfig overridden by environment variables.
//
// Supported variables:
//   - DATABASE_URL, PGURL, or DB_URL for a full PostgreSQL DSN
//   - PGHOST or DB_HOST
//   - PGPORT or DB_PORT
//   - PGUSER or DB_USER
//   - PGPASSWORD or DB_PASSWORD
//   - PGDATABASE or DB_NAME
//   - PGSSLMODE or DB_SSLMODE
//   - PGAPPNAME or DB_APPNAME
//   - MIGRATION_PATH or DB_MIGRATION_PATH
//   - MIGRATIONS_TABLE or DB_MIGRATIONS_TABLE
func ConfigFromEnv() Config {
	cfg := DefaultConfig()

	if value := firstEnv("DATABASE_URL", "PGURL", "DB_URL"); value != "" {
		cfg.DSN = value
	}

	if value := firstEnv("PGHOST", "DB_HOST"); value != "" {
		cfg.Host = value
	}
	if value := firstEnv("PGPORT", "DB_PORT"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			cfg.Port = parsed
		}
	}
	if value := firstEnv("PGUSER", "DB_USER"); value != "" {
		cfg.User = value
	}
	if value := firstEnv("PGPASSWORD", "DB_PASSWORD"); value != "" {
		cfg.Password = value
	}
	if value := firstEnv("PGDATABASE", "DB_NAME"); value != "" {
		cfg.Database = value
	}
	if value := firstEnv("PGSSLMODE", "DB_SSLMODE"); value != "" {
		cfg.SSLMode = value
	}
	if value := firstEnv("PGAPPNAME", "DB_APPNAME"); value != "" {
		cfg.ApplicationName = value
	}
	if value := firstEnv("MIGRATION_PATH", "DB_MIGRATION_PATH"); value != "" {
		cfg.MigrationPath = value
	}
	if value := firstEnv("MIGRATIONS_TABLE", "DB_MIGRATIONS_TABLE"); value != "" {
		cfg.MigrationsTable = value
	}

	return cfg
}

// ConnectPg connects to PostgreSQL using settings from ConfigFromEnv.
func (db *DBHandler) ConnectPg() error {
	return db.ConnectPgWithConfig(ConfigFromEnv())
}

// ConnectPgWithConfig connects to PostgreSQL using the provided config.
func (db *DBHandler) ConnectPgWithConfig(cfg Config) error {
	if db == nil {
		return fmt.Errorf("nil DBHandler")
	}

	options, err := pgOptions(cfg)
	if err != nil {
		return err
	}
	applyPoolDefaults(options)

	conn := pg.Connect(options)

	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := conn.Ping(pingCtx); err != nil {
		_ = conn.Close()
		db.Conn = nil
		return fmt.Errorf("ping postgres: %w", err)
	}

	db.Conn = conn
	return nil
}

// RunMigrations runs database migrations from cfg.MigrationPath using golang-migrate.
// The path must point to a directory containing migrations such as:
//
//	migrations/000001_create_users.up.sql
//	migrations/000001_create_users.down.sql
//
// go-pg does not expose a database/sql *sql.DB, so migrations open a short-lived
// lib/pq connection internally while application ORM calls continue to use db.Conn.
func (db *DBHandler) RunMigrations() error {
	return db.RunMigrationsWithConfig(ConfigFromEnv())
}

// RunMigrationsWithConfig runs database migrations using the provided config.
func (db *DBHandler) RunMigrationsWithConfig(cfg Config) error {
	if db == nil || db.Conn == nil {
		return fmt.Errorf("database is not connected")
	}

	cfg = cfg.withDefaults()

	migrationDB, err := sql.Open("postgres", cfg.migrationDSN())
	if err != nil {
		return fmt.Errorf("open postgres migration connection: %w", err)
	}
	defer migrationDB.Close()

	driver, err := postgres.WithInstance(migrationDB, &postgres.Config{
		MigrationsTable: cfg.MigrationsTable,
	})
	if err != nil {
		return fmt.Errorf("create postgres migrate driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", cfg.MigrationPath),
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("create migrations: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("run migrations: %w", err)
	}

	return nil
}

// Close closes the go-pg connection.
func (db *DBHandler) Close() error {
	if db == nil || db.Conn == nil {
		return nil
	}
	if err := db.Conn.Close(); err != nil {
		return err
	}
	db.Conn = nil
	return nil
}

// Model returns a go-pg ORM query builder for the provided model.
func (db *DBHandler) Model(model ...interface{}) *orm.Query {
	if db == nil || db.Conn == nil {
		return nil
	}
	return db.Conn.Model(model...)
}

// Begin starts a go-pg transaction.
func (db *DBHandler) Begin() (*pg.Tx, error) {
	if db == nil || db.Conn == nil {
		return nil, fmt.Errorf("database is not connected")
	}
	return db.Conn.Begin()
}

// Ping checks that the go-pg connection is alive.
func (db *DBHandler) Ping(ctx context.Context) error {
	if db == nil || db.Conn == nil {
		return fmt.Errorf("database is not connected")
	}
	return db.Conn.Ping(ctx)
}

func pgOptions(cfg Config) (*pg.Options, error) {
	if cfg.DSN != "" {
		return pg.ParseURL(cfg.DSN)
	}

	cfg = cfg.withDefaults()
	return &pg.Options{
		Addr:            net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)),
		User:            cfg.User,
		Password:        cfg.Password,
		Database:        cfg.Database,
		ApplicationName: cfg.ApplicationName,
		TLSConfig:       tlsConfigFromSSLMode(cfg.SSLMode),
	}, nil
}

func applyPoolDefaults(options *pg.Options) {
	options.PoolSize = envInt("DB_POOL_SIZE", options.PoolSize)
	options.MinIdleConns = envInt("DB_MIN_IDLE_CONNS", options.MinIdleConns)
	options.MaxRetries = envInt("DB_MAX_RETRIES", options.MaxRetries)
	options.DialTimeout = envDuration("DB_DIAL_TIMEOUT", options.DialTimeout)
	options.ReadTimeout = envDuration("DB_READ_TIMEOUT", options.ReadTimeout)
	options.WriteTimeout = envDuration("DB_WRITE_TIMEOUT", options.WriteTimeout)
	options.IdleTimeout = envDuration("DB_IDLE_TIMEOUT", options.IdleTimeout)
	options.IdleCheckFrequency = envDuration("DB_IDLE_CHECK_FREQUENCY", options.IdleCheckFrequency)
}

func (cfg Config) withDefaults() Config {
	if cfg.Host == "" {
		cfg.Host = DefaultPGHost
	}
	if cfg.Port == 0 {
		cfg.Port = DefaultPGPort
	}
	if cfg.User == "" {
		cfg.User = DefaultPGUser
	}
	if cfg.Password == "" {
		cfg.Password = DefaultPGPassword
	}
	if cfg.Database == "" {
		cfg.Database = DefaultPGDatabase
	}
	if cfg.SSLMode == "" {
		cfg.SSLMode = DefaultPGSSLMode
	}
	if cfg.ApplicationName == "" {
		cfg.ApplicationName = "users"
	}
	if cfg.MigrationPath == "" {
		cfg.MigrationPath = DefaultMigrationPath
	}
	if cfg.MigrationsTable == "" {
		cfg.MigrationsTable = DefaultMigrationsTable
	}
	return cfg
}

func (cfg Config) migrationDSN() string {
	if cfg.DSN != "" {
		return cfg.DSN
	}

	cfg = cfg.withDefaults()
	u := &url.URL{
		Scheme: "postgres",
		Host:   net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)),
		User:   url.UserPassword(cfg.User, cfg.Password),
		Path:   "/" + url.PathEscape(cfg.Database),
	}

	query := u.Query()
	query.Set("sslmode", cfg.SSLMode)
	if cfg.ApplicationName != "" {
		query.Set("application_name", cfg.ApplicationName)
	}
	u.RawQuery = query.Encode()

	return u.String()
}

func tlsConfigFromSSLMode(sslMode string) *tls.Config {
	switch strings.ToLower(sslMode) {
	case "", "disable":
		return nil
	case "allow", "prefer", "require":
		return &tls.Config{InsecureSkipVerify: true} //nolint:gosec
	case "verify-ca", "verify-full":
		return &tls.Config{}
	default:
		return nil
	}
}

func firstEnv(names ...string) string {
	for _, name := range names {
		if value := os.Getenv(name); value != "" {
			return value
		}
	}
	return ""
}

func envInt(name string, fallback int) int {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envDuration(name string, fallback time.Duration) time.Duration {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}
