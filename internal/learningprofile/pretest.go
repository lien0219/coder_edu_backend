package learningprofile

import (
	"errors"
	"fmt"
)

var (
	ErrPosttestBlocked    = errors.New("posttest is not available")
	ErrInvalidWave        = errors.New("invalid assessment wave")
	ErrUnknownInstrument  = errors.New("unknown learning profile instrument")
	ErrAlreadySubmitted   = errors.New("learning profile already submitted")
	ErrCatalogUnavailable = errors.New("official item bank is unavailable")
	ErrCatalogMismatch    = errors.New("official item bank does not match this instrument")
)

func IsOfficialInstrument(code string) bool {
	return code == InstrumentDL || code == InstrumentSDL
}

func RequireOfficialInstrument(code string) error {
	if !IsOfficialInstrument(code) {
		return ErrUnknownInstrument
	}
	return nil
}

func RequirePretestWave(wave string) error {
	switch wave {
	case WavePretest:
		return nil
	case WavePosttest:
		return ErrPosttestBlocked
	default:
		return ErrInvalidWave
	}
}

func RequireWave(wave string) error {
	switch wave {
	case WavePretest, WavePosttest:
		return nil
	default:
		return ErrInvalidWave
	}
}

func AdministrationStatus(hasRow bool, status string) string {
	if !hasRow || status == "" {
		return "not_started"
	}
	if status == StatusSubmitted {
		return StatusSubmitted
	}
	return StatusDraft
}

func ValidateDraftAnswers(instrumentCode string, answers map[string]int) error {
	if err := RequireOfficialInstrument(instrumentCode); err != nil {
		return err
	}
	items := ItemsByInstrument(instrumentCode)
	if len(items) == 0 {
		return ErrUnknownInstrument
	}
	allowed := make(map[string]ItemSpec, len(items))
	for _, item := range items {
		allowed[item.ItemCode] = item
	}
	for code, val := range answers {
		item, ok := allowed[code]
		if !ok {
			return fmt.Errorf("unexpected item %s", code)
		}
		if val < item.ScaleMin || val > item.ScaleMax {
			return fmt.Errorf("%s value %d outside %d-%d", code, val, item.ScaleMin, item.ScaleMax)
		}
	}
	return nil
}

func MatchOfficialItemCodes(instrumentCode string, itemCodes []string) error {
	if err := RequireOfficialInstrument(instrumentCode); err != nil {
		return err
	}
	want := ItemsByInstrument(instrumentCode)
	if len(want) == 0 {
		return ErrUnknownInstrument
	}
	if len(itemCodes) != len(want) {
		return fmt.Errorf("%w: want %d items got %d", ErrCatalogMismatch, len(want), len(itemCodes))
	}
	seen := make(map[string]struct{}, len(itemCodes))
	for _, code := range itemCodes {
		seen[code] = struct{}{}
	}
	for _, item := range want {
		if _, ok := seen[item.ItemCode]; !ok {
			return fmt.Errorf("%w: missing %s", ErrCatalogMismatch, item.ItemCode)
		}
	}
	if len(seen) != len(want) {
		return fmt.Errorf("%w: unexpected extra item codes", ErrCatalogMismatch)
	}
	return nil
}
