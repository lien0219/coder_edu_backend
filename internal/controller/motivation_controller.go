package controller

import (
	"bytes"
	"coder_edu_backend/internal/service"
	"coder_edu_backend/internal/util"
	"fmt"
	"io/ioutil"
	"strconv"

	"github.com/gin-gonic/gin"
)

type MotivationController struct {
	MotivationService *service.MotivationService
}

func NewMotivationController(motivationService *service.MotivationService) *MotivationController {
	return &MotivationController{MotivationService: motivationService}
}

// @Summary 获取当前显示的激励短句
// @Description 获取当前显示的每日激励短句
// @Tags 激励短句
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} util.Response
// @Router /api/motivation [get]
func (c *MotivationController) GetCurrentMotivation(ctx *gin.Context) {
	motivation, err := c.MotivationService.GetCurrentMotivation()
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, gin.H{"content": motivation})
}

// @Summary 获取所有激励短句
// @Description 获取系统中所有的激励短句（管理员权限）
// @Tags 激励短句
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} util.Response
// @Router /api/admin/motivations [get]
func (c *MotivationController) GetAllMotivations(ctx *gin.Context) {
	motivations, err := c.MotivationService.GetAllMotivations()
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, motivations)
}

// @Summary 创建新的激励短句
// @Description 创建新的激励短句（管理员权限）
// @Tags 激励短句
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param content body string true "激励短句内容"
// @Success 200 {object} util.Response
// @Router /api/admin/motivations [post]
func (c *MotivationController) CreateMotivation(ctx *gin.Context) {
	var req struct {
		Content string `json:"content" binding:"required,min=10,max=200"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	err := c.MotivationService.CreateMotivation(req.Content)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, gin.H{"message": "激励短句创建成功"})
}

// @Summary 更新激励短句
// @Description 更新激励短句内容和状态（管理员权限）
// @Tags 激励短句
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "激励短句ID"
// @Param motivation body model.Motivation true "激励短句信息"
// @Success 200 {object} util.Response
// @Router /api/admin/motivations/{id} [put]
func (c *MotivationController) UpdateMotivation(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		util.BadRequest(ctx, "无效的ID")
		return
	}

	body, _ := ioutil.ReadAll(ctx.Request.Body)
	fmt.Printf("原始请求体: %s\n", string(body))

	ctx.Request.Body = ioutil.NopCloser(bytes.NewBuffer(body))

	var req struct {
		Content   string `json:"content" binding:"required,min=10,max=200"`
		IsEnabled *bool  `json:"is_enabled" binding:"required"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	if req.IsEnabled == nil {
		util.BadRequest(ctx, "IsEnabled字段不能为空")
		return
	}

	err = c.MotivationService.UpdateMotivation(uint(id), req.Content, *req.IsEnabled)
	if err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	util.Success(ctx, gin.H{"message": "激励短句更新成功"})
}

// @Summary 删除激励短句
// @Description 删除激励短句（管理员权限）
// @Tags 激励短句
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "激励短句ID"
// @Success 200 {object} util.Response
// @Router /api/admin/motivations/{id} [delete]
func (c *MotivationController) DeleteMotivation(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		util.BadRequest(ctx, "无效的ID")
		return
	}

	err = c.MotivationService.DeleteMotivation(uint(id))
	if err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	util.Success(ctx, gin.H{"message": "激励短句删除成功"})
}

// @Summary 立即切换激励短句
// @Description 立即切换到指定的激励短句（管理员权限）
// @Tags 激励短句
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "激励短句ID"
// @Success 200 {object} util.Response
// @Router /api/admin/motivations/{id}/switch [post]
func (c *MotivationController) SwitchMotivation(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		util.BadRequest(ctx, "无效的ID")
		return
	}

	err = c.MotivationService.SwitchToMotivation(uint(id))
	if err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	util.Success(ctx, gin.H{"message": "激励短句切换成功"})
}
