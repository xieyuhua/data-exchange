package models

import (
	"encoding/json"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// 测试用模型：覆盖 datetime 字段行为
type dtDemo struct {
	ID        int64    `gorm:"primaryKey" json:"id"`
	Name      string   `gorm:"type:varchar(64)" json:"name"`
	CreatedAt DateTime `gorm:"type:datetime" json:"created_at"`
	UpdatedAt DateTime `gorm:"type:datetime" json:"updated_at"`
	EmptyTime DateTime `gorm:"type:datetime" json:"empty_time"` // 零值，验证 NULL / 空串
}

func (dtDemo) TableName() string { return "dt_demo" }

func TestDateTime_Behavior(t *testing.T) {
	// 1) MarshalJSON：零值输出 ""，正常值输出业务格式
	zero := DateTime(time.Time{})
	b, err := json.Marshal(zero)
	if err != nil {
		t.Fatalf("marshal zero: %v", err)
	}
	if string(b) != `""` {
		t.Fatalf("zero marshal 期望 \"\"，实际 %s", b)
	}
	now := DateTime(time.Date(2026, 7, 10, 16, 34, 5, 0, time.Local))
	b, _ = json.Marshal(now)
	if string(b) != `"2026-07-10 16:34:05"` {
		t.Fatalf("marshal now 期望 \"2026-07-10 16:34:05\"，实际 %s", b)
	}

	// 2) UnmarshalJSON：兼容业务格式与 RFC3339
	var in DateTime
	if err := json.Unmarshal([]byte(`"2026-07-10 16:34:05"`), &in); err != nil {
		t.Fatalf("unmarshal biz: %v", err)
	}
	if in.String() != "2026-07-10 16:34:05" {
		t.Fatalf("unmarshal biz 结果错误: %s", in.String())
	}
	if err := json.Unmarshal([]byte(`"2026-07-10T08:34:05Z"`), &in); err != nil {
		t.Fatalf("unmarshal rfc3339: %v", err)
	}

	// 3) GORM AutoMigrate 真实建表 + 读写往返（核心：列类型应为 datetime，存量文本可 Scan）
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&dtDemo{}); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	// 确认列类型
	type colInfo struct {
		Name string
		Type string
	}
	var cols []colInfo
	if err := db.Raw("SELECT name, type FROM pragma_table_info('dt_demo')").Scan(&cols).Error; err != nil {
		t.Fatalf("pragma: %v", err)
	}
	for _, c := range cols {
		if c.Name == "created_at" || c.Name == "updated_at" || c.Name == "empty_time" {
			if c.Type != "datetime" {
				t.Fatalf("列 %s 期望 datetime，实际 %s", c.Name, c.Type)
			}
		}
	}

	// 4) 写入并读回（包括零值 -> NULL）
	rec := &dtDemo{Name: "t1", CreatedAt: now, UpdatedAt: now, EmptyTime: zero}
	if err := db.Create(rec).Error; err != nil {
		t.Fatalf("create: %v", err)
	}
	var got dtDemo
	if err := db.First(&got, rec.ID).Error; err != nil {
		t.Fatalf("read: %v", err)
	}
	if got.CreatedAt.String() != "2026-07-10 16:34:05" {
		t.Fatalf("读回 created_at 错误: %q", got.CreatedAt.String())
	}
	if !got.EmptyTime.IsZero() {
		t.Fatalf("零值字段读回应仍为零值，实际 %q", got.EmptyTime.String())
	}

	// 5) Scan 兼容存量文本字符串（模拟旧 varchar 数据）
	var scanned dtDemo
	if err := db.Raw("SELECT id, name, '2026-01-02 03:04:05' AS created_at, '2026-01-02 03:04:05' AS updated_at, NULL AS empty_time FROM dt_demo WHERE id = ?", rec.ID).Scan(&scanned).Error; err != nil {
		t.Fatalf("scan text: %v", err)
	}
	if scanned.CreatedAt.String() != "2026-01-02 03:04:05" {
		t.Fatalf("Scan 文本结果错误: %q", scanned.CreatedAt.String())
	}
}
