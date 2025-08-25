package service

import (
	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/repository"
	"fmt"
	"strings"
	"time"
)

type CommunityService struct {
	PostRepo     *repository.PostRepository
	CommentRepo  *repository.CommentRepository
	QuestionRepo *repository.QuestionRepository
	AnswerRepo   *repository.AnswerRepository
	UserRepo     *repository.UserRepository
}

func NewCommunityService(
	postRepo *repository.PostRepository,
	commentRepo *repository.CommentRepository,
	questionRepo *repository.QuestionRepository,
	answerRepo *repository.AnswerRepository,
	userRepo *repository.UserRepository,
) *CommunityService {
	return &CommunityService{
		PostRepo:     postRepo,
		CommentRepo:  commentRepo,
		QuestionRepo: questionRepo,
		AnswerRepo:   answerRepo,
		UserRepo:     userRepo,
	}
}

type PostRequest struct {
	Title   string   `json:"title" binding:"required"`
	Content string   `json:"content" binding:"required"`
	Tags    []string `json:"tags"`
}

type QuestionRequest struct {
	Title   string   `json:"title" binding:"required"`
	Content string   `json:"content" binding:"required"`
	Tags    []string `json:"tags"`
}

type AnswerRequest struct {
	Content string `json:"content" binding:"required"`
}

func (s *CommunityService) GetPosts(page, limit int, tag, sort string) ([]model.Post, int, error) {
	offset := (page - 1) * limit
	return s.PostRepo.FindWithPagination(offset, limit, tag, sort)
}

func (s *CommunityService) CreatePost(userID uint, req PostRequest) (*model.Post, error) {
	post := &model.Post{
		Title:     req.Title,
		Content:   req.Content,
		AuthorID:  userID,
		Tags:      strings.Join(req.Tags, ","),
		CreatedAt: time.Now(),
	}

	err := s.PostRepo.Create(post)
	if err != nil {
		return nil, err
	}

	return post, nil
}

func (s *CommunityService) GetQuestions(page, limit int, tag string, solved *bool) ([]model.Question, int, error) {
	offset := (page - 1) * limit
	return s.QuestionRepo.FindWithPagination(offset, limit, tag, solved)
}

func (s *CommunityService) CreateQuestion(userID uint, req QuestionRequest) (*model.Question, error) {
	question := &model.Question{
		Title:     req.Title,
		Content:   req.Content,
		AuthorID:  userID,
		Tags:      strings.Join(req.Tags, ","),
		CreatedAt: time.Now(),
	}

	err := s.QuestionRepo.Create(question)
	if err != nil {
		return nil, err
	}

	return question, nil
}

func (s *CommunityService) AnswerQuestion(userID, questionID uint, req AnswerRequest) (*model.Answer, error) {
	answer := &model.Answer{
		QuestionID: questionID,
		AuthorID:   userID,
		Content:    req.Content,
		CreatedAt:  time.Now(),
	}

	err := s.AnswerRepo.Create(answer)
	if err != nil {
		return nil, err
	}

	return answer, nil
}

func (s *CommunityService) Upvote(userID uint, contentType string, contentID uint) error {
	switch contentType {
	case "post":
		return s.PostRepo.IncrementUpvotes(contentID)
	case "comment":
		return s.CommentRepo.IncrementUpvotes(contentID)
	case "answer":
		return s.AnswerRepo.IncrementUpvotes(contentID)
	default:
		return fmt.Errorf("unknown content type: %s", contentType)
	}
}
