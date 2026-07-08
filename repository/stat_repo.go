package repository

import "data-exchange/models"

// StatRepo 仪表盘统计计数数据访问
type StatRepo struct{}

// NewStatRepo 构建统计仓储
func NewStatRepo() *StatRepo { return &StatRepo{} }

// Counts 返回各维度计数：厂家数、启用任务数、启用FTP数、日志总数、成功数、失败数
func (r *StatRepo) Counts() (vendorCount, taskCount, ftpCount, logCount, successCount, failCount int64, err error) {
	if err = models.DB.Model(&models.Vendor{}).Count(&vendorCount).Error; err != nil {
		return
	}
	if err = models.DB.Model(&models.SQLTask{}).Where("enabled = 1").Count(&taskCount).Error; err != nil {
		return
	}
	if err = models.DB.Model(&models.FTPAccount{}).Where("enabled = 1").Count(&ftpCount).Error; err != nil {
		return
	}
	if err = models.DB.Model(&models.ExportLog{}).Count(&logCount).Error; err != nil {
		return
	}
	if err = models.DB.Model(&models.ExportLog{}).Where("status = 'success'").Count(&successCount).Error; err != nil {
		return
	}
	if err = models.DB.Model(&models.ExportLog{}).Where("status = 'failed'").Count(&failCount).Error; err != nil {
		return
	}
	return
}

// RecentLogs 获取最近若干条执行日志
func (r *StatRepo) RecentLogs(limit int) ([]models.ExportLog, error) {
	var logs []models.ExportLog
	if err := models.DB.Order("id DESC").Limit(limit).Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}
