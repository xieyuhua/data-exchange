package models

// SQLTaskHistory SQL 任务内容历史版本表：
// 当任务的 SQL 内容发生变更时保存变更前的旧版本快照，便于回溯与一键恢复历史数据。
type SQLTaskHistory struct {
	ID         int64  `gorm:"primaryKey" json:"id"`
	TaskID     int64  `gorm:"column:task_id;index;not null" json:"task_id"`
	TaskName   string `gorm:"default:''" json:"task_name"`               // 变更时的任务名快照
	SQLContent string `gorm:"type:longtext;not null;default:''" json:"sql_content"`
	ChangedBy  string `gorm:"type:varchar(128);default:''" json:"changed_by"` // 触发变更的用户名
	Remark     string `gorm:"type:varchar(255);default:''" json:"remark"`      // 备注（如“编辑保存”“恢复前快照”）
	CreatedAt  DateTime `gorm:"type:datetime" json:"created_at"`
}
