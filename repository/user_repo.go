package repository

import (
	"strings"
	"time"

	"data-exchange/models"
)

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

// Get 按 ID 获取用户
func (r *UserRepo) Get(id int64) (*models.User, error) {
	var u models.User
	if err := models.DB.First(&u, id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// UpdatePassword 更新指定用户的密码哈希
func (r *UserRepo) UpdatePassword(username, hash string) error {
	return models.DB.Model(&models.User{}).Where("username = ?", username).
		Update("password", hash).Error
}

// ListPaged 按关键字（用户名/昵称）分页查询用户，返回当前页数据与总记录数
func (r *UserRepo) ListPaged(keyword string, p Pagination) ([]models.User, int64, error) {
	p = p.Normalize()
	var list []models.User
	var total int64
	query := models.DB.Model(&models.User{})
	if kw := strings.TrimSpace(keyword); kw != "" {
		like := "%" + kw + "%"
		query = query.Where("username LIKE ? OR nickname LIKE ?", like, like)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Order("id ASC").Offset(p.Offset()).Limit(p.PageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// CountByUsername 统计同名用户数（用于唯一性校验，excludeID 为需排除的用户 ID）
func (r *UserRepo) CountByUsername(username string, excludeID int64) (int64, error) {
	var cnt int64
	q := models.DB.Model(&models.User{}).Where("username = ?", username)
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	if err := q.Count(&cnt).Error; err != nil {
		return 0, err
	}
	return cnt, nil
}

// Create 新建用户
func (r *UserRepo) Create(u *models.User) error {
	now := time.Now().Format("2006-01-02 15:04:05")
	u.CreatedAt = now
	u.UpdatedAt = now
	return models.DB.Create(u).Error
}

// Update 更新用户基础信息（不含密码）
func (r *UserRepo) Update(u *models.User) error {
	return models.DB.Model(&models.User{}).Where("id = ?", u.ID).Updates(map[string]interface{}{
		"nickname":   u.Nickname,
		"role":       u.Role,
		"status":     u.Status,
		"updated_at": time.Now().Format("2006-01-02 15:04:05"),
	}).Error
}

// UpdatePasswordByID 按 ID 重置密码哈希
func (r *UserRepo) UpdatePasswordByID(id int64, hash string) error {
	return models.DB.Model(&models.User{}).Where("id = ?", id).Updates(map[string]interface{}{
		"password":   hash,
		"updated_at": time.Now().Format("2006-01-02 15:04:05"),
	}).Error
}

// SetStatus 设置用户启用/禁用状态
func (r *UserRepo) SetStatus(id int64, status int) error {
	return models.DB.Model(&models.User{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":     status,
		"updated_at": time.Now().Format("2006-01-02 15:04:05"),
	}).Error
}

// Delete 按 ID 删除用户
func (r *UserRepo) Delete(id int64) error {
	return models.DB.Delete(&models.User{}, id).Error
}

// Count 统计用户总数
func (r *UserRepo) Count() (int64, error) {
	var cnt int64
	if err := models.DB.Model(&models.User{}).Count(&cnt).Error; err != nil {
		return 0, err
	}
	return cnt, nil
}
