package models

// User 系统用户表
type User struct {
	ID        int64  `gorm:"primaryKey" json:"id"`
	Username  string `gorm:"type:varchar(191);uniqueIndex;not null" json:"username"`
	Password  string `gorm:"not null;default:''" json:"-"` // 存储 bcrypt 哈希，不对外暴露
	Nickname  string `gorm:"default:''" json:"nickname"`
	Role      string `gorm:"default:'admin'" json:"role"` // admin 管理员（全部权限）/ viewer 只读
	Status    int    `gorm:"default:1" json:"status"`     // 1 启用 / 0 禁用
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}
