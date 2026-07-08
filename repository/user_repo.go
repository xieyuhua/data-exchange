package repository

import "data-exchange/models"

// UserRepo 用户数据访问
type UserRepo struct{}

// NewUserRepo 构建用户仓储
func NewUserRepo() *UserRepo { return &UserRepo{} }

// GetByUsername 按用户名获取用户（用于登录校验）
func (r *UserRepo) GetByUsername(username string) (*models.User, error) {
	var u models.User
	if err := models.DB.Where("username = ?", username).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}
