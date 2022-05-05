package cron

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	specialChars = []rune{'/', '-', ',', '*'}

	weekdaysNumberRegex = regexp.MustCompile(`(\d{1,2}w)|(w\d{1,2})`)
	lastDayOffsetRegex  = regexp.MustCompile(`l-(\d{1,2})`)
)

type (
	// ExpressionDescriptor represents the CRON expression descriptor.
	ExpressionDescriptor struct {
		isVerbose          bool
		isDOWStartsAtOne   bool
		is24HourTimeFormat bool

		logger  Logger
		parser  Parser
		locales map[LocaleType]Locale
	}

	// Logger is the logging interface for expression descriptor.
	Logger interface {
		Printf(format string, v ...interface{})
	}

	// Option allows to configure expression descriptor.
	Option func(exprDesc *ExpressionDescriptor)
)

// NewDescriptor returns a new CRON expression descriptor based on the list of options.
// If no options provided, a default CRON expression descriptor will be returned instead.
// By default, English (Locale_en) will always be loaded.
func NewDescriptor(options ...Option) (exprDesc *ExpressionDescriptor, err error) {
	exprDesc = &ExpressionDescriptor{}
	for _, option := range options {
		option(exprDesc)
	}

	// Init defaults
	if exprDesc.parser == nil {
		exprDesc.parser = &cronParser{
			isDOWStartsAtOne: exprDesc.isDOWStartsAtOne,
		}
	}

	// Always load EN locale so we can fallback to it
	if exprDesc.locales == nil {
		exprDesc.locales = make(map[LocaleType]Locale)
	}
	if _, ok := exprDesc.locales[Locale_en]; !ok {
		localeLoader, err := NewLocaleLoaders(Locale_en)
		if err != nil {
			return nil, fmt.Errorf("failed to init default locale EN: %w", err)
		}
		exprDesc.locales[Locale_en] = localeLoader[0]
	}

	return exprDesc, nil
}

// ToDescription converts the CRON expression to the human readable string in specified locale.
// If the specified locale had not been loaded by the CRON expression descriptor, the result will be
// returned in English (Locale_en) by default.
//
// To configure supported locales of the CRON expression descriptor, please see the SetLocales() option.
func (e *ExpressionDescriptor) ToDescription(expr string, loc LocaleType) (desc string, err error) {
	var exprParts []string
	if exprParts, err = e.parser.Parse(expr); err != nil {
		return "", fmt.Errorf("failed to parse CRON expression: %w", err)
	}

	locale := e.getLocale(loc)

	var timeSegment = e.getTimeOfDayDescription(exprParts, locale)
	var dayOfMonthDesc = e.getDayOfMonthDescription(exprParts, locale)
	var monthDesc = e.getMonthDescription(exprParts, locale)
	var dayOfWeekDesc = e.getDayOfWeekDescription(exprParts, locale)
	var yearDesc = e.getYearDescription(exprParts, locale)

	desc = timeSegment + dayOfMonthDesc + dayOfWeekDesc + monthDesc + yearDesc
	desc = transformVerbosity(desc, locale, e.isVerbose)
	desc = strings.Join(strings.Fields(desc), " ")
	desc = strings.Replace(desc, " ,", ",", -1)
	runes := []rune(desc)
	runes[0] = []rune(strings.ToUpper(string(runes[0])))[0]

	return string(runes), nil
}

func (e *ExpressionDescriptor) log(format string, v ...interface{}) {
	if e.logger == nil {
		return
	}
	e.logger.Printf(format, v...)
}

func (e *ExpressionDescriptor) verbose(format string, v ...interface{}) {
	if !e.isVerbose || e.logger == nil {
		return
	}
	e.logger.Printf(format, v...)
}

func (e *ExpressionDescriptor) getTimeOfDayDescription(exprParts []string, locale Locale) string {
	second := exprParts[0]
	minute := exprParts[1]
	hour := exprParts[2]
	var desc string

	if !containsAny(second, specialChars) && !containsAny(minute, specialChars) && !containsAny(hour, specialChars) {
		// specific time of day (i.e. 10:14:00)
		desc += locale.GetString(atSpace) + formatTime(hour, minute, second, locale, e.is24HourTimeFormat)
	} else if second == "" &&
		strings.Index(minute, "-") > -1 &&
		!(strings.Index(minute, ",") > -1) &&
		!(strings.Index(minute, "/") > -1) &&
		!containsAny(hour, specialChars) {
		// minute range in single hour (i.e. 0-10 11)
		idx := strings.Index(minute, "-")
		desc += sprintf(locale.GetString(everyMinuteBetweenX0AndX1),
			formatTime(hour, minute[:idx], "", locale, e.is24HourTimeFormat),
			formatTime(hour, minute[idx+1:], "", locale, e.is24HourTimeFormat))
	} else if second == "" &&
		strings.Index(hour, ",") > -1 &&
		strings.Index(hour, "-") == -1 &&
		strings.Index(hour, "/") == -1 &&
		!containsAny(minute, specialChars) {
		// hours list with single minute (i.e. 30 6,14,16)
		hourParts := strings.Split(hour, ",")
		desc += locale.GetString(at)
		for i, p := range hourParts {
			desc += " "
			desc += formatTime(p, minute, "", locale, e.is24HourTimeFormat)
			if i < len(hourParts)-2 {
				desc += ", "
			}
			if i == len(hourParts)-2 {
				desc += locale.GetString(spaceAnd)
			}
		}
	} else {
		// default time description
		secondDesc := e.getSecondsDescription(exprParts, locale)
		minuteDesc := e.getMinutesDescription(exprParts, locale)
		hourDesc := e.getHoursDescription(exprParts, locale)

		desc += secondDesc
		if desc != "" && minuteDesc != "" {
			desc += ", "
		}
		desc += minuteDesc

		if desc != "" && hourDesc != "" {
			desc += ", "
		}
		desc += hourDesc
	}

	return desc
}

func (e *ExpressionDescriptor) getSecondsDescription(exprParts []string, locale Locale) string {
	desc := getSegmentDescription(
		exprParts[0],
		locale.GetString(everySecond),
		func(s string) string {
			return s
		},
		func(s string) string {
			return sprintf(locale.GetString(everyX0Seconds), s)
		},
		func(s string) string {
			return locale.GetString(secondsX0ThroughX1PastTheMinute)
		},
		func(s string) string {
			if s == "" {
				return ""
			}
			sInt, _ := strconv.Atoi(s)
			if sInt < 20 {
				return locale.GetString(atX0SecondsPastTheMinute)
			}
			if msg := locale.GetString(atX0SecondsPastTheMinuteGt20); msg != "" {
				return msg
			}
			return locale.GetString(atX0SecondsPastTheMinute)
		},
		locale,
	)

	return desc
}

func (e *ExpressionDescriptor) getDayOfMonthDescription(exprParts []string, locale Locale) string {
	desc := ""
	dom := exprParts[3]

	switch dom {
	case "l":
		desc = locale.GetString(commaOnTheLastDayOfTheMonth)
	case "wl":
		fallthrough
	case "lw":
		desc = locale.GetString(commaOnTheLastWeekdayOfTheMonth)
	default:
		weekdaysNumberMatches := weekdaysNumberRegex.FindAllString(dom, -1)
		if len(weekdaysNumberMatches) > 0 {
			dayNumber, _ := strconv.Atoi(strings.Replace(weekdaysNumberMatches[0], "w", "", -1))
			dayStr := ""
			if dayNumber == 1 {
				dayStr = locale.GetString(firstWeekday)
			} else {
				dayStr = sprintf(locale.GetString(weekdayNearestDayX0), strconv.Itoa(dayNumber))
			}
			desc = sprintf(locale.GetString(commaOnTheX0OfTheMonth), dayStr)
			break
		}

		// Handle "last day offset" (i.e. L-5:  "5 days before the last day of the month")
		lastDayOffsetMatches := lastDayOffsetRegex.FindAllStringSubmatch(dom, -1)
		if len(lastDayOffsetMatches) > 0 {
			desc = sprintf(locale.GetString(commaDaysBeforeTheLastDayOfTheMonth), lastDayOffsetMatches[0][1])
			break
		}
		// * dayOfMonth and dayOfWeek specified so use dayOfWeek verbiage instead
		if dom == "*" && exprParts[5] != "*" {
			return ""
		}
		desc = getSegmentDescription(
			dom,
			locale.GetString(commaEveryDay),
			func(s string) string {
				if s == "l" {
					return locale.GetString(lastDay)
				}
				if msg := locale.GetString(dayX0); msg != "" {
					return sprintf(msg, s)
				}
				return s
			},
			func(s string) string {
				if s == "1" {
					return locale.GetString(commaEveryDay)
				}
				return locale.GetString(commaEveryX0Days)
			},
			func(s string) string {
				return locale.GetString(commaBetweenDayX0AndX1OfTheMonth)
			},
			func(s string) string {
				return locale.GetString(commaOnDayX0OfTheMonth)
			},
			locale,
		)
		break
	}

	return desc
}

func (e *ExpressionDescriptor) getMonthDescription(exprParts []string, locale Locale) string {
	monthNames := locale.GetSlice(monthsOfTheYear)

	desc := getSegmentDescription(
		exprParts[4],
		"",
		func(s string) string {
			sInt, _ := strconv.Atoi(s)
			return monthNames[sInt-1]
		},
		func(s string) string {
			sInt, _ := strconv.Atoi(s)
			if sInt == 1 {
				return "" // rather than "every 1 months" just return empty string
			}
			return sprintf(locale.GetString(commaEveryX0Months), s)

		},
		func(s string) string {
			if msg := locale.GetString(commaMonthX0ThroughMonthX1); msg != "" {
				return msg
			}
			return locale.GetString(commaX0ThroughX1)
		},
		func(s string) string {
			if msg := locale.GetString(commaOnlyInMonthX0); msg != "" {
				return msg
			}
			return locale.GetString(commaOnlyInX0)
		},
		locale,
	)

	return desc
}

func (e *ExpressionDescriptor) getDayOfWeekDescription(exprParts []string, locale Locale) string {
	daysOfWeekNames := locale.GetSlice(daysOfTheWeek)

	if exprParts[5] == "*" {
		// DOW is specified as * so we will not generate a description and defer to DOM part.
		// Otherwise, we could get a contradiction like "on day 1 of the month, every day"
		// or a dupe description like "every day, every day".
		return ""
	}
	desc := getSegmentDescription(
		exprParts[5],
		locale.GetString(commaEveryDay),
		func(s string) string {
			exp := s
			if idx := strings.Index(s, "#"); idx > -1 {
				exp = s[:idx]
			} else if strings.Index(s, "l") > -1 {
				exp = strings.Replace(exp, "l", "", -1)
			}
			expInt, _ := strconv.Atoi(exp)
			return daysOfWeekNames[expInt]
		},
		func(s string) string {
			sInt, _ := strconv.Atoi(s)
			if sInt == 1 {
				return "" // rather than "every 1 days" just return empty string
			}
			return sprintf(locale.GetString(commaEveryX0DaysOfTheWeek), s)
		},
		func(s string) string {
			return locale.GetString(commaX0ThroughX1)
		},
		func(s string) string {
			format := ""
			if idx := strings.Index(s, "#"); idx > -1 {
				dowOfMonthNum := s[idx+1:]
				dowOfMonthDesc := ""
				switch dowOfMonthNum {
				case "1":
					dowOfMonthDesc = locale.GetString(first)
				case "2":
					dowOfMonthDesc = locale.GetString(second)
				case "3":
					dowOfMonthDesc = locale.GetString(third)
				case "4":
					dowOfMonthDesc = locale.GetString(fourth)
				case "5":
					dowOfMonthDesc = locale.GetString(fifth)
				}
				format = locale.GetString(commaOnThe) + dowOfMonthDesc + locale.GetString(spaceX0OfTheMonth)
			} else if strings.Index(s, "l") > -1 {
				format = locale.GetString(commaOnTheLastX0OfTheMonth)
			} else {
				// If both DOM and DOW are specified, the cron will execute at either time.
				format = locale.GetString(commaAndOnX0)
				if exprParts[3] == "*" {
					format = locale.GetString(commaOnlyOnX0)
				}
			}

			return format
		},
		locale,
	)

	return desc
}

func (e *ExpressionDescriptor) getYearDescription(exprParts []string, locale Locale) string {
	desc := getSegmentDescription(
		exprParts[6],
		"",
		func(s string) string {
			return s // Note: Not handle the cases when year is not in full, e.g.: 93, 99
		},
		func(s string) string {
			return sprintf(locale.GetString(commaEveryX0Years), s)
		},
		func(s string) string {
			if msg := locale.GetString(commaYearX0ThroughYearX1); msg != "" {
				return msg
			}
			return locale.GetString(commaX0ThroughX1)
		},
		func(s string) string {
			if msg := locale.GetString(commaOnlyInYearX0); msg != "" {
				return msg
			}
			return locale.GetString(commaOnlyInX0)
		},
		locale,
	)

	return desc
}

func (e *ExpressionDescriptor) getLocale(loc LocaleType) Locale {
	v, ok := e.locales[loc]
	if !ok {
		return e.locales[Locale_en] // Fall back to default
	}
	return v
}

func containsAny(s string, matches []rune) bool {
	runes := []rune(s)
	for _, r := range runes {
		for _, c := range matches {
			if r == c {
				return true
			}
		}
	}

	return false
}

func formatTime(hour, minute, second string, locale Locale, isUse24HourTimeFormat bool) string {
	hourInt, _ := strconv.Atoi(hour)
	minuteInt, _ := strconv.Atoi(minute)
	period := ""
	isPeriodBeforeTime := false

	if !isUse24HourTimeFormat {
		isPeriodBeforeTime = locale.GetBool(confSetPeriodBeforeTime)
		period = getPeriod(hourInt, locale)
		if !isPeriodBeforeTime {
			period = " " + period
		}
		if hourInt > 12 {
			hourInt -= 12
		}
		if hourInt == 0 {
			hourInt = 12
		}
	}

	hour = fmt.Sprintf("%02d", hourInt)
	minute = fmt.Sprintf("%02d", minuteInt)
	ret := ""
	if isPeriodBeforeTime {
		ret += period + " "
	}
	ret += hour + ":" + minute
	if second != "" {
		secondInt, _ := strconv.Atoi(second)
		second = fmt.Sprintf("%02d", secondInt)
		ret += ":" + second
	}
	if !isPeriodBeforeTime {
		ret += period
	}
	return ret
}

func getPeriod(hour int, locale Locale) string {
	if hour >= 12 {
		period := locale.GetString(pm)
		if period == "" {
			return "PM"
		}
		return period
	}

	period := locale.GetString(am)
	if period == "" {
		return "AM"
	}
	return period
}

type getStringFunc func(string) string

func getSegmentDescription(expr, allDesc string,
	getSingleItemDescription,
	getIntervalDescriptionFormat,
	getBetweenDescriptionFormat,
	getDescriptionFormat getStringFunc,
	locale Locale) string {
	desc := ""
	if expr == "" {
		desc = ""
	} else if expr == "*" {
		desc = allDesc
	} else if !containsAny(expr, []rune{'/', '-', ','}) {
		desc = sprintf(getDescriptionFormat(expr), getSingleItemDescription(expr))
	} else if strings.Index(expr, "/") > -1 {
		segments := strings.Split(expr, "/")
		desc = sprintf(getIntervalDescriptionFormat(segments[1]), segments[1])

		// interval contains 'between' piece (i.e. 2-59/3 )
		if strings.Index(segments[0], "-") > -1 {
			betweenDesc := generateBetweenSegmentDescription(segments[0], getBetweenDescriptionFormat, getSingleItemDescription)
			if strings.Index(betweenDesc, ", ") != 0 {
				desc += ", "
			}
			desc += betweenDesc
		} else if !containsAny(segments[0], []rune{'*', ','}) {
			rangeDesc := sprintf(getDescriptionFormat(segments[0]), getSingleItemDescription(segments[0]))
			rangeDesc = strings.Replace(rangeDesc, ", ", "", 1)
			desc += sprintf(locale.GetString(commaStartingX0), rangeDesc)
		}
	} else if strings.Index(expr, ",") > -1 {
		segments := strings.Split(expr, ",")
		contentDesc := ""
		for i, seg := range segments {
			if i > 0 && len(segments) > 2 {
				contentDesc += ","
				if i < len(segments)-1 {
					contentDesc += " "
				}
			}

			if i > 0 && len(segments) > 1 && (i == len(segments)-1 || len(segments) == 2) {
				contentDesc += locale.GetString(spaceAnd) + " "
			}

			getBetweenFmtFunc := func(s string) string { return locale.GetString(commaX0ThroughX1) }
			if strings.Index(seg, "-") > -1 {
				betweenDesc := generateBetweenSegmentDescription(
					seg,
					getBetweenFmtFunc,
					getSingleItemDescription,
				)
				betweenDesc = strings.Replace(betweenDesc, ", ", "", 1)
				contentDesc += betweenDesc
			} else {
				contentDesc += getSingleItemDescription(seg)
			}
		}

		desc += sprintf(getDescriptionFormat(expr), contentDesc)
	} else if strings.Index(expr, "-") > -1 {
		desc = generateBetweenSegmentDescription(
			expr,
			getBetweenDescriptionFormat,
			getSingleItemDescription,
		)
	}

	return desc
}

func generateBetweenSegmentDescription(betweenDesc string, getBetweenDescriptionFormat, getSingleItemDescription getStringFunc) string {
	desc := ""
	betweenSegments := strings.Split(betweenDesc, "-")
	seg1 := getSingleItemDescription(betweenSegments[0])
	seg2 := getSingleItemDescription(betweenSegments[1])
	seg2 = strings.Replace(seg2, ":00", ":59", 1)
	desc += sprintf(getBetweenDescriptionFormat(betweenDesc), seg1, seg2)
	return desc
}

func (e *ExpressionDescriptor) getMinutesDescription(exprParts []string, locale Locale) string {
	second := exprParts[0]
	hour := exprParts[2]

	desc := getSegmentDescription(
		exprParts[1],
		locale.GetString(everyMinute),
		func(s string) string {
			return s
		},
		func(s string) string {
			return sprintf(locale.GetString(everyX0Minutes), s)
		},
		func(s string) string {
			return locale.GetString(minutesX0ThroughX1PastTheHour)
		},
		func(s string) string {
			if s == "0" && strings.Index(hour, "/") == -1 && second == "" {
				return locale.GetString(everyHour)
			}
			sInt, _ := strconv.Atoi(s)
			if sInt < 20 {
				return locale.GetString(atX0MinutesPastTheHour)
			}
			if msg := locale.GetString(atX0MinutesPastTheHourGt20); msg != "" {
				return msg
			}
			return locale.GetString(atX0MinutesPastTheHour)
		},
		locale)

	return desc
}

func (e *ExpressionDescriptor) getHoursDescription(exprParts []string, locale Locale) string {
	desc := getSegmentDescription(
		exprParts[2],
		locale.GetString(everyHour),
		func(s string) string {
			return formatTime(s, "0", "", locale, e.is24HourTimeFormat)
		},
		func(s string) string {
			return sprintf(locale.GetString(everyX0Hours), s)
		},
		func(s string) string {
			return locale.GetString(betweenX0AndX1)
		},
		func(s string) string {
			return locale.GetString(atX0)
		},
		locale,
	)

	return desc
}

func transformVerbosity(desc string, locale Locale, isVerbose bool) string {
	if isVerbose {
		return desc
	}

	eMinute := locale.GetString(everyMinute)
	eHour := locale.GetString(everyHour)
	eDay := locale.GetString(commaEveryDay)
	desc = strings.Replace(desc, ", "+eMinute, "", -1)
	desc = strings.Replace(desc, ", "+eHour, "", -1)
	desc = strings.Replace(desc, eDay, "", -1)
	return desc
}

func sprintf(tmpl string, values ...string) string {
	for _, v := range values {
		tmpl = strings.Replace(tmpl, "%s", v, 1)
	}
	return tmpl
}
