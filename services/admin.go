package services

import (
	"strings"

	"data-exchange/models"
	"data-exchange/repository"

	"golang.org/x/crypto/bcrypt"
)

// ==================== 用户 / 权限管理 ====================

// UserService 用户管理业务（账号权限管理）
type UserService struct{ repo *repository.UserRepo }

// NewUserService 构建用户服务
func NewUserService(repo *repository.UserRepo) *UserService { return &UserService{repo: repo} }

// ListPaged 分页查询用户
func (s *UserService) ListPaged(keyword string, page, pageSize int) ([]models.User, int64, error) {
	return s.repo.ListPaged(keyword, repository.Pagination{Page: page, PageSize: pageSize})
}

// Get 按 ID 获取用户
func (s *UserService) Get(id int64) (*models.User, error) { return s.repo.Get(id) }

// CreateParams 新增用户参数
type CreateParams struct {
	Username string
	Password string
	Nickname string
	Role     string
}

// Create 新增用户（用户名唯一、密码加密）
func (s *UserService) Create(p CreateParams) error {
	p.Username = strings.TrimSpace(p.Username)
	if p.Username == "" {
		return errInvalid("用户名不能为空")
	}
	if len(p.Password) < 6 {
		return errInvalid("密码长度至少 6 位")
	}
	if p.Role != "admin" && p.Role != "viewer" {
		p.Role = "viewer"
	}
	cnt, err := s.repo.CountByUsername(p.Username, 0)
	if err != nil {
		return err
	}
	if cnt > 0 {
		return errInvalid("用户名已存在")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(p.Password), bcrypt.DefaultCost)
	if err != nil {
		return errInvalid("密码加密失败: " + err.Error())
	}
	return s.repo.Create(&models.User{
		Username: p.Username,
		Password: string(hash),
		Nickname: p.Nickname,
		Role:     p.Role,
		Status:   1,
	})
}

// UpdateParams 更新用户参数
type UpdateParams struct {
	ID       int64
	Nickname string
	Role     string
	Status   int
}

// Update 更新用户基础信息（昵称、角色、状态）
func (s *UserService) Update(p UpdateParams) error {
	u, err := s.repo.Get(p.ID)
	if err != nil {
		return errInvalid("用户不存在")
	}
	if p.Role != "admin" && p.Role != "viewer" {
		p.Role = u.Role
	}
	// 保护：禁止把最后一个管理员降级或禁用，避免系统失去管理员
	if u.Role == "admin" && (p.Role != "admin" || p.Status == 0) {
		if last, e := s.isLastActiveAdmin(u.ID); e == nil && last {
			return errInvalid("系统必须保留至少一个启用的管理员")
		}
	}
	u.Nickname = p.Nickname
	u.Role = p.Role
	u.Status = p.Status
	return s.repo.Update(u)
}

// ResetPassword 重置指定用户密码
func (s *UserService) ResetPassword(id int64, newPassword string) error {
	if len(newPassword) < 6 {
		return errInvalid("密码长度至少 6 位")
	}
	if _, err := s.repo.Get(id); err != nil {
		return errInvalid("用户不存在")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return errInvalid("密码加密失败: " + err.Error())
	}
	return s.repo.UpdatePasswordByID(id, string(hash))
}

// Delete 删除用户（保护最后一个管理员）
func (s *UserService) Delete(id int64) error {
	u, err := s.repo.Get(id)
	if err != nil {
		return errInvalid("用户不存在")
	}
	if u.Role == "admin" {
		if last, e := s.isLastActiveAdmin(u.ID); e == nil && last {
			return errInvalid("系统必须保留至少一个启用的管理员")
		}
	}
	return s.repo.Delete(id)
}

// isLastActiveAdmin 判断 excludeID 之外是否还存在其他启用的管理员
func (s *UserService) isLastActiveAdmin(excludeID int64) (bool, error) {
	list, _, err := s.repo.ListPaged("", repository.Pagination{Page: 1, PageSize: 100000})
	if err != nil {
		return false, err
	}
	for _, u := range list {
		if u.ID != excludeID && u.Role == "admin" && u.Status == 1 {
			return false, nil
		}
	}
	return true, nil
}

// ==================== 操作日志 ====================

// OperationLogService 操作日志业务
type OperationLogService struct{ repo *repository.OperationLogRepo }

// NewOperationLogService 构建操作日志服务
func NewOperationLogService(repo *repository.OperationLogRepo) *OperationLogService {
	return &OperationLogService{repo: repo}
}

// Record 写入操作日志（失败仅忽略，不影响主流程）
func (s *OperationLogService) Record(l *models.OperationLog) {
	_ = s.repo.Create(l)
}

// List 分页查询操作日志
func (s *OperationLogService) List(username, keyword, success string, page, pageSize int) ([]models.OperationLog, int64, error) {
	return s.repo.List(repository.OperationLogFilter{
		Username: username, Keyword: keyword, Success: success,
		Pagination: repository.Pagination{Page: page, PageSize: pageSize},
	})
}

// Delete 删除单条操作日志
func (s *OperationLogService) Delete(id int64) error { return s.repo.Delete(id) }

// Clear 清空操作日志
func (s *OperationLogService) Clear() error { return s.repo.Clear() }

// ==================== SQL 任务历史 ====================

// SQLTaskHistoryService SQL 任务内容历史版本业务
type SQLTaskHistoryService struct {
	repo     *repository.SQLTaskHistoryRepo
	taskRepo *repository.SQLTaskRepo
}

// NewSQLTaskHistoryService 构建历史服务
func NewSQLTaskHistoryService(repo *repository.SQLTaskHistoryRepo, taskRepo *repository.SQLTaskRepo) *SQLTaskHistoryService {
	return &SQLTaskHistoryService{repo: repo, taskRepo: taskRepo}
}

// ListByTask 列出某任务的历史版本
func (s *SQLTaskHistoryService) ListByTask(taskID int64, page, pageSize int) ([]models.SQLTaskHistory, int64, error) {
	return s.repo.ListByTask(taskID, repository.Pagination{Page: page, PageSize: pageSize})
}

// Get 获取单个历史版本
func (s *SQLTaskHistoryService) Get(id int64) (*models.SQLTaskHistory, error) { return s.repo.Get(id) }

// Restore 将任务恢复到指定历史版本（恢复前会把当前版本再存一份历史，避免丢失现场）
func (s *SQLTaskHistoryService) Restore(historyID int64, changedBy string) (*models.SQLTask, error) {
	h, err := s.repo.Get(historyID)
	if err != nil {
		return nil, errInvalid("历史版本不存在")
	}
	task, err := s.taskRepo.Get(h.TaskID)
	if err != nil {
		return nil, errInvalid("任务不存在")
	}
	if task.SQLContent != h.SQLContent {
		// 保留恢复前的现场
		_ = s.repo.Create(&models.SQLTaskHistory{
			TaskID:     task.ID,
			TaskName:   task.TaskName,
			SQLContent: task.SQLContent,
			ChangedBy:  changedBy,
			Remark:     "恢复前快照",
		})
		if err := s.taskRepo.UpdateSQLContent(task.ID, h.SQLContent); err != nil {
			return nil, err
		}
		task.SQLContent = h.SQLContent
	}
	return task, nil
}

// RecordChange 在任务 SQL 内容变更前保存旧版本快照
func (s *SQLTaskHistoryService) RecordChange(task *models.SQLTask, changedBy, remark string) {
	_ = s.repo.Create(&models.SQLTaskHistory{
		TaskID:     task.ID,
		TaskName:   task.TaskName,
		SQLContent: task.SQLContent,
		ChangedBy:  changedBy,
		Remark:     remark,
	})
}
