package repository

import (
	"strings"

	"data-exchange/models"
)

// LogFilter 日志查询过滤条件
type LogFilter struct {
	Status string
	Keyword string
	Pagination
}

// ExportLogRepo 执行日志数据访问
type ExportLogRepo struct{}

// NewExportLogRepo 构建日志仓储
func NewExportLogRepo() *ExportLogRepo { return &ExportLogRepo{} }

// List 分页查询执行日志，返回日志列表与总数
func (r *ExportLogRepo) List(f LogFilter) ([]models.ExportLog, int64, error) {
	f.Pagination = f.Pagination.Normalize()
	query := models.DB.Model(&models.ExportLog{})
	if f.Status != "" {
		query = query.Where("export_logs.status = ?", f.Status)
	}
	if kw := strings.TrimSpace(f.Keyword); kw != "" {
		like := "%" + kw + "%"
		query = query.Joins("LEFT JOIN sql_tasks ON sql_tasks.id = export_logs.task_id").
			Joins("LEFT JOIN vendors ON vendors.id = export_logs.vendor_id").
			Where("export_logs.csv_filename LIKE ? OR export_logs.error_message LIKE ? OR sql_tasks.task_name LIKE ? OR vendors.name LIKE ?", like, like, like, like)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var logs []models.ExportLog
	if err := query.Order("export_logs.id DESC").Offset(f.Offset()).Limit(f.PageSize).Find(&logs).Error; err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}

// Get 按 ID 获取日志
func (r *ExportLogRepo) Get(id int64) (*models.ExportLog, error) {
	var l models.ExportLog
	if err := models.DB.First(&l, id).Error; err != nil {
		return nil, err
	}
	return &l, nil
}

// Delete 按 ID 删除日志
func (r *ExportLogRepo) Delete(id int64) error {
	return models.DB.Delete(&models.ExportLog{}, id).Error
}

// Clear 清空全部日志
func (r *ExportLogRepo) Clear() error {
	return models.DB.Where("1 = 1").Delete(&models.ExportLog{}).Error
}

// Create 写入执行日志
func (r *ExportLogRepo) Create(l *models.ExportLog) error {
	return models.DB.Create(l).Error
}
