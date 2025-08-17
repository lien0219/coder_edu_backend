package util

import (
	"coder_edu_backend/internal/model"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
)

type Claims struct {
	UserID uint           `json:"user_id"`
	Role   model.UserRole `json:"role"`
	Email  string         `json:"email"`
	jwt.StandardClaims
}

func GenerateJWT(user *model.User, secret string, expireHours time.Duration) (string, error) {
	expirationTime := time.Now().Add(expireHours)

	claims := &Claims{
		UserID: user.ID,
		Role:   user.Role,
		Email:  user.Email,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
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
	return user.(*Claims)
}
