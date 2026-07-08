package models

import (
	"log"
	"os"
	"path/filepath"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// ==================== GORM 模型 ====================

type Constant struct {
	ID          int64  `gorm:"primaryKey" json:"id"`
	Key         string `gorm:"uniqueIndex;not null" json:"key"`
	Value       string `gorm:"not null;default:''" json:"value"`
	Description string `gorm:"default:''" json:"description"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type DBConnection struct {
	ID           int64  `gorm:"primaryKey" json:"id"`
	Name         string `gorm:"not null" json:"name"`
	DBType       string `gorm:"column:db_type;not null;default:mysql" json:"db_type"`
	Host         string `gorm:"not null;default:''" json:"host"`
	Port         int    `gorm:"not null;default:3306" json:"port"`
	Username     string `gorm:"not null;default:''" json:"username"`
	Password     string `gorm:"not null;default:''" json:"password"`
	DatabaseName string `gorm:"not null;default:''" json:"database_name"`
	ExtraParams  string `gorm:"default:''" json:"extra_params"`
	Enabled      int    `gorm:"default:1" json:"enabled"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

type Vendor struct {
	ID          int64  `gorm:"primaryKey" json:"id"`
	Name        string `gorm:"not null" json:"name"`
	Code        string `gorm:"uniqueIndex;not null" json:"code"`
	Description string `gorm:"default:''" json:"description"`
	Enabled     int    `gorm:"default:1" json:"enabled"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type FTPAccount struct {
	ID         int64  `gorm:"primaryKey" json:"id"`
	VendorID   int64  `gorm:"column:vendor_id;not null;index" json:"vendor_id"`
	Name       string `gorm:"not null" json:"name"`
	Protocol   string `gorm:"not null;default:sftp" json:"protocol"`
	Host       string `gorm:"not null;default:''" json:"host"`
	Port       int    `gorm:"not null;default:22" json:"port"`
	Username   string `gorm:"not null;default:''" json:"username"`
	Password   string `gorm:"not null;default:''" json:"password"`
	RemotePath string `gorm:"default:/" json:"remote_path"`
	Enabled    int    `gorm:"default:1" json:"enabled"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`

	// 关联
	Vendor     *Vendor     `gorm:"foreignKey:VendorID" json:"-"`
	SQLTasks   []SQLTask   `gorm:"foreignKey:FTPAccountID" json:"-"`
}

type SQLTask struct {
	ID                  int64  `gorm:"primaryKey" json:"id"`
	VendorID            int64  `gorm:"column:vendor_id;not null;index" json:"vendor_id"`
	DBConnectionID      *int64 `gorm:"column:db_connection_id" json:"db_connection_id"`
	TaskName            string `gorm:"not null" json:"task_name"`
	SQLContent          string `gorm:"not null;default:''" json:"sql_content"`
	CSVFilenameTemplate string `gorm:"default:'{vendor_code}_{task_name}_{date}.csv'" json:"csv_filename_template"`
	CronExpression      string `gorm:"default:'0 2 * * *'" json:"cron_expression"`
	ExecutionMode       string `gorm:"default:export_only" json:"execution_mode"` // export_only, upload
	FTPAccountID        *int64 `gorm:"column:ftp_account_id" json:"ftp_account_id"`
	SortOrder           int    `gorm:"default:0" json:"sort_order"`
	Enabled             int    `gorm:"default:1" json:"enabled"`
	LastRunAt           string `json:"last_run_at"`
	LastStatus          string `json:"last_status"`
	CreatedAt           string `json:"created_at"`
	UpdatedAt           string `json:"updated_at"`

	// 关联
	Vendor        *Vendor        `gorm:"foreignKey:VendorID" json:"-"`
	DBConnection  *DBConnection  `gorm:"foreignKey:DBConnectionID" json:"-"`
	FTPAccount    *FTPAccount    `gorm:"foreignKey:FTPAccountID" json:"-"`
}

type SystemConfig struct {
	ID          int64  `gorm:"primaryKey" json:"id"`
	ConfigKey   string `gorm:"uniqueIndex;not null" json:"config_key"`
	ConfigValue string `gorm:"not null;default:''" json:"config_value"`
	Description string `gorm:"default:''" json:"description"`
	UpdatedAt   string `json:"updated_at"`
}

type ExportLog struct {
	ID            int64  `gorm:"primaryKey" json:"id"`
	TaskID        int64  `gorm:"column:task_id;not null;index" json:"task_id"`
	VendorID      int64  `gorm:"column:vendor_id;not null" json:"vendor_id"`
	Status        string `gorm:"not null;default:''" json:"status"` // success, failed
	ExecutionMode string `gorm:"default:''" json:"execution_mode"`
	CSVFilename   string `gorm:"default:''" json:"csv_filename"`
	FileSize      int64  `gorm:"default:0" json:"file_size"`
	RecordCount   int    `gorm:"default:0" json:"record_count"`
	ErrorMessage  string `gorm:"default:''" json:"error_message"`
	DurationMs    int64  `gorm:"default:0" json:"duration_ms"`
	StartedAt     string `json:"started_at"`
	FinishedAt    string `json:"finished_at"`
	CreatedAt     string `json:"created_at"`
}

// APIResponse 统一响应结构
type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Total   int64       `json:"total,omitempty"`
}

// InitDB 使用 GORM 初始化系统数据库
func InitDB(dbPath string) error {
	if dbPath == "" {
		dbPath = "data.db"
	}

	dir := filepath.Dir(dbPath)
	if dir != "" && dir != "." {
		os.MkdirAll(dir, 0755)
	}

	var err error
	DB, err = gorm.Open(sqlite.Open(dbPath+"?_journal_mode=WAL&_foreign_keys=on"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return err
	}

	sqlDB, _ := DB.DB()
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)

	log.Println("[DB] SQLite (GORM) 系统数据库连接成功:", dbPath)

	// AutoMigrate 自动建表
	if err := DB.AutoMigrate(
		&Constant{},
		&DBConnection{},
		&Vendor{},
		&FTPAccount{},
		&SQLTask{},
		&SystemConfig{},
		&ExportLog{},
	); err != nil {
		return err
	}

	initDefaultConfigs()
	return nil
}

func initDefaultConfigs() {
	type kv struct{ k, v, d string }
	defaults := []kv{
		{"backup_keep_count", "30", "保留备份文件的最大数量，超过则自动清理最旧的"},
		{"csv_output_dir", "./output", "CSV 导出文件存放目录"},
		{"backup_dir", "./backup", "文件备份目录"},
		{"csv_delimiter", ",", "CSV 字段分隔符，默认逗号"},
		{"csv_charset", "UTF-8", "CSV 文件字符集"},
		{"csv_bom", "true", "是否在 CSV 开头写入 UTF-8 BOM (true/false)"},
		{"date_format", "20060102", "文件名中的日期格式"},
		{"datetime_format", "20060102_150405", "文件名中的日期时间格式"},
		{"max_parallel_tasks", "3", "最大并行任务数"},
		{"notify_ding_enabled", "off", "钉钉失败提醒开关: on 开启 / off 关闭"},
		{"notify_ding_webhook", "", "钉钉机器人 Webhook 地址 (含 access_token)"},
		{"notify_ding_secret", "", "钉钉机器人加签密钥 (安全设置选择加签时填写，可空)"},
		{"notify_wx_enabled", "off", "企业微信失败提醒开关: on 开启 / off 关闭"},
		{"notify_wx_webhook", "", "企业微信群机器人 Webhook 地址 (含 key)"},
		{"notify_at", "", "失败提醒 @ 的成员手机号/userid，逗号分隔，@all 表示所有人"},
	}
	for _, d := range defaults {
		var cnt int64
		DB.Model(&SystemConfig{}).Where("config_key = ?", d.k).Count(&cnt)
		if cnt == 0 {
			DB.Create(&SystemConfig{ConfigKey: d.k, ConfigValue: d.v, Description: d.d})
		}
	}
}

func CloseDB() {
	if DB != nil {
		sqlDB, _ := DB.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}
}
