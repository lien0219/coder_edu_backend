package repository

import (
	"coder_edu_backend/internal/model"
	"time"

	"gorm.io/gorm"
)

type PostRepository struct {
	DB *gorm.DB
}

func NewPostRepository(db *gorm.DB) *PostRepository {
	return &PostRepository{DB: db}
}

func (r *PostRepository) FindWithPagination(offset, limit int, tag, search, tab string, userID uint) ([]model.Post, int, error) {
	var posts []model.Post
	var total int64

	query := r.DB.Model(&model.Post{})

	if tag != "" {
		query = query.Where("tags LIKE ?", "%"+tag+"%")
	}

	if search != "" {
		query = query.Where("title LIKE ? OR content LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	if tab == "my" && userID > 0 {
		query = query.Where("author_id = ?", userID)
	}

	// 计算总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 排序
	switch tab {
	case "popular":
		query = query.Order("(upvotes * 5 + views) DESC, created_at DESC")
	case "new":
		query = query.Order("created_at DESC")
	default:
		query = query.Order("created_at DESC")
	}

	// 分页查询
	err := query.Offset(offset).Limit(limit).
		Preload("Author").
		Preload("Comments").
		Find(&posts).Error
	if err != nil {
		return nil, 0, err
	}

	return posts, int(total), nil
}

func (r *PostRepository) Create(post *model.Post) error {
	return r.DB.Create(post).Error
}

func (r *PostRepository) Update(post *model.Post) error {
	return r.DB.Save(post).Error
}

func (r *PostRepository) Delete(id string) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		// 1. 删除帖子下的所有评论
		if err := tx.Where("post_id = ?", id).Delete(&model.Comment{}).Error; err != nil {
			return err
		}
		// 2. 删除帖子下的所有点赞 (物理删除，CommunityLike不支持软删除)
		if err := tx.Where("content_type = 'post' AND content_id = ?", id).Delete(&model.CommunityLike{}).Error; err != nil {
			return err
		}
		// 3. 删除帖子本身
		return tx.Delete(&model.Post{}, "id = ?", id).Error
	})
}

// 获取帖子详情（不再预加载全部评论）
func (r *PostRepository) FindByID(id string) (*model.Post, error) {
	var post model.Post
	err := r.DB.Preload("Author").
		First(&post, "id = ?", id).Error
	return &post, err
}

// 分页获取一级评论及其所有回复
func (r *PostRepository) FindCommentsWithPagination(postID string, offset, limit int) ([]model.Comment, int64, error) {
	var comments []model.Comment
	var total int64

	// 只统计一级评论的总数
	r.DB.Model(&model.Comment{}).Where("post_id = ? AND parent_id IS NULL", postID).Count(&total)

	// 先查出一级评论
	err := r.DB.Where("post_id = ? AND parent_id IS NULL", postID).
		Order("created_at DESC").
		Offset(offset).Limit(limit).
		Preload("Author").
		Find(&comments).Error

	if err != nil {
		return nil, 0, err
	}

	if len(comments) == 0 {
		return comments, total, nil
	}

	// 查出这些一级评论下的所有二级回复
	var parentIDs []string
	for _, c := range comments {
		parentIDs = append(parentIDs, c.ID)
	}

	var replies []model.Comment
	err = r.DB.Where("parent_id IN ?", parentIDs).
		Order("created_at ASC").
		Preload("Author").
		Preload("ReplyToUser").
		Find(&replies).Error

	if err != nil {
		return comments, total, nil
	}

	// 合并并返回。Service 层会负责将这些平铺的记录组装成树结构
	allComments := append(comments, replies...)

	return allComments, total, nil
}

func (r *PostRepository) HasLiked(userID uint, contentType string, contentID string) bool {
	if userID == 0 {
		return false
	}
	var count int64
	// 即使没有 DeletedAt，也可以正常工作
	r.DB.Model(&model.CommunityLike{}).
		Where("user_id = ? AND content_type = ? AND content_id = ?", userID, contentType, contentID).
		Count(&count)
	return count > 0
}

func (r *PostRepository) IncrementUpvotes(postID string) error {
	return r.DB.Model(&model.Post{}).
		Where("id = ?", postID).
		Update("upvotes", gorm.Expr("upvotes + 1")).
		Error
}

func (r *PostRepository) ToggleLike(userID uint, contentType string, contentID string) (bool, error) {
	var like model.CommunityLike
	// 使用 Unscoped() 确保即使在处理旧数据迁移或异常时也能正确处理记录
	result := r.DB.Where("user_id = ? AND content_type = ? AND content_id = ?", userID, contentType, contentID).First(&like)

	if result.Error == gorm.ErrRecordNotFound {
		// 点赞
		err := r.DB.Transaction(func(tx *gorm.DB) error {
			if err := tx.Create(&model.CommunityLike{UserID: userID, ContentType: contentType, ContentID: contentID}).Error; err != nil {
				return err
			}
			return tx.Model(r.getModel(contentType)).Where("id = ?", contentID).Update("upvotes", gorm.Expr("upvotes + 1")).Error
		})
		return true, err
	} else if result.Error != nil {
		return false, result.Error
	} else {
		// 取消点赞
		err := r.DB.Transaction(func(tx *gorm.DB) error {
			// 由于 model.CommunityLike 不再包含 DeletedAt，这里将执行物理删除
			if err := tx.Delete(&like).Error; err != nil {
				return err
			}
			return tx.Model(r.getModel(contentType)).Where("id = ?", contentID).Update("upvotes", gorm.Expr("upvotes - 1")).Error
		})
		return false, err
	}
}

func (r *PostRepository) getModel(contentType string) interface{} {
	switch contentType {
	case "post":
		return &model.Post{}
	case "comment":
		return &model.Comment{}
	case "answer":
		return &model.Answer{}
	case "resource":
		return &model.CommunityResource{}
	}
	return nil
}

type CommunityResourceRepository struct {
	DB *gorm.DB
}

func NewCommunityResourceRepository(db *gorm.DB) *CommunityResourceRepository {
	return &CommunityResourceRepository{DB: db}
}

func (r *CommunityResourceRepository) Create(resource *model.CommunityResource) error {
	return r.DB.Create(resource).Error
}

func (r *CommunityResourceRepository) FindWithPagination(offset, limit int, resourceType string, search string, sort string) ([]model.CommunityResource, int, error) {
	var resources []model.CommunityResource
	var total int64

	query := r.DB.Model(&model.CommunityResource{})

	if resourceType != "" {
		query = query.Where("type = ?", resourceType)
	}

	if search != "" {
		query = query.Where("title LIKE ? OR description LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 排序逻辑
	order := "created_at DESC"
	if sort == "views" {
		order = "view_count DESC"
	}

	err := query.Offset(offset).Limit(limit).
		Preload("Author").
		Order(order).
		Find(&resources).Error
	if err != nil {
		return nil, 0, err
	}

	return resources, int(total), nil
}

func (r *CommunityResourceRepository) FindByID(id string) (*model.CommunityResource, error) {
	var resource model.CommunityResource
	err := r.DB.Preload("Author").First(&resource, "id = ?", id).Error
	return &resource, err
}

func (r *CommunityResourceRepository) Delete(id string) error {
	return r.DB.Delete(&model.CommunityResource{}, "id = ?", id).Error
}

func (r *CommunityResourceRepository) IncrementDownload(id string) error {
	return r.DB.Model(&model.CommunityResource{}).Where("id = ?", id).
		Update("download_count", gorm.Expr("download_count + 1")).Error
}

func (r *CommunityResourceRepository) IncrementView(id string) error {
	return r.DB.Model(&model.CommunityResource{}).Where("id = ?", id).
		Update("view_count", gorm.Expr("view_count + 1")).Error
}

func (r *CommunityResourceRepository) GetTodayCount(userID uint) (int64, error) {
	var count int64
	today := time.Now().Format("2006-01-02")
	err := r.DB.Model(&model.CommunityResource{}).
		Where("author_id = ? AND created_at >= ?", userID, today).
		Count(&count).Error
	return count, err
}

type CommentRepository struct {
	DB *gorm.DB
}

func NewCommentRepository(db *gorm.DB) *CommentRepository {
	return &CommentRepository{DB: db}
}

func (r *CommentRepository) Create(comment *model.Comment) error {
	return r.DB.Create(comment).Error
}

func (r *CommentRepository) FindByID(id string) (*model.Comment, error) {
	var comment model.Comment
	err := r.DB.First(&comment, "id = ?", id).Error
	return &comment, err
}

func (r *CommentRepository) Delete(id string) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		// 1. 如果是删除一级评论，先删除该评论下的所有回复 (软删除)
		if err := tx.Where("parent_id = ?", id).Delete(&model.Comment{}).Error; err != nil {
			return err
		}
		// 2. 删除评论本身 (软删除)
		return tx.Delete(&model.Comment{}, "id = ?", id).Error
	})
}

func (r *CommentRepository) IncrementUpvotes(commentID string) error {
	return r.DB.Model(&model.Comment{}).
		Where("id = ?", commentID).
		Update("upvotes", gorm.Expr("upvotes + 1")).
		Error
}

type QuestionRepository struct {
	DB *gorm.DB
}

func NewQuestionRepository(db *gorm.DB) *QuestionRepository {
	return &QuestionRepository{DB: db}
}

func (r *QuestionRepository) FindWithPagination(offset, limit int, tag string, solved *bool) ([]model.Question, int, error) {
	var questions []model.Question
	var total int64

	query := r.DB.Model(&model.Question{})

	if tag != "" {
		query = query.Where("tags LIKE ?", "%"+tag+"%")
	}

	if solved != nil {
		query = query.Where("is_solved = ?", *solved)
	}

	// 计算总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 查询问题列表
	err := query.Offset(offset).Limit(limit).
		Preload("Author").
		Preload("Answers").
		Order("created_at desc").
		Find(&questions).Error
	if err != nil {
		return nil, 0, err
	}

	return questions, int(total), nil
}

func (r *QuestionRepository) Create(question *model.Question) error {
	return r.DB.Create(question).Error
}

type AnswerRepository struct {
	DB *gorm.DB
}

func NewAnswerRepository(db *gorm.DB) *AnswerRepository {
	return &AnswerRepository{DB: db}
}

func (r *AnswerRepository) Create(answer *model.Answer) error {
	return r.DB.Create(answer).Error
}

func (r *AnswerRepository) IncrementUpvotes(answerID string) error {
	return r.DB.Model(&model.Answer{}).
		Where("id = ?", answerID).
		Update("upvotes", gorm.Expr("upvotes + 1")).
		Error
}
