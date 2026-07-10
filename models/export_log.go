package models

// ExportLog 执行日志表
type ExportLog struct {
	ID            int64  `gorm:"primaryKey" json:"id"`
	TaskID        int64  `gorm:"column:task_id;not null;index" json:"task_id"`
	VendorID      int64  `gorm:"column:vendor_id;not null" json:"vendor_id"`
	Status        string `gorm:"type:varchar(32);not null;default:''" json:"status"` // success, failed
	ExecutionMode string `gorm:"type:varchar(32);default:''" json:"execution_mode"`
	CSVFilename   string `gorm:"type:varchar(255);default:''" json:"csv_filename"`
	FileSize      int64  `gorm:"default:0" json:"file_size"`
	RecordCount   int    `gorm:"default:0" json:"record_count"`
	ErrorMessage  string `gorm:"type:longtext" json:"error_message"`
	DurationMs    int64  `gorm:"default:0" json:"duration_ms"`
	StartedAt     DateTime `gorm:"type:datetime" json:"started_at"`
	FinishedAt    DateTime `gorm:"type:datetime" json:"finished_at"`
	CreatedAt     DateTime `gorm:"type:datetime" json:"created_at"`
}
