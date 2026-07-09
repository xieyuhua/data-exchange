package models

import (
	"log"
	"os"
	"path/filepath"

	"data-exchange/config"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB 全局数据库连接（由 InitDB 初始化）
var DB *gorm.DB

// APIResponse 统一响应结构
type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Total   int64       `json:"total,omitempty"`
}

// InitDB 使用 GORM 初始化系统数据库，支持 sqlite 与 mysql（由配置决定）
func InitDB(dbCfg config.DatabaseConfig) error {
	var err error
	switch dbCfg.Type {
	case "mysql":
		dsn := dbCfg.DSN()
		DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Warn),
		})
		if err != nil {
			return err
		}
		log.Println("[DB] MySQL (GORM) 系统数据库连接成功:", dbCfg.MySQL.Host+":"+itoa(dbCfg.MySQL.Port)+"/"+dbCfg.MySQL.Database)
	default:
		// sqlite
		dbPath := dbCfg.SQLitePath
		if dbPath == "" {
			dbPath = "data.db"
		}
		dir := filepath.Dir(dbPath)
		if dir != "" && dir != "." {
			os.MkdirAll(dir, 0755)
		}
		DB, err = gorm.Open(sqlite.Open(dbPath+"?_journal_mode=WAL&_foreign_keys=on"), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Warn),
		})
		if err != nil {
			return err
		}
		log.Println("[DB] SQLite (GORM) 系统数据库连接成功:", dbPath)
	}

	sqlDB, _ := DB.DB()
	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetMaxIdleConns(10)

	// AutoMigrate 自动建表并初始化默认数据
	if err := AutoMigrateAll(); err != nil {
		return err
	}

	return nil
}

// AutoMigrate 自动建表
func AutoMigrateAll() error {
	if err := DB.AutoMigrate(
		&Constant{},
		&DBConnection{},
		&Vendor{},
		&FTPAccount{},
		&SQLTask{},
		&SystemConfig{},
		&ExportLog{},
		&User{},
	); err != nil {
		return err
	}
	initDefaultConfigs()
	initDefaultUser()
	return nil
}

// initDefaultUser 初始化默认管理员账号 admin / admin2026（仅当无用户时）
func initDefaultUser() {
	var cnt int64
	DB.Model(&User{}).Count(&cnt)
	if cnt > 0 {
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("admin2026"), bcrypt.DefaultCost)
	if err != nil {
		log.Println("[DB] 生成默认管理员密码哈希失败:", err)
		return
	}
	if err := DB.Create(&User{Username: "admin", Password: string(hash), Nickname: "管理员", Role: "admin"}).Error; err != nil {
		log.Println("[DB] 创建默认管理员账号失败:", err)
		return
	}
	log.Println("[DB] 已创建默认管理员账号: admin / admin2026")
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
		{"page_size", "20", "列表每页显示条数（厂家/日志/文件列表），修改后对新打开的列表生效"},
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

// itoa 简单的整数转字符串，避免为单行日志额外引入 strconv 调用点
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
