package models

import (
	"database/sql/driver"
	"fmt"
	"strings"
	"time"
)

// DateTimeLayout 业务统一的时间字符串格式，与历史数据及前端展示保持一致。
const DateTimeLayout = "2006-01-02 15:04:05"

// DateTime 业务日期时间类型：
//   - 数据库层以 DATETIME 存储（实现 driver.Valuer，零值写入 NULL）；
//   - JSON 序列化输出 "2006-01-02 15:04:05" 字符串，零值输出空串 ""，以兼容前端现有展示逻辑；
//   - 反序列化兼容 "2006-01-02 15:04:05" 与 RFC3339 两种格式。
//
// 引入该类型后，时间字段在库中即为真正的 datetime 类型，而非 varchar 文本。
type DateTime time.Time

// Time 返回底层的 time.Time。
func (t DateTime) Time() time.Time { return time.Time(t) }

// IsZero 判断是否为未设置（零值）。
func (t DateTime) IsZero() bool { return t.Time().IsZero() }

// String 以标准布局输出，零值返回空串。
func (t DateTime) String() string {
	if t.IsZero() {
		return ""
	}
	return t.Time().Format(DateTimeLayout)
}

// MarshalJSON 输出 "2006-01-02 15:04:05"；零值输出 ""（保证前端 v-if 判断成立）。
func (t DateTime) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		return []byte(`""`), nil
	}
	return []byte(`"` + t.Time().Format(DateTimeLayout) + `"`), nil
}

// UnmarshalJSON 解析 JSON 中的时间字符串，兼容业务格式与 RFC3339。
func (t *DateTime) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "" || s == "null" {
		*t = DateTime(time.Time{})
		return nil
	}
	parsed, err := parseDateTime(s)
	if err != nil {
		return err
	}
	*t = DateTime(parsed)
	return nil
}

// Value 实现 driver.Valuer，供 GORM 写入数据库；零值写入 NULL。
func (t DateTime) Value() (driver.Value, error) {
	if t.IsZero() {
		return nil, nil
	}
	return t.Time(), nil
}

// Scan 实现 sql.Scanner，兼容 time.Time / string / []byte / nil，保证存量文本数据可读。
func (t *DateTime) Scan(v any) error {
	switch val := v.(type) {
	case nil:
		*t = DateTime(time.Time{})
	case time.Time:
		*t = DateTime(val)
	case []byte:
		if len(val) == 0 {
			*t = DateTime(time.Time{})
			return nil
		}
		parsed, err := parseDateTime(string(val))
		if err != nil {
			return err
		}
		*t = DateTime(parsed)
	case string:
		if val == "" {
			*t = DateTime(time.Time{})
			return nil
		}
		parsed, err := parseDateTime(val)
		if err != nil {
			return err
		}
		*t = DateTime(parsed)
	default:
		return fmt.Errorf("不支持的 DateTime.Scan 类型: %T", v)
	}
	return nil
}

// Now 返回当前时间的 DateTime。
func Now() DateTime { return DateTime(time.Now()) }

// parseDateTime 容忍多种常见时间格式的解析。
func parseDateTime(s string) (time.Time, error) {
	layouts := []string{
		DateTimeLayout,
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02",
	}
	var lastErr error
	for _, layout := range layouts {
		parsed, err := time.ParseInLocation(layout, s, time.Local)
		if err == nil {
			return parsed, nil
		}
		lastErr = err
	}
	return time.Time{}, lastErr
}
