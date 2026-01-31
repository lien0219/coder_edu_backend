package repository

import (
	"coder_edu_backend/internal/model"

	"gorm.io/gorm"
)

type ReflectionRepository struct {
	DB *gorm.DB
}

func NewReflectionRepository(db *gorm.DB) *ReflectionRepository {
	return &ReflectionRepository{DB: db}
}

func (r *ReflectionRepository) Save(reflection *model.Reflection) error {
	return r.DB.Save(reflection).Error
}

func (r *ReflectionRepository) FindByUserID(userID uint) (*model.Reflection, error) {
	var reflection model.Reflection
	err := r.DB.Where("user_id = ?", userID).First(&reflection).Error
	if err != nil {
		return nil, err
	}
	return &reflection, nil
}

func (r *ReflectionRepository) ListAll(name string, page, pageSize int) ([]model.Reflection, int64, error) {
	var users []model.User
	var total int64

	// 1. 先统计正常的学生总数
	query := r.DB.Model(&model.User{}).Where("role = ?", "student").Where("disabled = ?", false)
	if name != "" {
		query = query.Where("(name LIKE ? OR email LIKE ?)", "%"+name+"%", "%"+name+"%")
	}
	query.Count(&total)

	// 2. 分页查询学生
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Find(&users).Error; err != nil {
		return nil, 0, err
	}

	if len(users) == 0 {
		return []model.Reflection{}, total, nil
	}

	// 3. 获取这些学生的 ID 列表
	userIDs := make([]uint, len(users))
	for i, u := range users {
		userIDs[i] = u.ID
	}

	// 4. 查询已存在的反思记录
	var existingReflections []model.Reflection
	if err := r.DB.Where("user_id IN ?", userIDs).Find(&existingReflections).Error; err != nil {
		return nil, total, err
	}

	// 5. 将反思记录按 UserID 分组
	refMap := make(map[uint]model.Reflection)
	for _, ref := range existingReflections {
		refMap[ref.UserID] = ref
	}

	// 6. 合并结果
	result := make([]model.Reflection, len(users))
	for i, u := range users {
		if ref, ok := refMap[u.ID]; ok {
			ref.User = &users[i]
			result[i] = ref
		} else {
			result[i] = model.Reflection{
				UserID: u.ID,
				User:   &users[i],
			}
		}
	}

	return result, total, nil
}

func (r *ReflectionRepository) FindByID(id string) (*model.Reflection, error) {
	var reflection model.Reflection
	err := r.DB.Preload("User").First(&reflection, "id = ?", id).Error
	return &reflection, err
}
