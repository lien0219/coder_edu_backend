package service

import (
	"math"
	"sort"

	"coder_edu_backend/internal/model"
)

const DefaultKnowledgeMasteryPassingScore = 60

const (
	KnowledgeMasteryStatusMastered    = "mastered"
	KnowledgeMasteryStatusNeedsReview = "needs_review"
)

type KnowledgePointMasteryMeta struct {
	ID              string
	Title           string
	Order           int
	CompletionScore int
}

type KnowledgeMastery struct {
	KnowledgePointID        string `json:"knowledgePointId"`
	Title                   string `json:"title"`
	EarnedPoints            int    `json:"earnedPoints"`
	MaxPoints               int    `json:"maxPoints"`
	Percent                 int    `json:"percent"`
	PassingScore            int    `json:"passingScore"`
	Status                  string `json:"status"`
	QuestionOrders          []int  `json:"questionOrders"`
	IncorrectQuestionOrders []int  `json:"incorrectQuestionOrders"`
}

func ResolveKnowledgeMasteryPassingScore(completionScore int) int {
	if completionScore < 1 || completionScore > 100 {
		return DefaultKnowledgeMasteryPassingScore
	}
	return completionScore
}

func itemMaxPoints(item model.AssessmentItemResult, fallback map[uint]int) int {
	if item.MaxPoints > 0 {
		return item.MaxPoints
	}
	if fallback != nil {
		if pts, ok := fallback[item.QuestionID]; ok && pts > 0 {
			return pts
		}
	}
	if item.AutoResult == model.AutoResultCorrect && item.PointsAwarded > 0 {
		return item.PointsAwarded
	}
	return 0
}

func uniqueSortedPositive(nums []int) []int {
	seen := make(map[int]struct{}, len(nums))
	out := make([]int, 0, len(nums))
	for _, n := range nums {
		if n <= 0 {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	sort.Ints(out)
	return out
}

func masteryPercent(earned, max int) int {
	if max <= 0 {
		return 0
	}
	pct := int(math.Round(float64(earned) / float64(max) * 100))
	if pct < 0 {
		return 0
	}
	if pct > 100 {
		return 100
	}
	return pct
}

// AggregateKnowledgeMastery 按提交快照聚合客观题知识点掌握度，不读标准答案、不写库。
func AggregateKnowledgeMastery(
	items []model.AssessmentItemResult,
	metas map[string]KnowledgePointMasteryMeta,
	fallbackMaxPoints map[uint]int,
) []KnowledgeMastery {
	type bucket struct {
		earned    int
		max       int
		orders    []int
		incorrect []int
	}
	byID := map[string]*bucket{}
	for _, item := range items {
		if !countsTowardObjectiveMax(item.AutoResult) {
			continue
		}
		if len(item.KnowledgePointIDs) == 0 {
			continue
		}
		maxPts := itemMaxPoints(item, fallbackMaxPoints)
		incorrect := item.AutoResult == model.AutoResultWrong || item.AutoResult == model.AutoResultUnanswered
		seenKP := map[string]struct{}{}
		for _, kpID := range item.KnowledgePointIDs {
			if kpID == "" {
				continue
			}
			if _, dup := seenKP[kpID]; dup {
				continue
			}
			seenKP[kpID] = struct{}{}
			b := byID[kpID]
			if b == nil {
				b = &bucket{}
				byID[kpID] = b
			}
			b.earned += item.PointsAwarded
			b.max += maxPts
			if item.QuestionIndex > 0 {
				b.orders = append(b.orders, item.QuestionIndex)
				if incorrect {
					b.incorrect = append(b.incorrect, item.QuestionIndex)
				}
			}
		}
	}

	out := make([]KnowledgeMastery, 0, len(byID))
	for id, b := range byID {
		if b.max <= 0 {
			continue
		}
		meta := KnowledgePointMasteryMeta{}
		if metas != nil {
			meta = metas[id]
		}
		title := meta.Title
		if title == "" {
			title = id
		}
		passing := ResolveKnowledgeMasteryPassingScore(meta.CompletionScore)
		percent := masteryPercent(b.earned, b.max)
		status := KnowledgeMasteryStatusNeedsReview
		if percent >= passing {
			status = KnowledgeMasteryStatusMastered
		}
		out = append(out, KnowledgeMastery{
			KnowledgePointID:        id,
			Title:                   title,
			EarnedPoints:            b.earned,
			MaxPoints:               b.max,
			Percent:                 percent,
			PassingScore:            passing,
			Status:                  status,
			QuestionOrders:          uniqueSortedPositive(b.orders),
			IncorrectQuestionOrders: uniqueSortedPositive(b.incorrect),
		})
	}

	sort.SliceStable(out, func(i, j int) bool {
		oi, oj := 0, 0
		if metas != nil {
			oi = metas[out[i].KnowledgePointID].Order
			oj = metas[out[j].KnowledgePointID].Order
		}
		if oi != oj {
			return oi < oj
		}
		if out[i].Title != out[j].Title {
			return out[i].Title < out[j].Title
		}
		return out[i].KnowledgePointID < out[j].KnowledgePointID
	})
	return out
}
