package models

// DBConnection 数据库连接表
type DBConnection struct {
	ID           int64  `gorm:"primaryKey" json:"id"`
	Name         string `gorm:"not null" json:"name"`
	DBType       string `gorm:"column:db_type;not null;default:mysql" json:"db_type"`
	Host         string `gorm:"not null;default:''" json:"host"`
	Port         int    `gorm:"not null;default:3306" json:"port"`
	Username     string `gorm:"not null;default:''" json:"username"`
	Password     string `gorm:"not null;default:''" json:"password"`
	DatabaseName string `gorm:"not null;default:''" json:"database_name"`
	ExtraParams  string `gorm:"default:''" json:"extra_params"`
	Enabled      int    `gorm:"default:1" json:"enabled"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}
