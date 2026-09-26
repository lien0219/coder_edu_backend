package learningprofile

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

var expectedDimensionCounts = map[string]int{
	DimCriticalThinking:           26,
	DimProblemSolving:             18,
	DimKnowledgeTransfer:          12,
	DimLearningMotivation:         6,
	DimPlanningAndImplementation:  6,
	DimSelfManagement:             4,
	DimInterpersonalCommunication: 4,
}

func ValidateCatalog() error {
	items := OfficialItems()
	if len(items) != 76 {
		return fmt.Errorf("want 76 items, got %d", len(items))
	}
	dl := ItemsByInstrument(InstrumentDL)
	sdl := ItemsByInstrument(InstrumentSDL)
	if len(dl) != 56 {
		return fmt.Errorf("DL want 56, got %d", len(dl))
	}
	if len(sdl) != 20 {
		return fmt.Errorf("SDL want 20, got %d", len(sdl))
	}

	seen := map[string]struct{}{}
	counts := map[string]int{}
	for _, item := range items {
		if strings.TrimSpace(item.Stem) == "" {
			return fmt.Errorf("%s empty stem", item.ItemCode)
		}
		if HasNonPrintable(item.Stem) {
			return fmt.Errorf("%s stem has control characters", item.ItemCode)
		}
		if item.ScoringDirection != ScoringForward {
			return fmt.Errorf("%s direction %q", item.ItemCode, item.ScoringDirection)
		}
		if _, dup := seen[item.ItemCode]; dup {
			return fmt.Errorf("duplicate item code %s", item.ItemCode)
		}
		seen[item.ItemCode] = struct{}{}
		counts[item.PrimaryDimension]++
		if looksLikeDemographics(item.Stem) || looksLikeAppendix(item.Stem) {
			return fmt.Errorf("%s looks like demographics or appendix", item.ItemCode)
		}
	}

	if err := assertContinuous("CT", 26, seen); err != nil {
		return err
	}
	if err := assertContinuous("PS", 18, seen); err != nil {
		return err
	}
	if err := assertContinuous("KT", 12, seen); err != nil {
		return err
	}
	if err := assertContinuous("SDL", 20, seen); err != nil {
		return err
	}

	for dim, want := range expectedDimensionCounts {
		if counts[dim] != want {
			return fmt.Errorf("dimension %s want %d got %d", dim, want, counts[dim])
		}
	}

	for _, item := range dl {
		if item.ScaleMin != 1 || item.ScaleMax != 5 {
			return fmt.Errorf("%s scale %d-%d", item.ItemCode, item.ScaleMin, item.ScaleMax)
		}
	}
	for _, item := range sdl {
		if item.ScaleMin != 1 || item.ScaleMax != 7 {
			return fmt.Errorf("%s scale %d-%d", item.ItemCode, item.ScaleMin, item.ScaleMax)
		}
	}

	dlOpts := DLOptions()
	sdlOpts := SDLOptions()
	if err := assertOptionValues(dlOpts, 1, 5); err != nil {
		return fmt.Errorf("DL options: %w", err)
	}
	if err := assertOptionValues(sdlOpts, 1, 7); err != nil {
		return fmt.Errorf("SDL options: %w", err)
	}
	return nil
}

func ValidateOfficialAnswers(instrumentCode string, answers map[string]int) error {
	items := ItemsByInstrument(instrumentCode)
	if len(items) == 0 {
		return fmt.Errorf("unknown instrument %s", instrumentCode)
	}
	for _, item := range items {
		val, ok := answers[item.ItemCode]
		if !ok {
			return fmt.Errorf("missing %s", item.ItemCode)
		}
		if val < item.ScaleMin || val > item.ScaleMax {
			return fmt.Errorf("%s value %d outside %d-%d", item.ItemCode, val, item.ScaleMin, item.ScaleMax)
		}
	}
	if len(answers) != len(items) {
		return fmt.Errorf("unexpected extra answers: got %d want %d", len(answers), len(items))
	}
	return nil
}

func CanTransitionToSubmitted(currentStatus string) error {
	switch currentStatus {
	case StatusDraft, "":
		return nil
	case StatusSubmitted:
		return fmt.Errorf("already submitted")
	default:
		return fmt.Errorf("unknown status %s", currentStatus)
	}
}

func assertContinuous(prefix string, n int, seen map[string]struct{}) error {
	for i := 1; i <= n; i++ {
		code := prefix + strconv.Itoa(i)
		if _, ok := seen[code]; !ok {
			return fmt.Errorf("missing %s", code)
		}
	}
	return nil
}

func assertOptionValues(opts []LikertOption, min, max int) error {
	if len(opts) != max-min+1 {
		return fmt.Errorf("want %d options got %d", max-min+1, len(opts))
	}
	for i, opt := range opts {
		if opt.Value != min+i {
			return fmt.Errorf("option index %d value %d", i, opt.Value)
		}
		if strings.TrimSpace(opt.Label) == "" {
			return fmt.Errorf("option %d empty label", opt.Value)
		}
	}
	return nil
}

func looksLikeDemographics(stem string) bool {
	needles := []string{"性别", "年级", "专业类别", "学习C语言的时间", "自评成绩水平"}
	for _, n := range needles {
		if strings.Contains(stem, n) {
			return true
		}
	}
	return false
}

func looksLikeAppendix(stem string) bool {
	needles := []string{"KMO", "Bartlett", "Cronbach", "McDonald", "HTMT", "Fornell"}
	for _, n := range needles {
		if strings.Contains(stem, n) {
			return true
		}
	}
	return false
}

func HasNonPrintable(s string) bool {
	for _, r := range s {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return true
		}
	}
	return false
}
