package service

import (
	"fmt"
	"strings"
)

// brewDate represents a parsed DIY Dog first_brewed value ("MM/YYYY" or "YYYY").
type brewDate struct {
	Year     int
	Month    int // 1..12 when HasMonth is true; 0 otherwise
	HasMonth bool
}

// parseBrewDate parses strictly formatted "MM/YYYY" (month 01..12, 4-digit year)
// or "YYYY" (4-digit year) strings.
func parseBrewDate(raw string) (brewDate, error) {
	s := strings.TrimSpace(raw)
	if len(s) == 4 {
		year, ok := parseDigits(s)
		if !ok || year < 1000 {
			return brewDate{}, fmt.Errorf("%w: invalid date %q (expected MM/YYYY or YYYY)", ErrInvalidInput, raw)
		}
		return brewDate{Year: year}, nil
	}

	if len(s) == 7 && s[2] == '/' {
		month, okM := parseDigits(s[:2])
		year, okY := parseDigits(s[3:])
		if !okM || !okY || month < 1 || month > 12 || year < 1000 {
			return brewDate{}, fmt.Errorf("%w: invalid date %q (expected MM/YYYY or YYYY)", ErrInvalidInput, raw)
		}
		return brewDate{Year: year, Month: month, HasMonth: true}, nil
	}

	return brewDate{}, fmt.Errorf("%w: invalid date %q (expected MM/YYYY or YYYY)", ErrInvalidInput, raw)
}

func parseDigits(s string) (int, bool) {
	if len(s) == 0 {
		return 0, false
	}
	n := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			return 0, false
		}
		n = n*10 + int(c-'0')
	}
	return n, true
}

// lowerBoundKey converts a brewDate into a comparable month index for inclusive lower-bound checks.
// Year-only dates map to January of that year.
func (d brewDate) lowerBoundKey() int {
	m := d.Month
	if !d.HasMonth {
		m = 1
	}
	return d.Year*12 + m
}

// upperBoundKey converts a brewDate into a comparable month index for inclusive upper-bound checks.
// Year-only dates map to December of that year.
func (d brewDate) upperBoundKey() int {
	m := d.Month
	if !d.HasMonth {
		m = 12
	}
	return d.Year*12 + m
}

// compareBrewDate orders dates chronologically. Within the same year, year-only dates (Month == 0)
// precede month-specific dates (Month 1..12).
func compareBrewDate(a, b brewDate) int {
	if c := compareInt(a.Year, b.Year); c != 0 {
		return c
	}
	return compareInt(a.Month, b.Month)
}

type brewDateRange struct {
	hasAfter   bool
	afterDate  brewDate
	hasBefore  bool
	beforeDate brewDate
}

func parseBrewDateRange(afterRaw, beforeRaw string) (brewDateRange, error) {
	var (
		r   brewDateRange
		err error
	)
	if trimmed := strings.TrimSpace(afterRaw); trimmed != "" {
		r.afterDate, err = parseBrewDate(trimmed)
		if err != nil {
			return brewDateRange{}, fmt.Errorf("brewed_after: %w", err)
		}
		r.hasAfter = true
	}
	if trimmed := strings.TrimSpace(beforeRaw); trimmed != "" {
		r.beforeDate, err = parseBrewDate(trimmed)
		if err != nil {
			return brewDateRange{}, fmt.Errorf("brewed_before: %w", err)
		}
		r.hasBefore = true
	}
	return r, nil
}

func (r brewDateRange) active() bool {
	return r.hasAfter || r.hasBefore
}

func (r brewDateRange) matches(d brewDate, hasDate bool) bool {
	if !r.active() {
		return true
	}
	if !hasDate {
		return false
	}
	if r.hasAfter && d.lowerBoundKey() < r.afterDate.lowerBoundKey() {
		return false
	}
	if r.hasBefore && d.lowerBoundKey() > r.beforeDate.upperBoundKey() {
		return false
	}
	return true
}
