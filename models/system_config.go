package models

// SystemConfig 系统配置表
type SystemConfig struct {
	ID          int64  `gorm:"primaryKey" json:"id"`
	ConfigKey   string `gorm:"type:varchar(191);uniqueIndex;not null" json:"config_key"`
	ConfigValue string `gorm:"not null;default:''" json:"config_value"`
	Description string `gorm:"default:''" json:"description"`
	UpdatedAt   string `json:"updated_at"`
}
