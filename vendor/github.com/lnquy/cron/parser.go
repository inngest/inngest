package cron

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	InvalidExprError           = errors.New("invalid expression")
	InvalidExprSecondError     = errors.New("invalid expression, second part")
	InvalidExprMinuteError     = errors.New("invalid expression, minute part")
	InvalidExprHourError       = errors.New("invalid expression, hour part")
	InvalidExprDayOfMonthError = errors.New("invalid expression, day of month part")
	InvalidExprMonthError      = errors.New("invalid expression, month part")
	InvalidExprDayOfWeekError  = errors.New("invalid expression, day of week part")
	InvalidExprYearError       = errors.New("invalid expression, year part")
)

var (
	yearRegex = regexp.MustCompile(`\d{4}$`)

	everySecMinRegex = regexp.MustCompile(`[*/]`)
	everyHourRegex   = regexp.MustCompile(`[*\-,/]`)

	rangeRegex = regexp.MustCompile(`^[*\-,]`)

	invalidCharsDOWDOMRegex = regexp.MustCompile(`[a-km-vx-zA-KM-VX-Z]`)
)

var (
	zeroRune  int32 = 48
	sevenRune int32 = 55
)

var (
	days = map[string]int{
		"sun": 0,
		"mon": 1,
		"tue": 2,
		"wed": 3,
		"thu": 4,
		"fri": 5,
		"sat": 6,
	}

	months = map[string]int{
		"jan": 1,
		"feb": 2,
		"mar": 3,
		"apr": 4,
		"may": 5,
		"jun": 6,
		"jul": 7,
		"aug": 8,
		"sep": 9,
		"oct": 10,
		"nov": 11,
		"dec": 12,
	}
)

type (
	cronParser struct {
		isDOWStartsAtOne bool
	}

	// Parser represents the cron parser.
	Parser interface {
		Parse(expr string) (exprParts []string, err error)
	}
)

// Parse parses, normalizes and validates the CRON expression.
// If the CRON expression is valid, then the returned list always the normalized 7-part-CRON format.
// Example: "* 5 * * *" => ["", "*", "5", "*", "*", "*", ""]
func (p *cronParser) Parse(expr string) (exprParts []string, err error) {
	exprParts, err = p.extractExprParts(expr)
	if err != nil {
		return nil, fmt.Errorf("failed to extract expression parts: %w", err)
	}

	if err = p.normalize(exprParts); err != nil {
		return nil, fmt.Errorf("failed to normalize expression parts: %w", err)
	}

	if err = p.validate(exprParts); err != nil {
		return nil, fmt.Errorf("invalid CRON expression: %w", err)
	}
	return exprParts, nil
}

func (p *cronParser) extractExprParts(expr string) (exprParts []string, err error) {
	if strings.TrimSpace(expr) == "" {
		return nil, InvalidExprError
	}

	expr = strings.ToLower(expr)
	exprParts = make([]string, 7, 7)
	parts := strings.Fields(expr)

	switch {
	case len(parts) < 5:
		return nil, fmt.Errorf("expression has only %d part(s), at least 5 parts required: %w", len(parts), InvalidExprError)
	case len(parts) == 5:
		// Expression has 5 parts (standard POSIX CRON)
		// => Prepend 1 and append 1 empty part at the beginning and the end of exprParts
		copy(exprParts[1:], append(parts, ""))
	case len(parts) == 6:
		// Has year (last part) or second (first part)
		if yearRegex.MatchString(parts[5]) {
			// Year provided => Prepend 1 empty part at the beginning for second
			copy(exprParts[1:], parts)
			break
		}
		// Second provided => Last parts (year) is empty
		copy(exprParts, parts)
	case len(parts) > 7:
		return nil, fmt.Errorf("expression has %d parts, at most 7 parts allowed: %w", len(parts), InvalidExprError)
	default: // Expression has 7 parts
		exprParts = parts
	}

	return exprParts, nil
}

func (p *cronParser) normalize(exprParts []string) (err error) {
	second := exprParts[0]
	minute := exprParts[1]
	hour := exprParts[2]
	dayOfMonth := exprParts[3]
	month := exprParts[4]
	dayOfWeek := exprParts[5]
	year := exprParts[6]

	// Convert ? to * for DOM and DOW
	dayOfMonth = strings.Replace(dayOfMonth, "?", "*", 1)
	dayOfWeek = strings.Replace(dayOfWeek, "?", "*", 1)
	// Convert ? to * for hour. ? isn't valid for hour position but we can work around it
	hour = strings.Replace(hour, "?", "*", 1)

	// Convert 0/, 1/ to */
	if strings.Index(second, "0/") == 0 {
		second = strings.Replace(second, "0/", "*/", 1)
	}
	if strings.Index(minute, "0/") == 0 {
		minute = strings.Replace(minute, "0/", "*/", 1)
	}
	if strings.Index(hour, "0/") == 0 {
		hour = strings.Replace(hour, "0/", "*/", 1)
	}
	if strings.Index(dayOfMonth, "1/") == 0 {
		dayOfMonth = strings.Replace(dayOfMonth, "1/", "*/", 1)
	}
	if strings.Index(month, "1/") == 0 {
		month = strings.Replace(month, "1/", "*/", 1)
	}
	if strings.Index(dayOfWeek, "1/") == 0 {
		dayOfWeek = strings.Replace(dayOfWeek, "1/", "*/", 1)
	}
	if strings.Index(year, "1/") == 0 {
		year = strings.Replace(year, "1/", "*/", 1)
	}

	// Adjust DOW based on isDOWStartsAtZero option
	// Normalized DOW: 0=Sunday/6=Saturday
	dowRunes := []rune(dayOfWeek)
	for i, c := range dowRunes {
		if c == '/' || c == '#' { // Keep days after # and / as it is
			break
		}
		if c < zeroRune || c > sevenRune {
			continue
		}

		if !p.isDOWStartsAtOne {
			if c != sevenRune {
				continue
			}
			c = zeroRune // Accept 7 means Sunday too
		} else {
			if c == zeroRune {
				return fmt.Errorf("day of week starts at 1, must be from 1 to 7: %w", InvalidExprDayOfWeekError)
			}
			c -= 1 // Day of week start at 1 (Monday), so shift it 1
		}

		// Replace adjusted day of week
		dowRunes[i] = c
	}
	dayOfWeek = string(dowRunes)

	// Convert DOW 'L' to '6' (Saturday)
	if dayOfWeek == "l" {
		dayOfWeek = "6"
	}

	if strings.Index(dayOfMonth, "w") > -1 &&
		(strings.Index(dayOfMonth, ",") > -1 || strings.Index(dayOfMonth, "-") > -1) {
		return fmt.Errorf("the 'W' character can be specified only when the day-of-month is a single day, not a range or list of days: %w", InvalidExprDayOfMonthError)
	}

	// Convert DOW SUN-SAT format to 0-6 format
	for k, v := range days {
		dayOfWeek = strings.Replace(dayOfWeek, k, strconv.Itoa(v), 1)
	}

	// Convert DON JAN-DEC format to 1-12 format
	for k, v := range months {
		month = strings.Replace(month, k, strconv.Itoa(v), 1)
	}

	if second == "0" {
		second = ""
	}

	// If time interval or * (every) is specified for seconds or minutes and hours part is
	// single item, make it a "self-range" so the expression can be interpreted as
	// an interval 'between' range.
	// This will allow us to easily interpret an hour part as 'between' a second or minute duration.
	// For example:
	//    0-20/3 9 * * * => 0-20/3 9-9 * * * (9 => 9-9) => Every 3 minutes, minutes 0 through 20
	//       past the hour, between 09:00 AM and 09:59 AM
	//    */5 3 * * * => */5 3-3 * * * (3 => 3-3) => Every 5 minutes, between 03:00 AM and 03:59 AM
	if !everyHourRegex.MatchString(hour) &&
		(everySecMinRegex.MatchString(second) || everySecMinRegex.MatchString(minute)) {
		hour += "-" + hour
	}

	exprParts[0] = second
	exprParts[1] = minute
	exprParts[2] = hour
	exprParts[3] = dayOfMonth
	exprParts[4] = month
	exprParts[5] = dayOfWeek
	exprParts[6] = year

	// Loop through all parts and apply global normalization
	for i := range exprParts {
		if exprParts[i] == "*/1" {
			exprParts[i] = "*"
		}

		// Convert Month,DOW,Year step values with a starting value (i.e. not '*') to between expressions.
		// This allows us to reuse the between expression handling for step values.
		// For example:
		//   - month part '3/2' will be converted to '3-12/2' (every 2 months between March and December)
		//   - DOW part '3/2' will be converted to '3-6/2' (every 2 days between Tuesday and Saturday)
		if idx := strings.Index(exprParts[i], "/"); idx != -1 && !rangeRegex.MatchString(exprParts[i]) {
			var stepRangeThrough string
			switch i {
			case 4: // Month
				stepRangeThrough = "12"
			case 5: // Day of week
				stepRangeThrough = "6"
			case 6: // Year
				stepRangeThrough = "2099"
			}

			if stepRangeThrough == "" {
				continue
			}
			exprParts[i] = fmt.Sprintf("%s-%s/%s", exprParts[i][:idx], stepRangeThrough, exprParts[i][idx+1:])
		}
	}

	return nil
}

func (p *cronParser) validate(exprParts []string) (err error) {
	// Extract the numbers from s string
	buf := bytes.NewBuffer(make([]byte, 0, 8))
	getNumbersFunc := func(s string) (numbers []string) {
		for _, b := range s {
			if b >= '0' && b <= '9' {
				_, _ = buf.WriteRune(b)
			} else {
				if buf.Len() > 0 {
					numbers = append(numbers, buf.String())
					buf.Reset()
				}
			}
		}

		if buf.Len() > 0 {
			numbers = append(numbers, buf.String())
			buf.Reset()
		}
		return numbers
	}

	// Year
	// Check year first to reduce bound checking
	matches := getNumbersFunc(exprParts[6])
	if !isValidNumbers(matches, 1, 2099) {
		return fmt.Errorf("year contains invalid values: %w", InvalidExprYearError)
	}

	// Second
	matches = getNumbersFunc(exprParts[0])
	if !isValidNumbers(matches, 0, 59) {
		return fmt.Errorf("second contains invalid values: %w", InvalidExprSecondError)
	}
	// Minute
	matches = getNumbersFunc(exprParts[1])
	if !isValidNumbers(matches, 0, 59) {
		return fmt.Errorf("minute contains invalid values: %w", InvalidExprMinuteError)
	}
	// Hour
	matches = getNumbersFunc(exprParts[2])
	if !isValidNumbers(matches, 0, 23) {
		return fmt.Errorf("hour contains invalid values: %w", InvalidExprHourError)
	}
	// Day of month
	matches = getNumbersFunc(exprParts[3])
	if !isValidNumbers(matches, 1, 31) {
		return fmt.Errorf("DOM contains invalid values: %w", InvalidExprDayOfMonthError)
	}
	if invalidCharsDOWDOMRegex.MatchString(exprParts[3]) {
		return fmt.Errorf("DOM contains invalid values: %w", InvalidExprDayOfMonthError)
	}
	// Month
	matches = getNumbersFunc(exprParts[4])
	if !isValidNumbers(matches, 1, 12) {
		return fmt.Errorf("month contains invalid values: %w", InvalidExprMonthError)
	}
	// Day of week
	matches = getNumbersFunc(exprParts[5])
	if !isValidNumbers(matches, 0, 6) {
		return fmt.Errorf("DOW contains invalid values: %w", InvalidExprDayOfWeekError)
	}
	if invalidCharsDOWDOMRegex.MatchString(exprParts[5]) { // DOW
		return fmt.Errorf("DOW contains invalid values: %w", InvalidExprDayOfWeekError)
	}

	return nil
}

// isValidNumbers checks if all the numbers in the list is in range (lowerBound, upperBound).
func isValidNumbers(matches []string, lowerBound, upperBound int) bool {
	for _, m := range matches {
		num, err := strconv.Atoi(m)
		if err != nil {
			return false
		}
		if num < lowerBound || num > upperBound {
			return false
		}
	}
	return true
}
