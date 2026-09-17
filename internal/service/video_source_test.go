package service

import (
	"errors"
	"strings"
	"testing"

	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/util"
)

func TestParseResourceSourceType(t *testing.T) {
	st, err := ParseResourceSourceType("")
	if err != nil || st != model.ResourceSourceUpload {
		t.Fatalf("blank must be upload, got %q %v", st, err)
	}
	st, err = ParseResourceSourceType("UPLOAD")
	if err != nil || st != model.ResourceSourceUpload {
		t.Fatalf("upload: %q %v", st, err)
	}
	st, err = ParseResourceSourceType("external")
	if err != nil || st != model.ResourceSourceExternal {
		t.Fatalf("external: %q %v", st, err)
	}
	_, err = ParseResourceSourceType("oss")
	if !errors.Is(err, util.ErrVideoSourceInvalid) {
		t.Fatalf("illegal sourceType: %v", err)
	}
}

func TestValidateVideoSourceUploadStillAccepted(t *testing.T) {
	st, u, err := ValidateVideoSource("", "videos/abc.mp4")
	if err != nil || st != model.ResourceSourceUpload || u != "videos/abc.mp4" {
		t.Fatalf("relative upload: st=%s url=%s err=%v", st, u, err)
	}
	st, u, err = ValidateVideoSource(model.ResourceSourceUpload, "https://cdn.example.com/videos/abc.mp4")
	if err != nil || st != model.ResourceSourceUpload || u == "" {
		t.Fatalf("https upload: st=%s url=%s err=%v", st, u, err)
	}
}

func TestValidateVideoSourceExternalHTTPSAndHTTP(t *testing.T) {
	st, _, err := ValidateVideoSource("external", "https://www.youtube.com/watch?v=dQw4w9WgXcQ")
	if err != nil || st != model.ResourceSourceExternal {
		t.Fatalf("youtube: st=%s err=%v", st, err)
	}
	_, _, err = ValidateVideoSource(model.ResourceSourceExternal, "https://www.bilibili.com/video/BV1xx411c7mD")
	if err != nil {
		t.Fatalf("bilibili: %v", err)
	}
	_, _, err = ValidateVideoSource("external", "http://example.com/lesson")
	if err != nil {
		t.Fatalf("http external must be allowed; client warns, got %v", err)
	}
}

func TestValidateVideoSourceRejectsEmptyJavascriptDataFile(t *testing.T) {
	_, _, err := ValidateVideoSource("external", "   ")
	if !errors.Is(err, util.ErrVideoURLEmpty) {
		t.Fatalf("empty: %v", err)
	}
	_, _, err = ValidateVideoSource("upload", "")
	if !errors.Is(err, util.ErrVideoURLEmpty) {
		t.Fatalf("empty upload: %v", err)
	}
	_, _, err = ValidateVideoSource("external", "javascript:alert(1)")
	if !errors.Is(err, util.ErrVideoURLUnsupportedProtocol) {
		t.Fatalf("javascript: %v", err)
	}
	_, _, err = ValidateVideoSource("upload", "javascript:alert(1)")
	if !errors.Is(err, util.ErrVideoURLUnsupportedProtocol) {
		t.Fatalf("javascript upload: %v", err)
	}
	_, _, err = ValidateVideoSource("external", "data:text/html,hi")
	if !errors.Is(err, util.ErrVideoURLUnsupportedProtocol) {
		t.Fatalf("data: %v", err)
	}
	_, _, err = ValidateVideoSource("external", "file:///tmp/a.mp4")
	if !errors.Is(err, util.ErrVideoURLUnsupportedProtocol) {
		t.Fatalf("file: %v", err)
	}
}

func TestValidateVideoSourceRejectsIllegalSourceType(t *testing.T) {
	_, _, err := ValidateVideoSource("ftp", "https://example.com/a")
	if !errors.Is(err, util.ErrVideoSourceInvalid) {
		t.Fatalf("illegal sourceType: %v", err)
	}
}

func TestValidateVideoSourceAllowsURLBetween255And2048(t *testing.T) {
	long := "https://example.com/" + strings.Repeat("a", 300)
	if len(long) <= 255 || len(long) >= model.MaxVideoURLLength {
		t.Fatalf("fixture length %d", len(long))
	}
	st, u, err := ValidateVideoSource("external", long)
	if err != nil || st != model.ResourceSourceExternal || u != long {
		t.Fatalf("long url: st=%s err=%v", st, err)
	}
	tooLong := "https://example.com/" + strings.Repeat("b", model.MaxVideoURLLength)
	_, _, err = ValidateVideoSource("external", tooLong)
	if !errors.Is(err, util.ErrVideoURLInvalid) {
		t.Fatalf("oversize: %v", err)
	}
}

func TestNormalizeVideoPayloadCreateAndUpdate(t *testing.T) {
	createUpload := map[string]interface{}{
		"title": "old upload",
		"url":   "videos/keep.mp4",
	}
	if err := NormalizeVideoPayload(createUpload); err != nil {
		t.Fatal(err)
	}
	if createUpload["sourceType"] != model.ResourceSourceUpload {
		t.Fatalf("missing sourceType must be upload, got %+v", createUpload["sourceType"])
	}

	createExternal := map[string]interface{}{
		"sourceType": "external",
		"url":        "https://example.com/watch?v=1",
	}
	if err := NormalizeVideoPayload(createExternal); err != nil {
		t.Fatal(err)
	}
	if createExternal["sourceType"] != model.ResourceSourceExternal {
		t.Fatalf("create external source %+v", createExternal["sourceType"])
	}

	editExternal := map[string]interface{}{
		"sourceType": "external",
		"url":        "https://www.youtube.com/watch?v=updated",
		"title":      "edited",
	}
	if err := NormalizeVideoPayload(editExternal); err != nil {
		t.Fatal(err)
	}

	titleOnly := map[string]interface{}{"title": "keep source"}
	if err := NormalizeVideoPayload(titleOnly); err != nil {
		t.Fatal(err)
	}
	if _, ok := titleOnly["sourceType"]; ok {
		t.Fatal("title-only update must not invent sourceType")
	}

	bad := map[string]interface{}{"sourceType": "external", "url": "javascript:alert(1)"}
	if err := NormalizeVideoPayload(bad); !errors.Is(err, util.ErrVideoURLUnsupportedProtocol) {
		t.Fatalf("payload javascript: %v", err)
	}

	illegal := map[string]interface{}{"sourceType": "mirror", "url": "https://example.com/a"}
	if err := NormalizeVideoPayload(illegal); !errors.Is(err, util.ErrVideoSourceInvalid) {
		t.Fatalf("payload illegal source: %v", err)
	}
}
