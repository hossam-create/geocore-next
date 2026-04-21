package backup

import (
	"log/slog"
	"time"

	"gorm.io/gorm"
)

// StartBackupJobs launches:
//   - Daily full backup at midnight UTC
//   - Weekly backup every Sunday at 01:00 UTC
//   - Monthly backup on the 1st of each month at 02:00 UTC
//   - Weekly validation job every Sunday at 03:00 UTC
//   - Daily retention sweep at 04:00 UTC
//
// Returns a stop function that shuts down all tickers.
func StartBackupJobs(db *gorm.DB, cfg *BackupConfig, alert AlertFunc) func() {
	stop := make(chan struct{})

	go func() {
		for {
			now := time.Now().UTC()

			// Next midnight UTC.
			next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
			select {
			case <-time.After(time.Until(next)):
				runDaily(db, cfg, alert)
			case <-stop:
				return
			}
		}
	}()

	go func() {
		for {
			now := time.Now().UTC()
			daysUntilSunday := (int(time.Sunday) - int(now.Weekday()) + 7) % 7
			if daysUntilSunday == 0 {
				daysUntilSunday = 7
			}
			nextSunday := time.Date(now.Year(), now.Month(), now.Day()+daysUntilSunday, 1, 0, 0, 0, time.UTC)
			select {
			case <-time.After(time.Until(nextSunday)):
				runWeekly(db, cfg, alert)
			case <-stop:
				return
			}
		}
	}()

	go func() {
		for {
			now := time.Now().UTC()
			nextFirst := time.Date(now.Year(), now.Month()+1, 1, 2, 0, 0, 0, time.UTC)
			select {
			case <-time.After(time.Until(nextFirst)):
				runMonthly(db, cfg, alert)
			case <-stop:
				return
			}
		}
	}()

	go func() {
		for {
			now := time.Now().UTC()
			daysUntilSunday := (int(time.Sunday) - int(now.Weekday()) + 7) % 7
			if daysUntilSunday == 0 {
				daysUntilSunday = 7
			}
			nextValidation := time.Date(now.Year(), now.Month(), now.Day()+daysUntilSunday, 3, 0, 0, 0, time.UTC)
			select {
			case <-time.After(time.Until(nextValidation)):
				runValidation(db, cfg, alert)
			case <-stop:
				return
			}
		}
	}()

	go func() {
		for {
			now := time.Now().UTC()
			nextRetention := time.Date(now.Year(), now.Month(), now.Day()+1, 4, 0, 0, 0, time.UTC)
			select {
			case <-time.After(time.Until(nextRetention)):
				if cfg.IsConfigured() {
					ApplyRetentionPolicy(db, cfg)
				}
			case <-stop:
				return
			}
		}
	}()

	slog.Info("backup: jobs scheduled (daily/weekly/monthly/validation/retention)")
	return func() { close(stop) }
}

// AlertFunc is a callback for backup events (wired to security/alerting).
type AlertFunc func(event, message string)

func runDaily(db *gorm.DB, cfg *BackupConfig, alert AlertFunc) {
	slog.Info("backup: running daily backup")
	_, err := RunFullBackup(db, cfg, BackupDaily)
	if err != nil && alert != nil {
		alert("backup_failure", "Daily backup failed: "+err.Error())
	}
}

func runWeekly(db *gorm.DB, cfg *BackupConfig, alert AlertFunc) {
	slog.Info("backup: running weekly backup")
	_, err := RunFullBackup(db, cfg, BackupWeekly)
	if err != nil && alert != nil {
		alert("backup_failure", "Weekly backup failed: "+err.Error())
	}
}

func runMonthly(db *gorm.DB, cfg *BackupConfig, alert AlertFunc) {
	slog.Info("backup: running monthly backup")
	_, err := RunFullBackup(db, cfg, BackupMonthly)
	if err != nil && alert != nil {
		alert("backup_failure", "Monthly backup failed: "+err.Error())
	}
}

func runValidation(db *gorm.DB, cfg *BackupConfig, alert AlertFunc) {
	slog.Info("backup: running weekly validation")
	if !cfg.IsConfigured() {
		return
	}
	res := ValidateLatestBackup(db, cfg)
	if !res.Passed && alert != nil {
		alert("restore_failure", "Backup validation failed: "+res.Error)
	}
}
