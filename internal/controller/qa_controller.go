package controller

import (
	"coder_edu_backend/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

type QAController struct {
	qaService *service.QAService
}

func NewQAController(qaService *service.QAService) *QAController {
	return &QAController{qaService: qaService}
}

// Ask 处理 AI 问答请求
// @Summary AI 知识库问答
// @Description 先检索知识库，如果没有则调用大模型回答
// @Tags QA
// @Accept json
// @Produce json
// @Param request body service.AskRequest true "问题内容"
// @Success 200 {object} service.AskResponse
// @Router /api/qa/ask [post]
func (c *QAController) Ask(ctx *gin.Context) {
	var req service.AskRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 流式响应
	stream, source, errChan := c.qaService.AskStream(req.Question)

	// 设置SSE响应头
	ctx.Header("Content-Type", "text/event-stream")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Connection", "keep-alive")
	ctx.Header("Transfer-Encoding", "chunked")

	// 实时发送源信息
	ctx.SSEvent("source", source)
	ctx.Writer.Flush()

	// 循环读取并发送AI回答内容
	for content := range stream {
		ctx.SSEvent("message", content)
		ctx.Writer.Flush()
	}

	// 检查是否有错误发生
	if err := <-errChan; err != nil {
		ctx.SSEvent("error", err.Error())
		ctx.Writer.Flush()
	}

	ctx.SSEvent("end", "done")
	ctx.Writer.Flush()
}
