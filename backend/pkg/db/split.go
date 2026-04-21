// Package db provides database connection management with read/write split.
package db

import (
	"context"
	"log/slog"
	"sync"

	"gorm.io/gorm"
)

// SplitDB manages separate read and write database connections.
type SplitDB struct {
	mu     sync.RWMutex
	write  *gorm.DB
	read   *gorm.DB
}

// NewSplitDB creates a read/write split database manager.
// If readDB is nil, all queries fall back to writeDB.
func NewSplitDB(writeDB, readDB *gorm.DB) *SplitDB {
	return &SplitDB{
		write: writeDB,
		read:  readDB,
	}
}

// Write returns the write (primary) database connection.
func (d *SplitDB) Write() *gorm.DB {
	return d.write
}

// Read returns the read (replica) database connection.
// Falls back to write if no replica is configured.
func (d *SplitDB) Read() *gorm.DB {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.read != nil {
		return d.read
	}
	return d.write
}

// SetRead updates the read replica connection (for failover).
func (d *SplitDB) SetRead(db *gorm.DB) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.read = db
	slog.Info("db: read replica updated")
}

// HealthCheck verifies both connections are alive.
func (d *SplitDB) HealthCheck(ctx context.Context) (writeOK, readOK bool) {
	if sqlDB, err := d.write.DB(); err == nil {
		writeOK = sqlDB.PingContext(ctx) == nil
	}
	if d.read != nil {
		if sqlDB, err := d.read.DB(); err == nil {
			readOK = sqlDB.PingContext(ctx) == nil
		}
	} else {
		readOK = writeOK // no replica = read goes to write
	}
	return
}
