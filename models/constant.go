package models

// Constant 系统常量表
type Constant struct {
	ID          int64  `gorm:"primaryKey" json:"id"`
	Key         string `gorm:"uniqueIndex;not null" json:"key"`
	Value       string `gorm:"not null;default:''" json:"value"`
	Description string `gorm:"default:''" json:"description"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}
