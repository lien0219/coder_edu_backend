package controller

import (
	"coder_edu_backend/internal/service"
	"coder_edu_backend/internal/util"

	"github.com/gin-gonic/gin"
)

type KnowledgePointController struct {
	Service *service.KnowledgePointService
}

func NewKnowledgePointController(svc *service.KnowledgePointService) *KnowledgePointController {
	return &KnowledgePointController{Service: svc}
}

// @Summary 创建知识点 (老师/管理员)
// @Tags 知识点
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body service.CreateKnowledgePointRequest true "知识点信息"
// @Success 201 {object} util.Response
// @Router /api/teacher/knowledge-points [post]
func (c *KnowledgePointController) Create(ctx *gin.Context) {
	var req service.CreateKnowledgePointRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	kp, err := c.Service.CreateKnowledgePoint(req)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Created(ctx, kp)
}

// @Summary 获取知识点列表 (老师/管理员)
// @Tags 知识点
// @Produce json
// @Security BearerAuth
// @Param title query string false "标题筛选"
// @Success 200 {object} util.Response
// @Router /api/teacher/knowledge-points [get]
func (c *KnowledgePointController) List(ctx *gin.Context) {
	title := ctx.Query("title")

	kps, err := c.Service.ListKnowledgePoints(title)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, kps)
}

// @Summary 更新知识点 (老师/管理员)
// @Tags 知识点
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "知识点ID"
// @Param body body service.CreateKnowledgePointRequest true "知识点信息"
// @Success 200 {object} util.Response
// @Router /api/teacher/knowledge-points/{id} [put]
func (c *KnowledgePointController) Update(ctx *gin.Context) {
	id := ctx.Param("id")
	var req service.CreateKnowledgePointRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	kp, err := c.Service.UpdateKnowledgePoint(id, req)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, kp)
}

// @Summary 删除知识点 (老师/管理员)
// @Tags 知识点
// @Produce json
// @Security BearerAuth
// @Param id path string true "知识点ID"
// @Success 200 {object} util.Response
// @Router /api/teacher/knowledge-points/{id} [delete]
func (c *KnowledgePointController) Delete(ctx *gin.Context) {
	id := ctx.Param("id")

	if err := c.Service.DeleteKnowledgePoint(id); err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, gin.H{"deleted": id})
}
