package repository

import (
	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/util"
	"strings"

	"gorm.io/gorm"
)

func uniqueKnowledgePointIDs(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func assertKnowledgePointsExist(db *gorm.DB, ids []string) error {
	ids = uniqueKnowledgePointIDs(ids)
	if len(ids) == 0 {
		return nil
	}
	var count int64
	if err := db.Model(&model.KnowledgePoint{}).Where("id IN ?", ids).Count(&count).Error; err != nil {
		return err
	}
	if int(count) != len(ids) {
		return util.ErrInvalidKnowledgePoint
	}
	return nil
}

func (r *AssessmentRepository) AssertKnowledgePointsExist(ids []string) error {
	return assertKnowledgePointsExist(r.DB, ids)
}

func (r *AssessmentRepository) ReplaceQuestionKnowledgePoints(questionID uint, ids []string) error {
	ids = uniqueKnowledgePointIDs(ids)
	if err := assertKnowledgePointsExist(r.DB, ids); err != nil {
		return err
	}
	if err := r.DB.Where("assessment_question_id = ?", questionID).
		Delete(&model.AssessmentQuestionKnowledgePoint{}).Error; err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	rows := make([]model.AssessmentQuestionKnowledgePoint, len(ids))
	for i, id := range ids {
		rows[i] = model.AssessmentQuestionKnowledgePoint{
			AssessmentQuestionID: questionID,
			KnowledgePointID:     id,
		}
	}
	return r.DB.Create(&rows).Error
}

func (r *AssessmentRepository) ListKnowledgePointIDsByQuestionIDs(questionIDs []uint) (map[uint][]string, error) {
	out := make(map[uint][]string, len(questionIDs))
	if len(questionIDs) == 0 {
		return out, nil
	}
	var rows []model.AssessmentQuestionKnowledgePoint
	if err := r.DB.Where("assessment_question_id IN ?", questionIDs).
		Order("assessment_question_id ASC, knowledge_point_id ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.AssessmentQuestionID] = append(out[row.AssessmentQuestionID], row.KnowledgePointID)
	}
	return out, nil
}

func (r *LearningPathRepository) WithTx(fn func(*LearningPathRepository) error) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		return fn(&LearningPathRepository{DB: tx})
	})
}

func (r *LearningPathRepository) AssertKnowledgePointsExist(ids []string) error {
	return assertKnowledgePointsExist(r.DB, ids)
}

func (r *LearningPathRepository) ReplaceMaterialKnowledgePoints(materialID string, ids []string) error {
	ids = uniqueKnowledgePointIDs(ids)
	if err := assertKnowledgePointsExist(r.DB, ids); err != nil {
		return err
	}
	if err := r.DB.Where("material_id = ?", materialID).
		Delete(&model.LearningPathMaterialKnowledgePoint{}).Error; err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	rows := make([]model.LearningPathMaterialKnowledgePoint, len(ids))
	for i, id := range ids {
		rows[i] = model.LearningPathMaterialKnowledgePoint{
			MaterialID:       materialID,
			KnowledgePointID: id,
		}
	}
	return r.DB.Create(&rows).Error
}

func (r *LearningPathRepository) ListKnowledgePointIDsByMaterialIDs(materialIDs []string) (map[string][]string, error) {
	out := make(map[string][]string, len(materialIDs))
	if len(materialIDs) == 0 {
		return out, nil
	}
	var rows []model.LearningPathMaterialKnowledgePoint
	if err := r.DB.Where("material_id IN ?", materialIDs).
		Order("material_id ASC, knowledge_point_id ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.MaterialID] = append(out[row.MaterialID], row.KnowledgePointID)
	}
	return out, nil
}

func (r *LearningPathRepository) ListMaterialKnowledgePointsByKnowledgePointIDs(kpIDs []string) ([]model.LearningPathMaterialKnowledgePoint, error) {
	if len(kpIDs) == 0 {
		return nil, nil
	}
	var rows []model.LearningPathMaterialKnowledgePoint
	err := r.DB.Where("knowledge_point_id IN ?", kpIDs).
		Order("material_id ASC, knowledge_point_id ASC").
		Find(&rows).Error
	return rows, err
}

func (r *LearningPathRepository) FindMaterialsByIDs(ids []string) ([]model.LearningPathMaterial, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var ms []model.LearningPathMaterial
	err := r.DB.Where("id IN ?", ids).Find(&ms).Error
	return ms, err
}

func (r *AssessmentRepository) FindKnowledgePointTitles(ids []string) (map[string]string, error) {
	out := make(map[string]string, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	var kps []model.KnowledgePoint
	if err := r.DB.Select("id", "title").Where("id IN ?", ids).Find(&kps).Error; err != nil {
		return nil, err
	}
	for _, kp := range kps {
		out[kp.ID] = kp.Title
	}
	return out, nil
}

func (r *AssessmentRepository) FindKnowledgePointMasteryMeta(ids []string) (map[string]model.KnowledgePoint, error) {
	out := make(map[string]model.KnowledgePoint, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	var kps []model.KnowledgePoint
	if err := r.DB.Select("id", "title", "`order`", "completion_score").Where("id IN ?", ids).Find(&kps).Error; err != nil {
		return nil, err
	}
	for _, kp := range kps {
		out[kp.ID] = kp
	}
	return out, nil
}
