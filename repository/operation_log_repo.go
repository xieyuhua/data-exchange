package repository

import (
	"strings"
	"time"

	"data-exchange/models"
)

// OperationLogFilter 操作日志查询过滤条件
type OperationLogFilter struct {
	Username string
	Keyword  string
	Success  string // "1" / "0" / ""
	Pagination
}

// OperationLogRepo 操作日志数据访问
type OperationLogRepo struct{}

// NewOperationLogRepo 构建操作日志仓储
func NewOperationLogRepo() *OperationLogRepo { return &OperationLogRepo{} }

// Create 写入一条操作日志
func (r *OperationLogRepo) Create(l *models.OperationLog) error {
	if l.CreatedAt == "" {
		l.CreatedAt = time.Now().Format("2006-01-02 15:04:05")
	}
	return models.DB.Create(l).Error
}

// List 分页查询操作日志，返回列表与总记录数
func (r *OperationLogRepo) List(f OperationLogFilter) ([]models.OperationLog, int64, error) {
	f.Pagination = f.Pagination.Normalize()
	query := models.DB.Model(&models.OperationLog{})
	if u := strings.TrimSpace(f.Username); u != "" {
		query = query.Where("username = ?", u)
	}
	if f.Success == "1" {
		query = query.Where("success = 1")
	} else if f.Success == "0" {
		query = query.Where("success = 0")
	}
	if kw := strings.TrimSpace(f.Keyword); kw != "" {
		like := "%" + kw + "%"
		query = query.Where("action LIKE ? OR module LIKE ? OR path LIKE ? OR detail LIKE ? OR username LIKE ?",
			like, like, like, like, like)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []models.OperationLog
	if err := query.Order("id DESC").Offset(f.Offset()).Limit(f.PageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// Delete 按 ID 删除操作日志
func (r *OperationLogRepo) Delete(id int64) error {
	return models.DB.Delete(&models.OperationLog{}, id).Error
}

// Clear 清空全部操作日志
func (r *OperationLogRepo) Clear() error {
	return models.DB.Where("1 = 1").Delete(&models.OperationLog{}).Error
}
