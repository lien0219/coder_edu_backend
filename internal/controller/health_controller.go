package controller

import (
	"coder_edu_backend/internal/util"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type HealthController struct {
	DB *gorm.DB
}

func NewHealthController(db *gorm.DB) *HealthController {
	return &HealthController{DB: db}
}

// @Summary 健康检查
// @Description 检查服务状态
// @Tags 系统
// @Produce json
// @Success 200 {object} util.Response
// @Router /health [get]
func (c *HealthController) HealthCheck(ctx *gin.Context) {
	// 检查数据库连接
	sqlDB, err := c.DB.DB()
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	if err := sqlDB.Ping(); err != nil {
		util.Error(ctx, http.StatusServiceUnavailable, "Database unavailable")
		return
	}

	util.Success(ctx, gin.H{
		"status": "ok",
		"components": gin.H{
			"database": "up",
		},
	})
}
