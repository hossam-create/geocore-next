package monetization

import "gorm.io/gorm"

// defaultCommissionRate is used when no PlatformSettings row exists yet.
const defaultCommissionRate = 0.05

// GetSettings returns the platform settings row, creating it with defaults
// if it does not yet exist (idempotent).
func GetSettings(db *gorm.DB) PlatformSettings {
	var s PlatformSettings
	if err := db.First(&s).Error; err != nil {
		// Row doesn't exist — return defaults without writing, so we never
		// depend on a successful DB write during a read path.
		return PlatformSettings{
			CommissionRate: defaultCommissionRate,
			BoostFeeUSD:    BoostFee,
		}
	}
	return s
}

// SeedDefaultSettings inserts the default row if the table is empty.
// Called once at startup after AutoMigrate.
func SeedDefaultSettings(db *gorm.DB) {
	var count int64
	db.Model(&PlatformSettings{}).Count(&count)
	if count == 0 {
		db.Create(&PlatformSettings{
			CommissionRate: defaultCommissionRate,
			BoostFeeUSD:    BoostFee,
		})
	}
}
