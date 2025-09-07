package controller

import (
	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/service"
	"coder_edu_backend/internal/util"
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// UserController 处理用户相关的HTTP请求
type UserController struct {
	UserService *service.UserService
}

// NewUserController 创建一个新的用户控制器实例
func NewUserController(userService *service.UserService) *UserController {
	return &UserController{
		UserService: userService,
	}
}

// UpdateUserRequest 定义用户更新请求结构
// swagger:model UpdateUserRequest
type UpdateUserRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Role     string `json:"role" binding:"required,oneof=student teacher admin"`
	Language string `json:"language"`
	Password string `json:"password"`
	Disabled bool   `json:"disabled"`
}

// GetUsers godoc
// @Summary 获取用户列表
// @Description 获取用户列表，支持分页和筛选
// @Tags 用户管理
// @Accept  json
// @Produce  json
// @Security ApiKeyAuth
// @Param   page query int false "页码" default(1)
// @Param   pageSize query int false "每页条数" default(10)
// @Param   role query string false "角色筛选"
// @Param   status query string false "状态筛选"
// @Param   search query string false "搜索关键词"
// @Param   startDate query string false "开始日期"
// @Param   endDate query string false "结束日期"
// @Success 200 {object} util.Response{data=[]model.User} "成功"
// @Failure 401 {object} util.Response "未授权"
// @Failure 500 {object} util.Response "服务器内部错误"
// @Router /api/admin/users [get]
func (c *UserController) GetUsers(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("pageSize", "10"))
	role := ctx.Query("role")
	status := ctx.Query("status")
	search := ctx.Query("search")
	startDateStr := ctx.Query("startDate")
	endDateStr := ctx.Query("endDate")

	var startDate, endDate time.Time
	var err error

	if startDateStr != "" {
		startDate, err = time.Parse(time.RFC3339, startDateStr)
		if err != nil {
			util.BadRequest(ctx, "无效的开始日期格式")
			return
		}
	}

	if endDateStr != "" {
		endDate, err = time.Parse(time.RFC3339, endDateStr)
		if err != nil {
			util.BadRequest(ctx, "无效的结束日期格式")
			return
		}
	}

	filter := service.UserFilter{
		Role:      role,
		Status:    status,
		Search:    search,
		StartDate: startDate,
		EndDate:   endDate,
	}

	users, total, err := c.UserService.GetUsers(page, pageSize, filter)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, gin.H{
		"items": users,
		"total": total,
		"page":  page,
		"pages": (total + pageSize - 1) / pageSize,
	})
}

// GetUser godoc
// @Summary 获取单个用户信息
// @Description 根据ID获取用户详细信息
// @Tags 用户管理
// @Accept  json
// @Produce  json
// @Security ApiKeyAuth
// @Param   id path int true "用户ID"
// @Success 200 {object} util.Response{data=model.User} "成功"
// @Failure 400 {object} util.Response "请求参数错误"
// @Failure 401 {object} util.Response "未授权"
// @Failure 404 {object} util.Response "用户不存在"
// @Router /api/admin/users/{id} [get]
func (c *UserController) GetUser(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		util.BadRequest(ctx, "无效的用户ID")
		return
	}

	user, err := c.UserService.GetUserByID(uint(id))
	if err != nil {
		util.NotFound(ctx)
		return
	}

	util.Success(ctx, user)
}

// UpdateUser godoc
// @Summary 更新用户信息
// @Description 更新用户的详细信息
// @Tags 用户管理
// @Accept  json
// @Produce  json
// @Security ApiKeyAuth
// @Param   id path int true "用户ID"
// @Param   body body UpdateUserRequest true "用户更新信息"
// @Success 200 {object} util.Response{data=model.User} "成功"
// @Failure 400 {object} util.Response "请求参数错误"
// @Failure 401 {object} util.Response "未授权"
// @Failure 404 {object} util.Response "用户不存在"
// @Router /api/admin/users/{id} [put]
func (c *UserController) UpdateUser(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		util.BadRequest(ctx, "无效的用户ID")
		return
	}

	var req UpdateUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	user := &model.User{
		Name:     req.Name,
		Email:    req.Email,
		Role:     model.UserRole(req.Role),
		Language: req.Language,
		Disabled: req.Disabled,
	}
	user.ID = uint(id)

	// 如果提供了密码，则更新密码
	if req.Password != "" {
		err := c.UserService.UpdateUserWithPassword(user, req.Password)
		if err != nil {
			if err.Error() == "用户不存在" {
				util.NotFound(ctx)
			} else {
				util.InternalServerError(ctx)
			}
			return
		}
	} else {
		// 如果没有提供密码，使用原有的更新方法
		if err := c.UserService.UpdateUser(user); err != nil {
			if err.Error() == "用户不存在" {
				util.NotFound(ctx)
			} else {
				util.InternalServerError(ctx)
			}
			return
		}
	}

	updatedUser, _ := c.UserService.GetUserByID(uint(id))
	util.Success(ctx, updatedUser)
}

// ResetPassword godoc
// @Summary 重置用户密码
// @Description 重置用户密码并返回临时密码
// @Tags 用户管理
// @Accept  json
// @Produce  json
// @Security ApiKeyAuth
// @Param   id path int true "用户ID"
// @Success 200 {object} util.Response{data=string} "成功"
// @Failure 400 {object} util.Response "请求参数错误"
// @Failure 401 {object} util.Response "未授权"
// @Failure 404 {object} util.Response "用户不存在"
// @Router /api/admin/users/{id}/reset-password [post]
func (c *UserController) ResetPassword(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		util.BadRequest(ctx, "无效的用户ID")
		return
	}

	tempPassword, err := c.UserService.ResetPassword(uint(id))
	if err != nil {
		if err.Error() == "用户不存在" {
			util.NotFound(ctx)
		} else {
			util.InternalServerError(ctx)
		}
		return
	}

	util.Success(ctx, gin.H{"tempPassword": tempPassword})
}

// DeleteUser godoc
// @Summary 删除用户
// @Description 删除指定的用户
// @Tags 用户管理
// @Accept  json
// @Produce  json
// @Security ApiKeyAuth
// @Param   id path int true "用户ID"
// @Success 200 {object} util.Response "成功"
// @Failure 400 {object} util.Response "请求参数错误"
// @Failure 401 {object} util.Response "未授权"
// @Failure 404 {object} util.Response "用户不存在"
// @Router /api/admin/users/{id} [delete]
func (c *UserController) DeleteUser(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		util.BadRequest(ctx, "无效的用户ID")
		return
	}

	if err := c.UserService.DeleteUser(uint(id)); err != nil {
		if err.Error() == "用户不存在" {
			util.NotFound(ctx)
		} else {
			util.InternalServerError(ctx)
		}
		return
	}

	util.Success(ctx, gin.H{"message": "用户已成功删除"})
}

// DisableUser godoc
// @Summary 禁用/启用用户
// @Description 禁用或启用指定的用户
// @Tags 用户管理
// @Accept  json
// @Produce  json
// @Security ApiKeyAuth
// @Param   id path int true "用户ID"
// @Param   disable query bool true "是否禁用"
// @Success 200 {object} util.Response "成功"
// @Failure 400 {object} util.Response "请求参数错误"
// @Failure 401 {object} util.Response "未授权"
// @Failure 404 {object} util.Response "用户不存在"
// @Router /api/admin/users/{id}/disable [post]
func (c *UserController) DisableUser(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		util.BadRequest(ctx, "无效的用户ID")
		return
	}

	disableStr := ctx.Query("disable")
	disable := disableStr == "true"

	if err := c.UserService.DisableUser(uint(id), disable); err != nil {
		if err.Error() == "用户不存在" {
			util.NotFound(ctx)
		} else {
			util.InternalServerError(ctx)
		}
		return
	}

	status := "启用"
	if disable {
		status = "禁用"
	}

	util.Success(ctx, gin.H{"message": fmt.Sprintf("用户已成功%s", status)})
}
