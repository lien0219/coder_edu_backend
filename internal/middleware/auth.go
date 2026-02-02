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
		tokenString := ""
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			tokenString = strings.TrimPrefix(authHeader, "Bearer ")
		}

		if tokenString == "" {
			tokenString = c.Query("token")
		}

		if tokenString == "" {
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
			// 允许管理员拥有所有教师权限：管理员直接放行
			if string(user.Role) == string(model.Admin) {
				hasRole = true
				break
			}
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

type UserActivityRepo interface {
	UpdateLastSeen(userID uint) error
}

func ActivityMiddleware(repo UserActivityRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := util.GetUserFromContext(c)
		if claims != nil {
			// 异步更新，不阻塞主流程
			go repo.UpdateLastSeen(claims.UserID)
		}
		c.Next()
	}
}
