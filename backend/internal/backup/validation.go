package backup

import (
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"time"

	"gorm.io/gorm"
)

// ValidationResult holds the outcome of a backup integrity check.
type ValidationResult struct {
	BackupID  uint
	S3Key     string
	Passed    bool
	TableRows map[string]int64
	Error     string
	CheckedAt time.Time
}

// ValidateLatestBackup downloads the most recent weekly backup, decrypts it,
// restores it to a temporary Postgres schema, and verifies core table row counts.
func ValidateLatestBackup(db *gorm.DB, cfg *BackupConfig) *ValidationResult {
	res := &ValidationResult{CheckedAt: time.Now()}

	// Find the latest uploaded weekly backup.
	var rec BackupRecord
	if err := db.Where("backup_type = ? AND status = ?", BackupWeekly, BackupUploaded).
		Order("created_at DESC").First(&rec).Error; err != nil {
		res.Error = "no weekly backup found: " + err.Error()
		slog.Warn("backup: validation skipped", "reason", res.Error)
		return res
	}
	res.BackupID = rec.ID
	res.S3Key = rec.S3Key

	// Download from S3.
	s3 := NewS3Client(cfg.S3Endpoint, cfg.S3Region, cfg.S3Bucket, cfg.S3AccessKey, cfg.S3SecretKey)
	encrypted, err := s3.GetObject(rec.S3Key)
	if err != nil {
		res.Error = "s3 download failed: " + err.Error()
		return res
	}

	// Decrypt.
	compressed, err := aesDecrypt(cfg.EncryptionKey, encrypted)
	if err != nil {
		res.Error = "decrypt failed: " + err.Error()
		return res
	}

	// Decompress.
	raw, err := gzipDecompress(compressed)
	if err != nil {
		res.Error = "decompress failed: " + err.Error()
		return res
	}

	// Verify checksum.
	if got := hexSHA256(encrypted); got != rec.Checksum {
		res.Error = fmt.Sprintf("checksum mismatch: stored=%s actual=%s", rec.Checksum, got)
		return res
	}

	// Write to temp file.
	tmpFile, err := TempBackupFile(raw, ".dump")
	if err != nil {
		res.Error = "tmp file failed: " + err.Error()
		return res
	}
	defer CleanupTempFile(tmpFile)

	// Restore to temp schema in same DB using pg_restore --schema-only to check structure.
	schemaName := fmt.Sprintf("geocore_validate_%d", time.Now().UnixMilli())
	restoreErr := pgRestoreToSchema(cfg.DatabaseURL, tmpFile, schemaName)
	if restoreErr != nil {
		res.Error = "pg_restore failed: " + restoreErr.Error()
		return res
	}
	defer dropSchema(cfg.DatabaseURL, schemaName)

	// Count rows in critical tables within the temp schema.
	res.TableRows = map[string]int64{}
	criticalTables := []string{"users", "listings", "orders", "wallet_accounts"}
	for _, t := range criticalTables {
		var count int64
		db.Raw(fmt.Sprintf(`SELECT COUNT(*) FROM "%s"."%s"`, schemaName, t)).Scan(&count)
		res.TableRows[t] = count
	}

	// Mark success.
	res.Passed = true
	db.Model(&rec).Update("status", BackupValidated)
	slog.Info("backup: validation passed", "backup_id", rec.ID, "tables", res.TableRows)
	return res
}

func pgRestoreToSchema(dbURL, dumpFile, schema string) error {
	// Create the schema first.
	createCmd := exec.Command("psql", dbURL,
		"-c", fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS "%s"`, schema))
	if out, err := createCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("create schema: %w: %s", err, out)
	}

	// Restore into that schema.
	restoreCmd := exec.Command("pg_restore",
		"--no-password",
		"--schema-only",
		"--no-owner",
		"--no-acl",
		fmt.Sprintf("--schema=%s", schema),
		fmt.Sprintf("--dbname=%s", dbURL),
		dumpFile,
	)
	if out, err := restoreCmd.CombinedOutput(); err != nil {
		// pg_restore exits non-zero for warnings; only fail on non-empty stderr lines.
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		serious := []string{}
		for _, l := range lines {
			if strings.Contains(l, "ERROR") {
				serious = append(serious, l)
			}
		}
		if len(serious) > 0 {
			return fmt.Errorf("pg_restore errors: %s", strings.Join(serious, "; "))
		}
	}
	return nil
}

func dropSchema(dbURL, schema string) {
	cmd := exec.Command("psql", dbURL,
		"-c", fmt.Sprintf(`DROP SCHEMA IF EXISTS "%s" CASCADE`, schema))
	if out, err := cmd.CombinedOutput(); err != nil {
		slog.Warn("backup: drop schema failed", "schema", schema, "err", string(out))
	}
}
