package util

import "errors"

var (
	ErrUserNotFound            = errors.New("用户不存在")
	ErrEmailRegistered         = errors.New("该邮箱已被注册")
	ErrPermissionDenied        = errors.New("permission denied")
	ErrLevelNotFound           = errors.New("level not found")
	ErrLevelNotAccessible      = errors.New("level not accessible")
	ErrLevelNotYetAvailable    = errors.New("level not yet available")
	ErrLevelNoLongerAvailable  = errors.New("level no longer available")
	ErrAttemptNotFound         = errors.New("attempt not found")
	ErrTestNotPublished        = errors.New("test not published or not accessible")
	ErrTestAlreadySubmitted    = errors.New("test already submitted")
	ErrDailyShareLimit         = errors.New("daily share limit reached (max 3)")
	ErrUnauthorized            = errors.New("unauthorized")
	ErrInvalidRequest          = errors.New("invalid request")
	ErrAttemptLimitReached     = errors.New("您已达到该关卡的最大尝试次数限制")
	ErrTitleRequired           = errors.New("title required")
	ErrAbilityRequired         = errors.New("at least one ability must be selected")
	ErrVisibleToRequired       = errors.New("visibleTo must be provided when visibleScope is 'specific'")
	ErrQuestionTypeRequired    = errors.New("questionType required")
	ErrContentRequired         = errors.New("content required")
	ErrQuestionNotBelong       = errors.New("question not belong to level")
	ErrInvalidVideoExt         = errors.New("文件格式不支持，请上传有效的视频文件")
	ErrInvalidIconExt          = errors.New("文件格式不支持，请上传PNG、JPG或SVG格式")
	ErrUploadProgressNotFound  = errors.New("upload progress not found")
	ErrInvalidRequestFormat    = errors.New("invalid request format")
	ErrAnswersFieldMissing     = errors.New("answers field missing")
	ErrAnswersFieldMustBeArray = errors.New("answers field must be array")
	ErrResourceNotFound        = errors.New("resource not found")
	ErrNoPublishedAssessment     = errors.New("暂无测验")
	ErrAssessmentPaperStale      = errors.New("试卷已更新，请重新开始测验")
	ErrAssessmentSubmitInvalid   = errors.New("提交内容无效")
	ErrAssessmentRetestDenied    = errors.New("您已完成测试，暂不可重测")
	ErrAssessmentSubmitMissingMeta   = errors.New("请提交领取的试卷ID、试卷版本和请求编号")
	ErrAssessmentIdempotencyConflict = errors.New("重复提交内容与原记录不一致")
	ErrAssessmentPaperUnavailable    = errors.New("试卷未发布或暂无有效题目")
)

const (
	ErrCodeAssessmentPaperStale          = "ASSESSMENT_PAPER_STALE"
	ErrCodeAssessmentIdempotencyConflict = "ASSESSMENT_IDEMPOTENCY_CONFLICT"
	ErrCodeAssessmentPaperUnavailable    = "ASSESSMENT_PAPER_UNAVAILABLE"
)
