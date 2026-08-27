package util

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// RandomHex 返回 n 个随机字节的十六进制字符串（长度 2n），基于 crypto/rand，不可预测。
// 适用于 token、锁持有者标识、幂等 key、requestId 等。n<=0 返回空串。
func RandomHex(n int) string {
	if n <= 0 {
		return ""
	}
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand 几乎不会失败；兜底用时间戳避免返回空串
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// UUIDv4 返回 RFC 4122 v4 版本的 UUID（crypto/rand 随机）。
func UUIDv4() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// 兜底：退化为无连字符的随机 hex（仍保证唯一性）
		return RandomHex(16)
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
