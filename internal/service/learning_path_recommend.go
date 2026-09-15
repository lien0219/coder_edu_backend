package service

import (
	"encoding/json"
	"errors"
	"sort"

	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/util"
)

const (
	RecStatusOK                      = "ok"
	RecStatusNoConfirmedDiagnosis    = "no_confirmed_diagnosis"
	RecStatusNoItemResults           = "no_item_results"
	RecStatusNoWeakEvidence          = "no_weak_evidence"
	RecStatusNoMappedKnowledgePoints = "no_mapped_knowledge_points"
	RecStatusLegacyNoSnapshot        = "legacy_no_snapshot"
	RecStatusNoMatchingMaterials     = "no_matching_materials"
)

const (
	RecCauseWrong      = "wrong"
	RecCauseUnanswered = "unanswered"
)

type MaterialRecommendationReason struct {
	Cause               string `json:"cause"`
	QuestionID          uint   `json:"questionId"`
	QuestionIndex       int    `json:"questionIndex,omitempty"`
	QuestionSummary     string `json:"questionSummary,omitempty"`
	KnowledgePointID    string `json:"knowledgePointId"`
	KnowledgePointTitle string `json:"knowledgePointTitle"`
}

type RecommendedMaterial struct {
	ID            string                         `json:"id"`
	Title         string                         `json:"title"`
	Level         int                            `json:"level"`
	Points        int                            `json:"points"`
	ChapterNumber int                            `json:"chapterNumber"`
	IsCompleted   bool                           `json:"isCompleted"`
	Reasons       []MaterialRecommendationReason `json:"reasons"`
}

type MaterialRecommendationBlock struct {
	Status string                `json:"status"`
	Reason string                `json:"reason"`
	Items  []RecommendedMaterial `json:"items"`
}

func emptyRecommendation(status, reason string) MaterialRecommendationBlock {
	return MaterialRecommendationBlock{
		Status: status,
		Reason: reason,
		Items:  []RecommendedMaterial{},
	}
}

func (s *LearningPathService) loadConfirmedDiagnosis(userID uint) (*model.AssessmentSubmission, error) {
	paper, qs, err := s.AssessmentRepo.FindPublishedAssessmentWithQuestions()
	if err != nil {
		if errors.Is(err, util.ErrNoPublishedAssessment) {
			return nil, nil
		}
		return nil, err
	}
	version, err := ComputePaperVersion(paper.ID, qs)
	if err != nil {
		return nil, err
	}
	return s.AssessmentRepo.FindConfirmedDiagnosis(userID, paper.ID, version)
}

func recommendedLevelOf(sub *model.AssessmentSubmission) int {
	if sub == nil {
		return 0
	}
	return sub.RecommendedLevel
}

func (s *LearningPathService) BuildMaterialRecommendations(userID uint, confirmed *model.AssessmentSubmission, completedMap map[string]bool) (MaterialRecommendationBlock, error) {
	if confirmed == nil || confirmed.RecommendedLevel <= 0 {
		return emptyRecommendation(RecStatusNoConfirmedDiagnosis, "暂无当前试卷的有效诊断，无法生成补学推荐"), nil
	}
	if len(confirmed.ItemResults) == 0 {
		return emptyRecommendation(RecStatusNoItemResults, "有效诊断缺少逐题结果，无法定位薄弱知识点"), nil
	}

	var items []model.AssessmentItemResult
	if err := json.Unmarshal(confirmed.ItemResults, &items); err != nil || items == nil {
		return emptyRecommendation(RecStatusNoItemResults, "有效诊断缺少逐题结果，无法定位薄弱知识点"), nil
	}

	type clue struct {
		questionID      uint
		questionIndex   int
		questionSummary string
		cause           string
		kpIDs           []string
	}
	var clues []clue
	hasCatchUpSignal := false
	missingSnapshot := 0
	explicitSnapshot := 0
	for _, item := range items {
		switch item.AutoResult {
		case model.AutoResultWrong, model.AutoResultUnanswered:
			hasCatchUpSignal = true
			if item.KnowledgePointIDs == nil {
				missingSnapshot++
				continue
			}
			explicitSnapshot++
			cause := RecCauseWrong
			if item.AutoResult == model.AutoResultUnanswered {
				cause = RecCauseUnanswered
			}
			clues = append(clues, clue{
				questionID:      item.QuestionID,
				questionIndex:   item.QuestionIndex,
				questionSummary: item.QuestionSummary,
				cause:           cause,
				kpIDs:           item.KnowledgePointIDs,
			})
		}
	}
	if !hasCatchUpSignal {
		return emptyRecommendation(RecStatusNoWeakEvidence, "当前有效诊断没有答错或未作答的客观题，暂无补学线索"), nil
	}

	kpSet := make(map[string]struct{})
	for _, c := range clues {
		for _, id := range c.kpIDs {
			if id == "" {
				continue
			}
			kpSet[id] = struct{}{}
		}
	}
	if len(kpSet) == 0 {
		if explicitSnapshot == 0 && missingSnapshot > 0 {
			return emptyRecommendation(RecStatusLegacyNoSnapshot, "历史答卷缺少知识点快照，不能按当前题目关联重新解释"), nil
		}
		return emptyRecommendation(RecStatusNoMappedKnowledgePoints, "错题或未答题尚未关联知识点，无法匹配补学资料"), nil
	}

	kpIDs := make([]string, 0, len(kpSet))
	for id := range kpSet {
		kpIDs = append(kpIDs, id)
	}
	sort.Strings(kpIDs)

	titles, err := s.AssessmentRepo.FindKnowledgePointTitles(kpIDs)
	if err != nil {
		return MaterialRecommendationBlock{}, err
	}
	liveIDs := make([]string, 0, len(kpIDs))
	for _, id := range kpIDs {
		if _, ok := titles[id]; ok {
			liveIDs = append(liveIDs, id)
		}
	}
	if len(liveIDs) == 0 {
		return emptyRecommendation(RecStatusNoMatchingMaterials, "快照中的知识点已删除，无法匹配补学资料"), nil
	}

	links, err := s.Repo.ListMaterialKnowledgePointsByKnowledgePointIDs(liveIDs)
	if err != nil {
		return MaterialRecommendationBlock{}, err
	}
	if len(links) == 0 {
		return emptyRecommendation(RecStatusNoMatchingMaterials, "薄弱知识点暂无匹配的学习资料"), nil
	}

	materialIDs := make([]string, 0, len(links))
	seenMat := make(map[string]struct{})
	for _, link := range links {
		if _, ok := seenMat[link.MaterialID]; ok {
			continue
		}
		seenMat[link.MaterialID] = struct{}{}
		materialIDs = append(materialIDs, link.MaterialID)
	}

	materials, err := s.Repo.FindMaterialsByIDs(materialIDs)
	if err != nil {
		return MaterialRecommendationBlock{}, err
	}
	materialByID := make(map[string]model.LearningPathMaterial, len(materials))
	for _, m := range materials {
		materialByID[m.ID] = m
	}

	type agg struct {
		material model.LearningPathMaterial
		reasons  []MaterialRecommendationReason
	}
	aggs := make(map[string]*agg)
	liveSummary := map[uint]string{}
	for _, link := range links {
		m, ok := materialByID[link.MaterialID]
		if !ok {
			continue
		}
		if confirmed.RecommendedLevel <= 0 || m.Level > confirmed.RecommendedLevel {
			continue
		}
		a := aggs[m.ID]
		if a == nil {
			a = &agg{material: m}
			aggs[m.ID] = a
		}
		for _, c := range clues {
			for _, kpID := range c.kpIDs {
				if kpID != link.KnowledgePointID {
					continue
				}
				title := titles[kpID]
				if title == "" {
					title = kpID
				}
				summary := c.questionSummary
				if summary == "" {
					if cached, ok := liveSummary[c.questionID]; ok {
						summary = cached
					} else if q, err := s.AssessmentRepo.FindQuestionByID(c.questionID); err == nil && q != nil {
						summary = questionSummary(*q)
						liveSummary[c.questionID] = summary
					} else {
						liveSummary[c.questionID] = ""
					}
				}
				a.reasons = append(a.reasons, MaterialRecommendationReason{
					Cause:               c.cause,
					QuestionID:          c.questionID,
					QuestionIndex:       c.questionIndex,
					QuestionSummary:     summary,
					KnowledgePointID:    kpID,
					KnowledgePointTitle: title,
				})
			}
		}
	}

	out := make([]RecommendedMaterial, 0, len(aggs))
	for _, a := range aggs {
		if len(a.reasons) == 0 {
			continue
		}
		a.reasons = dedupeRecommendationReasons(a.reasons)
		out = append(out, RecommendedMaterial{
			ID:            a.material.ID,
			Title:         a.material.Title,
			Level:         a.material.Level,
			Points:        a.material.Points,
			ChapterNumber: a.material.ChapterNumber,
			IsCompleted:   completedMap[a.material.ID],
			Reasons:       a.reasons,
		})
	}
	if len(out) == 0 {
		return emptyRecommendation(RecStatusNoMatchingMaterials, "薄弱知识点暂无当前等级可访问的匹配资料"), nil
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].IsCompleted != out[j].IsCompleted {
			return !out[i].IsCompleted && out[j].IsCompleted
		}
		if len(out[i].Reasons) != len(out[j].Reasons) {
			return len(out[i].Reasons) > len(out[j].Reasons)
		}
		if out[i].ChapterNumber != out[j].ChapterNumber {
			return out[i].ChapterNumber < out[j].ChapterNumber
		}
		if out[i].ID != out[j].ID {
			return out[i].ID < out[j].ID
		}
		return false
	})
	return MaterialRecommendationBlock{
		Status: RecStatusOK,
		Reason: "根据当前有效诊断的错题与未答题推荐补学资料",
		Items:  out,
	}, nil
}

func dedupeRecommendationReasons(in []MaterialRecommendationReason) []MaterialRecommendationReason {
	type key struct {
		cause string
		qid   uint
		kp    string
	}
	seen := make(map[key]struct{}, len(in))
	out := make([]MaterialRecommendationReason, 0, len(in))
	for _, r := range in {
		k := key{cause: r.Cause, qid: r.QuestionID, kp: r.KnowledgePointID}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, r)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Cause != out[j].Cause {
			return out[i].Cause < out[j].Cause
		}
		if out[i].QuestionID != out[j].QuestionID {
			return out[i].QuestionID < out[j].QuestionID
		}
		return out[i].KnowledgePointID < out[j].KnowledgePointID
	})
	return out
}
