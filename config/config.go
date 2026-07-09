package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// DatabaseConfig 系统数据库配置
type DatabaseConfig struct {
	Type       string      `yaml:"type"`       // sqlite 或 mysql
	SQLitePath string      `yaml:"sqlite_path"` // sqlite 文件路径
	MySQL      MySQLConfig `yaml:"mysql"`
}

// MySQLConfig MySQL 连接配置
type MySQLConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
	Params   string `yaml:"params"`
}

// Config 应用配置
type Config struct {
	Database DatabaseConfig `yaml:"database"`
	// WebRoot 前端静态资源目录(外部目录)。为空则使用编译期嵌入的前端资源。
	// 该目录应包含 index.html 与 assets/ (即 vite 构建产物，通常指向 static/ 目录)。
	WebRoot string `yaml:"web_root"`
	// AutoMigrate 是否启用启动时的自动建表(含默认数据初始化)。
	// 使用 *bool 以区分"未配置"(nil，按默认 true)与显式 false。
	// 设为 false 可跳过 AutoMigrate（适用于表结构由外部/手动维护的场景）。
	AutoMigrate *bool `yaml:"auto_migrate"`
}

// 默认配置（全局单例，供其他包读取）
var AppConfig = Config{
	Database: DatabaseConfig{
		Type:       "sqlite",
		SQLitePath: "data.db",
		MySQL: MySQLConfig{
			Host:     "127.0.0.1",
			Port:     3306,
			User:     "root",
			Password: "",
			Database: "data_exchange",
			Params:   "charset=utf8mb4&parseTime=True&loc=Local",
		},
	},
}

// Load 从指定路径加载配置文件（yaml）。文件不存在时使用默认配置。
func Load(path string) error {
	if path == "" {
		path = "config.yaml"
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// 配置文件不存在，使用内置默认配置
			return nil
		}
		return fmt.Errorf("读取配置文件失败: %w", err)
	}
	if err := yaml.Unmarshal(data, &AppConfig); err != nil {
		return fmt.Errorf("解析配置文件失败: %w", err)
	}
	// 兜底默认值
	if AppConfig.Database.Type == "" {
		AppConfig.Database.Type = "sqlite"
	}
	if AppConfig.Database.Type == "sqlite" && AppConfig.Database.SQLitePath == "" {
		AppConfig.Database.SQLitePath = "data.db"
	}
	if AppConfig.Database.Type == "mysql" {
		if AppConfig.Database.MySQL.Host == "" {
			AppConfig.Database.MySQL.Host = "127.0.0.1"
		}
		if AppConfig.Database.MySQL.Port == 0 {
			AppConfig.Database.MySQL.Port = 3306
		}
		if AppConfig.Database.MySQL.Params == "" {
			AppConfig.Database.MySQL.Params = "charset=utf8mb4&parseTime=True&loc=Local"
		}
	}
	return nil
}

// DSN 返回当前数据库类型的连接串
func (c *DatabaseConfig) DSN() string {
	if c.Type == "mysql" {
		m := c.MySQL
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?%s", m.User, m.Password, m.Host, m.Port, m.Database, m.Params)
	}
	return c.SQLitePath
}

// ShouldAutoMigrate 返回是否执行启动时的自动建表。
// 未配置 auto_migrate 时默认 true；显式设置为 false 时返回 false。
func ShouldAutoMigrate() bool {
	return AppConfig.AutoMigrate == nil || *AppConfig.AutoMigrate
}
