package repository

import "data-exchange/models"

// ConstantRepo 系统常量数据访问
type ConstantRepo struct{}

// NewConstantRepo 构建常量仓储
func NewConstantRepo() *ConstantRepo { return &ConstantRepo{} }

// List 列出全部系统常量
func (r *ConstantRepo) List() ([]models.Constant, error) {
	var list []models.Constant
	if err := models.DB.Order("id").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// Save 新增或更新常量
func (r *ConstantRepo) Save(c *models.Constant) error {
	if c.ID == 0 {
		return models.DB.Create(c).Error
	}
	return models.DB.Model(&models.Constant{}).Where("id = ?", c.ID).Updates(map[string]interface{}{
		"key": c.Key, "value": c.Value, "description": c.Description,
	}).Error
}

// Delete 按 ID 删除常量
func (r *ConstantRepo) Delete(id int64) error {
	return models.DB.Delete(&models.Constant{}, id).Error
}
