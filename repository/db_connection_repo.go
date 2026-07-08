package repository

import "data-exchange/models"

// DBConnectionRepo 数据库连接数据访问
type DBConnectionRepo struct{}

// NewDBConnectionRepo 构建连接仓储
func NewDBConnectionRepo() *DBConnectionRepo { return &DBConnectionRepo{} }

// List 列出全部数据库连接
func (r *DBConnectionRepo) List() ([]models.DBConnection, error) {
	var list []models.DBConnection
	if err := models.DB.Order("id DESC").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// Get 按 ID 获取数据库连接
func (r *DBConnectionRepo) Get(id int64) (*models.DBConnection, error) {
	var c models.DBConnection
	if err := models.DB.First(&c, id).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

// Create 新建数据库连接
func (r *DBConnectionRepo) Create(c *models.DBConnection) error {
	return models.DB.Create(c).Error
}

// Update 更新数据库连接（指定字段）
func (r *DBConnectionRepo) Update(c *models.DBConnection) error {
	return models.DB.Model(&models.DBConnection{}).Where("id = ?", c.ID).Updates(map[string]interface{}{
		"name": c.Name, "db_type": c.DBType, "host": c.Host,
		"port": c.Port, "username": c.Username, "password": c.Password,
		"database_name": c.DatabaseName, "extra_params": c.ExtraParams, "enabled": c.Enabled,
	}).Error
}

// Delete 按 ID 删除数据库连接
func (r *DBConnectionRepo) Delete(id int64) error {
	return models.DB.Delete(&models.DBConnection{}, id).Error
}
