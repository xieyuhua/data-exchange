package models

// SQLTask SQL 任务表
type SQLTask struct {
	ID                  int64  `gorm:"primaryKey" json:"id"`
	VendorID            int64  `gorm:"column:vendor_id;not null;index" json:"vendor_id"`
	DBConnectionID      *int64 `gorm:"column:db_connection_id" json:"db_connection_id"`
	TaskName            string `gorm:"type:varchar(255);not null" json:"task_name"`
	SQLContent          string `gorm:"type:longtext;not null" json:"sql_content"`
	CSVFilenameTemplate string `gorm:"type:varchar(255);default:'{vendor_code}_{task_name}_{date}.csv'" json:"csv_filename_template"`
	CronExpression      string `gorm:"type:varchar(64);default:'0 2 * * *'" json:"cron_expression"`
	ExecutionMode       string `gorm:"type:varchar(32);default:'export_only'" json:"execution_mode"` // export_only, upload, import_db
	FTPAccountID        *int64 `gorm:"column:ftp_account_id" json:"ftp_account_id"`
	// 数据导入数据库(import_db)相关配置
	TargetDBConnectionID *int64 `gorm:"column:target_db_connection_id" json:"target_db_connection_id"` // 目标库连接，空则复用源库连接
	TargetTableName      string `gorm:"column:target_table_name;type:varchar(191);default:''" json:"target_table_name"` // 目标表名
	FieldMapping         string `gorm:"column:field_mapping;type:longtext" json:"field_mapping"`              // JSON: {"target_col": "source_header", ...}
	ImportMode           string `gorm:"column:import_mode;type:varchar(32);default:'append'" json:"import_mode"`          // append 追加 / truncate 先清空再写入
	SortOrder    int    `gorm:"default:0" json:"sort_order"`
	Enabled      int    `gorm:"default:1" json:"enabled"`
	LastRunAt           DateTime `gorm:"type:datetime" json:"last_run_at"`
	LastStatus          string `gorm:"type:varchar(32)" json:"last_status"`
	CreatedAt           DateTime `gorm:"type:datetime" json:"created_at"`
	UpdatedAt           DateTime `gorm:"type:datetime" json:"updated_at"`

	// 关联
	Vendor       *Vendor       `gorm:"foreignKey:VendorID" json:"-"`
	DBConnection *DBConnection `gorm:"foreignKey:DBConnectionID" json:"-"`
	FTPAccount   *FTPAccount   `gorm:"foreignKey:FTPAccountID" json:"-"`
}
