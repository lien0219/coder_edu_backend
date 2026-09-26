package learningprofile

import (
	"encoding/json"
	"testing"
)

func completeAnswers(instrumentCode string, value int) map[string]int {
	answers := map[string]int{}
	for _, item := range ItemsByInstrument(instrumentCode) {
		answers[item.ItemCode] = value
	}
	return answers
}

func TestFreezeSnapshotRetainsLikertOptions(t *testing.T) {
	dl, err := FreezeAnswers(InstrumentDL, completeAnswers(InstrumentDL, 4))
	if err != nil {
		t.Fatal(err)
	}
	if dl.InstrumentCode != InstrumentDL || dl.InstrumentVersion != "v1" {
		t.Fatalf("DL identity %+v", dl)
	}
	if dl.ScaleMin != 1 || dl.ScaleMax != 5 || len(dl.Items) != 56 {
		t.Fatalf("DL paper %+v items=%d", dl, len(dl.Items))
	}
	wantDL := DLOptions()
	if len(dl.Items[0].Options) != 5 {
		t.Fatalf("DL options %d", len(dl.Items[0].Options))
	}
	for i, opt := range dl.Items[0].Options {
		if opt != wantDL[i] {
			t.Fatalf("DL option %d got %+v want %+v", i, opt, wantDL[i])
		}
	}
	if dl.Items[0].RawValue != 4 {
		t.Fatalf("DL raw %d", dl.Items[0].RawValue)
	}

	sdl, err := FreezeAnswers(InstrumentSDL, completeAnswers(InstrumentSDL, 6))
	if err != nil {
		t.Fatal(err)
	}
	if sdl.InstrumentCode != InstrumentSDL || sdl.InstrumentVersion != "v1" {
		t.Fatalf("SDL identity %+v", sdl)
	}
	if sdl.ScaleMin != 1 || sdl.ScaleMax != 7 || len(sdl.Items) != 20 {
		t.Fatalf("SDL paper %+v items=%d", sdl, len(sdl.Items))
	}
	wantSDL := SDLOptions()
	if len(sdl.Items[0].Options) != 7 {
		t.Fatalf("SDL options %d", len(sdl.Items[0].Options))
	}
	for i, opt := range sdl.Items[0].Options {
		if opt != wantSDL[i] {
			t.Fatalf("SDL option %d got %+v want %+v", i, opt, wantSDL[i])
		}
	}
	if sdl.Items[19].RawValue != 6 || sdl.Items[19].ItemCode != "SDL20" {
		t.Fatalf("SDL last %+v", sdl.Items[19])
	}
}

func TestFreezeSnapshotMatchesCatalogItems(t *testing.T) {
	for _, inst := range Instruments() {
		paper, err := FreezeAnswers(inst.Code, completeAnswers(inst.Code, inst.ScaleMin))
		if err != nil {
			t.Fatal(err)
		}
		catalog := ItemsByInstrument(inst.Code)
		if len(paper.Items) != len(catalog) || len(paper.Items) != inst.ItemCount {
			t.Fatalf("%s freeze %d catalog %d spec %d", inst.Code, len(paper.Items), len(catalog), inst.ItemCount)
		}
		if paper.InstrumentCode != inst.Code || paper.InstrumentVersion != inst.Version || paper.Program != inst.Program {
			t.Fatalf("header %+v vs %+v", paper, inst)
		}
		for i, item := range catalog {
			got := paper.Items[i]
			if got.ItemCode != item.ItemCode || got.SourceItemNo != item.SourceItemNo || got.Stem != item.Stem {
				t.Fatalf("%s identity mismatch %+v vs %+v", item.ItemCode, got, item)
			}
			if got.PrimaryDimension != item.PrimaryDimension || got.SecondaryDimension != item.SecondaryDimension {
				t.Fatalf("%s dimension mismatch %+v vs %+v", item.ItemCode, got, item)
			}
			if got.ScaleMin != item.ScaleMin || got.ScaleMax != item.ScaleMax || got.ScoringDirection != item.ScoringDirection {
				t.Fatalf("%s scale/direction mismatch %+v vs %+v", item.ItemCode, got, item)
			}
			if got.SortOrder != item.SortOrder || got.RawValue != inst.ScaleMin {
				t.Fatalf("%s sort/raw %+v", item.ItemCode, got)
			}
			if len(got.Options) != len(inst.Options) {
				t.Fatalf("%s options %d", item.ItemCode, len(got.Options))
			}
			for j, opt := range inst.Options {
				if got.Options[j] != opt {
					t.Fatalf("%s option %d %+v", item.ItemCode, j, got.Options[j])
				}
			}
		}
		raw, err := json.Marshal(paper)
		if err != nil {
			t.Fatal(err)
		}
		var round FrozenPaper
		if err := json.Unmarshal(raw, &round); err != nil {
			t.Fatal(err)
		}
		if round.Items[0].Options[0].Label == "" || round.Items[0].Options[0].Value != inst.ScaleMin {
			t.Fatalf("json roundtrip options %+v", round.Items[0].Options)
		}
	}
}

func TestFreezeShownPaperUsesDatabaseOptionLabels(t *testing.T) {
	inst, ok := InstrumentByCode(InstrumentSDL)
	if !ok {
		t.Fatal("missing SDL spec")
	}
	catalog := ItemsByInstrument(InstrumentSDL)
	shown := make([]ShownItem, 0, len(catalog))
	custom := []LikertOption{{Value: 1, Label: "库内标签1"}, {Value: 2, Label: "库内标签2"}}
	for i, item := range catalog {
		opts := SDLOptions()
		if i == 0 {
			opts = append([]LikertOption{}, custom...)
			opts = append(opts, SDLOptions()[2:]...)
		}
		shown = append(shown, ShownItem{Item: item, Options: opts})
	}
	paper, err := FreezeShownPaper(inst, shown, completeAnswers(InstrumentSDL, 4))
	if err != nil {
		t.Fatal(err)
	}
	if paper.Items[0].Options[0].Label != "库内标签1" || paper.Items[0].Options[1].Label != "库内标签2" {
		t.Fatalf("did not keep DB labels %+v", paper.Items[0].Options[:2])
	}
	if paper.Items[1].Options[3].Label != "一般 / 不确定" {
		t.Fatalf("SDL option 4 %+v", paper.Items[1].Options[3])
	}
}

func TestFreezeDoesNotChangeScoringInputs(t *testing.T) {
	answers := completeAnswers(InstrumentDL, 5)
	paper, err := FreezeAnswers(InstrumentDL, answers)
	if err != nil {
		t.Fatal(err)
	}
	frozenAnswers := map[string]int{}
	for _, item := range paper.Items {
		frozenAnswers[item.ItemCode] = item.RawValue
	}
	fromLive, err := ScoreInstrument(InstrumentDL, answers)
	if err != nil {
		t.Fatal(err)
	}
	fromFrozen, err := ScoreInstrument(InstrumentDL, frozenAnswers)
	if err != nil {
		t.Fatal(err)
	}
	if len(fromLive) != len(fromFrozen) {
		t.Fatalf("score len %d vs %d", len(fromLive), len(fromFrozen))
	}
	for i := range fromLive {
		if fromLive[i] != fromFrozen[i] {
			t.Fatalf("score %d %+v vs %+v", i, fromLive[i], fromFrozen[i])
		}
	}
}
