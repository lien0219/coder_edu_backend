package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/util"
)

type optionRef struct {
	Index int
	Label string
	Text  string
}

type paperVersionQuestion struct {
	ID           uint            `json:"id"`
	QuestionType string          `json:"questionType"`
	Options      json.RawMessage `json:"options"`
	Answer       string          `json:"answer"`
	Points       int             `json:"points"`
	UpdatedAt    int64           `json:"updatedAt"`
}

type paperVersionPayload struct {
	AssessmentID uint                   `json:"assessmentId"`
	Questions    []paperVersionQuestion `json:"questions"`
}

// ComputePaperVersion 只哈希试卷内容（题干选项、标准答案、分值、题目更新时间）。
// 题目–知识点关联不进入版本：改映射不得使已确认诊断失效。
// 推荐使用提交时写入 item_results.knowledgePointIds 的快照，避免旧答卷套用新关联。
func ComputePaperVersion(assessmentID uint, questions []model.AssessmentQuestion) (string, error) {
	payload := paperVersionPayload{
		AssessmentID: assessmentID,
		Questions:    make([]paperVersionQuestion, 0, len(questions)),
	}
	sorted := append([]model.AssessmentQuestion(nil), questions...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })
	for _, q := range sorted {
		payload.Questions = append(payload.Questions, paperVersionQuestion{
			ID:           q.ID,
			QuestionType: q.QuestionType,
			Options:      q.Options,
			Answer:       q.Answer,
			Points:       q.Points,
			UpdatedAt:    q.UpdatedAt.UnixNano(),
		})
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func parseOptions(raw json.RawMessage) []optionRef {
	if len(raw) == 0 {
		return nil
	}
	var opts []map[string]interface{}
	if err := json.Unmarshal(raw, &opts); err != nil || len(opts) == 0 {
		return nil
	}
	refs := make([]optionRef, 0, len(opts))
	for i, opt := range opts {
		refs = append(refs, optionRef{
			Index: i,
			Label: stringifyOpt(opt["label"]),
			Text:  firstNonEmpty(stringifyOpt(opt["text"]), stringifyOpt(opt["label"])),
		})
	}
	return refs
}

func stringifyOpt(v interface{}) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case float64:
		if t == float64(int(t)) {
			return strconv.Itoa(int(t))
		}
		return strings.TrimSpace(fmt.Sprint(t))
	default:
		return strings.TrimSpace(fmt.Sprint(t))
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func questionSummary(q model.AssessmentQuestion) string {
	s := strings.TrimSpace(q.Title)
	if s == "" {
		s = strings.TrimSpace(q.Content)
	}
	s = strings.Join(strings.Fields(s), " ")
	runes := []rune(s)
	const max = 40
	if len(runes) > max {
		return string(runes[:max]) + "…"
	}
	return s
}

func isTrueToken(s string) bool {
	switch strings.TrimSpace(s) {
	case "true", "True", "TRUE", "正确", "对":
		return true
	default:
		return false
	}
}

func isFalseToken(s string) bool {
	switch strings.TrimSpace(s) {
	case "false", "False", "FALSE", "错误", "错":
		return true
	default:
		return false
	}
}

func optionIndexByToken(options []optionRef, token string) (int, bool) {
	token = strings.TrimSpace(token)
	if token == "" || len(options) == 0 {
		return -1, false
	}
	if n, err := strconv.Atoi(token); err == nil && n >= 0 && n < len(options) {
		return n, true
	}
	matches := []int{}
	for _, opt := range options {
		if opt.Label == token || opt.Text == token {
			matches = append(matches, opt.Index)
		}
	}
	if len(matches) == 1 {
		return matches[0], true
	}
	if isTrueToken(token) || isFalseToken(token) {
		wantTrue := isTrueToken(token)
		boolMatches := []int{}
		for _, opt := range options {
			if wantTrue && (isTrueToken(opt.Text) || isTrueToken(opt.Label)) {
				boolMatches = append(boolMatches, opt.Index)
			}
			if !wantTrue && (isFalseToken(opt.Text) || isFalseToken(opt.Label)) {
				boolMatches = append(boolMatches, opt.Index)
			}
		}
		if len(boolMatches) == 1 {
			return boolMatches[0], true
		}
	}
	return -1, false
}

func resolveAnswerKeyIndex(q model.AssessmentQuestion) (index int, ambiguous bool) {
	options := parseOptions(q.Options)
	key := strings.TrimSpace(q.Answer)
	if key == "" {
		return -1, true
	}
	if len(options) == 0 {
		// 无选项时 0/1 不能当作 true/false，避免误判。
		if _, err := strconv.Atoi(key); err == nil {
			return -1, true
		}
		if isTrueToken(key) || isFalseToken(key) {
			return -1, false
		}
		return -1, true
	}
	idx, ok := optionIndexByToken(options, key)
	if !ok {
		return -1, true
	}
	return idx, false
}

func canonicalMultipleIndexes(q model.AssessmentQuestion, raw string, studentSide bool) (indexes []int, invalid bool, ambiguous bool) {
	options := parseOptions(q.Options)
	parts := strings.Split(raw, ",")
	seen := map[int]struct{}{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if len(options) == 0 {
			if studentSide {
				return nil, true, false
			}
			return nil, false, true
		}
		idx, ok := optionIndexByToken(options, part)
		if !ok {
			if studentSide {
				return nil, true, false
			}
			return nil, false, true
		}
		seen[idx] = struct{}{}
	}
	for idx := range seen {
		indexes = append(indexes, idx)
	}
	sort.Ints(indexes)
	return indexes, false, false
}

type scoredItem struct {
	Result model.AssessmentItemResult
}

func scoreQuestion(q model.AssessmentQuestion, rawAnswer string) (scoredItem, error) {
	item := model.AssessmentItemResult{
		QuestionID:   q.ID,
		QuestionType: q.QuestionType,
		MaxPoints:    q.Points,
	}
	rawAnswer = strings.TrimSpace(rawAnswer)
	switch q.QuestionType {
	case "essay", "code":
		item.AutoResult = model.AutoResultPendingManual
		return scoredItem{Result: item}, nil
	case "single_choice", "true_false":
		if rawAnswer == "" {
			item.AutoResult = model.AutoResultUnanswered
			return scoredItem{Result: item}, nil
		}
		options := parseOptions(q.Options)
		if len(options) == 0 {
			if q.QuestionType == "true_false" {
				key := strings.TrimSpace(q.Answer)
				if _, err := strconv.Atoi(key); err == nil || key == "" {
					item.AutoResult = model.AutoResultUnscoredAmbiguous
					item.NeedsTeacherFix = true
					return scoredItem{Result: item}, nil
				}
				studentTrue := isTrueToken(rawAnswer)
				studentFalse := isFalseToken(rawAnswer)
				if !studentTrue && !studentFalse {
					return scoredItem{}, fmt.Errorf("%w: 题目 %d 选项无效", util.ErrAssessmentSubmitInvalid, q.ID)
				}
				keyTrue := isTrueToken(key)
				keyFalse := isFalseToken(key)
				if !keyTrue && !keyFalse {
					item.AutoResult = model.AutoResultUnscoredAmbiguous
					item.NeedsTeacherFix = true
					return scoredItem{Result: item}, nil
				}
				if (studentTrue && keyTrue) || (studentFalse && keyFalse) {
					item.AutoResult = model.AutoResultCorrect
					item.PointsAwarded = q.Points
				} else {
					item.AutoResult = model.AutoResultWrong
				}
				return scoredItem{Result: item}, nil
			}
			return scoredItem{}, fmt.Errorf("%w: 题目 %d 选项无效", util.ErrAssessmentSubmitInvalid, q.ID)
		}
		studentIdx, ok := optionIndexByToken(options, rawAnswer)
		if !ok {
			return scoredItem{}, fmt.Errorf("%w: 题目 %d 选项无效", util.ErrAssessmentSubmitInvalid, q.ID)
		}
		keyIdx, keyAmbiguous := resolveAnswerKeyIndex(q)
		if keyAmbiguous {
			item.AutoResult = model.AutoResultUnscoredAmbiguous
			item.NeedsTeacherFix = true
			return scoredItem{Result: item}, nil
		}
		if studentIdx == keyIdx {
			item.AutoResult = model.AutoResultCorrect
			item.PointsAwarded = q.Points
		} else {
			item.AutoResult = model.AutoResultWrong
		}
		return scoredItem{Result: item}, nil
	case "multiple_choice":
		if rawAnswer == "" {
			item.AutoResult = model.AutoResultUnanswered
			return scoredItem{Result: item}, nil
		}
		studentIdx, invalid, _ := canonicalMultipleIndexes(q, rawAnswer, true)
		if invalid {
			return scoredItem{}, fmt.Errorf("%w: 题目 %d 选项无效", util.ErrAssessmentSubmitInvalid, q.ID)
		}
		keyIdx, keyInvalid, keyAmbiguous := canonicalMultipleIndexes(q, q.Answer, false)
		if keyInvalid || keyAmbiguous {
			item.AutoResult = model.AutoResultUnscoredAmbiguous
			item.NeedsTeacherFix = true
			return scoredItem{Result: item}, nil
		}
		if sameIntSlice(studentIdx, keyIdx) {
			item.AutoResult = model.AutoResultCorrect
			item.PointsAwarded = q.Points
		} else {
			item.AutoResult = model.AutoResultWrong
		}
		return scoredItem{Result: item}, nil
	default:
		// fill_blank 及其他客观填空：可自动比，空为未作答。
		if rawAnswer == "" {
			item.AutoResult = model.AutoResultUnanswered
			return scoredItem{Result: item}, nil
		}
		if strings.TrimSpace(q.Answer) == "" {
			item.AutoResult = model.AutoResultUnscoredAmbiguous
			item.NeedsTeacherFix = true
			return scoredItem{Result: item}, nil
		}
		if rawAnswer == strings.TrimSpace(q.Answer) {
			item.AutoResult = model.AutoResultCorrect
			item.PointsAwarded = q.Points
		} else {
			item.AutoResult = model.AutoResultWrong
		}
		return scoredItem{Result: item}, nil
	}
}

func sameIntSlice(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func countsTowardObjectiveMax(result string) bool {
	switch result {
	case model.AutoResultCorrect, model.AutoResultWrong, model.AutoResultUnanswered:
		return true
	default:
		return false
	}
}

func validateAndScoreAnswers(questions []model.AssessmentQuestion, answers []model.QuestionAnswer) ([]model.AssessmentItemResult, int, int, int, error) {
	if err := validateAnswerEnvelope(questions, answers); err != nil {
		return nil, 0, 0, 0, err
	}
	answerByID := map[uint]string{}
	for _, ans := range answers {
		answerByID[ans.QuestionID] = ans.Answer
	}
	results := make([]model.AssessmentItemResult, 0, len(questions))
	autoScore := 0
	objectiveMax := 0
	pendingManual := 0
	for i, q := range questions {
		scored, err := scoreQuestion(q, answerByID[q.ID])
		if err != nil {
			return nil, 0, 0, 0, err
		}
		scored.Result.QuestionIndex = i + 1
		scored.Result.QuestionSummary = questionSummary(q)
		results = append(results, scored.Result)
		autoScore += scored.Result.PointsAwarded
		if scored.Result.AutoResult == model.AutoResultPendingManual {
			pendingManual++
		}
		if countsTowardObjectiveMax(scored.Result.AutoResult) {
			objectiveMax += q.Points
		}
	}
	return results, autoScore, objectiveMax, pendingManual, nil
}

func validateAnswerEnvelope(questions []model.AssessmentQuestion, answers []model.QuestionAnswer) error {
	questionIDs := map[uint]struct{}{}
	for _, q := range questions {
		questionIDs[q.ID] = struct{}{}
	}
	seen := map[uint]struct{}{}
	for _, ans := range answers {
		if _, ok := questionIDs[ans.QuestionID]; !ok {
			return fmt.Errorf("%w: 题目 %d 不属于本试卷", util.ErrAssessmentSubmitInvalid, ans.QuestionID)
		}
		if _, dup := seen[ans.QuestionID]; dup {
			return fmt.Errorf("%w: 题目 %d 重复提交", util.ErrAssessmentSubmitInvalid, ans.QuestionID)
		}
		seen[ans.QuestionID] = struct{}{}
	}
	return nil
}
