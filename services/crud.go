package services

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"data-exchange/models"
	"data-exchange/repository"
)

// ==================== 系统常量 ====================

// ConstantService 系统常量业务
type ConstantService struct{ repo *repository.ConstantRepo }

// NewConstantService 构建常量服务
func NewConstantService(repo *repository.ConstantRepo) *ConstantService { return &ConstantService{repo: repo} }

// List 列出全部常量
func (s *ConstantService) List() ([]models.Constant, error) { return s.repo.List() }

// Save 保存常量
func (s *ConstantService) Save(c *models.Constant) error { return s.repo.Save(c) }

// Delete 删除常量
func (s *ConstantService) Delete(id int64) error { return s.repo.Delete(id) }

// ==================== 数据库连接 ====================

// DBConnectionService 数据库连接业务
type DBConnectionService struct{ repo *repository.DBConnectionRepo }

// NewDBConnectionService 构建连接服务
func NewDBConnectionService(repo *repository.DBConnectionRepo) *DBConnectionService {
	return &DBConnectionService{repo: repo}
}

// List 列出全部数据库连接
func (s *DBConnectionService) List() ([]models.DBConnection, error) { return s.repo.List() }

// Save 保存（新增/更新）数据库连接
func (s *DBConnectionService) Save(c *models.DBConnection) error {
	if c.Name == "" {
		return errInvalid("连接名称不能为空")
	}
	if c.ID == 0 {
		return s.repo.Create(c)
	}
	return s.repo.Update(c)
}

// Delete 删除数据库连接
func (s *DBConnectionService) Delete(id int64) error { return s.repo.Delete(id) }

// ==================== 厂家 ====================

// VendorService 厂家业务
type VendorService struct{ repo *repository.VendorRepo }

// NewVendorService 构建厂家服务
func NewVendorService(repo *repository.VendorRepo) *VendorService { return &VendorService{repo: repo} }

// List 按关键字列出厂家
func (s *VendorService) List(keyword string) ([]models.Vendor, error) { return s.repo.List(keyword) }

// ListPaged 按关键字分页列出厂家，返回数据与总记录数
func (s *VendorService) ListPaged(keyword string, page, pageSize int) ([]models.Vendor, int64, error) {
	return s.repo.ListPaged(keyword, repository.Pagination{Page: page, PageSize: pageSize})
}

// Save 保存（新增/更新）厂家，校验名称/编码唯一性
func (s *VendorService) Save(v *models.Vendor) error {
	if v.Name == "" || v.Code == "" {
		return errInvalid("厂家名称和编码不能为空")
	}
	err := func() error {
		if v.ID == 0 {
			return s.repo.Create(v)
		}
		return s.repo.Update(v)
	}()
	if err != nil && strings.Contains(err.Error(), "UNIQUE") {
		return errInvalid("厂家编码已存在")
	}
	return err
}

// Delete 删除厂家（级联任务与 FTP）
func (s *VendorService) Delete(id int64) error { return s.repo.Delete(id) }

// ==================== SQL 任务 ====================

// ConfigKeyMaxTasksPerVendor 每个厂家允许的最大任务数配置键
const ConfigKeyMaxTasksPerVendor = "max_tasks_per_vendor"

// defaultMaxTasksPerVendor 任务数上限默认值（可在系统配置中修改）
const defaultMaxTasksPerVendor = 4

// VendorTasksResult 厂家任务列表结果（含关联名与上限）
type VendorTasksResult struct {
	Tasks []TaskWithNames `json:"data"`
	Max   int             `json:"max"`
}

// TaskWithNames 任务及其关联名称（供前端回显）
type TaskWithNames struct {
	models.SQLTask
	DBConnectionName string `json:"db_connection_name"`
	FTPAccountName   string `json:"ftp_account_name"`
}

// SQLTaskService SQL 任务业务
type SQLTaskService struct {
	app       *App
	repo      *repository.SQLTaskRepo
	dbRepo    *repository.DBConnectionRepo
	ftpRepo   *repository.FTPAccountRepo
	vendorRepo *repository.VendorRepo
}

// NewSQLTaskService 构建任务服务
func NewSQLTaskService(
	app *App,
	repo *repository.SQLTaskRepo,
	dbRepo *repository.DBConnectionRepo,
	ftpRepo *repository.FTPAccountRepo,
	vendorRepo *repository.VendorRepo,
) *SQLTaskService {
	return &SQLTaskService{app: app, repo: repo, dbRepo: dbRepo, ftpRepo: ftpRepo, vendorRepo: vendorRepo}
}

// maxTasksPerVendor 读取每个厂家允许的最大任务数（来自系统配置，缺省为默认4）
func (s *SQLTaskService) maxTasksPerVendor() int {
	n, err := strconv.Atoi(s.app.GetConfigWithDefault(ConfigKeyMaxTasksPerVendor, strconv.Itoa(defaultMaxTasksPerVendor)))
	if err != nil || n <= 0 {
		return defaultMaxTasksPerVendor
	}
	return n
}

// ListByVendor 列出某厂家的任务（填充关联名）
func (s *SQLTaskService) ListByVendor(vendorID int64) (*VendorTasksResult, error) {
	tasks, err := s.repo.ListByVendor(vendorID)
	if err != nil {
		return nil, err
	}
	result := &VendorTasksResult{Max: s.maxTasksPerVendor()}
	for _, t := range tasks {
		result.Tasks = append(result.Tasks, s.decorate(t))
	}
	return result, nil
}

// Get 获取单个任务详情（填充关联名）
func (s *SQLTaskService) Get(id int64) (*TaskWithNames, error) {
	t, err := s.repo.Get(id)
	if err != nil {
		return nil, err
	}
	twn := s.decorate(*t)
	return &twn, nil
}

// Save 保存（新增/更新）任务，包含上限校验与调度注册
func (s *SQLTaskService) Save(t *models.SQLTask) error {
	if t.ID == 0 {
		cnt, err := s.repo.CountByVendor(t.VendorID)
		if err != nil {
			return err
		}
		max := s.maxTasksPerVendor()
		if int(cnt) >= max {
			return errInvalid(fmt.Sprintf("每个厂家最多设置%d个SQL任务", max))
		}
		if err := s.repo.Create(t); err != nil {
			return err
		}
		if t.Enabled == 1 && t.CronExpression != "" {
			s.app.AddTaskToScheduler(t.ID, t.CronExpression)
		}
		return nil
	}
	if err := s.repo.Update(t); err != nil {
		return err
	}
	s.app.RemoveTaskFromScheduler(t.ID)
	if t.Enabled == 1 && t.CronExpression != "" {
		s.app.AddTaskToScheduler(t.ID, t.CronExpression)
	}
	return nil
}

// Delete 删除任务（并移除调度）
func (s *SQLTaskService) Delete(id int64) error {
	s.app.RemoveTaskFromScheduler(id)
	return s.repo.Delete(id)
}

// Toggle 切换任务启用状态，返回新状态
func (s *SQLTaskService) Toggle(id int64) (int, error) {
	t, err := s.repo.Get(id)
	if err != nil {
		return 0, err
	}
	newEnabled := 0
	if t.Enabled == 0 {
		newEnabled = 1
	}
	if err := s.repo.SetEnabled(id, newEnabled); err != nil {
		return 0, err
	}
	if newEnabled == 1 && t.CronExpression != "" {
		s.app.AddTaskToScheduler(id, t.CronExpression)
	} else {
		s.app.RemoveTaskFromScheduler(id)
	}
	return newEnabled, nil
}

func (s *SQLTaskService) decorate(t models.SQLTask) TaskWithNames {
	twn := TaskWithNames{SQLTask: t}
	if t.DBConnectionID != nil {
		if dbc, err := s.dbRepo.Get(*t.DBConnectionID); err == nil {
			twn.DBConnectionName = dbc.Name
		}
	}
	if t.FTPAccountID != nil {
		if fa, err := s.ftpRepo.Get(*t.FTPAccountID); err == nil {
			twn.FTPAccountName = fa.Name
		}
	}
	return twn
}

// ==================== FTP / SFTP 账号 ====================

// FTPWithVendor FTP 账号及其厂家名
type FTPWithVendor struct {
	models.FTPAccount
	VendorName string `json:"vendor_name"`
}

// FTPAccountService FTP 账号业务
type FTPAccountService struct {
	repo       *repository.FTPAccountRepo
	vendorRepo *repository.VendorRepo
}

// NewFTPAccountService 构建 FTP 服务
func NewFTPAccountService(repo *repository.FTPAccountRepo, vendorRepo *repository.VendorRepo) *FTPAccountService {
	return &FTPAccountService{repo: repo, vendorRepo: vendorRepo}
}

// List 列出 FTP 账号（填充厂家名）
func (s *FTPAccountService) List(vendorID string) ([]FTPWithVendor, error) {
	list, err := s.repo.List(vendorID)
	if err != nil {
		return nil, err
	}
	var result []FTPWithVendor
	for _, a := range list {
		fwv := FTPWithVendor{FTPAccount: a}
		if v, err := s.vendorRepo.Get(a.VendorID); err == nil {
			fwv.VendorName = v.Name
		}
		result = append(result, fwv)
	}
	return result, nil
}

// Save 保存（新增/更新）FTP 账号
func (s *FTPAccountService) Save(a *models.FTPAccount) error {
	if a.ID == 0 {
		return s.repo.Create(a)
	}
	return s.repo.Update(a)
}

// Delete 删除 FTP 账号
func (s *FTPAccountService) Delete(id int64) error { return s.repo.Delete(id) }

// ==================== 系统配置 ====================

// ConfigItem 配置项（前端提交结构）
type ConfigItem struct {
	ID          int64  `json:"id"`
	Key         string `json:"config_key"`
	Value       string `json:"config_value"`
	Description string `json:"description"`
}

// SystemConfigService 系统配置业务
type SystemConfigService struct{ repo *repository.SystemConfigRepo }

// NewSystemConfigService 构建配置服务
func NewSystemConfigService(repo *repository.SystemConfigRepo) *SystemConfigService {
	return &SystemConfigService{repo: repo}
}

// Save 批量保存配置（upsert + 内存缓存）
func (s *SystemConfigService) Save(items []ConfigItem) error {
	for _, item := range items {
		if item.Key == "" {
			return errInvalid("配置键名不能为空")
		}
		if err := s.repo.Upsert(repository.ConfigUpsert{
			ID: item.ID, Key: item.Key, Value: item.Value, Description: item.Description,
		}); err != nil {
			return err
		}
		if err := s.repo.Set(item.Key, item.Value); err != nil {
			return err
		}
	}
	return nil
}

// List 列出全部配置
func (s *SystemConfigService) List() ([]models.SystemConfig, error) { return s.repo.ListAll() }

// ==================== 执行日志 ====================

// LogWithNames 日志及其关联名称
type LogWithNames struct {
	models.ExportLog
	TaskName   string `json:"task_name"`
	VendorName string `json:"vendor_name"`
}

// ExportLogService 执行日志业务
type ExportLogService struct {
	repo       *repository.ExportLogRepo
	taskRepo   *repository.SQLTaskRepo
	vendorRepo *repository.VendorRepo
}

// NewExportLogService 构建日志服务
func NewExportLogService(
	repo *repository.ExportLogRepo,
	taskRepo *repository.SQLTaskRepo,
	vendorRepo *repository.VendorRepo,
) *ExportLogService {
	return &ExportLogService{repo: repo, taskRepo: taskRepo, vendorRepo: vendorRepo}
}

// List 分页查询日志（填充关联名）
func (s *ExportLogService) List(page, pageSize int, status, keyword string) ([]LogWithNames, int64, error) {
	logs, total, err := s.repo.List(repository.LogFilter{
		Status: status, Keyword: keyword,
		Pagination: repository.Pagination{Page: page, PageSize: pageSize},
	})
	if err != nil {
		return nil, 0, err
	}
	var result []LogWithNames
	for _, l := range logs {
		result = append(result, s.decorate(l))
	}
	return result, total, nil
}

// Delete 删除单条日志
func (s *ExportLogService) Delete(id int64) error { return s.repo.Delete(id) }

// Clear 清空全部日志
func (s *ExportLogService) Clear() error { return s.repo.Clear() }

func (s *ExportLogService) decorate(l models.ExportLog) LogWithNames {
	lwn := LogWithNames{ExportLog: l}
	if t, err := s.taskRepo.Get(l.TaskID); err == nil {
		lwn.TaskName = t.TaskName
	}
	if v, err := s.vendorRepo.Get(l.VendorID); err == nil {
		lwn.VendorName = v.Name
	}
	return lwn
}

// ==================== 仪表盘 ====================

// DashboardService 仪表盘业务
type DashboardService struct {
	app        *App
	statRepo   *repository.StatRepo
	taskRepo   *repository.SQLTaskRepo
	vendorRepo *repository.VendorRepo
}

// NewDashboardService 构建仪表盘服务
func NewDashboardService(app *App, statRepo *repository.StatRepo, taskRepo *repository.SQLTaskRepo, vendorRepo *repository.VendorRepo) *DashboardService {
	return &DashboardService{app: app, statRepo: statRepo, taskRepo: taskRepo, vendorRepo: vendorRepo}
}

// Stats 汇总仪表盘统计数据
func (s *DashboardService) Stats() (map[string]interface{}, error) {
	vendorCount, taskCount, ftpCount, logCount, successCount, failCount, err := s.statRepo.Counts()
	if err != nil {
		return nil, err
	}

	recentLogs, err := s.statRepo.RecentLogs(10)
	if err != nil {
		return nil, err
	}

	var recent []LogWithNames
	for _, l := range recentLogs {
		recent = append(recent, s.decorateLog(l))
	}

	backupKeep := s.app.GetConfigWithDefault("backup_keep_count", "30")

	return map[string]interface{}{
		"vendor_count":  vendorCount,
		"task_count":    taskCount,
		"ftp_count":     ftpCount,
		"log_count":     logCount,
		"success_count": successCount,
		"fail_count":    failCount,
		"recent_logs":   recent,
		"backup_keep":   backupKeep,
		"current_time":  time.Now().Format("2006-01-02 15:04:05"),
	}, nil
}

func (s *DashboardService) decorateLog(l models.ExportLog) LogWithNames {
	lwn := LogWithNames{ExportLog: l}
	if t, err := s.taskRepo.Get(l.TaskID); err == nil {
		lwn.TaskName = t.TaskName
	}
	if v, err := s.vendorRepo.Get(l.VendorID); err == nil {
		lwn.VendorName = v.Name
	}
	return lwn
}

// ==================== 业务错误 ====================

type invalidError struct{ msg string }

func (e *invalidError) Error() string { return e.msg }

func errInvalid(msg string) error { return &invalidError{msg: msg} }
