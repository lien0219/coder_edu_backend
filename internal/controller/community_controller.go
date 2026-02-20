package controller

import (
	"coder_edu_backend/internal/service"
	"coder_edu_backend/internal/util"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type CommunityController struct {
	CommunityService *service.CommunityService
}

func NewCommunityController(communityService *service.CommunityService) *CommunityController {
	return &CommunityController{CommunityService: communityService}
}

// @Summary 获取讨论帖子
// @Description 获取社区讨论帖子列表，支持搜索和分类
// @Tags 社区
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param limit query int false "每页数量" default(20)
// @Param search query string false "搜索关键词"
// @Param tab query string false "分类" Enums(new, popular, my) default(new)
// @Param tag query string false "标签筛选"
// @Success 200 {object} util.Response
// @Router /api/community/posts [get]
func (c *CommunityController) GetPosts(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "20"))
	search := ctx.Query("search")
	tab := ctx.DefaultQuery("tab", "new")
	tag := ctx.Query("tag")

	var userID uint
	user := util.GetUserFromContext(ctx)
	if user != nil {
		userID = user.UserID
	}

	if tab == "my" && userID == 0 {
		util.Unauthorized(ctx)
		return
	}

	posts, total, err := c.CommunityService.GetPosts(page, limit, tag, search, tab, userID)
	if err != nil {
		util.LogInternalError(ctx, err)
		return
	}

	util.Success(ctx, gin.H{
		"items": posts,
		"total": total,
	})
}

// @Summary 获取帖子列表(独立分页接口)
// @Description 专门用于“查看更多”页面的帖子列表接口，支持分页和筛选
// @Tags 社区
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param limit query int false "每页数量" default(10)
// @Param search query string false "搜索关键词"
// @Param tab query string false "分类" Enums(new, popular, my)
// @Param tag query string false "标签筛选"
// @Success 200 {object} util.Response
// @Router /api/community/posts/list [get]
func (c *CommunityController) ListPosts(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "10"))
	search := ctx.Query("search")
	tab := ctx.DefaultQuery("tab", "new")
	tag := ctx.Query("tag")

	var userID uint
	user := util.GetUserFromContext(ctx)
	if user != nil {
		userID = user.UserID
	}

	// 如果访问我的，必须登录
	if tab == "my" && userID == 0 {
		util.Unauthorized(ctx)
		return
	}

	posts, total, err := c.CommunityService.GetPosts(page, limit, tag, search, tab, userID)
	if err != nil {
		util.LogInternalError(ctx, err)
		return
	}

	util.Success(ctx, gin.H{
		"items": posts,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// @Summary 获取帖子详情
// @Description 获取帖子主体内容以及评论树结构
// @Tags 社区
// @Accept json
// @Produce json
// @Param id path string true "帖子ID"
// @Success 200 {object} util.Response
// @Router /api/community/posts/{id} [get]
func (c *CommunityController) GetPostDetail(ctx *gin.Context) {
	id := ctx.Param("id")
	var userID uint
	user := util.GetUserFromContext(ctx)
	if user != nil {
		userID = user.UserID
	}

	detail, err := c.CommunityService.GetPostDetail(id, userID, ctx.ClientIP())
	if err != nil {
		util.LogInternalError(ctx, err)
		return
	}

	util.Success(ctx, detail)
}

// @Summary 获取帖子分页评论
// @Description 分页获取帖子的评论列表（树形结构）
// @Tags 社区
// @Accept json
// @Produce json
// @Param id path string true "帖子ID"
// @Param page query int false "页码" default(1)
// @Param limit query int false "每页数量" default(10)
// @Success 200 {object} util.Response
// @Router /api/community/posts/{id}/comments [get]
func (c *CommunityController) GetPostComments(ctx *gin.Context) {
	id := ctx.Param("id")
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "10"))

	var userID uint
	if user := util.GetUserFromContext(ctx); user != nil {
		userID = user.UserID
	}

	comments, total, err := c.CommunityService.GetPostComments(id, page, limit, userID)
	if err != nil {
		util.LogInternalError(ctx, err)
		return
	}

	util.Success(ctx, gin.H{
		"items": comments,
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
		util.BadRequest(ctx, "请求格式错误")
		return
	}

	// 安全性校验
	if req.Title == "" || req.Content == "" {
		util.BadRequest(ctx, "标题和内容不能为空")
		return
	}
	if len(req.Title) > 100 {
		util.BadRequest(ctx, "标题字数过多（最多100个字符）")
		return
	}
	if len(req.Content) > 5000 {
		util.BadRequest(ctx, "讨论内容过长")
		return
	}

	post, err := c.CommunityService.CreatePost(user.UserID, req)
	if err != nil {
		util.LogInternalError(ctx, err)
		return
	}

	ctx.JSON(200, util.Response{
		Code:    200,
		Message: "讨论发布成功",
		Data:    post,
	})
}

// @Summary 更新讨论帖子
// @Description 更新已有的讨论帖子
// @Tags 社区
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "帖子ID"
// @Param post body service.PostRequest true "帖子内容"
// @Success 200 {object} util.Response
// @Router /api/community/posts/{id} [put]
func (c *CommunityController) UpdatePost(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	postID := ctx.Param("id")
	var req service.PostRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, "请求格式错误")
		return
	}

	if req.Title == "" || req.Content == "" {
		util.BadRequest(ctx, "标题和内容不能为空")
		return
	}

	post, err := c.CommunityService.UpdatePost(user.UserID, postID, req, user.Role)
	if err != nil {
		if errors.Is(err, util.ErrPermissionDenied) {
			util.Forbidden(ctx)
		} else {
			util.LogInternalError(ctx, err)
		}
		return
	}

	ctx.JSON(200, util.Response{
		Code:    200,
		Message: "讨论更新成功",
		Data:    post,
	})
}

// @Summary 删除讨论帖子
// @Description 删除已有的讨论帖子及其关联内容
// @Tags 社区
// @Security BearerAuth
// @Param id path string true "帖子ID"
// @Success 200 {object} util.Response
// @Router /api/community/posts/{id} [delete]
func (c *CommunityController) DeletePost(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	postID := ctx.Param("id")
	err := c.CommunityService.DeletePost(user.UserID, postID, user.Role)
	if err != nil {
		if errors.Is(err, util.ErrPermissionDenied) {
			util.Forbidden(ctx)
		} else {
			util.LogInternalError(ctx, err)
		}
		return
	}

	util.Success(ctx, gin.H{"message": "Post deleted successfully"})
}

// @Summary 发表评论/回复
// @Description 在帖子下发表评论，或回复某条评论
// @Tags 社区
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "帖子ID"
// @Param comment body service.CommentCreateRequest true "评论内容"
// @Success 200 {object} util.Response
// @Router /api/community/posts/{id}/comments [post]
func (c *CommunityController) CreateComment(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	postID := ctx.Param("id")
	var req service.CommentCreateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, "请求格式错误")
		return
	}

	if req.Content == "" {
		util.BadRequest(ctx, "内容不能为空")
		return
	}

	res, err := c.CommunityService.CreateComment(user.UserID, postID, req)
	if err != nil {
		util.LogInternalError(ctx, err)
		return
	}

	ctx.JSON(200, util.Response{
		Code:    200,
		Message: "评论发表成功",
		Data:    res,
	})
}

// @Summary 删除评论/回复
// @Description 删除自己的评论或回复。如果删除的是一级评论，其下的所有回复也会被级联删除。
// @Tags 社区
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "评论ID"
// @Success 200 {object} util.Response
// @Router /api/community/comments/{id} [delete]
func (c *CommunityController) DeleteComment(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	commentID := ctx.Param("id")
	err := c.CommunityService.DeleteComment(user.UserID, commentID, user.Role)
	if err != nil {
		if errors.Is(err, util.ErrPermissionDenied) {
			util.Forbidden(ctx)
		} else {
			util.LogInternalError(ctx, err)
		}
		return
	}

	util.Success(ctx, gin.H{"message": "Comment deleted successfully"})
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
// @Param questionId path string true "问题ID"
// @Param answer body service.AnswerRequest true "回答内容"
// @Success 200 {object} util.Response
// @Router /api/community/questions/{questionId}/answers [post]
func (c *CommunityController) AnswerQuestion(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	questionID := ctx.Param("questionId")
	if questionID == "" {
		util.BadRequest(ctx, "Invalid question ID")
		return
	}

	var req service.AnswerRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	answer, err := c.CommunityService.AnswerQuestion(user.UserID, questionID, req)
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
// @Param id path string true "内容ID"
// @Success 200 {object} util.Response
// @Router /api/community/{type}/{id}/upvote [post]
func (c *CommunityController) Upvote(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	contentType := ctx.Param("type")
	contentID := ctx.Param("id")
	if contentID == "" {
		util.BadRequest(ctx, "Invalid content ID")
		return
	}

	isLiked, err := c.CommunityService.Upvote(user.UserID, contentType, contentID)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	msg := "Upvoted successfully"
	if !isLiked {
		msg = "Unliked successfully"
	}
	util.Success(ctx, gin.H{
		"message": msg,
		"isLiked": isLiked,
	})
}

// @Summary 获取资源列表
// @Description 获取社区分享的资源列表
// @Tags 社区
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param limit query int false "每页数量" default(10)
// @Param type query string false "资源类型" Enums(pdf, video, word, article)
// @Param search query string false "搜索关键词"
// @Param sort query string false "排序方式(views:按观看量从高到低)"
// @Success 200 {object} util.Response
// @Router /api/community/resources [get]
func (c *CommunityController) GetResources(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "10"))
	resourceType := ctx.Query("type")
	search := ctx.Query("search")
	sort := ctx.Query("sort")

	var userID uint
	if user := util.GetUserFromContext(ctx); user != nil {
		userID = user.UserID
	}

	resources, total, err := c.CommunityService.GetResources(page, limit, resourceType, search, userID, sort)
	if err != nil {
		util.LogInternalError(ctx, err)
		return
	}

	util.Success(ctx, gin.H{
		"items": resources,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// @Summary 获取资源详情
// @Description 获取资源的详细内容，并增加观看量
// @Tags 社区
// @Accept json
// @Produce json
// @Param id path string true "资源ID"
// @Success 200 {object} util.Response
// @Router /api/community/resources/{id} [get]
func (c *CommunityController) GetResourceDetail(ctx *gin.Context) {
	id := ctx.Param("id")
	var userID uint
	if user := util.GetUserFromContext(ctx); user != nil {
		userID = user.UserID
	}

	detail, err := c.CommunityService.GetResourceDetail(id, userID)
	if err != nil {
		if errors.Is(err, util.ErrResourceNotFound) {
			util.NotFound(ctx)
			return
		}
		util.LogInternalError(ctx, err)
		return
	}

	util.Success(ctx, detail)
}

// @Summary 分享资源
// @Description 上传并分享一个资源（文件或文章）
// @Tags 社区
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param resource body service.ResourceShareRequest true "资源内容"
// @Success 200 {object} util.Response
// @Router /api/community/resources [post]
func (c *CommunityController) CreateResource(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	var req service.ResourceShareRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, "请求格式错误")
		return
	}

	res, err := c.CommunityService.CreateResource(user.UserID, user.Role, req)
	if err != nil {
		if errors.Is(err, util.ErrDailyShareLimit) {
			util.BadRequest(ctx, "每天最多只能分享3次资源")
		} else {
			util.LogInternalError(ctx, err)
		}
		return
	}

	util.Success(ctx, res)
}

// @Summary 上传资源文件
// @Description 上传PDF、视频、Word文件到服务器
// @Tags 社区
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param file formData file true "资源文件"
// @Success 200 {object} util.Response
// @Router /api/community/resources/upload [post]
func (c *CommunityController) UploadResourceFile(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	file, err := ctx.FormFile("file")
	if err != nil {
		util.BadRequest(ctx, "未找到上传的文件")
		return
	}

	fileURL, err := c.CommunityService.UploadResourceFile(ctx, file)
	if err != nil {
		util.LogInternalError(ctx, err)
		return
	}

	util.Success(ctx, gin.H{
		"url": fileURL,
	})
}

// @Summary 下载资源文件
// @Description 下载资源文件，并增加下载量
// @Tags 社区
// @Param id path string true "资源ID"
// @Success 200 {string} string "文件流"
// @Router /api/community/resources/{id}/download [get]
func (c *CommunityController) DownloadResource(ctx *gin.Context) {
	id := ctx.Param("id")
	fileURL, err := c.CommunityService.DownloadResource(id)
	if err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	if strings.HasPrefix(fileURL, "http") {
		ctx.Redirect(http.StatusFound, fileURL)
		return
	}

	// 如果是本地路径或MinIO路径
	if strings.HasPrefix(fileURL, "/api/community/resources/files/") {
		filename := strings.TrimPrefix(fileURL, "/api/community/resources/files/")
		filepath := "resource_file/" + filename
		if _, err := os.Stat(filepath); err == nil {
			ctx.File(filepath)
			return
		}
	}

	if strings.HasPrefix(fileURL, "/uploads/") {
		filename := strings.TrimPrefix(fileURL, "/uploads/")
		filepath := filepath.Join(c.CommunityService.Cfg.Storage.LocalPath, filename)
		if _, err := os.Stat(filepath); err == nil {
			ctx.File(filepath)
			return
		}
	}

	// 其他情况（如 MinIO），如果配置了静态服务则可以访问，或者这里可以实现MinIO下载逻辑
	// 暂时重定向
	ctx.Redirect(http.StatusFound, fileURL)
}

// @Summary 删除资源
// @Description 删除分享的资源，仅限老师或管理员
// @Tags 社区
// @Security BearerAuth
// @Param id path string true "资源ID"
// @Success 200 {object} util.Response
// @Router /api/community/resources/{id} [delete]
func (c *CommunityController) DeleteResource(ctx *gin.Context) {
	user := util.GetUserFromContext(ctx)
	if user == nil {
		util.Unauthorized(ctx)
		return
	}

	id := ctx.Param("id")
	err := c.CommunityService.DeleteResource(id, user.UserID, user.Role)
	if err != nil {
		if errors.Is(err, util.ErrPermissionDenied) {
			util.Forbidden(ctx)
		} else {
			util.LogInternalError(ctx, err)
		}
		return
	}

	util.Success(ctx, gin.H{"message": "Resource deleted successfully"})
}
