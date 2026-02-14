package service

import (
	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/repository"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

type CommunityService struct {
	PostRepo     *repository.PostRepository
	CommentRepo  *repository.CommentRepository
	QuestionRepo *repository.QuestionRepository
	AnswerRepo   *repository.AnswerRepository
	UserRepo     *repository.UserRepository
	ResourceRepo *repository.CommunityResourceRepository
	Redis        *redis.Client
}

func NewCommunityService(
	postRepo *repository.PostRepository,
	commentRepo *repository.CommentRepository,
	questionRepo *repository.QuestionRepository,
	answerRepo *repository.AnswerRepository,
	userRepo *repository.UserRepository,
	resourceRepo *repository.CommunityResourceRepository,
	rdb *redis.Client,
) *CommunityService {
	return &CommunityService{
		PostRepo:     postRepo,
		CommentRepo:  commentRepo,
		QuestionRepo: questionRepo,
		AnswerRepo:   answerRepo,
		UserRepo:     userRepo,
		ResourceRepo: resourceRepo,
		Redis:        rdb,
	}
}

type PostRequest struct {
	Title   string   `json:"title" binding:"required"`
	Content string   `json:"content" binding:"required"`
	Tags    []string `json:"tags"`
}

type PostResponse struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Content      string    `json:"content"`
	Author       string    `json:"author"`
	Avatar       string    `json:"avatar"`
	Tags         []string  `json:"tags"`
	CreatedAt    time.Time `json:"createdAt"`
	Likes        int       `json:"likes"`
	Views        int       `json:"views"`
	CommentCount int       `json:"commentCount"`
}

type QuestionRequest struct {
	Title   string   `json:"title" binding:"required"`
	Content string   `json:"content" binding:"required"`
	Tags    []string `json:"tags"`
}

type AnswerRequest struct {
	Content string `json:"content" binding:"required"`
}

type ResourceShareRequest struct {
	Title       string                      `json:"title" binding:"required"`
	Description string                      `json:"description"`
	Type        model.CommunityResourceType `json:"type" binding:"required"`
	Content     string                      `json:"content"` // 用于手写文章
	FileURL     string                      `json:"fileUrl"` // 用于文件上传
}

type ResourceResponse struct {
	ID            string                      `json:"id"`
	Title         string                      `json:"title"`
	Description   string                      `json:"description"`
	Author        string                      `json:"author"`
	AuthorID      uint                        `json:"authorId"`
	Type          model.CommunityResourceType `json:"type"`
	FileURL       string                      `json:"fileUrl"`
	Content       string                      `json:"content,omitempty"`
	DownloadCount int                         `json:"downloadCount"`
	ViewCount     int                         `json:"viewCount"`
	Likes         int                         `json:"likes"`
	CreatedAt     time.Time                   `json:"createdAt"`
	IsLiked       bool                        `json:"isLiked"`
}

type CommentCreateRequest struct {
	Content  string  `json:"content" binding:"required,max=1000"`
	ParentID *string `json:"parentId"` // 一级评论的 ID
	ToUserID *uint   `json:"toUserId"` // 被回复者的用户 ID
}

type ReplyResponse struct {
	ID        string    `json:"id"`
	Author    string    `json:"author"`
	AuthorID  uint      `json:"authorId"`
	Avatar    string    `json:"avatar"`
	Content   string    `json:"content"`
	ToUser    string    `json:"toUser,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	Likes     int       `json:"likes"`
	IsLiked   bool      `json:"isLiked"`
}

type CommentResponse struct {
	ID        string          `json:"id"`
	Author    string          `json:"author"`
	AuthorID  uint            `json:"authorId"`
	Avatar    string          `json:"avatar"`
	Content   string          `json:"content"`
	ToUser    string          `json:"toUser,omitempty"`
	CreatedAt time.Time       `json:"createdAt"`
	Likes     int             `json:"likes"`
	Replies   []ReplyResponse `json:"replies"`
	IsLiked   bool            `json:"isLiked"`
}

type DiscussionDetailResponse struct {
	PostResponse
	IsLiked  bool              `json:"isLiked"`
	Comments []CommentResponse `json:"comments"`
}

func (s *CommunityService) GetPosts(page, limit int, tag, search, tab string, userID uint) ([]PostResponse, int, error) {
	offset := (page - 1) * limit
	posts, total, err := s.PostRepo.FindWithPagination(offset, limit, tag, search, tab, userID)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]PostResponse, len(posts))
	for i, post := range posts {
		tags := []string{}
		if post.Tags != "" {
			tags = strings.Split(post.Tags, ",")
		}

		responses[i] = PostResponse{
			ID:           post.ID,
			Title:        post.Title,
			Content:      post.Content,
			Author:       post.Author.Name,
			Avatar:       post.Author.Avatar,
			Tags:         tags,
			CreatedAt:    post.CreatedAt,
			Likes:        post.Upvotes,
			Views:        post.Views,
			CommentCount: len(post.Comments),
		}
	}

	return responses, total, nil
}

func (s *CommunityService) GetPostDetail(postID string, userID uint, ip string) (*DiscussionDetailResponse, error) {
	post, err := s.PostRepo.FindByID(postID)
	if err != nil {
		return nil, err
	}

	// 防刷机制 (Redis)
	var userKey string
	if userID > 0 {
		userKey = fmt.Sprintf("post_v:%s:u:%d", postID, userID)
	} else {
		userKey = fmt.Sprintf("post_v:%s:ip:%s", postID, ip)
	}

	ctx := context.Background()
	// 使用 SetNX (If Not Exists) 设置标识，有效期 10 分钟
	isNewVisit, _ := s.Redis.SetNX(ctx, userKey, "1", 10*time.Minute).Result()

	// 异步增加阅读量
	if isNewVisit {
		go func(pid string) {
			s.PostRepo.DB.Model(&model.Post{}).Where("id = ?", pid).Update("views", gorm.Expr("views + 1"))
		}(post.ID)
		post.Views += 1 // 本次返回的数据 +1
	}

	tags := []string{}
	if post.Tags != "" {
		tags = strings.Split(post.Tags, ",")
	}

	// 统计总评论数（包含回复）
	var commentCount int64
	s.PostRepo.DB.Model(&model.Comment{}).Where("post_id = ?", postID).Count(&commentCount)

	res := &DiscussionDetailResponse{
		PostResponse: PostResponse{
			ID:           post.ID,
			Title:        post.Title,
			Content:      post.Content,
			Author:       post.Author.Name,
			Avatar:       post.Author.Avatar,
			Tags:         tags,
			CreatedAt:    post.CreatedAt,
			Likes:        post.Upvotes,
			Views:        post.Views,
			CommentCount: int(commentCount),
		},
		IsLiked:  s.PostRepo.HasLiked(userID, "post", post.ID),
		Comments: []CommentResponse{}, // 详情页不再直接返回评论列表，由前端分页请求
	}

	return res, nil
}

func (s *CommunityService) GetPostComments(postID string, page, limit int, userID uint) ([]CommentResponse, int64, error) {
	offset := (page - 1) * limit
	allComments, total, err := s.PostRepo.FindCommentsWithPagination(postID, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	// 转换并组装树形结构
	commentMap := make(map[string]*CommentResponse)
	var rootComments []CommentResponse

	// 第一遍：识别一级评论
	for _, c := range allComments {
		if c.ParentID == nil {
			cr := &CommentResponse{
				ID:        c.ID,
				Author:    c.Author.Name,
				AuthorID:  c.AuthorID,
				Avatar:    c.Author.Avatar,
				Content:   c.Content,
				CreatedAt: c.CreatedAt,
				Likes:     c.Upvotes,
				Replies:   []ReplyResponse{},
				IsLiked:   s.PostRepo.HasLiked(userID, "comment", c.ID),
			}
			commentMap[c.ID] = cr
			rootComments = append(rootComments, *cr)
		}
	}

	// 第二遍：填充二级回复
	for _, c := range allComments {
		if c.ParentID != nil {
			if parent, ok := commentMap[*c.ParentID]; ok {
				toUser := ""
				if c.ReplyToUser != nil {
					toUser = c.ReplyToUser.Name
				}
				parent.Replies = append(parent.Replies, ReplyResponse{
					ID:        c.ID,
					Author:    c.Author.Name,
					AuthorID:  c.AuthorID,
					Avatar:    c.Author.Avatar,
					Content:   c.Content,
					ToUser:    toUser,
					CreatedAt: c.CreatedAt,
					Likes:     c.Upvotes,
					IsLiked:   s.PostRepo.HasLiked(userID, "comment", c.ID),
				})
			}
		}
	}

	// 同步 Map 里的更新到结果切片
	for i := range rootComments {
		if updated, ok := commentMap[rootComments[i].ID]; ok {
			rootComments[i] = *updated
		}
	}

	return rootComments, total, nil
}

func (s *CommunityService) CreatePost(userID uint, req PostRequest) (*PostResponse, error) {
	user, err := s.UserRepo.FindByID(userID)
	if err != nil {
		return nil, err
	}

	post := &model.Post{
		Title:    req.Title,
		Content:  req.Content,
		AuthorID: userID,
		Tags:     strings.Join(req.Tags, ","),
	}

	err = s.PostRepo.Create(post)
	if err != nil {
		return nil, err
	}

	return &PostResponse{
		ID:        post.ID,
		Title:     post.Title,
		Content:   post.Content,
		Author:    user.Name,
		Avatar:    user.Avatar,
		Tags:      req.Tags,
		CreatedAt: post.CreatedAt,
		Likes:     post.Upvotes,
		Views:     post.Views,
	}, nil
}

func (s *CommunityService) UpdatePost(userID uint, postID string, req PostRequest, userRole model.UserRole) (*PostResponse, error) {
	post, err := s.PostRepo.FindByID(postID)
	if err != nil {
		return nil, err
	}

	// 作者本人或管理员可以修改
	if post.AuthorID != userID && userRole != model.Admin {
		return nil, fmt.Errorf("permission denied")
	}

	post.Title = req.Title
	post.Content = req.Content
	post.Tags = strings.Join(req.Tags, ",")

	if err := s.PostRepo.Update(post); err != nil {
		return nil, err
	}

	user, _ := s.UserRepo.FindByID(post.AuthorID)

	return &PostResponse{
		ID:        post.ID,
		Title:     post.Title,
		Content:   post.Content,
		Author:    user.Name,
		Avatar:    user.Avatar,
		Tags:      req.Tags,
		CreatedAt: post.CreatedAt,
		Likes:     post.Upvotes,
		Views:     post.Views,
	}, nil
}

func (s *CommunityService) DeletePost(userID uint, postID string, userRole model.UserRole) error {
	post, err := s.PostRepo.FindByID(postID)
	if err != nil {
		return err
	}

	// 作者本人或管理员可以删除
	if post.AuthorID != userID && userRole != model.Admin {
		return fmt.Errorf("permission denied")
	}

	return s.PostRepo.Delete(postID)
}

func (s *CommunityService) CreateComment(userID uint, postID string, req CommentCreateRequest) (*CommentResponse, error) {
	user, err := s.UserRepo.FindByID(userID)
	if err != nil {
		return nil, err
	}

	comment := &model.Comment{
		PostID:     postID,
		AuthorID:   userID,
		Content:    req.Content,
		ParentID:   req.ParentID,
		ReplyToUID: req.ToUserID,
	}

	if err := s.CommentRepo.Create(comment); err != nil {
		return nil, err
	}

	toUser := ""
	if req.ToUserID != nil {
		target, err := s.UserRepo.FindByID(*req.ToUserID)
		if err == nil {
			toUser = target.Name
		}
	}

	return &CommentResponse{
		ID:        comment.ID,
		Author:    user.Name,
		AuthorID:  userID,
		Avatar:    user.Avatar,
		Content:   comment.Content,
		CreatedAt: comment.CreatedAt,
		Likes:     0,
		IsLiked:   false,
		ToUser:    toUser,
		Replies:   []ReplyResponse{},
	}, nil
}

func (s *CommunityService) DeleteComment(userID uint, commentID string, userRole model.UserRole) error {
	comment, err := s.CommentRepo.FindByID(commentID)
	if err != nil {
		return err
	}

	// 权限检查：只有作者本人或管理员可以删除
	if comment.AuthorID != userID && userRole != model.Admin {
		return fmt.Errorf("permission denied")
	}

	return s.CommentRepo.Delete(commentID)
}

func (s *CommunityService) GetQuestions(page, limit int, tag string, solved *bool) ([]model.Question, int, error) {
	offset := (page - 1) * limit
	return s.QuestionRepo.FindWithPagination(offset, limit, tag, solved)
}

func (s *CommunityService) CreateQuestion(userID uint, req QuestionRequest) (*model.Question, error) {
	question := &model.Question{
		Title:    req.Title,
		Content:  req.Content,
		AuthorID: userID,
		Tags:     strings.Join(req.Tags, ","),
	}

	err := s.QuestionRepo.Create(question)
	if err != nil {
		return nil, err
	}

	return question, nil
}

func (s *CommunityService) AnswerQuestion(userID uint, questionID string, req AnswerRequest) (*model.Answer, error) {
	answer := &model.Answer{
		QuestionID: questionID,
		AuthorID:   userID,
		Content:    req.Content,
	}

	err := s.AnswerRepo.Create(answer)
	if err != nil {
		return nil, err
	}

	return answer, nil
}

func (s *CommunityService) Upvote(userID uint, contentType string, contentID string) (bool, error) {
	return s.PostRepo.ToggleLike(userID, contentType, contentID)
}

func (s *CommunityService) GetResources(page, limit int, resourceType string, search string, userID uint, sort string) ([]ResourceResponse, int, error) {
	offset := (page - 1) * limit
	resources, total, err := s.ResourceRepo.FindWithPagination(offset, limit, resourceType, search, sort)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]ResourceResponse, len(resources))
	for i, r := range resources {
		responses[i] = ResourceResponse{
			ID:            r.ID,
			Title:         r.Title,
			Description:   r.Description,
			Author:        r.Author.Name,
			AuthorID:      r.AuthorID,
			Type:          r.Type,
			FileURL:       r.FileURL,
			DownloadCount: r.DownloadCount,
			ViewCount:     r.ViewCount,
			Likes:         r.Upvotes,
			CreatedAt:     r.CreatedAt,
			IsLiked:       s.PostRepo.HasLiked(userID, "resource", r.ID),
		}
	}
	return responses, total, nil
}

func (s *CommunityService) CreateResource(userID uint, role model.UserRole, req ResourceShareRequest) (*ResourceResponse, error) {
	// 学生限额检查
	if role == model.Student {
		count, err := s.ResourceRepo.GetTodayCount(userID)
		if err != nil {
			return nil, err
		}
		if count >= 3 {
			return nil, fmt.Errorf("daily share limit reached (max 3)")
		}
	}

	resource := &model.CommunityResource{
		Title:       req.Title,
		Description: req.Description,
		AuthorID:    userID,
		Type:        req.Type,
		FileURL:     req.FileURL,
		Content:     req.Content,
	}

	if err := s.ResourceRepo.Create(resource); err != nil {
		return nil, err
	}

	user, _ := s.UserRepo.FindByID(userID)

	return &ResourceResponse{
		ID:            resource.ID,
		Title:         resource.Title,
		Description:   resource.Description,
		Author:        user.Name,
		AuthorID:      userID,
		Type:          resource.Type,
		FileURL:       resource.FileURL,
		DownloadCount: 0,
		ViewCount:     0,
		Likes:         0,
		CreatedAt:     resource.CreatedAt,
	}, nil
}

func (s *CommunityService) GetResourceDetail(id string, userID uint) (*ResourceResponse, error) {
	resource, err := s.ResourceRepo.FindByID(id)
	if err != nil {
		return nil, err
	}

	// 增加观看量 (Redis 去重，防止同一用户重复增加)
	viewKey := fmt.Sprintf("resource_view:%s:%d", id, userID)
	if userID == 0 {
		// 这里暂定未登录用户每次进入都增加
		s.ResourceRepo.IncrementView(id)
		resource.ViewCount++
	} else {
		// 已登录用户，10分钟内不重复计算观看量
		success, _ := s.Redis.SetNX(context.Background(), viewKey, "1", 10*time.Minute).Result()
		if success {
			s.ResourceRepo.IncrementView(id)
			resource.ViewCount++
		}
	}

	return &ResourceResponse{
		ID:            resource.ID,
		Title:         resource.Title,
		Description:   resource.Description,
		Author:        resource.Author.Name,
		AuthorID:      resource.AuthorID,
		Type:          resource.Type,
		FileURL:       resource.FileURL,
		Content:       resource.Content,
		DownloadCount: resource.DownloadCount,
		ViewCount:     resource.ViewCount,
		Likes:         resource.Upvotes,
		CreatedAt:     resource.CreatedAt,
		IsLiked:       s.PostRepo.HasLiked(userID, "resource", resource.ID),
	}, nil
}

func (s *CommunityService) DownloadResource(id string) (string, error) {
	resource, err := s.ResourceRepo.FindByID(id)
	if err != nil {
		return "", err
	}

	if resource.Type == model.ResourceArticle {
		return "", fmt.Errorf("articles cannot be downloaded")
	}

	s.ResourceRepo.IncrementDownload(id)
	return resource.FileURL, nil
}

func (s *CommunityService) DeleteResource(id string, userID uint, role model.UserRole) error {
	_, err := s.ResourceRepo.FindByID(id)
	if err != nil {
		return err
	}

	// 只允许老师或者管理员可以删除
	if role != model.Teacher && role != model.Admin {
		return fmt.Errorf("permission denied")
	}

	// 如果有物理文件，可以选择是否删除物理文件。这里暂时只删除数据库记录
	return s.ResourceRepo.Delete(id)
}
