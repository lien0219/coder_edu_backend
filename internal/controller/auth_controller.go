package controller

import (
	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/service"
	"coder_edu_backend/internal/util"
	"errors"

	"github.com/gin-gonic/gin"
)

type AuthController struct {
	AuthService    *service.AuthService
	UserService    *service.UserService
	CaptchaService *service.CaptchaService
	IsRelease      bool // 是否为生产环境
}

func NewAuthController(authService *service.AuthService, userService *service.UserService, captchaService *service.CaptchaService, isRelease bool) *AuthController {
	return &AuthController{
		AuthService:    authService,
		UserService:    userService,
		CaptchaService: captchaService,
		IsRelease:      isRelease,
	}
}

// RegisterRequest defines model for registration
// swagger:model RegisterRequest
type RegisterRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Role     string `json:"role" binding:"required,oneof=student teacher"`
}

// Register godoc
// @Summary 注册新用户
// @Description 使用提供的信息注册新用户
// @Tags 认证
// @Accept  json
// @Produce  json
// @Param   body body RegisterRequest true "用户注册信息"
// @Success 201 {object} util.Response{data=object} "创建成功"
// @Failure 400 {object} util.Response "请求参数错误"
// @Failure 409 {object} util.Response "邮箱已被注册"
// @Failure 500 {object} util.Response "服务器内部错误"
// @Router /api/register [post]
func (c *AuthController) Register(ctx *gin.Context) {
	var req RegisterRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	user := &model.User{
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
		Role:     model.UserRole(req.Role),
	}

	if err := c.AuthService.Register(user); err != nil {
		if errors.Is(err, util.ErrEmailRegistered) {
			util.Error(ctx, 409, "该邮箱已被注册")
		} else {
			util.LogInternalError(ctx, err)
		}
		return
	}

	util.Created(ctx, gin.H{"id": user.ID})
}

// CaptchaVerifyRequest 验证码校验请求
type CaptchaVerifyRequest struct {
	Trajectory []service.TrajectoryPoint `json:"trajectory"`
	Duration   int                       `json:"duration"`
}

// VerifyCaptcha godoc
// @Summary 验证码校验
// @Description 后端根据滑动轨迹判断是否为真人
// @Tags 认证
// @Accept  json
// @Produce  json
// @Param   body body CaptchaVerifyRequest true "轨迹数据"
// @Success 200 {object} util.Response{data=object} "验证通过"
// @Failure 400 {object} util.Response "验证失败"
// @Router /api/auth/captcha/verify [post]
func (c *AuthController) VerifyCaptcha(ctx *gin.Context) {
	var req CaptchaVerifyRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	token, err := c.CaptchaService.VerifyTrajectory(req.Trajectory, req.Duration)
	if err != nil {
		util.Error(ctx, 400, "人机验证失败: "+err.Error())
		return
	}

	util.Success(ctx, gin.H{"captcha_token": token})
}

// CheckCaptchaSkip godoc
// @Summary 检查是否可以跳过验证码
// @Description 检查请求中的 trust_device_token Cookie 是否有效
// @Tags 认证
// @Accept  json
// @Produce  json
// @Success 200 {object} util.Response{data=object} "成功"
// @Router /api/auth/captcha/check-skip [get]
func (c *AuthController) CheckCaptchaSkip(ctx *gin.Context) {
	cookie, err := ctx.Cookie("trust_device_token")
	if err != nil {
		util.Success(ctx, gin.H{"shouldVerify": true})
		return
	}

	_, valid := c.CaptchaService.VerifyTrustDeviceToken(cookie)
	util.Success(ctx, gin.H{"shouldVerify": !valid})
}

// swagger:model LoginRequest
type LoginRequest struct {
	Email        string `json:"email" binding:"required,email"`
	Password     string `json:"password" binding:"required"`
	CaptchaToken string `json:"captcha_token"`
	RememberMe   bool   `json:"rememberMe"`
}

// Login godoc
// @Summary 用户登录
// @Description 验证用户身份并返回JWT令牌
// @Tags 认证
// @Accept  json
// @Produce  json
// @Param   body body LoginRequest true "用户登录凭据"
// @Success 200 {object} util.Response{data=object} "成功"
// @Failure 400 {object} util.Response "请求参数错误"
// @Failure 401 {object} util.Response "未授权"
// @Failure 403 {object} util.Response "验证码错误"
// @Router /api/login [post]
func (c *AuthController) Login(ctx *gin.Context) {
	var req LoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	// 1. 检查是否可以免验证
	cookie, _ := ctx.Cookie("trust_device_token")
	_, isTrusted := c.CaptchaService.VerifyTrustDeviceToken(cookie)

	// 2. 如果不满足免验证条件，则必须校验 captcha_token
	if !isTrusted {
		if req.CaptchaToken == "" || !c.CaptchaService.ValidateToken(req.CaptchaToken) {
			util.Error(ctx, 403, "请先完成人机验证")
			return
		}
	}

	token, err := c.AuthService.Login(req.Email, req.Password)
	if err != nil {
		util.Unauthorized(ctx)
		return
	}

	// 3. 如果勾选了“记住我”，生成可信设备 Token 并设置 Cookie
	if req.RememberMe {
		user, _ := c.AuthService.UserRepo.FindByEmail(req.Email)
		if user != nil {
			trustToken, err := c.CaptchaService.GenerateTrustDeviceToken(user.ID)
			if err == nil {
				// 设置 HttpOnly Cookie，有效期 15 天；生产环境启用 Secure 标志
				ctx.SetCookie("trust_device_token", trustToken, 15*24*3600, "/", "", c.IsRelease, true)
			}
		}
	}

	util.Success(ctx, gin.H{"token": token})
}

// GetProfile godoc
// @Summary 获取当前用户资料
// @Description 获取当前已认证用户的个人资料
// @Tags 认证
// @Accept  json
// @Produce  json
// @Security ApiKeyAuth
// @Success 200 {object} util.Response{data=model.User} "Success"
// @Failure 401 {object} util.Response "Unauthorized"
// @Router /api/profile [get]
func (c *AuthController) GetProfile(ctx *gin.Context) {
	user := c.AuthService.GetCurrentUser(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	// 用户签到状态
	isCheckedInToday, err := c.UserService.IsCheckedInToday(user.ID)
	if err != nil {
		isCheckedInToday = false
	}

	profile := gin.H{
		"id":               user.ID,
		"name":             user.Name,
		"email":            user.Email,
		"avatar":           user.Avatar,
		"role":             user.Role,
		"xp":               user.XP,
		"language":         user.Language,
		"createdAt":        user.CreatedAt,
		"isCheckedInToday": isCheckedInToday,
	}

	util.Success(ctx, profile)
}
