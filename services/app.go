package services

import (
	"data-exchange/models"
	"data-exchange/repository"
	"fmt"
	"strconv"
	"strings"
)

// App 聚合所有业务服务、执行器与工具能力，作为依赖注入的根。
// 由 NewApp 统一装配 repository / service / executor / scheduler，杜绝散落的包级函数直连。
type App struct {
	// 配置仓储（聚合根持有的共享实例，供各服务与工具方法复用）
	ConfigRepo *repository.SystemConfigRepo

	// 业务服务（持有各自依赖的仓储）
	Constant     *ConstantService
	DBConnection *DBConnectionService
	Vendor       *VendorService
	Task         *SQLTaskService
	FTP          *FTPAccountService
	Config       *SystemConfigService
	Log          *ExportLogService
	Dashboard    *DashboardService
	User         *UserService
	OpLog        *OperationLogService
	TaskHistory  *SQLTaskHistoryService

	// 执行引擎与调度
	Executor  *TaskExecutor
	Scheduler *Scheduler

	// 并发工作池（信号量控制）
	Pool *TaskWorkerPool
}

// NewApp 装配整个应用的服务层（repository → service → executor → scheduler）
func NewApp() *App {
	// 1. 仓储实例（共享单例）
	constantRepo := repository.NewConstantRepo()
	dbConnRepo := repository.NewDBConnectionRepo()
	vendorRepo := repository.NewVendorRepo()
	ftpRepo := repository.NewFTPAccountRepo()
	taskRepo := repository.NewSQLTaskRepo()
	cfgRepo := repository.NewSystemConfigRepo()
	logRepo := repository.NewExportLogRepo()
	statRepo := repository.NewStatRepo()
	userRepo := repository.NewUserRepo()
	opLogRepo := repository.NewOperationLogRepo()
	taskHistoryRepo := repository.NewSQLTaskHistoryRepo()

	// 2. 聚合根外壳（先建壳，便于业务服务反向引用 App 以获取配置/调度能力）
	app := &App{ConfigRepo: cfgRepo}

	// 3. 业务服务（持有各自依赖的仓储 + 反向引用 App）
	app.Constant = NewConstantService(constantRepo)
	app.DBConnection = NewDBConnectionService(dbConnRepo)
	app.Vendor = NewVendorService(vendorRepo)
	app.Task = NewSQLTaskService(app, taskRepo, dbConnRepo, ftpRepo, vendorRepo, taskHistoryRepo)
	app.FTP = NewFTPAccountService(ftpRepo, vendorRepo)
	app.Config = NewSystemConfigService(cfgRepo)
	app.Log = NewExportLogService(logRepo, taskRepo, vendorRepo)
	app.Dashboard = NewDashboardService(app, statRepo, taskRepo, vendorRepo)
	app.User = NewUserService(userRepo)
	app.OpLog = NewOperationLogService(opLogRepo)
	app.TaskHistory = NewSQLTaskHistoryService(taskHistoryRepo, taskRepo)

	// 4. 任务执行器（供 worker pool 与按名并发执行，反向引用 App 以获取配置/常量）
	executor := NewTaskExecutor(app, taskRepo, vendorRepo, logRepo, ftpRepo)
	app.Executor = executor

	// 5. 调度器与工作池
	app.Scheduler = NewScheduler(taskRepo)
	app.Pool = newWorkerPool(getMaxParallel(app), executor)
	SetTaskExecutor(executor)

	return app
}

// UserRepo 暴露用户仓储供鉴权使用
func (a *App) UserRepo() *repository.UserRepo { return repository.NewUserRepo() }

// ==================== 系统配置（App 方法，替代原包级函数） ====================

// GetConfig 读取配置项
func (a *App) GetConfig(key string) string { return a.ConfigRepo.Get(key) }

// SetConfig 写入配置项
func (a *App) SetConfig(key, value string) error { return a.ConfigRepo.Set(key, value) }

// EnsureDefaults 预置系统默认配置项（仅当 key 不存在时写入，不覆盖用户设置）
func (a *App) EnsureDefaults() {
	a.ConfigRepo.Ensure(ConfigKeyMaxTasksPerVendor, strconv.Itoa(defaultMaxTasksPerVendor), "每个厂家允许的最大 SQL 任务数")
}

// GetAllConfigs 列出全部配置
func (a *App) GetAllConfigs() ([]models.SystemConfig, error) { return a.ConfigRepo.ListAll() }

// GetConfigWithDefault 读取配置项，缺省时返回默认值
func (a *App) GetConfigWithDefault(key, defaultVal string) string {
	val := a.GetConfig(key)
	if val == "" {
		return defaultVal
	}
	return val
}

// ==================== 常量函数（App 方法，替代原包级函数） ====================

// GetAllConstants 列全部常量
func (a *App) GetAllConstants() ([]models.Constant, error) { return a.Constant.List() }

// GetConstantMap 构建 key->value 映射
func (a *App) GetConstantMap() (map[string]string, error) {
	constants, err := a.Constant.List()
	if err != nil {
		return nil, err
	}
	m := make(map[string]string, len(constants))
	for _, c := range constants {
		m[c.Key] = c.Value
	}
	return m, nil
}

// ReplaceConstants 将 SQL 中的 {{key}} 替换为常量值
func (a *App) ReplaceConstants(sqlContent string) string {
	constants, err := a.GetConstantMap()
	if err != nil {
		return sqlContent
	}
	result := sqlContent
	for key, value := range constants {
		result = strings.ReplaceAll(result, fmt.Sprintf("{{%s}}", key), value)
	}
	return result
}
