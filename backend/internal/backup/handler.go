package backup

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Handler holds dependencies for backup admin endpoints.
type Handler struct {
	db  *gorm.DB
	cfg *BackupConfig
}

func NewHandler(db *gorm.DB, cfg *BackupConfig) *Handler {
	return &Handler{db: db, cfg: cfg}
}

// ListBackupsHandler — GET /admin/system/backups
// Query params: ?type=daily|weekly|monthly&limit=20
func (h *Handler) ListBackupsHandler(c *gin.Context) {
	q := h.db.Model(&BackupRecord{}).Order("created_at DESC")
	if t := c.Query("type"); t != "" {
		q = q.Where("backup_type = ?", t)
	}
	limit := 20
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	var records []BackupRecord
	q.Limit(limit).Find(&records)
	c.JSON(http.StatusOK, gin.H{"backups": records, "count": len(records)})
}

// TriggerBackupHandler — POST /admin/system/backups/trigger
// Body: {"type": "daily|weekly|monthly|manual"}
func (h *Handler) TriggerBackupHandler(c *gin.Context) {
	var body struct {
		Type string `json:"type" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	bType := BackupType(body.Type)
	switch bType {
	case BackupDaily, BackupWeekly, BackupMonthly, BackupManual:
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid type"})
		return
	}
	go func() {
		_, _ = RunFullBackup(h.db, h.cfg, bType)
	}()
	c.JSON(http.StatusAccepted, gin.H{"message": "backup queued", "type": bType})
}

// RestoreHandler — POST /admin/system/restore
// Body: {"backup_id": 42}
// Downloads, decrypts, and restores the selected backup.
func (h *Handler) RestoreHandler(c *gin.Context) {
	var body struct {
		BackupID uint `json:"backup_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var rec BackupRecord
	if err := h.db.First(&rec, body.BackupID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "backup record not found"})
		return
	}
	if rec.Status == BackupFailed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot restore a failed backup"})
		return
	}
	if !h.cfg.IsConfigured() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "S3 not configured"})
		return
	}

	// Run restore asynchronously — this is a heavy operation.
	go func() {
		s3 := NewS3Client(h.cfg.S3Endpoint, h.cfg.S3Region, h.cfg.S3Bucket, h.cfg.S3AccessKey, h.cfg.S3SecretKey)
		encrypted, err := s3.GetObject(rec.S3Key)
		if err != nil {
			h.db.Model(&rec).Update("error_msg", "restore download failed: "+err.Error())
			return
		}
		compressed, err := aesDecrypt(h.cfg.EncryptionKey, encrypted)
		if err != nil {
			h.db.Model(&rec).Update("error_msg", "restore decrypt failed: "+err.Error())
			return
		}
		raw, err := gzipDecompress(compressed)
		if err != nil {
			h.db.Model(&rec).Update("error_msg", "restore decompress failed: "+err.Error())
			return
		}
		tmpFile, err := TempBackupFile(raw, ".dump")
		if err != nil {
			h.db.Model(&rec).Update("error_msg", "restore tmp file failed: "+err.Error())
			return
		}
		defer CleanupTempFile(tmpFile)

		// Full pg_restore — overwrites current DB. USE WITH EXTREME CAUTION.
		if err := pgRestoreToSchema(h.cfg.DatabaseURL, tmpFile, "public"); err != nil {
			h.db.Model(&rec).Update("error_msg", "pg_restore failed: "+err.Error())
			return
		}
		h.db.Model(&rec).Update("status", BackupValidated)
	}()

	c.JSON(http.StatusAccepted, gin.H{
		"message":   "restore started — this operation runs in the background",
		"backup_id": rec.ID,
		"s3_key":    rec.S3Key,
	})
}

// ValidateHandler — POST /admin/system/backups/validate
func (h *Handler) ValidateHandler(c *gin.Context) {
	if !h.cfg.IsConfigured() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "S3 not configured"})
		return
	}
	result := ValidateLatestBackup(h.db, h.cfg)
	status := http.StatusOK
	if !result.Passed {
		status = http.StatusUnprocessableEntity
	}
	c.JSON(status, result)
}
