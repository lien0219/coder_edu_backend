package service

import "coder_edu_backend/internal/model"

// 客观题推荐等级阈值目前只定义在本文件。
// 后续若改为配置表，应只改 RecommendObjectiveLevel，不要在调用方或前端重复分档。
const (
	objectiveLevelPctBasic        = 60.0 // [0, 60) → 1 基础
	objectiveLevelPctElementary   = 75.0 // [60, 75) → 2 初级/巩固
	objectiveLevelPctIntermediate = 90.0 // [75, 90) → 3 中级/进阶
	// [90, 100] → 4 高级/拓展
)

// RecommendObjectiveLevel 按客观得分百分比返回推荐等级 1–4。
// 使用实际百分比（float64(earned)/float64(max)*100），不按整数分直接判断。
// 例：40/60 ≈ 66.67% → 2；60%、75%、90% 分别进入 2、3、4。
// max<=0 时不得自动推荐。earned 限制在 [0, max]。
func RecommendObjectiveLevel(earned, max int) (level int, ok bool) {
	if max <= 0 {
		return 0, false
	}
	if earned < 0 {
		earned = 0
	}
	if earned > max {
		earned = max
	}
	percent := float64(earned) / float64(max) * 100
	switch {
	case percent < objectiveLevelPctBasic:
		return 1, true
	case percent < objectiveLevelPctElementary:
		return 2, true
	case percent < objectiveLevelPctIntermediate:
		return 3, true
	default:
		return 4, true
	}
}

func isAutoGradableObjectiveType(questionType string) bool {
	switch questionType {
	case "single_choice", "multiple_choice", "true_false", "fill_blank", "fill_in":
		return true
	default:
		return false
	}
}

func isDeterminateAutoResult(result string) bool {
	switch result {
	case model.AutoResultCorrect, model.AutoResultWrong, model.AutoResultUnanswered:
		return true
	default:
		return false
	}
}

// CanAutoCompleteAssessment 仅当整卷都是现有评分器可确定自动评分的客观题、且 objectiveMax>0 时为 true。
// 未作答客观题计 0 分，仍视为可自动评分，不因此转为人工审核。
// essay/code、ambiguous、pending_manual 或空卷不得自动完成。
func CanAutoCompleteAssessment(questions []model.AssessmentQuestion, results []model.AssessmentItemResult, objectiveMax int) bool {
	if len(questions) == 0 || objectiveMax <= 0 || len(results) != len(questions) {
		return false
	}
	for i, q := range questions {
		if !isAutoGradableObjectiveType(q.QuestionType) {
			return false
		}
		if !isDeterminateAutoResult(results[i].AutoResult) {
			return false
		}
	}
	return true
}

func applyObjectiveAutoComplete(questions []model.AssessmentQuestion, results []model.AssessmentItemResult, autoScore, objectiveMax int) (status, scoringStatus string, level int) {
	status = model.SubmissionStatusPending
	scoringStatus = model.ScoringStatusAwaitingTeacher
	if !CanAutoCompleteAssessment(questions, results, objectiveMax) {
		return status, scoringStatus, 0
	}
	level, ok := RecommendObjectiveLevel(autoScore, objectiveMax)
	if !ok {
		return status, scoringStatus, 0
	}
	return model.SubmissionStatusCompleted, model.ScoringStatusSystemCompleted, level
}

func shouldPreserveObjectiveAutoScore(sub *model.AssessmentSubmission) bool {
	return sub != nil && sub.PendingManualCount == 0 && sub.ObjectiveMax > 0
}
