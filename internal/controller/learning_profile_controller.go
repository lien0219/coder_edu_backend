package controller

import (
	"errors"
	"net/http"

	"coder_edu_backend/internal/learningprofile"
	"coder_edu_backend/internal/service"
	"coder_edu_backend/internal/util"

	"github.com/gin-gonic/gin"
)

type learningProfileAPI interface {
	GetMe(userID uint) (*service.LearningProfileMe, error)
	GetPaper(userID uint, instrumentCode, wave string) (*service.LearningProfilePaper, error)
	SaveDraft(userID uint, instrumentCode, wave string, answers map[string]int) (*service.LearningProfileWriteResult, error)
	Submit(userID uint, instrumentCode, wave string, answers map[string]int) (*service.LearningProfileWriteResult, error)
	GetSdlProfile(userID uint) (*service.SdlLearningProfile, error)
}

type LearningProfileController struct {
	svc learningProfileAPI
}

func NewLearningProfileController(svc learningProfileAPI) *LearningProfileController {
	return &LearningProfileController{svc: svc}
}

type learningProfileWriteBody struct {
	Answers map[string]int `json:"answers"`
}

func (ctrl *LearningProfileController) GetMe(c *gin.Context) {
	userID, ok := learningProfileSelfUserID(c)
	if !ok {
		return
	}
	data, err := ctrl.svc.GetMe(userID)
	if err != nil {
		writeLearningProfileError(c, err)
		return
	}
	util.Success(c, data)
}

func (ctrl *LearningProfileController) GetSdlProfile(c *gin.Context) {
	userID, ok := learningProfileSelfUserID(c)
	if !ok {
		return
	}
	data, err := ctrl.svc.GetSdlProfile(userID)
	if err != nil {
		writeLearningProfileError(c, err)
		return
	}
	util.Success(c, data)
}

func (ctrl *LearningProfileController) GetPaper(c *gin.Context) {
	userID, code, wave, ok := ctrl.paperContext(c)
	if !ok {
		return
	}
	data, err := ctrl.svc.GetPaper(userID, code, wave)
	if err != nil {
		writeLearningProfileError(c, err)
		return
	}
	util.Success(c, data)
}

func (ctrl *LearningProfileController) SaveDraft(c *gin.Context) {
	userID, code, wave, ok := ctrl.paperContext(c)
	if !ok {
		return
	}
	answers, ok := bindLearningProfileAnswers(c)
	if !ok {
		return
	}
	data, err := ctrl.svc.SaveDraft(userID, code, wave, answers)
	if err != nil {
		writeLearningProfileError(c, err)
		return
	}
	util.Success(c, data)
}

func (ctrl *LearningProfileController) Submit(c *gin.Context) {
	userID, code, wave, ok := ctrl.paperContext(c)
	if !ok {
		return
	}
	answers, ok := bindLearningProfileAnswers(c)
	if !ok {
		return
	}
	data, err := ctrl.svc.Submit(userID, code, wave, answers)
	if err != nil {
		writeLearningProfileError(c, err)
		return
	}
	util.Success(c, data)
}

func (ctrl *LearningProfileController) paperContext(c *gin.Context) (uint, string, string, bool) {
	userID, ok := learningProfileSelfUserID(c)
	if !ok {
		return 0, "", "", false
	}
	code := c.Param("code")
	if err := learningprofile.RequireOfficialInstrument(code); err != nil {
		util.BadRequest(c, err.Error())
		return 0, "", "", false
	}
	wave := c.Param("wave")
	if err := learningprofile.RequireWave(wave); err != nil {
		writeLearningProfileError(c, err)
		return 0, "", "", false
	}
	return userID, code, wave, true
}

func learningProfileSelfUserID(c *gin.Context) (uint, bool) {
	user := util.GetUserFromContext(c)
	if user == nil || user.UserID == 0 {
		util.Unauthorized(c)
		return 0, false
	}
	return user.UserID, true
}

func bindLearningProfileAnswers(c *gin.Context) (map[string]int, bool) {
	var body learningProfileWriteBody
	if err := c.ShouldBindJSON(&body); err != nil {
		util.BadRequest(c, "invalid request")
		return nil, false
	}
	if body.Answers == nil {
		body.Answers = map[string]int{}
	}
	return body.Answers, true
}

func writeLearningProfileError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, util.ErrUnauthorized):
		util.Unauthorized(c)
	case errors.Is(err, learningprofile.ErrPosttestBlocked):
		util.ErrorWithCode(c, http.StatusForbidden, "POSTTEST_BLOCKED", err.Error())
	case errors.Is(err, learningprofile.ErrAlreadySubmitted):
		util.ErrorWithCode(c, http.StatusConflict, "ALREADY_SUBMITTED", err.Error())
	case errors.Is(err, learningprofile.ErrUnknownInstrument),
		errors.Is(err, learningprofile.ErrInvalidWave):
		util.BadRequest(c, err.Error())
	case errors.Is(err, learningprofile.ErrCatalogUnavailable),
		errors.Is(err, learningprofile.ErrCatalogMismatch):
		util.ErrorWithCode(c, http.StatusServiceUnavailable, "CATALOG_UNAVAILABLE", err.Error())
	default:
		util.BadRequest(c, err.Error())
	}
}
