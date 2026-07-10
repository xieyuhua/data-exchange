package models

import (
	"fmt"
	"strings"
	"sync"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// GenSchemaSQL 生成建表语句与初始数据 SQL 文本，供手动导入。
// dialect 为 "sqlite" 或 "mysql"，对应 config 中的数据库类型。
// 当 auto_migrate 设为 false 时，可用本函数产物自行维护表结构与初始数据。
func GenSchemaSQL(dialect string) (string, error) {
	// 借助对应方言的 Dialector 获取准确的字段类型映射（DataTypeOf）
	var sd gorm.Dialector
	switch dialect {
	case "mysql":
		sd = mysql.Open("")
	default:
		sd = sqlite.Open("")
	}

	models := []interface{}{
		&Constant{}, &DBConnection{}, &Vendor{}, &FTPAccount{},
		&SQLTask{}, &SystemConfig{}, &ExportLog{}, &User{},
		&OperationLog{}, &SQLTaskHistory{},
	}

	var b strings.Builder
	b.WriteString("-- 数据交换系统 建表与初始数据 SQL（由系统生成，供手动导入）\n")
	b.WriteString(fmt.Sprintf("-- 方言: %s\n", dialect))
	b.WriteString("-- 用途: 将 auto_migrate 设为 false 后，由 DBA/运维手动执行本文件完成建表与初始化\n\n")

	for _, m := range models {
		sc, err := schema.Parse(m, &sync.Map{}, schema.NamingStrategy{})
		if err != nil {
			return "", fmt.Errorf("解析模型 %T 失败: %w", m, err)
		}
		ddl, err := buildCreateTable(sc, sd, dialect)
		if err != nil {
			return "", err
		}
		b.WriteString(ddl)
		b.WriteString(";\n\n")
	}

	// ---- 初始数据 ----
	b.WriteString("-- ==================== 初始数据 ====================\n\n")

	cfgTable := tableNameOf(&SystemConfig{})
	b.WriteString(fmt.Sprintf("-- 默认系统配置 (%s)\n", cfgTable))
	for _, d := range defaultConfigs() {
		b.WriteString(fmt.Sprintf("INSERT INTO `%s` (`config_key`, `config_value`, `description`) VALUES ('%s', '%s', '%s');\n",
			cfgTable, escapeSQL(d.K), escapeSQL(d.V), escapeSQL(d.D)))
	}
	b.WriteString("\n")

	userTable := tableNameOf(&User{})
	hash, err := bcrypt.GenerateFromPassword([]byte("admin2026"), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("生成默认管理员密码哈希失败: %w", err)
	}
	b.WriteString(fmt.Sprintf("-- 默认管理员账号 admin / admin2026 (%s)\n", userTable))
	b.WriteString(fmt.Sprintf("INSERT INTO `%s` (`username`, `password`, `nickname`, `role`) VALUES ('admin', '%s', '管理员', 'admin');\n",
		userTable, string(hash)))

	return b.String(), nil
}

// tableNameOf 返回模型对应的实际表名（含自定义 TableName 方法）
func tableNameOf(m interface{}) string {
	sc, err := schema.Parse(m, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		return ""
	}
	return sc.Table
}

// buildCreateTable 根据 schema 与方言生成 CREATE TABLE 语句
func buildCreateTable(sc *schema.Schema, sd gorm.Dialector, dialect string) (string, error) {
	var cols []string
	var pkCols []string

	for _, f := range sc.Fields {
		if f.DBName == "" {
			continue
		}
		typ := sd.DataTypeOf(f)
		col := fmt.Sprintf("  `%s` %s", f.DBName, typ)

		if f.PrimaryKey {
			pkCols = append(pkCols, "`"+f.DBName+"`")
			if f.AutoIncrement {
				if dialect == "mysql" {
					col += " AUTO_INCREMENT"
				} else {
					// sqlite 自增主键：类型需为 INTEGER 且加 AUTOINCREMENT
					col = fmt.Sprintf("  `%s` INTEGER", f.DBName)
					if !strings.Contains(strings.ToUpper(typ), "PRIMARY") {
						// 单列自增主键直接内联
					}
					col += " PRIMARY KEY AUTOINCREMENT"
				}
			}
		} else {
			if f.NotNull {
				col += " NOT NULL"
			}
			if f.Unique {
				col += " UNIQUE"
			}
			if f.DefaultValue != "" {
				col += " DEFAULT " + f.DefaultValue
			}
		}
		cols = append(cols, col)
	}

	// 复合主键
	if len(pkCols) > 1 {
		cols = append(cols, "  PRIMARY KEY ("+strings.Join(pkCols, ", ")+")")
	}

	return fmt.Sprintf("CREATE TABLE `%s` (\n%s\n)", sc.Table, strings.Join(cols, ",\n")), nil
}

func escapeSQL(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
