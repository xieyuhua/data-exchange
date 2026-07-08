package models

// SQLTask SQL 任务表
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
	Vendor       *Vendor       `gorm:"foreignKey:VendorID" json:"-"`
	DBConnection *DBConnection `gorm:"foreignKey:DBConnectionID" json:"-"`
	FTPAccount   *FTPAccount   `gorm:"foreignKey:FTPAccountID" json:"-"`
}
