package middleware

import (
	"coder_edu_backend/internal/config"
	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/util"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			util.Unauthorized(c)
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			util.Unauthorized(c)
			c.Abort()
			return
		}

		cfg := c.MustGet("config").(*config.Config)
		claims, err := util.ParseJWT(tokenString, cfg.JWT.Secret)
		if err != nil {
			fmt.Printf("JWT解析错误: %v\n", err)
			util.Unauthorized(c)
			c.Abort()
			return
		}

		c.Set("user", claims)
		c.Next()
	}
}

func RoleMiddleware(roles ...model.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := util.GetUserFromContext(c)
		if user == nil {
			util.Unauthorized(c)
			c.Abort()
			return
		}

		hasRole := false
		for _, role := range roles {
			if user.Role == role {
				hasRole = true
				break
			}
		}

		if !hasRole {
			util.Forbidden(c)
			c.Abort()
			return
		}

		c.Next()
	}
}
