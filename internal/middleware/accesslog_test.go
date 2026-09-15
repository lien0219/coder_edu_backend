package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

const fakeToken = "secret-test-token"

func TestRedactChatWSToken(t *testing.T) {
	got := RedactRequestPath("/api/chat/ws?token=" + fakeToken)
	if strings.Contains(got, fakeToken) {
		t.Fatalf("token leaked in %q", got)
	}
	if !strings.Contains(got, "token="+RedactedQueryValue) {
		t.Fatalf("want redacted token, got %q", got)
	}
	if !strings.HasPrefix(got, "/api/chat/ws?") {
		t.Fatalf("path changed: %q", got)
	}
}

func TestRedactKeepsNonSensitiveQuery(t *testing.T) {
	got := RedactRequestPath("/api/example?page=1")
	if got != "/api/example?page=1" {
		t.Fatalf("got %q", got)
	}
}

func TestRedactMixedQuery(t *testing.T) {
	got := RedactRequestPath("/api/example?page=1&token=" + fakeToken)
	if strings.Contains(got, fakeToken) {
		t.Fatalf("token leaked in %q", got)
	}
	if !strings.Contains(got, "page=1") {
		t.Fatalf("lost page: %q", got)
	}
	if !strings.Contains(got, "token="+RedactedQueryValue) {
		t.Fatalf("want redacted token, got %q", got)
	}
}

func TestRedactRepeatedTokens(t *testing.T) {
	got := RedactRequestPath("/api/chat/ws?token=" + fakeToken + "&token=secret-test-token-2")
	if strings.Contains(got, fakeToken) || strings.Contains(got, "secret-test-token-2") {
		t.Fatalf("token leaked in %q", got)
	}
	if strings.Count(got, RedactedQueryValue) < 2 {
		t.Fatalf("expected both tokens redacted, got %q", got)
	}
}

func TestRedactNamedSecrets(t *testing.T) {
	names := []string{"access_token", "authorization", "api_key", "apikey"}
	for _, name := range names {
		got := RedactRequestPath("/api/example?" + name + "=" + fakeToken)
		if strings.Contains(got, fakeToken) {
			t.Fatalf("%s leaked in %q", name, got)
		}
		if !strings.Contains(got, name+"="+RedactedQueryValue) {
			t.Fatalf("%s not redacted: %q", name, got)
		}
	}
}

func TestRedactQueryKeyCaseInsensitive(t *testing.T) {
	got := RedactRequestPath("/api/chat/ws?TOKEN=" + fakeToken + "&Access_Token=secret-test-token-2")
	if strings.Contains(got, fakeToken) || strings.Contains(got, "secret-test-token-2") {
		t.Fatalf("token leaked in %q", got)
	}
	if strings.Count(got, RedactedQueryValue) < 2 {
		t.Fatalf("case variants not fully redacted: %q", got)
	}
}

func TestRedactURLEncodedQuery(t *testing.T) {
	got := RedactRequestPath("/api/chat/ws?token=secret-test-token%2Dextra&%74oken=" + fakeToken)
	if strings.Contains(got, fakeToken) || strings.Contains(got, "secret-test-token-extra") || strings.Contains(got, "secret-test-token%2Dextra") {
		t.Fatalf("encoded token leaked in %q", got)
	}
	if !strings.Contains(got, RedactedQueryValue) {
		t.Fatalf("encoded token not redacted: %q", got)
	}
}

func TestRedactPathWithoutQueryUnchanged(t *testing.T) {
	if got := RedactRequestPath("/api/chat/ws"); got != "/api/chat/ws" {
		t.Fatalf("got %q", got)
	}
}

func TestRedactEmptySensitiveValue(t *testing.T) {
	got := RedactRequestPath("/api/chat/ws?token=")
	if got != "/api/chat/ws?token="+RedactedQueryValue {
		t.Fatalf("empty token should still redact, got %q", got)
	}
}

func TestSafeAccessLogFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var buf bytes.Buffer
	router := gin.New()
	router.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		Formatter: SafeAccessLogFormatter,
		Output:    &buf,
	}))
	router.Use(SafeRecovery())
	router.GET("/api/chat/ws", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	router.GET("/api/example", func(c *gin.Context) {
		c.Status(http.StatusCreated)
	})
	if len(router.Routes()) != 2 {
		t.Fatalf("expected chat ws and example routes to remain registered")
	}

	req := httptest.NewRequest(http.MethodGet, "/api/chat/ws?token="+fakeToken, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	out := buf.String()
	if strings.Contains(out, fakeToken) {
		t.Fatalf("access log leaked token")
	}
	if !strings.Contains(out, RedactedQueryValue) {
		t.Fatalf("access log missing redaction: %q", out)
	}
	if !strings.Contains(out, "GET") || !strings.Contains(out, "/api/chat/ws") {
		t.Fatalf("access log missing method/path: %q", out)
	}
	if !strings.Contains(out, " 200 ") && !strings.Contains(out, "| 200 |") && !strings.Contains(out, "200") {
		t.Fatalf("access log missing status: %q", out)
	}

	buf.Reset()
	req = httptest.NewRequest(http.MethodGet, "/api/example?page=1", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	out = buf.String()
	if !strings.Contains(out, "page=1") {
		t.Fatalf("non-sensitive query dropped: %q", out)
	}
	if !strings.Contains(out, "201") {
		t.Fatalf("status missing: %q", out)
	}
}

func TestSafeRecoveryDoesNotDumpToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var errBuf bytes.Buffer
	prev := gin.DefaultErrorWriter
	gin.DefaultErrorWriter = &errBuf
	t.Cleanup(func() { gin.DefaultErrorWriter = prev })

	router := gin.New()
	router.Use(SafeRecovery())
	router.GET("/api/chat/ws", func(c *gin.Context) {
		panic("test-panic")
	})
	req := httptest.NewRequest(http.MethodGet, "/api/chat/ws?token="+fakeToken, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d", w.Code)
	}
	out := errBuf.String()
	if strings.Contains(out, fakeToken) {
		t.Fatalf("recovery dump leaked token")
	}
	if !strings.Contains(out, "/api/chat/ws") {
		t.Fatalf("recovery missing path: %q", out)
	}
}

func TestSafeAccessLogFormatterUsesRedactedPath(t *testing.T) {
	line := SafeAccessLogFormatter(gin.LogFormatterParams{
		TimeStamp:  time.Date(2026, 9, 16, 4, 0, 0, 0, time.UTC),
		StatusCode: 200,
		Latency:    time.Millisecond,
		ClientIP:   "127.0.0.1",
		Method:     http.MethodGet,
		Path:       "/api/chat/ws?token=" + fakeToken,
	})
	if strings.Contains(line, fakeToken) {
		t.Fatalf("formatter leaked token")
	}
	if !strings.Contains(line, "[GIN]") || !strings.Contains(line, "GET") || !strings.Contains(line, "200") {
		t.Fatalf("formatter missing diagnostics: %q", line)
	}
}
