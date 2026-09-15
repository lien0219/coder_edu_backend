package middleware

import (
	"fmt"
	"net/http"
	"net/url"
	"runtime/debug"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const RedactedQueryValue = "[REDACTED]"

var sensitiveQueryKeys = map[string]struct{}{
	"token":         {},
	"access_token":  {},
	"authorization": {},
	"api_key":       {},
	"apikey":        {},
}

func isSensitiveQueryKey(key string) bool {
	_, ok := sensitiveQueryKeys[strings.ToLower(strings.TrimSpace(key))]
	return ok
}

func pathAndQuery(requestPath string) (string, string) {
	path := requestPath
	rawQuery := ""
	if i := strings.IndexByte(requestPath, '?'); i >= 0 {
		path = requestPath[:i]
		rawQuery = requestPath[i+1:]
	}
	if parsed, err := url.Parse(requestPath); err == nil {
		if parsed.Path != "" {
			path = parsed.Path
		}
		rawQuery = parsed.RawQuery
	}
	return path, rawQuery
}

func encodeQueryForLog(values url.Values) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, key := range keys {
		escapedKey := url.QueryEscape(key)
		sensitive := isSensitiveQueryKey(key)
		for _, value := range values[key] {
			if b.Len() > 0 {
				b.WriteByte('&')
			}
			b.WriteString(escapedKey)
			b.WriteByte('=')
			if sensitive {
				b.WriteString(RedactedQueryValue)
			} else {
				b.WriteString(url.QueryEscape(value))
			}
		}
	}
	return b.String()
}

// RedactRequestPath 解析 path 与 query，将敏感查询参数值替换为 [REDACTED]。
// 解析失败时省略 query，避免把原始串写入日志。
func RedactRequestPath(requestPath string) string {
	if requestPath == "" {
		return requestPath
	}
	path, rawQuery := pathAndQuery(requestPath)
	if rawQuery == "" {
		return path
	}
	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		return path
	}
	encoded := encodeQueryForLog(values)
	if encoded == "" {
		return path
	}
	return path + "?" + encoded
}

func SafeAccessLogFormatter(param gin.LogFormatterParams) string {
	param.Path = RedactRequestPath(param.Path)

	var statusColor, methodColor, resetColor string
	if param.IsOutputColor() {
		statusColor = param.StatusCodeColor()
		methodColor = param.MethodColor()
		resetColor = param.ResetColor()
	}

	if param.Latency > time.Minute {
		param.Latency = param.Latency.Truncate(time.Second)
	}

	return fmt.Sprintf("[GIN] %v |%s %3d %s| %13v | %15s |%s %-7s %s %#v\n%s",
		param.TimeStamp.Format("2006/01/02 - 15:04:05"),
		statusColor, param.StatusCode, resetColor,
		param.Latency,
		param.ClientIP,
		methodColor, param.Method, resetColor,
		param.Path,
		param.ErrorMessage,
	)
}

func SafeLogger() gin.HandlerFunc {
	return gin.LoggerWithFormatter(SafeAccessLogFormatter)
}

func SafeRecovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		safePath := RedactRequestPath(c.Request.URL.RequestURI())
		fmt.Fprintf(gin.DefaultErrorWriter, "[Recovery] %s panic recovered:\n%s %s\n%v\n%s\n",
			time.Now().Format("2006/01/02 - 15:04:05"),
			c.Request.Method,
			safePath,
			recovered,
			debug.Stack(),
		)
		c.AbortWithStatus(http.StatusInternalServerError)
	})
}
