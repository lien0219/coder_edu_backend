package controller

import (
	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/service"
	"coder_edu_backend/internal/util"

	"github.com/gin-gonic/gin"
)

type AuthController struct {
	AuthService *service.AuthService
	UserService *service.UserService
}

func NewAuthController(authService *service.AuthService, userService *service.UserService) *AuthController {
	return &AuthController{
		AuthService: authService,
		UserService: userService,
	}
}

// RegisterRequest defines model for registration
// swagger:model RegisterRequest
type RegisterRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Role     string `json:"role" binding:"required,oneof=student teacher admin"`
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
		if err.Error() == "该邮箱已被注册" {
			util.Error(ctx, 409, "该邮箱已被注册")
		} else {
			// 添加日志记录错误详情
			util.LogInternalError(ctx, err)
		}
		return
	}

	util.Created(ctx, gin.H{"id": user.ID})
}

// LoginRequest defines model for login
// swagger:model LoginRequest
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
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
// @Router /api/login [post]
func (c *AuthController) Login(ctx *gin.Context) {
	var req LoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	token, err := c.AuthService.Login(req.Email, req.Password)
	if err != nil {
		util.Unauthorized(ctx)
		return
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
		// 如果出错，默认设置为未签到
		isCheckedInToday = false
	}

	profile := gin.H{
		"id":               user.ID,
		"name":             user.Name,
		"email":            user.Email,
		"role":             user.Role,
		"xp":               user.XP,
		"language":         user.Language,
		"createdAt":        user.CreatedAt,
		"isCheckedInToday": isCheckedInToday,
	}

	util.Success(ctx, profile)
}
