package util

import "errors"

var (
	ErrUserNotFound           = errors.New("用户不存在")
	ErrEmailRegistered        = errors.New("该邮箱已被注册")
	ErrPermissionDenied       = errors.New("permission denied")
	ErrLevelNotFound          = errors.New("level not found")
	ErrLevelNotAccessible     = errors.New("level not accessible")
	ErrLevelNotYetAvailable   = errors.New("level not yet available")
	ErrLevelNoLongerAvailable = errors.New("level no longer available")
	ErrAttemptNotFound        = errors.New("attempt not found")
	ErrTestNotPublished       = errors.New("test not published or not accessible")
	ErrTestAlreadySubmitted   = errors.New("test already submitted")
	ErrDailyShareLimit        = errors.New("daily share limit reached (max 3)")
)
