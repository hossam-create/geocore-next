package db

import (
	"fmt"
	"log/slog"

	"gorm.io/gorm"
)

// MigrationRule: NEVER drop columns, NEVER rename directly.
// ALWAYS add + dual-write. This helper enforces expand-safe migrations.

// AddColumnIfNotExists adds a column to a table only if it doesn't already exist.
// This is the safe alternative to AutoMigrate which can cause unexpected schema changes.
func AddColumnIfNotExists(db *gorm.DB, tableName, column, typ string) error {
	if db.Migrator().HasColumn(tableName, column) {
		slog.Debug("db: column already exists, skipping", "table", tableName, "column", column)
		return nil
	}
	sql := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", tableName, column, typ)
	if err := db.Exec(sql).Error; err != nil {
		slog.Error("db: failed to add column", "table", tableName, "column", column, "error", err)
		return err
	}
	slog.Info("db: column added", "table", tableName, "column", column, "type", typ)
	return nil
}

// AddIndexIfNotExists creates an index only if it doesn't already exist.
func AddIndexIfNotExists(db *gorm.DB, tableName, indexName, columns string) error {
	if db.Migrator().HasIndex(tableName, indexName) {
		slog.Debug("db: index already exists, skipping", "table", tableName, "index", indexName)
		return nil
	}
	sql := fmt.Sprintf("CREATE INDEX %s ON %s (%s)", indexName, tableName, columns)
	if err := db.Exec(sql).Error; err != nil {
		slog.Error("db: failed to create index", "table", tableName, "index", indexName, "error", err)
		return err
	}
	slog.Info("db: index created", "table", tableName, "index", indexName)
	return nil
}

// CreateTableIfNotExists creates a table from a GORM model only if it doesn't exist.
func CreateTableIfNotExists(db *gorm.DB, model interface{}) error {
	if db.Migrator().HasTable(model) {
		return nil
	}
	if err := db.AutoMigrate(model); err != nil {
		slog.Error("db: failed to create table", "error", err)
		return err
	}
	slog.Info("db: table created", "model", fmt.Sprintf("%T", model))
	return nil
}

// SafeMigrate runs a set of expand-safe migrations.
// Each migration is idempotent — safe to run multiple times.
func SafeMigrate(db *gorm.DB, migrations []func(*gorm.DB) error) error {
	for i, m := range migrations {
		if err := m(db); err != nil {
			return fmt.Errorf("migration %d failed: %w", i, err)
		}
	}
	slog.Info("db: all migrations applied successfully")
	return nil
}
