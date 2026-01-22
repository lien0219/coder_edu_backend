package util

import (
	"strconv"
)

// MustParseUint 将字符串转换为无符号整数，解析失败时返回 0
func MustParseUint(s string) uint {
	id, _ := strconv.ParseUint(s, 10, 32)
	return uint(id)
}
