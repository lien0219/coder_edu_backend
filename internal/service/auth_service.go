package service

import (
	"coder_edu_backend/internal/config"
	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/repository"
	"coder_edu_backend/internal/util"
	"errors"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService struct {
	UserRepo *repository.UserRepository
	Cfg      *config.Config
}

func NewAuthService(userRepo *repository.UserRepository, cfg *config.Config) *AuthService {
	return &AuthService{
		UserRepo: userRepo,
		Cfg:      cfg,
	}
}

func (s *AuthService) Register(user *model.User) error {
	_, err := s.UserRepo.FindByEmail(user.Email)
	if err == nil {
		return errors.New("该邮箱已被注册")
	} else if err != gorm.ErrRecordNotFound {
		return err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.Password = string(hashedPassword)
	return s.UserRepo.Create(user)
}

func (s *AuthService) Login(email, password string) (string, error) {
	user, err := s.UserRepo.FindByEmail(email)
	if err != nil {
		return "", errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", errors.New("invalid credentials")
	}

	return util.GenerateJWT(user, s.Cfg.JWT.Secret, s.Cfg.JWT.ExpireTime)
}

func (s *AuthService) GetCurrentUser(c *gin.Context) *model.User {
	claims := util.GetUserFromContext(c)
	if claims == nil {
		return nil
	}

	user, _ := s.UserRepo.FindByID(claims.UserID)
	return user
}
