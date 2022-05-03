package cron

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lnquy/cron/i18n"
)

const (
	LocaleAll LocaleType = "all"

	Locale_cs    LocaleType = "cs"
	Locale_da    LocaleType = "da"
	Locale_de    LocaleType = "de"
	Locale_en    LocaleType = "en"
	Locale_es    LocaleType = "es"
	Locale_fa    LocaleType = "fa"
	Locale_fi    LocaleType = "fi"
	Locale_fr    LocaleType = "fr"
	Locale_he    LocaleType = "he"
	Locale_it    LocaleType = "it"
	Locale_ja    LocaleType = "ja"
	Locale_ko    LocaleType = "ko"
	Locale_nb    LocaleType = "nb"
	Locale_nl    LocaleType = "nl"
	Locale_pl    LocaleType = "pl"
	Locale_pt_BR LocaleType = "pt_BR"
	Locale_ro    LocaleType = "ro"
	Locale_ru    LocaleType = "ru"
	Locale_sk    LocaleType = "sk"
	Locale_sl    LocaleType = "sl"
	Locale_sv    LocaleType = "sv"
	Locale_sw    LocaleType = "sw"
	Locale_tr    LocaleType = "tr"
	Locale_uk    LocaleType = "uk"
	Locale_zh_CN LocaleType = "zh_CN"
	Locale_zh_TW LocaleType = "zh_TW"
)

var (
	allLocales = []LocaleType{
		Locale_cs,
		Locale_da,
		Locale_de,
		Locale_en,
		Locale_es,
		Locale_fa,
		Locale_fi,
		Locale_fr,
		Locale_he,
		Locale_it,
		Locale_ja,
		Locale_ko,
		Locale_nb,
		Locale_nl,
		Locale_pl,
		Locale_pt_BR,
		Locale_ro,
		Locale_ru,
		Locale_sk,
		Locale_sl,
		Locale_sv,
		Locale_sw,
		Locale_tr,
		Locale_uk,
		Locale_zh_CN,
		Locale_zh_TW,
	}
)

type (
	LocaleType string
	LocaleKey  string

	// Locale is the interface to a specific i18n.
	// The methods of Locale interface returns the corresponding i18n of a specific key.
	Locale interface {
		GetLocaleType() (typ LocaleType)
		GetBool(key LocaleKey) (value bool)
		GetString(key LocaleKey) (value string)
		GetSlice(key LocaleKey) (values []string)
	}

	// LocaleLoader holds the map of i18n strings in a specific language (localType).
	LocaleLoader struct {
		localeType LocaleType
		data       map[string]interface{}
	}
)

func NewLocaleLoaders(types ...LocaleType) (loaders []Locale, err error) {
	loaders = make([]Locale, 0)
	for _, typ := range types {
		got, err := newLocaleLoader(typ)
		if err != nil {
			return nil, fmt.Errorf("failed to init locale for %s", typ)
		}
		loaders = append(loaders, got...)
	}
	return loaders, nil
}

func newLocaleLoader(typ LocaleType) (loaders []Locale, err error) {
	var rawData string
	localeMap := make(map[string]interface{}, 60)
	switch typ {
	case Locale_cs:
		rawData = i18n.Locale_cs
	case Locale_da:
		rawData = i18n.Locale_da
	case Locale_de:
		rawData = i18n.Locale_de
	case Locale_en:
		rawData = i18n.Locale_en
	case Locale_es:
		rawData = i18n.Locale_es
	case Locale_fa:
		rawData = i18n.Locale_fa
	case Locale_fi:
		rawData = i18n.Locale_fi
	case Locale_fr:
		rawData = i18n.Locale_fr
	case Locale_he:
		rawData = i18n.Locale_he
	case Locale_it:
		rawData = i18n.Locale_it
	case Locale_ja:
		rawData = i18n.Locale_ja
	case Locale_ko:
		rawData = i18n.Locale_ko
	case Locale_nb:
		rawData = i18n.Locale_nb
	case Locale_nl:
		rawData = i18n.Locale_nl
	case Locale_pl:
		rawData = i18n.Locale_pl
	case Locale_pt_BR:
		rawData = i18n.Locale_pt_BR
	case Locale_ro:
		rawData = i18n.Locale_ro
	case Locale_ru:
		rawData = i18n.Locale_ru
	case Locale_sk:
		rawData = i18n.Locale_sk
	case Locale_sl:
		rawData = i18n.Locale_sl
	case Locale_sv:
		rawData = i18n.Locale_sv
	case Locale_sw:
		rawData = i18n.Locale_sw
	case Locale_tr:
		rawData = i18n.Locale_tr
	case Locale_uk:
		rawData = i18n.Locale_uk
	case Locale_zh_CN:
		rawData = i18n.Locale_zh_CN
	case Locale_zh_TW:
		rawData = i18n.Locale_zh_TW
	case LocaleAll:
		loaders = make([]Locale, 0, len(allLocales))
		for _, l := range allLocales {
			got, err := newLocaleLoader(l)
			if err != nil {
				return nil, fmt.Errorf("failed to init locale loader for %s: %w", l, err)
			}
			loaders = append(loaders, got...)
		}
		return loaders, nil
	default:
		return nil, fmt.Errorf("unsupported locale: %s", typ)
	}

	// Load a single locale
	if err = json.Unmarshal([]byte(rawData), &localeMap); err != nil {
		return nil, fmt.Errorf("failed to decode locale map, locale=%s: %w", typ, err)
	}

	// Handle slice data
	type sliceData struct {
		DaysOfTheWeek   []string `json:"daysOfTheWeek"`
		MonthsOfTheYear []string `json:"monthsOfTheYear"`
	}
	sld := sliceData{}
	if err = json.Unmarshal([]byte(rawData), &sld); err != nil {
		return nil, fmt.Errorf("failed to decode slice locale map, locale=%s: %w", typ, err)
	}
	localeMap[string(daysOfTheWeek)] = sld.DaysOfTheWeek
	localeMap[string(monthsOfTheYear)] = sld.MonthsOfTheYear

	loaders = []Locale{
		&LocaleLoader{localeType: typ, data: localeMap},
	}
	return loaders, nil
}

func (l *LocaleLoader) GetLocaleType() (typ LocaleType) {
	return l.localeType
}

func (l *LocaleLoader) GetBool(key LocaleKey) (value bool) {
	casted, ok := l.data[string(key)].(bool)
	if !ok {
		return false
	}
	return casted
}

func (l *LocaleLoader) GetString(key LocaleKey) (value string) {
	casted, ok := l.data[string(key)].(string)
	if !ok {
		return ""
	}
	return casted
}

func (l *LocaleLoader) GetSlice(key LocaleKey) (values []string) {
	casted, ok := l.data[string(key)].([]string)
	if !ok {
		return nil
	}
	return casted
}

var (
	// Config
	confSetPeriodBeforeTime LocaleKey = "confSetPeriodBeforeTime"

	// Keys
	everyMinute                         LocaleKey = "everyMinute"
	everyHour                           LocaleKey = "everyHour"
	atSpace                             LocaleKey = "atSpace"
	everyMinuteBetweenX0AndX1           LocaleKey = "everyMinuteBetweenX0AndX1"
	at                                  LocaleKey = "at"
	spaceAnd                            LocaleKey = "spaceAnd"
	everySecond                         LocaleKey = "everySecond"
	everyX0Seconds                      LocaleKey = "everyX0Seconds"
	secondsX0ThroughX1PastTheMinute     LocaleKey = "secondsX0ThroughX1PastTheMinute"
	atX0SecondsPastTheMinute            LocaleKey = "atX0SecondsPastTheMinute"
	everyX0Minutes                      LocaleKey = "everyX0Minutes"
	minutesX0ThroughX1PastTheHour       LocaleKey = "minutesX0ThroughX1PastTheHour"
	atX0MinutesPastTheHour              LocaleKey = "atX0MinutesPastTheHour"
	everyX0Hours                        LocaleKey = "everyX0Hours"
	betweenX0AndX1                      LocaleKey = "betweenX0AndX1"
	atX0                                LocaleKey = "atX0"
	commaEveryDay                       LocaleKey = "commaEveryDay"
	commaEveryX0DaysOfTheWeek           LocaleKey = "commaEveryX0DaysOfTheWeek"
	commaX0ThroughX1                    LocaleKey = "commaX0ThroughX1"
	first                               LocaleKey = "first"
	second                              LocaleKey = "second"
	third                               LocaleKey = "third"
	fourth                              LocaleKey = "fourth"
	fifth                               LocaleKey = "fifth"
	commaOnThe                          LocaleKey = "commaOnThe"
	spaceX0OfTheMonth                   LocaleKey = "spaceX0OfTheMonth"
	lastDay                             LocaleKey = "lastDay"
	commaOnTheLastX0OfTheMonth          LocaleKey = "commaOnTheLastX0OfTheMonth"
	commaOnlyOnX0                       LocaleKey = "commaOnlyOnX0"
	commaAndOnX0                        LocaleKey = "commaAndOnX0"
	commaEveryX0Months                  LocaleKey = "commaEveryX0Months"
	commaOnlyInX0                       LocaleKey = "commaOnlyInX0"
	commaOnTheLastDayOfTheMonth         LocaleKey = "commaOnTheLastDayOfTheMonth"
	commaOnTheLastWeekdayOfTheMonth     LocaleKey = "commaOnTheLastWeekdayOfTheMonth"
	commaDaysBeforeTheLastDayOfTheMonth LocaleKey = "commaDaysBeforeTheLastDayOfTheMonth"
	firstWeekday                        LocaleKey = "firstWeekday"
	weekdayNearestDayX0                 LocaleKey = "weekdayNearestDayX0"
	commaOnTheX0OfTheMonth              LocaleKey = "commaOnTheX0OfTheMonth"
	commaEveryX0Days                    LocaleKey = "commaEveryX0Days"
	commaBetweenDayX0AndX1OfTheMonth    LocaleKey = "commaBetweenDayX0AndX1OfTheMonth"
	commaOnDayX0OfTheMonth              LocaleKey = "commaOnDayX0OfTheMonth"
	commaEveryHour                      LocaleKey = "commaEveryHour" // Not used yet
	commaEveryX0Years                   LocaleKey = "commaEveryX0Years"
	commaStartingX0                     LocaleKey = "commaStartingX0"
	daysOfTheWeek                       LocaleKey = "daysOfTheWeek"
	atX0SecondsPastTheMinuteGt20        LocaleKey = "atX0SecondsPastTheMinuteGt20"
	atX0MinutesPastTheHourGt20          LocaleKey = "atX0MinutesPastTheHourGt20"
	commaMonthX0ThroughMonthX1          LocaleKey = "commaMonthX0ThroughMonthX1"
	commaOnlyInMonthX0                  LocaleKey = "commaOnlyInMonthX0"
	commaYearX0ThroughYearX1            LocaleKey = "commaYearX0ThroughYearX1"
	dayX0                               LocaleKey = "dayX0"
	monthsOfTheYear                     LocaleKey = "monthsOfTheYear"
	pm                                  LocaleKey = "pm"
	am                                  LocaleKey = "am"
	commaOnlyInYearX0                   LocaleKey = "commaOnlyInYearX0"
)

func ParseLocale(s string) (l LocaleType, err error) {
	switch strings.ToLower(s) {
	case "cs":
		return Locale_cs, nil
	case "da":
		return Locale_da, nil
	case "de":
		return Locale_de, nil
	case "en":
		return Locale_en, nil
	case "es":
		return Locale_es, nil
	case "fa":
		return Locale_fa, nil
	case "fi":
		return Locale_fi, nil
	case "fr":
		return Locale_fr, nil
	case "he":
		return Locale_he, nil
	case "it":
		return Locale_it, nil
	case "ja":
		return Locale_ja, nil
	case "ko":
		return Locale_ko, nil
	case "nb":
		return Locale_nb, nil
	case "nl":
		return Locale_nl, nil
	case "pl":
		return Locale_pl, nil
	case "pt", "pt_br":
		return Locale_pt_BR, nil
	case "ro":
		return Locale_ro, nil
	case "ru":
		return Locale_ru, nil
	case "sk":
		return Locale_sk, nil
	case "sl":
		return Locale_sl, nil
	case "sv":
		return Locale_sv, nil
	case "sw":
		return Locale_sw, nil
	case "tr":
		return Locale_tr, nil
	case "uk":
		return Locale_uk, nil
	case "zh", "zh_cn":
		return Locale_zh_CN, nil
	case "zh_tw":
		return Locale_zh_TW, nil
	default:
		return "", fmt.Errorf("unsupported locale: %s", s)
	}
}
