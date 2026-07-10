package models

// Vendor 厂家表
type Vendor struct {
	ID          int64  `gorm:"primaryKey" json:"id"`
	Name        string `gorm:"type:varchar(255);not null" json:"name"`
	Code        string `gorm:"type:varchar(191);uniqueIndex;not null" json:"code"`
	Description string `gorm:"type:varchar(512);default:''" json:"description"`
	Enabled     int    `gorm:"type:int;default:1" json:"enabled"`
	CreatedAt   DateTime `gorm:"type:datetime" json:"created_at"`
	UpdatedAt   DateTime `gorm:"type:datetime" json:"updated_at"`
}
