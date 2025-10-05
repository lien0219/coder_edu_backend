package repository

import (
	"coder_edu_backend/internal/model"
)

// Delete 删除练习题题目
func (r *ExerciseQuestionRepository) Delete(id uint) error {
	return r.DB.Delete(&model.ExerciseQuestion{}, id).Error
}

// UpdateQuestion 更新练习题题目信息
func (r *ExerciseQuestionRepository) UpdateQuestion(question *model.ExerciseQuestion) error {
	return r.DB.Model(question).Updates(question).Error
}

// FindAllByCategoryID 根据分类ID查找所有练习题题目
func (r *ExerciseQuestionRepository) FindAllByCategoryID(categoryID uint) ([]model.ExerciseQuestion, error) {
	var questions []model.ExerciseQuestion
	err := r.DB.Where("category_id = ?", categoryID).Find(&questions).Error
	return questions, err
}

// FindQuestionsByCategoryIDWithPagination 根据分类ID查找练习题题目，支持分页
func (r *ExerciseQuestionRepository) FindQuestionsByCategoryIDWithPagination(categoryID uint, page, limit int) ([]model.ExerciseQuestion, int, error) {
	var questions []model.ExerciseQuestion
	var total int64

	err := r.DB.Model(&model.ExerciseQuestion{}).Where("category_id = ?", categoryID).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	err = r.DB.Where("category_id = ?", categoryID).Offset(offset).Limit(limit).Find(&questions).Error

	return questions, int(total), err
}

// 添加FindByID方法
func (r *ExerciseQuestionRepository) FindByID(id uint) (*model.ExerciseQuestion, error) {
	var question model.ExerciseQuestion
	err := r.DB.First(&question, id).Error
	if err != nil {
		return nil, err
	}
	return &question, nil
}
