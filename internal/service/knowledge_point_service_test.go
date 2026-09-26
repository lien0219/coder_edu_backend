package service

import (
	"errors"
	"testing"

	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/util"
)

func TestNormalizeKnowledgePointVideosUploadAndExternal(t *testing.T) {
	videos := []CreateVideoResourceRequest{
		{Title: "local", URL: "videos/a.mp4"},
		{Title: "link", URL: "https://www.bilibili.com/video/BV1", SourceType: "external"},
	}
	if err := NormalizeKnowledgePointVideos(videos); err != nil {
		t.Fatalf("normalize mixed: %v", err)
	}
	if videos[0].SourceType != model.ResourceSourceUpload {
		t.Fatalf("missing sourceType must become upload, got %q", videos[0].SourceType)
	}
	if videos[1].SourceType != model.ResourceSourceExternal {
		t.Fatalf("external: %q", videos[1].SourceType)
	}
}

func TestNormalizeKnowledgePointVideosRejectsIllegalSource(t *testing.T) {
	videos := []CreateVideoResourceRequest{
		{Title: "bad", URL: "https://example.com/a.mp4", SourceType: "oss"},
	}
	if err := NormalizeKnowledgePointVideos(videos); !errors.Is(err, util.ErrVideoSourceInvalid) {
		t.Fatalf("illegal sourceType: %v", err)
	}
}

func TestNormalizeKnowledgePointVideosRejectsJavascript(t *testing.T) {
	videos := []CreateVideoResourceRequest{
		{Title: "xss", URL: "javascript:alert(1)", SourceType: "external"},
	}
	if err := NormalizeKnowledgePointVideos(videos); !errors.Is(err, util.ErrVideoURLUnsupportedProtocol) {
		t.Fatalf("javascript: %v", err)
	}
}
