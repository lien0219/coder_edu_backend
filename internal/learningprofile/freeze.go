package learningprofile

import "fmt"

// FrozenPaper is the official-submit payload stored in administrations.item_snapshot.
// It keeps the instrument identity plus every shown option label/value with the raw answer.
type FrozenPaper struct {
	InstrumentCode    string       `json:"instrumentCode"`
	InstrumentVersion string       `json:"instrumentVersion"`
	InstrumentName    string       `json:"instrumentName"`
	Program           string       `json:"program"`
	ScaleMin          int          `json:"scaleMin"`
	ScaleMax          int          `json:"scaleMax"`
	Items             []FrozenItem `json:"items"`
}

type FrozenItem struct {
	ItemCode           string         `json:"itemCode"`
	SourceItemNo       string         `json:"sourceItemNo"`
	Stem               string         `json:"stem"`
	PrimaryDimension   string         `json:"primaryDimension"`
	SecondaryDimension string         `json:"secondaryDimension"`
	ScaleMin           int            `json:"scaleMin"`
	ScaleMax           int            `json:"scaleMax"`
	ScoringDirection   string         `json:"scoringDirection"`
	SortOrder          int            `json:"sortOrder"`
	Options            []LikertOption `json:"options"`
	RawValue           int            `json:"rawValue"`
}

// ShownItem is one live paper item as loaded from the official DB bank.
type ShownItem struct {
	Item    ItemSpec
	Options []LikertOption
}

func FreezeAnswers(instrumentCode string, answers map[string]int) (FrozenPaper, error) {
	inst, ok := InstrumentByCode(instrumentCode)
	if !ok {
		return FrozenPaper{}, fmt.Errorf("unknown instrument %s", instrumentCode)
	}
	if err := ValidateOfficialAnswers(instrumentCode, answers); err != nil {
		return FrozenPaper{}, err
	}
	paper := FrozenPaper{
		InstrumentCode:    inst.Code,
		InstrumentVersion: inst.Version,
		InstrumentName:    inst.Name,
		Program:           inst.Program,
		ScaleMin:          inst.ScaleMin,
		ScaleMax:          inst.ScaleMax,
	}
	for _, item := range ItemsByInstrument(instrumentCode) {
		paper.Items = append(paper.Items, FrozenItem{
			ItemCode:           item.ItemCode,
			SourceItemNo:       item.SourceItemNo,
			Stem:               item.Stem,
			PrimaryDimension:   item.PrimaryDimension,
			SecondaryDimension: item.SecondaryDimension,
			ScaleMin:           item.ScaleMin,
			ScaleMax:           item.ScaleMax,
			ScoringDirection:   item.ScoringDirection,
			SortOrder:          item.SortOrder,
			Options:            copyOptions(inst.Options),
			RawValue:           answers[item.ItemCode],
		})
	}
	return paper, nil
}

func copyOptions(opts []LikertOption) []LikertOption {
	out := make([]LikertOption, len(opts))
	copy(out, opts)
	return out
}

// FreezeShownPaper snapshots the items actually shown from the official bank,
// including per-item option labels stored in the database.
func FreezeShownPaper(inst InstrumentSpec, items []ShownItem, answers map[string]int) (FrozenPaper, error) {
	if inst.Code == "" || inst.ItemCount <= 0 {
		return FrozenPaper{}, fmt.Errorf("invalid instrument")
	}
	codes := make([]string, 0, len(items))
	for _, shown := range items {
		codes = append(codes, shown.Item.ItemCode)
	}
	if err := MatchOfficialItemCodes(inst.Code, codes); err != nil {
		return FrozenPaper{}, err
	}
	if err := ValidateOfficialAnswers(inst.Code, answers); err != nil {
		return FrozenPaper{}, err
	}
	paper := FrozenPaper{
		InstrumentCode:    inst.Code,
		InstrumentVersion: inst.Version,
		InstrumentName:    inst.Name,
		Program:           inst.Program,
		ScaleMin:          inst.ScaleMin,
		ScaleMax:          inst.ScaleMax,
	}
	for _, shown := range items {
		if len(shown.Options) == 0 {
			return FrozenPaper{}, fmt.Errorf("%s missing options", shown.Item.ItemCode)
		}
		item := shown.Item
		paper.Items = append(paper.Items, FrozenItem{
			ItemCode:           item.ItemCode,
			SourceItemNo:       item.SourceItemNo,
			Stem:               item.Stem,
			PrimaryDimension:   item.PrimaryDimension,
			SecondaryDimension: item.SecondaryDimension,
			ScaleMin:           item.ScaleMin,
			ScaleMax:           item.ScaleMax,
			ScoringDirection:   item.ScoringDirection,
			SortOrder:          item.SortOrder,
			Options:            copyOptions(shown.Options),
			RawValue:           answers[item.ItemCode],
		})
	}
	return paper, nil
}
