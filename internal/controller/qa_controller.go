package controller

import (
	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/service"
	"coder_edu_backend/internal/util"
	"io"
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
	// 从上下文获取当前用户信息
	user, exists := ctx.Get("user")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}
	claims := user.(*util.Claims)
	userID := claims.UserID

	// 1. Redis频率限制校验
	allowed, err := c.qaService.CheckRateLimit(userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "系统繁忙，请稍后再试"})
		return
	}
	if !allowed {
		ctx.JSON(http.StatusTooManyRequests, gin.H{"error": "提问太频繁了，请休息一分钟再来吧"})
		return
	}

	var req service.AskRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 流式响应，传入userID和sessionID支持多轮对话
	stream, source, errChan := c.qaService.AskStream(userID, req.Question, req.SessionID)
	if stream == nil {
		// 处理 AskStream 返回 nil 的情况（如触发敏感词）
		if err := <-errChan; err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "message": err.Error()})
			return
		}
	}

	// SSE响应头
	ctx.Header("Content-Type", "text/event-stream")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Connection", "keep-alive")
	ctx.Header("Transfer-Encoding", "chunked")

	// 实时发送源信息
	ctx.SSEvent("source", source)
	ctx.Writer.Flush()

	// 处理流式响应，支持客户端断开检测
	ctx.Stream(func(w io.Writer) bool {
		select {
		case content, ok := <-stream:
			if !ok {
				// 流结束，检查是否有错误
				select {
				case err := <-errChan:
					if err != nil {
						ctx.SSEvent("error", err.Error())
					}
				default:
				}
				ctx.SSEvent("end", "done")
				return false
			}
			ctx.SSEvent("message", content)
			return true
		case err := <-errChan:
			if err != nil {
				ctx.SSEvent("error", err.Error())
			}
			ctx.SSEvent("end", "done")
			return false
		case <-ctx.Request.Context().Done():
			return false
		}
	})
}

// GetHistory 获取 AI 问答历史记录
// @Summary 获取 AI 问答历史
// @Tags QA
// @Security ApiKeyAuth
// @Param page query int false "页码"
// @Param limit query int false "每页数量"
// @Param sessionId query string false "会话 ID"
// @Success 200 {object} gin.H
// @Router /api/qa/history [get]
func (c *QAController) GetHistory(ctx *gin.Context) {
	// 从上下文获取当前用户信息
	user, exists := ctx.Get("user")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}
	claims := user.(*util.Claims)
	userID := claims.UserID
	sessionID := ctx.Query("sessionId")

	page := 1
	limit := 20

	var histories []model.AIQAHistory
	var total int64
	var err error

	if sessionID != "" {
		// 如果传了 sessionId，只查该会话的历史
		db := c.qaService.GetDB().Model(&model.AIQAHistory{}).Where("user_id = ? AND session_id = ?", userID, sessionID)
		db.Count(&total)
		err = db.Order("created_at desc").Offset((page - 1) * limit).Limit(limit).Find(&histories).Error
	} else {
		// 否则查该用户所有的历史
		histories, total, err = c.qaService.GetHistory(userID, limit, (page-1)*limit)
	}

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"items": histories,
		"total": total,
	})
}

// GetHistoryDetail 获取特定会话的历史详情
// @Summary 获取会话历史详情
// @Tags QA
// @Security ApiKeyAuth
// @Param sessionId query string true "会话 ID"
// @Success 200 {object} []model.AIQAHistory
// @Router /api/qa/history/detail [get]
func (c *QAController) GetHistoryDetail(ctx *gin.Context) {
	// 从上下文获取当前用户信息
	user, exists := ctx.Get("user")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}
	claims := user.(*util.Claims)
	userID := claims.UserID
	sessionID := ctx.Query("sessionId")

	if sessionID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "sessionId 不能为空"})
		return
	}

	var histories []model.AIQAHistory
	err := c.qaService.GetDB().Where("user_id = ? AND session_id = ?", userID, sessionID).
		Order("created_at asc").Find(&histories).Error

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, histories)
}

// DeleteSession 删除指定会话的所有历史记录
// @Summary 删除 AI 问答会话
// @Description 根据 sessionId 删除该会话的所有历史记录
// @Tags QA
// @Security ApiKeyAuth
// @Param sessionId path string true "会话 ID"
// @Success 200 {object} gin.H
// @Router /api/qa/history/{sessionId} [delete]
func (c *QAController) DeleteSession(ctx *gin.Context) {
	user, exists := ctx.Get("user")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}
	claims := user.(*util.Claims)
	userID := claims.UserID
	sessionID := ctx.Param("sessionId")

	if sessionID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "sessionId 不能为空"})
		return
	}

	if err := c.qaService.DeleteSession(userID, sessionID); err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "会话已删除"})
}

// GetWeeklyReport 获取学习周报 (SSE)
// @Summary 获取学习周报
// @Description 生成并获取用户的学习周报，采用 SSE 流式返回
// @Tags QA
// @Security ApiKeyAuth
// @Produce text/event-stream
// @Success 200 {string} string "SSE stream"
// @Router /api/qa/report/weekly [get]
func (c *QAController) GetWeeklyReport(ctx *gin.Context) {
	user, exists := ctx.Get("user")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}
	claims := user.(*util.Claims)
	userID := claims.UserID

	out, errChan := c.qaService.GenerateWeeklyReport(userID)

	ctx.Header("Content-Type", "text/event-stream")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Connection", "keep-alive")
	ctx.Header("Transfer-Encoding", "chunked")

	ctx.Stream(func(w io.Writer) bool {
		select {
		case content, ok := <-out:
			if !ok {
				ctx.SSEvent("message", "[DONE]")
				return false
			}
			ctx.SSEvent("message", content)
			return true
		case err := <-errChan:
			if err != nil {
				ctx.SSEvent("error", err.Error())
			}
			return false
		case <-ctx.Request.Context().Done():
			return false
		}
	})
}

// DiagnoseRequest 代码诊断请求
type DiagnoseRequest struct {
	QuestionID    uint   `json:"questionId" binding:"required"`
	Code          string `json:"code" binding:"required"`
	CompilerError string `json:"compilerError"`
}

// @Summary AI 代码自动诊断
// @Description 结合题目背景、用户代码和编译器报错信息，提供启发式的代码诊断建议
// @Tags QA
// @Accept json
// @Produce text/event-stream
// @Security ApiKeyAuth
// @Param request body DiagnoseRequest true "代码诊断请求参数"
// @Success 200 {string} string "SSE stream"
// @Router /api/qa/diagnose [post]
func (c *QAController) DiagnoseCode(ctx *gin.Context) {
	user, exists := ctx.Get("user")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}
	claims := user.(*util.Claims)
	userID := claims.UserID

	var req DiagnoseRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	out, errChan := c.qaService.DiagnoseCode(userID, req.QuestionID, req.Code, req.CompilerError)

	ctx.Header("Content-Type", "text/event-stream")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Connection", "keep-alive")

	ctx.Stream(func(w io.Writer) bool {
		select {
		case content, ok := <-out:
			if !ok {
				ctx.SSEvent("message", "[DONE]")
				return false
			}
			ctx.SSEvent("message", content)
			return true
		case err := <-errChan:
			if err != nil {
				ctx.SSEvent("error", err.Error())
			}
			return false
		case <-ctx.Request.Context().Done():
			return false
		}
	})
}
