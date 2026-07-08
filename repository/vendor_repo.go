package repository

import (
	"strings"

	"data-exchange/models"
)

// VendorRepo 厂家数据访问
type VendorRepo struct{}

// NewVendorRepo 构建厂家仓储
func NewVendorRepo() *VendorRepo { return &VendorRepo{} }

// List 按关键字（名称/编码/描述）模糊搜索厂家（不分页，兼容旧调用）
func (r *VendorRepo) List(keyword string) ([]models.Vendor, error) {
	list, _, err := r.ListPaged(keyword, Pagination{Page: 1, PageSize: 100000})
	return list, err
}

// ListPaged 按关键字分页查询厂家，返回当前页数据与总记录数
func (r *VendorRepo) ListPaged(keyword string, p Pagination) ([]models.Vendor, int64, error) {
	p = p.Normalize()
	var list []models.Vendor
	var total int64
	query := models.DB.Model(&models.Vendor{})
	if kw := strings.TrimSpace(keyword); kw != "" {
		like := "%" + kw + "%"
		query = query.Where("name LIKE ? OR code LIKE ? OR description LIKE ?", like, like, like)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Order("id DESC").Offset(p.Offset()).Limit(p.PageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// Get 按 ID 获取厂家
func (r *VendorRepo) Get(id int64) (*models.Vendor, error) {
	var v models.Vendor
	if err := models.DB.First(&v, id).Error; err != nil {
		return nil, err
	}
	return &v, nil
}

// Create 新建厂家
func (r *VendorRepo) Create(v *models.Vendor) error {
	return models.DB.Create(v).Error
}

// Update 更新厂家
func (r *VendorRepo) Update(v *models.Vendor) error {
	return models.DB.Model(&models.Vendor{}).Where("id = ?", v.ID).Updates(map[string]interface{}{
		"name": v.Name, "code": v.Code, "description": v.Description, "enabled": v.Enabled,
	}).Error
}

// Delete 删除厂家及其关联任务、FTP 账号（级联）
func (r *VendorRepo) Delete(id int64) error {
	if err := models.DB.Where("vendor_id = ?", id).Delete(&models.SQLTask{}).Error; err != nil {
		return err
	}
	if err := models.DB.Where("vendor_id = ?", id).Delete(&models.FTPAccount{}).Error; err != nil {
		return err
	}
	return models.DB.Delete(&models.Vendor{}, id).Error
}
