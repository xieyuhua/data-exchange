package models

// ExportLog 执行日志表
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
