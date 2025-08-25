package repository

import (
	"coder_edu_backend/internal/model"

	"gorm.io/gorm"
)

type PostRepository struct {
	DB *gorm.DB
}

func NewPostRepository(db *gorm.DB) *PostRepository {
	return &PostRepository{DB: db}
}

func (r *PostRepository) FindWithPagination(offset, limit int, tag, sort string) ([]model.Post, int, error) {
	var posts []model.Post
	var total int64

	query := r.DB.Model(&model.Post{})

	if tag != "" {
		query = query.Where("tags LIKE ?", "%"+tag+"%")
	}

	// 计算总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 排序
	switch sort {
	case "popular":
		query = query.Order("upvotes desc, created_at desc")
	default:
		query = query.Order("created_at desc")
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

func (r *PostRepository) IncrementUpvotes(postID uint) error {
	return r.DB.Model(&model.Post{}).
		Where("id = ?", postID).
		Update("upvotes", gorm.Expr("upvotes + 1")).
		Error
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

func (r *CommentRepository) IncrementUpvotes(commentID uint) error {
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

func (r *AnswerRepository) IncrementUpvotes(answerID uint) error {
	return r.DB.Model(&model.Answer{}).
		Where("id = ?", answerID).
		Update("upvotes", gorm.Expr("upvotes + 1")).
		Error
}
