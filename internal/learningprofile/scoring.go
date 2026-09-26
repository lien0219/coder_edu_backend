package learningprofile

import (
	"fmt"
	"math"
	"time"
)

const posttestDelay = 7 * 24 * time.Hour

type DimensionScore struct {
	DimensionCode string
	RawMean       float64
	ScaleMin      int
	ScaleMax      int
	DisplayScore  float64
	ItemCount     int
}

func DisplayScore(rawMean float64, scaleMin, scaleMax int) (float64, error) {
	span := float64(scaleMax - scaleMin)
	if span <= 0 || math.IsNaN(rawMean) || math.IsInf(rawMean, 0) {
		return 0, fmt.Errorf("invalid scale or mean")
	}
	if rawMean < float64(scaleMin) || rawMean > float64(scaleMax) {
		return 0, fmt.Errorf("raw mean %.4f outside [%d, %d]", rawMean, scaleMin, scaleMax)
	}
	return (rawMean - float64(scaleMin)) / span * 100, nil
}

func IsStoredDisplayScore(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0 && value <= 100
}

func Mean(values []int) (float64, error) {
	if len(values) == 0 {
		return 0, fmt.Errorf("empty values")
	}
	sum := 0
	for _, v := range values {
		sum += v
	}
	return float64(sum) / float64(len(values)), nil
}

func ScoreInstrument(instrumentCode string, answers map[string]int) ([]DimensionScore, error) {
	if err := ValidateOfficialAnswers(instrumentCode, answers); err != nil {
		return nil, err
	}
	grouped := map[string][]int{}
	scale := map[string][2]int{}
	for _, item := range ItemsByInstrument(instrumentCode) {
		grouped[item.PrimaryDimension] = append(grouped[item.PrimaryDimension], answers[item.ItemCode])
		scale[item.PrimaryDimension] = [2]int{item.ScaleMin, item.ScaleMax}
	}
	var out []DimensionScore
	for _, dim := range DimensionOrder() {
		vals, ok := grouped[dim]
		if !ok {
			continue
		}
		raw, err := Mean(vals)
		if err != nil {
			return nil, err
		}
		minmax := scale[dim]
		display, err := DisplayScore(raw, minmax[0], minmax[1])
		if err != nil {
			return nil, err
		}
		out = append(out, DimensionScore{
			DimensionCode: dim,
			RawMean:       raw,
			ScaleMin:      minmax[0],
			ScaleMax:      minmax[1],
			DisplayScore:  display,
			ItemCount:     len(vals),
		})
	}
	return out, nil
}

// PosttestOpenAt returns the shared posttest open instant, or nil if either pretest is missing.
func PosttestOpenAt(dlSubmittedAt, sdlSubmittedAt *time.Time) *time.Time {
	if dlSubmittedAt == nil || sdlSubmittedAt == nil {
		return nil
	}
	latest := *dlSubmittedAt
	if sdlSubmittedAt.After(latest) {
		latest = *sdlSubmittedAt
	}
	open := latest.Add(posttestDelay)
	return &open
}

func PosttestIsOpen(now time.Time, dlSubmittedAt, sdlSubmittedAt *time.Time) bool {
	openAt := PosttestOpenAt(dlSubmittedAt, sdlSubmittedAt)
	if openAt == nil {
		return false
	}
	return !now.Before(*openAt)
}

func CheckPosttestOpen(now time.Time, dlSubmittedAt, sdlSubmittedAt *time.Time) error {
	if !PosttestIsOpen(now, dlSubmittedAt, sdlSubmittedAt) {
		return ErrPosttestBlocked
	}
	return nil
}
