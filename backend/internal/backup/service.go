package backup

import (
	"bytes"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"
)

// BackupType classifies retention bucket.
type BackupType string

const (
	BackupDaily   BackupType = "daily"
	BackupWeekly  BackupType = "weekly"
	BackupMonthly BackupType = "monthly"
	BackupManual  BackupType = "manual"
)

// BackupStatus tracks lifecycle.
type BackupStatus string

const (
	BackupPending   BackupStatus = "pending"
	BackupUploaded  BackupStatus = "uploaded"
	BackupValidated BackupStatus = "validated"
	BackupFailed    BackupStatus = "failed"
)

// BackupRecord persists metadata for every backup run.
type BackupRecord struct {
	ID         uint         `gorm:"primaryKey;autoIncrement"  json:"id"`
	Filename   string       `gorm:"size:256;not null;index"   json:"filename"`
	S3Key      string       `gorm:"size:512;not null"         json:"s3_key"`
	SizeBytes  int64        `gorm:"default:0"                 json:"size_bytes"`
	BackupType BackupType   `gorm:"size:16;not null;index"    json:"backup_type"`
	Status     BackupStatus `gorm:"size:16;not null"          json:"status"`
	Checksum   string       `gorm:"size:128"                  json:"checksum"`
	ErrorMsg   string       `gorm:"type:text"                 json:"error_msg,omitempty"`
	CreatedAt  time.Time    `json:"created_at"`
}

func (BackupRecord) TableName() string { return "backup_records" }

// BackupConfig holds all backup-related settings sourced from ENV.
type BackupConfig struct {
	DatabaseURL   string
	S3Endpoint    string
	S3Region      string
	S3Bucket      string
	S3AccessKey   string
	S3SecretKey   string
	S3Prefix      string
	EncryptionKey []byte // 32 bytes for AES-256-GCM
}

// BackupConfigFromEnv reads config from environment variables.
func BackupConfigFromEnv() *BackupConfig {
	keyB64 := os.Getenv("BACKUP_ENCRYPTION_KEY")
	key, _ := base64.StdEncoding.DecodeString(keyB64)
	if len(key) != 32 {
		key = make([]byte, 32) // zero key in dev (noop encryption)
	}
	prefix := os.Getenv("BACKUP_S3_PREFIX")
	if prefix == "" {
		prefix = "geocore-backups/"
	}
	endpoint := os.Getenv("BACKUP_S3_ENDPOINT")
	if endpoint == "" {
		endpoint = "https://s3.amazonaws.com"
	}
	region := os.Getenv("BACKUP_S3_REGION")
	if region == "" {
		region = "us-east-1"
	}
	return &BackupConfig{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		S3Endpoint:  endpoint,
		S3Region:    region,
		S3Bucket:    os.Getenv("BACKUP_S3_BUCKET"),
		S3AccessKey: os.Getenv("BACKUP_S3_KEY_ID"),
		S3SecretKey: os.Getenv("BACKUP_S3_KEY_SECRET"),
		S3Prefix:    prefix,
		EncryptionKey: key,
	}
}

// IsConfigured returns false when the S3 bucket is not set (skip uploads safely).
func (c *BackupConfig) IsConfigured() bool {
	return c.S3Bucket != "" && c.S3AccessKey != ""
}

// RunFullBackup executes pg_dump → gzip → AES-256-GCM encrypt → S3 upload.
func RunFullBackup(db *gorm.DB, cfg *BackupConfig, bType BackupType) (*BackupRecord, error) {
	now := time.Now().UTC()
	filename := fmt.Sprintf("%s_%s_%s.sql.gz.enc",
		bType, now.Format("2006-01-02T150405"), randomSuffix())
	s3Key := cfg.S3Prefix + string(bType) + "/" + filename

	record := &BackupRecord{
		Filename:   filename,
		S3Key:      s3Key,
		BackupType: bType,
		Status:     BackupPending,
		CreatedAt:  now,
	}
	if err := db.Create(record).Error; err != nil {
		return nil, fmt.Errorf("backup: failed to create record: %w", err)
	}

	slog.Info("backup: starting", "type", bType, "file", filename)

	// 1. pg_dump → raw bytes.
	dumpData, err := pgDump(cfg.DatabaseURL)
	if err != nil {
		markFailed(db, record, err)
		return record, fmt.Errorf("backup: pg_dump failed: %w", err)
	}

	// 2. Gzip compress.
	compressed, err := gzipCompress(dumpData)
	if err != nil {
		markFailed(db, record, err)
		return record, fmt.Errorf("backup: gzip failed: %w", err)
	}

	// 3. AES-256-GCM encrypt.
	encrypted, err := aesEncrypt(cfg.EncryptionKey, compressed)
	if err != nil {
		markFailed(db, record, err)
		return record, fmt.Errorf("backup: encrypt failed: %w", err)
	}

	// 4. Upload to S3.
	if cfg.IsConfigured() {
		s3 := NewS3Client(cfg.S3Endpoint, cfg.S3Region, cfg.S3Bucket, cfg.S3AccessKey, cfg.S3SecretKey)
		if err := s3.PutObject(s3Key, encrypted, "application/octet-stream"); err != nil {
			markFailed(db, record, err)
			return record, fmt.Errorf("backup: s3 upload failed: %w", err)
		}
	} else {
		slog.Warn("backup: S3 not configured, skipping upload")
	}

	record.SizeBytes = int64(len(encrypted))
	record.Checksum = hexSHA256(encrypted)
	record.Status = BackupUploaded
	db.Save(record)

	slog.Info("backup: completed", "type", bType, "size_bytes", record.SizeBytes, "key", s3Key)
	return record, nil
}

// ApplyRetentionPolicy deletes S3 objects + DB records older than the policy window.
// Policy: daily → 7 days, weekly → 28 days, monthly → 90 days.
func ApplyRetentionPolicy(db *gorm.DB, cfg *BackupConfig) {
	policy := map[BackupType]time.Duration{
		BackupDaily:   7 * 24 * time.Hour,
		BackupWeekly:  28 * 24 * time.Hour,
		BackupMonthly: 90 * 24 * time.Hour,
	}
	s3 := NewS3Client(cfg.S3Endpoint, cfg.S3Region, cfg.S3Bucket, cfg.S3AccessKey, cfg.S3SecretKey)
	for bType, retention := range policy {
		cutoff := time.Now().Add(-retention)
		var old []BackupRecord
		db.Where("backup_type = ? AND created_at < ?", bType, cutoff).Find(&old)
		for _, rec := range old {
			if cfg.IsConfigured() {
				if err := s3.DeleteObject(rec.S3Key); err != nil {
					slog.Warn("backup: retention delete failed", "key", rec.S3Key, "err", err)
					continue
				}
			}
			db.Delete(&rec)
			slog.Info("backup: retention cleaned", "key", rec.S3Key)
		}
	}
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func pgDump(dbURL string) ([]byte, error) {
	if dbURL == "" {
		dbURL = os.Getenv("DATABASE_URL")
	}
	cmd := exec.Command("pg_dump", "--no-password", "--format=custom", dbURL)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%w: %s", err, stderr.String())
	}
	return out.Bytes(), nil
}

func gzipCompress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return nil, err
	}
	if _, err := w.Write(data); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func gzipDecompress(data []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

func aesEncrypt(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func aesDecrypt(key, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(ciphertext) < gcm.NonceSize() {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce := ciphertext[:gcm.NonceSize()]
	return gcm.Open(nil, nonce, ciphertext[gcm.NonceSize():], nil)
}

func randomSuffix() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return strings.ToUpper(fmt.Sprintf("%x", b))
}

func markFailed(db *gorm.DB, rec *BackupRecord, err error) {
	rec.Status = BackupFailed
	rec.ErrorMsg = err.Error()
	db.Save(rec)
	slog.Error("backup: failed", "file", rec.Filename, "err", err)
}

// TempBackupFile writes bytes to a temp file and returns the path.
func TempBackupFile(data []byte, ext string) (string, error) {
	f, err := os.CreateTemp("", "geocore-restore-*"+ext)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := f.Write(data); err != nil {
		return "", err
	}
	return f.Name(), nil
}

// CleanupTempFile removes a temp file, ignoring errors.
func CleanupTempFile(path string) {
	_ = os.Remove(path)
	_ = os.Remove(filepath.Dir(path))
}
