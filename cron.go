package cron

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type (
	bitset8  uint8
	bitset16 uint16
	bitset32 uint32
	bitset64 uint64

	fieldBounds struct {
		min, max int
	}

	Cron struct {
		minute bitset64
		hour   bitset32
		dom    bitset32
		month  bitset16
		dow    bitset8
		tz     *time.Location
	}
)

const (
	yearLimit = 5
)

var (
	boundMinute = fieldBounds{0, 59}
	boundHour   = fieldBounds{0, 24}
	boundDOM    = fieldBounds{1, 31}
	boundMonth  = fieldBounds{1, 12}
	boundDOW    = fieldBounds{0, 6}

	ErrInvalidExpression = errors.New("invalid cron expression")
	ErrMaxYearLimit      = errors.New("there is no date matching the expression within the year limit")
)

// returns the same result as Parse, but it panics when the syntax of expression is wrong
func MustParse(expr string, tz *time.Location) *Cron {
	c, err := Parse(expr, tz)
	if err != nil {
		panic(err)
	}

	return c
}

// parses the expression and returns a new schedule representing the given spec
//
// it returns an error when the syntax of expression is wrong
func Parse(expr string, tz *time.Location) (*Cron, error) {
	fields := strings.Fields(strings.TrimSpace(expr))
	if len(fields) != 5 {
		return nil, ErrInvalidExpression
	}

	minute, err := parseField[bitset64](fields[0], boundMinute)
	if err != nil {
		return nil, err
	}

	hour, err := parseField[bitset32](fields[1], boundHour)
	if err != nil {
		return nil, err
	}

	dom, err := parseField[bitset32](fields[2], boundDOM)
	if err != nil {
		return nil, err
	}

	month, err := parseField[bitset16](fields[3], boundMonth)
	if err != nil {
		return nil, err
	}

	dow, err := parseField[bitset8](fields[4], boundDOW)
	if err != nil {
		return nil, err
	}

	return &Cron{
		minute: minute,
		hour:   hour,
		dom:    dom,
		month:  month,
		dow:    dow,
		tz:     tz,
	}, nil
}

// returns an int with the bits set to 1 depending on the frecuency setted for the field, or an error if the field expression is invalid
//
// for dow = 7 => 1111111b = 127d
func parseField[T bitset8 | bitset16 | bitset32 | bitset64](field string, bounds fieldBounds) (T, error) {
	var result T = 0

	// split by , and do a binary summatory (OR) of the results
	fieldParts := strings.Split(field, ",")
	for i := 0; i < len(fieldParts); i++ {
		fieldPart := fieldParts[i]

		partialResult, err := parseFieldPart[T](fieldPart, bounds)
		if err != nil {
			return 0, err
		}

		result = result | partialResult
	}

	return result, nil
}

// returns an int with the bits set to 1 depending on the frecuency setted for the field part, or an error if the field expression is invalid
//
// fPart = number | number "-" number [ "/" number ]
func parseFieldPart[T bitset8 | bitset16 | bitset32 | bitset64](fPart string, fBounds fieldBounds) (T, error) {
	// replace "*" into "min-max".
	newexpr := strings.Replace(fPart, "*", fmt.Sprintf("%d-%d", fBounds.min, fBounds.max), 1)

	// split by /
	rangeAndStep := strings.Split(newexpr, "/")
	if len(rangeAndStep) > 2 {
		return 0, ErrInvalidExpression
	}

	hasStep := len(rangeAndStep) == 2

	/// parse the range
	// split by -
	lowAndHigh := strings.Split(rangeAndStep[0], "-")
	if len(lowAndHigh) > 2 {
		return 0, ErrInvalidExpression
	}

	// get the begining of the range
	begin, err := strconv.Atoi(lowAndHigh[0])
	if err != nil {
		return 0, ErrInvalidExpression
	}

	var end int

	// get the end of the range
	// "n/step" = "n-max/step"
	if len(lowAndHigh) == 1 && hasStep {
		end = fBounds.max
	} else if len(lowAndHigh) == 1 && !hasStep {
		end = begin
	} else if len(lowAndHigh) == 2 {
		end, err = strconv.Atoi(lowAndHigh[1])
		if err != nil {
			return 0, ErrInvalidExpression
		}
	}

	/// parse the step
	step := 1
	if hasStep {
		step, err = strconv.Atoi(rangeAndStep[1])
		if err != nil {
			return 0, ErrInvalidExpression
		}
	}

	return buildBitset[T](begin, end, step), nil
}

// creates the bit set
func buildBitset[T bitset8 | bitset16 | bitset32 | bitset64](min, max, step int) T {
	var b T

	for i := min; i <= max; i += step {
		b = b | (1 << i)
	}

	return b
}

// returns the next time that matches the expression in the timezone of the input
func (s *Cron) Next(t time.Time) (time.Time, error) {
	// flag to reset the time only once
	resetted := false

	t = t.In(s.tz)

	// calculates the max possible year for the loop
	maxYear := t.Year() + yearLimit

	// set the sec and nsec to 0 and add a minute (the closest match)
	t = t.Truncate(time.Minute).Add(1 * time.Minute)

loop:
	if t.Year() > maxYear {
		return time.Time{}, ErrMaxYearLimit
	}

	year := t.Year()
	// find the first month matching the expression
	for 1<<t.Month()&s.month == 0 {
		// if the month value have to be increased, reset the less significant time parts to 0 (only once)
		if !resetted {
			resetted = true
			t = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
		}

		// add a month to the date
		t = t.AddDate(0, 1, 0)

		// if the year changed, continue the loop to ensure the maxYear condition
		if t.Year() != year {
			goto loop
		}
	}

	month := t.Month()
	// find the first day matching the expression (day of week and day of month)
	for 1<<t.Day()&s.dom == 0 || 1<<int(t.Weekday())&s.dow == 0 {
		// if the day value have to be increased, reset the less significant time parts to 0 (only once)
		if !resetted {
			resetted = true
			t = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
		}

		// add a day to the date
		t = t.AddDate(0, 0, 1)

		// if the month changed, run the loop again to ensure the maxYear and month conditions
		if t.Month() != month {
			goto loop
		}
	}

	day := t.Day()
	// find the first day matching the expression
	for 1<<t.Hour()&s.hour == 0 {
		// if the hour value have to be increased, reset the less significant time parts to 0 (only once)
		if !resetted {
			resetted = true
			t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, t.Location())
		}

		// add an hour to the date
		t = t.Add(1 * time.Hour)

		// if the month changed, run the loop again to ensure the maxYear, month and day conditions
		if t.Day() != day {
			goto loop
		}
	}

	hour := t.Hour()
	// find the first minute matching the expression
	for 1<<t.Minute()&s.minute == 0 {
		// reset not needed (is done at the begining)

		// add a minute to the date
		t = t.Add(1 * time.Minute)

		// if the hour changed, run the loop again to ensure the maxYear, month, day and hour conditions
		if t.Hour() != hour {
			goto loop
		}
	}

	return t, nil
}
