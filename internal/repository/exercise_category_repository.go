package repository

import (
	"coder_edu_backend/internal/model"
)

// Delete 删除练习题分类
func (r *ExerciseCategoryRepository) Delete(id uint) error {
	return r.DB.Delete(&model.ExerciseCategory{}, id).Error
}
