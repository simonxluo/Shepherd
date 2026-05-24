package storage

import (
	"fmt"
	"time"
)

// generateID generates a unique ID with a prefix
func generateID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

// timeNow returns the current UTC time
func timeNow() time.Time {
	return time.Now().UTC()
}

// derefInt64 安全解引用 *int64，如果为 nil 返回默认值
func derefInt64(p *int64, def int64) int64 {
	if p == nil {
		return def
	}
	return *p
}

// derefString 安全解引用 *string，如果为 nil 返回默认值
func derefString(p *string, def string) string {
	if p == nil {
		return def
	}
	return *p
}

// derefFloat64 安全解引用 *float64，如果为 nil 返回默认值
func derefFloat64(p *float64, def float64) float64 {
	if p == nil {
		return def
	}
	return *p
}

// int64Ptr 从 int64 创建指针
func int64Ptr(v int64) *int64 {
	return &v
}

// stringPtr 从 string 创建指针（仅在非空时）
func stringPtr(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

// timeToUnixPtr 将 *time.Time 转换为 *int64（Unix 时间戳）
func timeToUnixPtr(t *time.Time) *int64 {
	if t == nil {
		return nil
	}
	v := t.Unix()
	return &v
}
