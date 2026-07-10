package models

// OperationLog 操作日志表：记录用户对系统的写操作（新增/修改/删除/执行等），便于审计与问题排查
type OperationLog struct {
	ID         int64  `gorm:"primaryKey" json:"id"`
	UserID     int64  `gorm:"index" json:"user_id"`
	Username   string `gorm:"type:varchar(128);index;default:''" json:"username"`
	Action     string `gorm:"type:varchar(128);default:''" json:"action"`   // 操作描述（中文，如“新增厂家”）
	Module     string `gorm:"type:varchar(64);default:''" json:"module"`    // 所属模块（如“厂家管理”）
	Method     string `gorm:"type:varchar(16);default:''" json:"method"`    // HTTP 方法
	Path       string `gorm:"type:varchar(255);default:''" json:"path"`     // 请求路径
	Detail     string `gorm:"type:longtext" json:"detail"`       // 请求参数摘要
	IP         string `gorm:"type:varchar(64);default:''" json:"ip"`        // 客户端 IP
	Status     int    `gorm:"default:0" json:"status"`                     // 业务响应码（0 成功 / 非 0 失败）
	Success    int    `gorm:"default:1" json:"success"`                    // 1 成功 / 0 失败
	DurationMs int64  `gorm:"default:0" json:"duration_ms"`                 // 耗时（毫秒）
	CreatedAt  DateTime `gorm:"type:datetime" json:"created_at"`
}
