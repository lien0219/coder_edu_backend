package util

import (
	"coder_edu_backend/internal/model"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID uint           `json:"user_id"`
	Role   model.UserRole `json:"role"`
	Email  string         `json:"email"`
	jwt.RegisteredClaims
}

func GenerateJWT(user *model.User, secret string, expiration time.Duration) (string, error) {
	expirationTime := time.Now().Add(expiration)

	claims := &Claims{
		UserID: user.ID,
		Role:   user.Role,
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func ParseJWT(tokenString, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, err
}

func GetUserFromContext(c *gin.Context) *Claims {
	user, exists := c.Get("user")
	if !exists {
		return nil
	}
	claims, ok := user.(*Claims)
	if !ok {
		return nil
	}
	return claims
}
