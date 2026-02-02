package util

import (
	"coder_edu_backend/pkg/logger"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// PageResponse 分页响应结构
type PageResponse struct {
	List  interface{} `json:"list"`
	Total int64       `json:"total"`
	Page  int         `json:"page"`
	Limit int         `json:"limit"`
}

func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "success",
		Data:    data,
	})
}

func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, Response{
		Code:    http.StatusCreated,
		Message: "created",
		Data:    data,
	})
}

func Error(c *gin.Context, code int, message string) {
	c.JSON(code, Response{
		Code:    code,
		Message: message,
	})
}

func Unauthorized(c *gin.Context) {
	Error(c, http.StatusUnauthorized, "Unauthorized")
}

func Forbidden(c *gin.Context) {
	Error(c, http.StatusForbidden, "Forbidden")
}

func BadRequest(c *gin.Context, message string) {
	Error(c, http.StatusBadRequest, message)
}

func NotFound(c *gin.Context) {
	Error(c, http.StatusNotFound, "Resource not found")
}

func InternalServerError(c *gin.Context) {
	Error(c, http.StatusInternalServerError, "Internal server error")
}

func LogInternalError(c *gin.Context, err error) {
	logger.Log.Error("Internal server error", zap.Error(err))
	InternalServerError(c)
}
