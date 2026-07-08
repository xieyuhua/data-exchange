package repository

import "data-exchange/models"

// ConfigUpsert 配置写入/更新请求
type ConfigUpsert struct {
	ID          int64
	Key         string
	Value       string
	Description string
}

// SystemConfigRepo 系统配置数据访问
type SystemConfigRepo struct{}

// NewSystemConfigRepo 构建配置仓储
func NewSystemConfigRepo() *SystemConfigRepo { return &SystemConfigRepo{} }

// Get 按 key 读取配置值，不存在返回空串
func (r *SystemConfigRepo) Get(key string) string {
	var c models.SystemConfig
	if err := models.DB.Where("config_key = ?", key).First(&c).Error; err != nil {
		return ""
	}
	return c.ConfigValue
}

// Set 写入配置（按 key upsert）
func (r *SystemConfigRepo) Set(key, value string) error {
	var c models.SystemConfig
	result := models.DB.Where("config_key = ?", key).First(&c)
	if result.Error != nil {
		c = models.SystemConfig{ConfigKey: key, ConfigValue: value}
		return models.DB.Create(&c).Error
	}
	return models.DB.Model(&models.SystemConfig{}).Where("id = ?", c.ID).Update("config_value", value).Error
}

// ListAll 列出全部配置
func (r *SystemConfigRepo) ListAll() ([]models.SystemConfig, error) {
	var list []models.SystemConfig
	if err := models.DB.Order("id").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// Ensure 仅当 key 不存在时插入默认配置，不覆盖已有设置
func (r *SystemConfigRepo) Ensure(key, value, desc string) error {
	var cnt int64
	if err := models.DB.Model(&models.SystemConfig{}).Where("config_key = ?", key).Count(&cnt).Error; err != nil {
		return err
	}
	if cnt > 0 {
		return nil
	}
	return models.DB.Create(&models.SystemConfig{ConfigKey: key, ConfigValue: value, Description: desc}).Error
}

// Upsert 按主键或 key 写入/更新配置
func (r *SystemConfigRepo) Upsert(item ConfigUpsert) error {
	if item.ID > 0 {
		return models.DB.Model(&models.SystemConfig{}).
			Where("id = ?", item.ID).
			Updates(map[string]interface{}{"config_value": item.Value, "description": item.Description}).Error
	}
	var cnt int64
	models.DB.Model(&models.SystemConfig{}).Where("config_key = ?", item.Key).Count(&cnt)
	if cnt > 0 {
		return models.DB.Model(&models.SystemConfig{}).
			Where("config_key = ?", item.Key).
			Updates(map[string]interface{}{"config_value": item.Value, "description": item.Description}).Error
	}
	return models.DB.Create(&models.SystemConfig{
		ConfigKey: item.Key, ConfigValue: item.Value, Description: item.Description,
	}).Error
}
