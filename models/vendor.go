package models

// Vendor 厂家表
type Vendor struct {
	ID          int64  `gorm:"primaryKey" json:"id"`
	Name        string `gorm:"not null" json:"name"`
	Code        string `gorm:"uniqueIndex;not null" json:"code"`
	Description string `gorm:"default:''" json:"description"`
	Enabled     int    `gorm:"default:1" json:"enabled"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}
