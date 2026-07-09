package models

// User 系统用户表
type User struct {
	ID        int64  `gorm:"primaryKey" json:"id"`
	Username  string `gorm:"type:varchar(191);uniqueIndex;not null" json:"username"`
	Password  string `gorm:"not null;default:''" json:"-"` // 存储 bcrypt 哈希，不对外暴露
	Nickname  string `gorm:"default:''" json:"nickname"`
	Role      string `gorm:"default:'admin'" json:"role"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}
