package controller

import (
	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/service"
	"coder_edu_backend/internal/util"

	"github.com/gin-gonic/gin"
)

type AuthController struct {
	AuthService *service.AuthService
}

func NewAuthController(authService *service.AuthService) *AuthController {
	return &AuthController{AuthService: authService}
}

// RegisterRequest defines model for registration
// swagger:model RegisterRequest
type RegisterRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Role     string `json:"role" binding:"required,oneof=student teacher admin"`
}

// Register godoc
// @Summary Register a new user
// @Description Register a new user with the given information
// @Tags auth
// @Accept  json
// @Produce  json
// @Param   body body RegisterRequest true "User registration information"
// @Success 201 {object} util.Response{data=object} "Created"
// @Failure 400 {object} util.Response "Bad Request"
// @Failure 500 {object} util.Response "Internal Server Error"
// @Router /api/register [post]
func (c *AuthController) Register(ctx *gin.Context) {
	var req RegisterRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	user := &model.User{
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
		Role:     model.UserRole(req.Role),
	}

	if err := c.AuthService.Register(user); err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Created(ctx, gin.H{"id": user.ID})
}

// LoginRequest defines model for login
// swagger:model LoginRequest
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// Login godoc
// @Summary User login
// @Description Authenticate user and return JWT token
// @Tags auth
// @Accept  json
// @Produce  json
// @Param   body body LoginRequest true "User login credentials"
// @Success 200 {object} util.Response{data=object} "Success"
// @Failure 400 {object} util.Response "Bad Request"
// @Failure 401 {object} util.Response "Unauthorized"
// @Router /api/login [post]
func (c *AuthController) Login(ctx *gin.Context) {
	var req LoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	token, err := c.AuthService.Login(req.Email, req.Password)
	if err != nil {
		util.Unauthorized(ctx)
		return
	}

	util.Success(ctx, gin.H{"token": token})
}

// GetProfile godoc
// @Summary Get current user profile
// @Description Get the profile of the currently authenticated user
// @Tags auth
// @Accept  json
// @Produce  json
// @Security ApiKeyAuth
// @Success 200 {object} util.Response{data=model.User} "Success"
// @Failure 401 {object} util.Response "Unauthorized"
// @Router /api/profile [get]
func (c *AuthController) GetProfile(ctx *gin.Context) {
	user := c.AuthService.GetCurrentUser(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	profile := gin.H{
		"id":        user.ID,
		"name":      user.Name,
		"email":     user.Email,
		"role":      user.Role,
		"xp":        user.XP,
		"language":  user.Language,
		"createdAt": user.CreatedAt,
	}

	util.Success(ctx, profile)
}
