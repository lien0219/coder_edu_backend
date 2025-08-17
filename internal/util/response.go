package util

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response represents the standard API response format
// swagger:model Response
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Success creates a successful response
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
