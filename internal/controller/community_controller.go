package controller

import (
	"coder_edu_backend/internal/service"
	"coder_edu_backend/internal/util"
	"strconv"

	"github.com/gin-gonic/gin"
)

type CommunityController struct {
	CommunityService *service.CommunityService
}

func NewCommunityController(communityService *service.CommunityService) *CommunityController {
	return &CommunityController{CommunityService: communityService}
}

// @Summary 获取讨论帖子
// @Description 获取社区讨论帖子列表
// @Tags 社区
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param limit query int false "每页数量" default(20)
// @Param tag query string false "标签筛选"
// @Param sort query string false "排序方式" Enums(new, popular) default(new)
// @Success 200 {object} util.Response
// @Router /api/community/posts [get]
func (c *CommunityController) GetPosts(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "20"))
	tag := ctx.Query("tag")
	sort := ctx.DefaultQuery("sort", "new")

	posts, total, err := c.CommunityService.GetPosts(page, limit, tag, sort)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, gin.H{
		"posts": posts,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// @Summary 创建讨论帖子
// @Description 创建新的讨论帖子
// @Tags 社区
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param post body service.PostRequest true "帖子内容"
// @Success 200 {object} util.Response
// @Router /api/community/posts [post]
func (c *CommunityController) CreatePost(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	var req service.PostRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	post, err := c.CommunityService.CreatePost(user.UserID, req)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Created(ctx, post)
}

// @Summary 获取问题列表
// @Description 获取问答区问题列表
// @Tags 社区
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param limit query int false "每页数量" default(20)
// @Param tag query string false "标签筛选"
// @Param solved query bool false "是否已解决"
// @Success 200 {object} util.Response
// @Router /api/community/questions [get]
func (c *CommunityController) GetQuestions(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "20"))
	tag := ctx.Query("tag")
	solvedStr := ctx.Query("solved")

	var solved *bool
	if solvedStr != "" {
		s := solvedStr == "true"
		solved = &s
	}

	questions, total, err := c.CommunityService.GetQuestions(page, limit, tag, solved)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, gin.H{
		"questions": questions,
		"total":     total,
		"page":      page,
		"limit":     limit,
	})
}

// @Summary 创建问题
// @Description 创建新的问题
// @Tags 社区
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param question body service.QuestionRequest true "问题内容"
// @Success 200 {object} util.Response
// @Router /api/community/questions [post]
func (c *CommunityController) CreateQuestion(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	var req service.QuestionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	question, err := c.CommunityService.CreateQuestion(user.UserID, req)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Created(ctx, question)
}

// @Summary 回答问题
// @Description 回答一个问题
// @Tags 社区
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param questionId path int true "问题ID"
// @Param answer body service.AnswerRequest true "回答内容"
// @Success 200 {object} util.Response
// @Router /api/community/questions/{questionId}/answers [post]
func (c *CommunityController) AnswerQuestion(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	questionIDStr := ctx.Param("questionId")
	questionID, err := strconv.Atoi(questionIDStr)
	if err != nil {
		util.BadRequest(ctx, "Invalid question ID")
		return
	}

	var req service.AnswerRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	answer, err := c.CommunityService.AnswerQuestion(user.UserID, uint(questionID), req)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Created(ctx, answer)
}

// @Summary 点赞内容
// @Description 给帖子、评论或回答点赞
// @Tags 社区
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param type path string true "内容类型" Enums(post, comment, answer)
// @Param id path int true "内容ID"
// @Success 200 {object} util.Response
// @Router /api/community/{type}/{id}/upvote [post]
func (c *CommunityController) Upvote(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	contentType := ctx.Param("type")
	contentIDStr := ctx.Param("id")
	contentID, err := strconv.Atoi(contentIDStr)
	if err != nil {
		util.BadRequest(ctx, "Invalid content ID")
		return
	}

	err = c.CommunityService.Upvote(user.UserID, contentType, uint(contentID))
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, gin.H{"message": "Upvoted successfully"})
}
