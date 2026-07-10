package models

// Constant 系统常量表
type Constant struct {
	ID          int64  `gorm:"primaryKey" json:"id"`
	Key         string `gorm:"type:varchar(191);uniqueIndex;not null" json:"key"`
	Value       string `gorm:"not null;default:''" json:"value"`
	Description string `gorm:"default:''" json:"description"`
	CreatedAt   DateTime `gorm:"type:datetime" json:"created_at"`
	UpdatedAt   DateTime `gorm:"type:datetime" json:"updated_at"`
}
