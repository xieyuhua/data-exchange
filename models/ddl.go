package models

import (
	"fmt"
	"strconv"
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

	models := []any{
		&Constant{}, &DBConnection{}, &Vendor{}, &FTPAccount{},
		&SQLTask{}, &SystemConfig{}, &ExportLog{}, &User{},
		&OperationLog{}, &SQLTaskHistory{},
	}

	var b strings.Builder
	b.WriteString("-- 数据交换系统 建表与初始数据 SQL（由系统生成，供手动导入）\n")
	fmt.Fprintf(&b, "-- 方言: %s\n", dialect)
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
	fmt.Fprintf(&b, "-- 默认系统配置 (%s)\n", cfgTable)
	for _, d := range defaultConfigs() {
		fmt.Fprintf(&b, "INSERT INTO `%s` (`config_key`, `config_value`, `description`) VALUES ('%s', '%s', '%s');\n",
			cfgTable, escapeSQL(d.K), escapeSQL(d.V), escapeSQL(d.D))
	}
	b.WriteString("\n")

	userTable := tableNameOf(&User{})
	hash, err := bcrypt.GenerateFromPassword([]byte("admin2026"), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("生成默认管理员密码哈希失败: %w", err)
	}
	fmt.Fprintf(&b, "-- 默认管理员账号 admin / admin2026 (%s)\n", userTable)
	fmt.Fprintf(&b, "INSERT INTO `%s` (`username`, `password`, `nickname`, `role`) VALUES ('admin', '%s', '管理员', 'admin');\n",
		userTable, string(hash))

	return b.String(), nil
}

// tableNameOf 返回模型对应的实际表名（含自定义 TableName 方法）
func tableNameOf(m any) string {
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
		// 方言级类型修正，使生成的建表 SQL 更规范、贴合实际业务体量：
		typ = normalizeColumnType(typ, f, dialect)
		col := fmt.Sprintf("  `%s` %s", f.DBName, typ)

		if f.PrimaryKey {
			pkCols = append(pkCols, "`"+f.DBName+"`")
			if f.AutoIncrement {
				if dialect == "mysql" {
					// MySQL 自增主键：bigint + PRIMARY KEY + AUTO_INCREMENT
					col = fmt.Sprintf("  `%s` bigint PRIMARY KEY AUTO_INCREMENT", f.DBName)
				} else {
					// sqlite 自增主键：必须为 INTEGER PRIMARY KEY AUTOINCREMENT
					col = fmt.Sprintf("  `%s` INTEGER PRIMARY KEY AUTOINCREMENT", f.DBName)
				}
			} else {
				col = fmt.Sprintf("  `%s` %s PRIMARY KEY", f.DBName, typ)
			}
		} else {
			if f.NotNull {
				col += " NOT NULL"
			}
			if f.Unique {
				col += " UNIQUE"
			}
			if f.DefaultValue != "" {
				col += " DEFAULT " + quoteDefault(f.DefaultValue)
			}
		}
		cols = append(cols, col)
	}

	// 复合主键：单列主键已在列内联 PRIMARY KEY，无需重复
	if len(pkCols) > 1 {
		cols = append(cols, "  PRIMARY KEY ("+strings.Join(pkCols, ", ")+")")
	}

	return fmt.Sprintf("CREATE TABLE `%s` (\n%s\n)", sc.Table, strings.Join(cols, ",\n")), nil
}

// normalizeColumnType 对方言推导出的列类型做针对性修正，使建表 SQL 更规范：
//  1. SQLite 没有 longtext / mediumtext 等类型，统一降级为 text（SQLite 文本列本质都是 text）。
//  2. MySQL 下 GORM 默认把 Go 的 int 映射成 bigint；对于非主键、非自增的普通 int 字段
//     （状态码、计数、排序等）强制使用 int，更省空间、索引更优，且足以覆盖业务取值范围。
func normalizeColumnType(typ string, f *schema.Field, dialect string) string {
	upper := strings.ToUpper(typ)
	if dialect == "sqlite" {
		// SQLite 仅支持 TEXT，longtext/mediumtext 均不在其类型亲和里，降级为 TEXT。
		if strings.Contains(upper, "LONGTEXT") || strings.Contains(upper, "MEDIUMTEXT") {
			return "text"
		}
		return typ
	}
	if dialect == "mysql" {
		// 仅对非自增、非主键的普通 Go int 字段做 int 加固，避免 bigint 膨胀。
		// GORM 把 Go 的 int 归类为 schema.Int（DataType 字段），据此判断比反射字段名更稳定。
		if !f.PrimaryKey && !f.AutoIncrement &&
			f.DataType == schema.Int &&
			(strings.HasPrefix(upper, "BIGINT") || strings.HasPrefix(upper, "INT")) {
			return "int"
		}
	}
	return typ
}

// quoteDefault 为字符串类型的默认值补上单引号。
// 已带单引号、数值、或 SQL 表达式/函数（如 CURRENT_TIMESTAMP、NOW()）则原样返回。
func quoteDefault(d string) string {
	s := strings.TrimSpace(d)
	if s == "" {
		return d
	}
	if strings.HasPrefix(s, "'") && strings.HasSuffix(s, "'") {
		return d
	}
	if isNumeric(s) {
		return d
	}
	upper := strings.ToUpper(s)
	for _, expr := range []string{"CURRENT_TIMESTAMP", "CURRENT_DATE", "CURRENT_TIME", "NOW(", "GETDATE(", "UUID(", "NULL"} {
		if strings.HasPrefix(upper, expr) {
			return d
		}
	}
	// 其余视为字符串字面量，补单引号并转义内部单引号
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// isNumeric 判断字符串是否为整数或小数（用于跳过数值类型的默认值加引号）
func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

func escapeSQL(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
