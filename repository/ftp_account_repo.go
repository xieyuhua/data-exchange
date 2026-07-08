package repository

import "data-exchange/models"

// FTPAccountRepo FTP 账号数据访问
type FTPAccountRepo struct{}

// NewFTPAccountRepo 构建 FTP 仓储
func NewFTPAccountRepo() *FTPAccountRepo { return &FTPAccountRepo{} }

// List 列出 FTP 账号，可按 vendorID 过滤
func (r *FTPAccountRepo) List(vendorID string) ([]models.FTPAccount, error) {
	var list []models.FTPAccount
	query := models.DB.Order("id")
	if vendorID != "" {
		query = query.Where("vendor_id = ?", vendorID)
	}
	if err := query.Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// Get 按 ID 获取 FTP 账号
func (r *FTPAccountRepo) Get(id int64) (*models.FTPAccount, error) {
	var a models.FTPAccount
	if err := models.DB.First(&a, id).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

// Create 新建 FTP 账号
func (r *FTPAccountRepo) Create(a *models.FTPAccount) error {
	return models.DB.Create(a).Error
}

// Update 更新 FTP 账号
func (r *FTPAccountRepo) Update(a *models.FTPAccount) error {
	return models.DB.Model(&models.FTPAccount{}).Where("id = ?", a.ID).Updates(map[string]interface{}{
		"vendor_id": a.VendorID, "name": a.Name, "protocol": a.Protocol,
		"host": a.Host, "port": a.Port, "username": a.Username,
		"password": a.Password, "remote_path": a.RemotePath, "enabled": a.Enabled,
	}).Error
}

// Delete 按 ID 删除 FTP 账号
func (r *FTPAccountRepo) Delete(id int64) error {
	return models.DB.Delete(&models.FTPAccount{}, id).Error
}
