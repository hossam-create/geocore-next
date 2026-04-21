package ops

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CronSchedule struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	Name        string     `gorm:"uniqueIndex;not null" json:"name"`
	Description string     `json:"description"`
	Schedule    string     `gorm:"not null" json:"schedule"`
	Action      string     `gorm:"not null" json:"action"`
	Payload     string     `gorm:"type:text" json:"payload"`
	Enabled     bool       `gorm:"default:true" json:"enabled"`
	LastRunAt   *time.Time `json:"last_run_at"`
	LastRunOK   *bool      `json:"last_run_ok"`
	LastRunErr  string     `json:"last_run_err,omitempty"`
	NextRunAt   *time.Time `json:"next_run_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type AlertRule struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	Name        string     `gorm:"not null" json:"name"`
	Metric      string     `gorm:"not null" json:"metric"`
	Condition   string     `gorm:"not null" json:"condition"`
	Threshold   float64    `json:"threshold"`
	Window      string     `gorm:"default:'1h'" json:"window"`
	Enabled     bool       `gorm:"default:true" json:"enabled"`
	LastFiredAt *time.Time `json:"last_fired_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type OpsConfig struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	Key       string    `gorm:"uniqueIndex;not null" json:"key"`
	Value     string    `gorm:"type:text" json:"-"`
	IsSecret  bool      `gorm:"default:false" json:"is_secret"`
	UpdatedAt time.Time `json:"updated_at"`
	UpdatedBy string    `json:"updated_by"`
}

func (OpsConfig) TableName() string   { return "ops_configs" }
func (AlertRule) TableName() string   { return "ops_alert_rules" }
func (CronSchedule) TableName() string { return "ops_cron_schedules" }

type AlertHistory struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	RuleID    uuid.UUID `gorm:"type:uuid;index" json:"rule_id"`
	RuleName  string    `json:"rule_name"`
	Metric    string    `json:"metric"`
	Value     float64   `json:"value"`
	Threshold float64   `json:"threshold"`
	FiredAt   time.Time `json:"fired_at"`
}

func (AlertHistory) TableName() string { return "ops_alert_history" }

func AutoMigrateOps(db *gorm.DB) error {
	return db.AutoMigrate(&OpsConfig{}, &CronSchedule{}, &AlertRule{}, &AlertHistory{})
}
