package repository

import (
	"time"

	"data-exchange/models"
)

// SQLTaskHistoryRepo SQL 任务历史版本数据访问
type SQLTaskHistoryRepo struct{}

// NewSQLTaskHistoryRepo 构建历史仓储
func NewSQLTaskHistoryRepo() *SQLTaskHistoryRepo { return &SQLTaskHistoryRepo{} }

// Create 写入一条历史版本
func (r *SQLTaskHistoryRepo) Create(h *models.SQLTaskHistory) error {
	if h.CreatedAt.IsZero() {
		h.CreatedAt = models.DateTime(time.Now())
	}
	return models.DB.Create(h).Error
}

// ListByTask 列出某任务的历史版本（按时间倒序），支持分页
func (r *SQLTaskHistoryRepo) ListByTask(taskID int64, p Pagination) ([]models.SQLTaskHistory, int64, error) {
	p = p.Normalize()
	query := models.DB.Model(&models.SQLTaskHistory{}).Where("task_id = ?", taskID)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []models.SQLTaskHistory
	if err := query.Order("id DESC").Offset(p.Offset()).Limit(p.PageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// Get 按 ID 获取历史版本
func (r *SQLTaskHistoryRepo) Get(id int64) (*models.SQLTaskHistory, error) {
	var h models.SQLTaskHistory
	if err := models.DB.First(&h, id).Error; err != nil {
		return nil, err
	}
	return &h, nil
}

// DeleteByTask 删除某任务的全部历史（任务删除时级联）
func (r *SQLTaskHistoryRepo) DeleteByTask(taskID int64) error {
	return models.DB.Where("task_id = ?", taskID).Delete(&models.SQLTaskHistory{}).Error
}
