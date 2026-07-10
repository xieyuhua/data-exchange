package models

// FTPAccount FTP/SFTP 账号表
type FTPAccount struct {
	ID         int64  `gorm:"primaryKey" json:"id"`
	VendorID   int64  `gorm:"column:vendor_id;not null;index" json:"vendor_id"`
	Name       string `gorm:"type:varchar(255);not null" json:"name"`
	Protocol   string `gorm:"type:varchar(16);not null;default:'sftp'" json:"protocol"`
	Host       string `gorm:"type:varchar(255);not null;default:''" json:"host"`
	Port       int    `gorm:"not null;default:22" json:"port"`
	Username   string `gorm:"type:varchar(255);not null;default:''" json:"username"`
	Password   string `gorm:"type:varchar(255);not null;default:''" json:"password"`
	RemotePath string `gorm:"type:varchar(255);default:'/'" json:"remote_path"`
	Enabled    int    `gorm:"default:1" json:"enabled"`
	CreatedAt  DateTime `gorm:"type:datetime" json:"created_at"`
	UpdatedAt  DateTime `gorm:"type:datetime" json:"updated_at"`

	// 关联
	Vendor   *Vendor   `gorm:"foreignKey:VendorID" json:"-"`
	SQLTasks []SQLTask `gorm:"foreignKey:FTPAccountID" json:"-"`
}
