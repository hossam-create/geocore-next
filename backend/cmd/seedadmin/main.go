// cmd/seedadmin — create or upsert a super_admin user for local development.
//
// Usage:
//   go run ./cmd/seedadmin --email=admin@geocore.local --password=ChangeMe123!
//
// Reads DATABASE_URL (or DB_* vars) from environment, same resolution as the
// main API binary. Idempotent: if a user with the same email already exists,
// its password is reset and role is elevated to super_admin.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/geocore-next/backend/internal/security"
)

func main() {
	email := flag.String("email", "admin@geocore.local", "admin email")
	password := flag.String("password", "ChangeMe123!", "admin password (≥8 chars)")
	name := flag.String("name", "Local Admin", "display name")
	flag.Parse()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		host := envOr("DB_HOST", "localhost")
		port := envOr("DB_PORT", "5432")
		user := envOr("DB_USER", "geocore")
		pass := envOr("DB_PASSWORD", "geocore_secret")
		db := envOr("DB_NAME", "geocore")
		ssl := envOr("DB_SSLMODE", "disable")
		dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			host, port, user, pass, db, ssl)
	}

	gormDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}

	hash, err := security.HashPassword(*password)
	if err != nil {
		log.Fatalf("hash password: %v", err)
	}

	// Upsert by email: either update existing row or create new.
	var existingID uuid.UUID
	row := gormDB.Raw(`SELECT id FROM users WHERE email = ?`, *email).Row()
	_ = row.Scan(&existingID)

	if existingID != uuid.Nil {
		if err := gormDB.Exec(`
			UPDATE users
			SET password_hash = ?, role = 'super_admin', is_active = TRUE,
			    is_banned = FALSE, email_verified = TRUE, name = ?, updated_at = ?
			WHERE id = ?`,
			hash, *name, time.Now(), existingID).Error; err != nil {
			log.Fatalf("update admin: %v", err)
		}
		fmt.Printf("✓ Updated existing user → super_admin\n")
		fmt.Printf("  id       : %s\n", existingID)
		fmt.Printf("  email    : %s\n", *email)
		fmt.Printf("  password : %s\n", *password)
		return
	}

	newID := uuid.New()
	if err := gormDB.Exec(`
		INSERT INTO users (id, name, email, password_hash, role, is_active, is_banned,
		                   email_verified, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'super_admin', TRUE, FALSE, TRUE, ?, ?)`,
		newID, *name, *email, hash, time.Now(), time.Now()).Error; err != nil {
		log.Fatalf("insert admin: %v", err)
	}

	fmt.Printf("✓ Created super_admin user\n")
	fmt.Printf("  id       : %s\n", newID)
	fmt.Printf("  email    : %s\n", *email)
	fmt.Printf("  password : %s\n", *password)
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
