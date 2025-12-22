package controller

import (
	"coder_edu_backend/internal/service"
	"coder_edu_backend/internal/util"

	"github.com/gin-gonic/gin"
)

type KnowledgeTagController struct {
	Service *service.KnowledgeTagService
}

func NewKnowledgeTagController(s *service.KnowledgeTagService) *KnowledgeTagController {
	return &KnowledgeTagController{Service: s}
}

// @Summary 获取知识点标签列表
// @Tags 知识点
// @Produce json
// @Security BearerAuth
// @Success 200 {object} util.Response
// @Router /api/knowledge-tags [get]
func (c *KnowledgeTagController) ListTags(ctx *gin.Context) {
	tags, err := c.Service.ListTags()
	if err != nil {
		util.InternalServerError(ctx)
		return
	}
	util.Success(ctx, tags)
}
