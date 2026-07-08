package repository

import "data-exchange/models"

// SQLTaskRepo SQL 任务数据访问
type SQLTaskRepo struct{}

// NewSQLTaskRepo 构建任务仓储
func NewSQLTaskRepo() *SQLTaskRepo { return &SQLTaskRepo{} }

// ListByVendor 列出某厂家的全部任务（按 sort_order, id 排序）
func (r *SQLTaskRepo) ListByVendor(vendorID int64) ([]models.SQLTask, error) {
	var list []models.SQLTask
	if err := models.DB.Where("vendor_id = ?", vendorID).Order("sort_order, id").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// CountByVendor 统计某厂家的任务数
func (r *SQLTaskRepo) CountByVendor(vendorID int64) (int64, error) {
	var cnt int64
	if err := models.DB.Model(&models.SQLTask{}).Where("vendor_id = ?", vendorID).Count(&cnt).Error; err != nil {
		return 0, err
	}
	return cnt, nil
}

// Get 按 ID 获取任务
func (r *SQLTaskRepo) Get(id int64) (*models.SQLTask, error) {
	var t models.SQLTask
	if err := models.DB.First(&t, id).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

// Create 新建任务
func (r *SQLTaskRepo) Create(t *models.SQLTask) error {
	return models.DB.Create(t).Error
}

// Update 更新任务字段
func (r *SQLTaskRepo) Update(t *models.SQLTask) error {
	return models.DB.Model(&models.SQLTask{}).Where("id = ?", t.ID).Updates(map[string]interface{}{
		"vendor_id": t.VendorID, "db_connection_id": t.DBConnectionID,
		"task_name": t.TaskName, "sql_content": t.SQLContent,
		"csv_filename_template": t.CSVFilenameTemplate, "cron_expression": t.CronExpression,
		"execution_mode": t.ExecutionMode, "ftp_account_id": t.FTPAccountID,
		"sort_order": t.SortOrder, "enabled": t.Enabled,
	}).Error
}

// Delete 按 ID 删除任务
func (r *SQLTaskRepo) Delete(id int64) error {
	return models.DB.Delete(&models.SQLTask{}, id).Error
}

// SetEnabled 设置任务启用状态
func (r *SQLTaskRepo) SetEnabled(id int64, enabled int) error {
	return models.DB.Model(&models.SQLTask{}).Where("id = ?", id).Update("enabled", enabled).Error
}

// UpdateLastRun 更新任务最近执行信息
func (r *SQLTaskRepo) UpdateLastRun(id int64, lastRunAt, lastStatus string) error {
	return models.DB.Model(&models.SQLTask{}).Where("id = ?", id).Updates(map[string]interface{}{
		"last_run_at": lastRunAt,
		"last_status": lastStatus,
	}).Error
}

// ListEnabledByName 列出某名称且启用、含 cron 的任务（供按名并发执行）
func (r *SQLTaskRepo) ListEnabledByName(name string) ([]models.SQLTask, error) {
	var list []models.SQLTask
	if err := models.DB.Where("task_name = ? AND enabled = 1", name).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// LoadAllEnabled 加载全部启用且配置了 cron 的任务（供调度器）
func (r *SQLTaskRepo) LoadAllEnabled() ([]models.SQLTask, error) {
	var list []models.SQLTask
	if err := models.DB.Preload("Vendor").
		Where("enabled = 1 AND cron_expression != ''").
		Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}
