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

		// fmt.Printf("用户角色: '%s' (长度: %d), 允许的角色列表: %v\n", user.Role, len(user.Role), roles)

		hasRole := false
		for _, role := range roles {
			// fmt.Printf("比较角色: 用户角色 '%s' vs 允许角色 '%s'\n", user.Role, role)
			if string(user.Role) == string(role) {
				hasRole = true
				// fmt.Printf("角色匹配成功: %s\n", role)
				break
			}
		}

		if !hasRole {
			// fmt.Printf("角色验证失败: 用户角色 '%s' 不在允许的角色列表 %v 中\n", user.Role, roles)
			util.Forbidden(c)
			c.Abort()
			return
		}
		// fmt.Printf("角色验证成功，允许访问\n")
		c.Next()
	}
}
