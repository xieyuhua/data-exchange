package models

// DBConnection 数据库连接表
type DBConnection struct {
	ID           int64  `gorm:"primaryKey" json:"id"`
	Name         string `gorm:"type:varchar(255);not null" json:"name"`
	DBType       string `gorm:"column:db_type;type:varchar(16);not null;default:'mysql'" json:"db_type"`
	Host         string `gorm:"type:varchar(255);not null;default:''" json:"host"`
	Port         int    `gorm:"not null;default:3306" json:"port"`
	Username     string `gorm:"type:varchar(255);not null;default:''" json:"username"`
	Password     string `gorm:"type:varchar(255);not null;default:''" json:"password"`
	DatabaseName string `gorm:"type:varchar(255);not null;default:''" json:"database_name"`
	ExtraParams  string `gorm:"type:text" json:"extra_params"`
	Enabled      int    `gorm:"default:1" json:"enabled"`
	CreatedAt    DateTime `gorm:"type:datetime" json:"created_at"`
	UpdatedAt    DateTime `gorm:"type:datetime" json:"updated_at"`
}
