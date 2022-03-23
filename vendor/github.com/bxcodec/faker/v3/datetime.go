package faker

import (
	"fmt"
	"reflect"
	"time"
)

var century = []string{"I", "II", "III", "IV", "V", "VI", "VII", "VIII", "IX", "X", "XI", "XII", "XIII", "XIV", "XV", "XVI", "XVII", "XVIII", "XIX", "XX", "XXI"}
var timezones = []string{
	"Australia/Adelaide",
	"Australia/Broken_Hill",
	"Australia/South",
	"Australia/Yancowinna",
	"Australia/Darwin",
	"Australia/North",
	"America/Eirunepe",
	"America/Porto_Acre",
	"America/Rio_Branco",
	"Brazil/Acre",
	"Australia/Eucla",
	"Australia/ACT",
	"Australia/Canberra",
	"Australia/Currie",
	"Australia/Hobart",
	"Australia/Melbourne",
	"Australia/NSW",
	"Australia/Sydney",
	"Australia/Tasmania",
	"Australia/Victoria",
	"Australia/Brisbane",
	"Australia/Lindeman",
	"Australia/Queensland",
	"Asia/Kabul",
	"America/Anchorage",
	"America/Juneau",
	"America/Nome",
	"America/Sitka",
	"America/Yakutat",
	"US/Alaska",
	"Asia/Almaty",
	"America/Boa_Vista",
	"America/Campo_Grande",
	"America/Cuiaba",
	"America/Manaus",
	"America/Porto_Velho",
	"Asia/Yerevan",
	"Brazil/West",
	"Asia/Anadyr",
	"Asia/Aqtau",
	"Asia/Aqtobe",
	"America/Argentina/Buenos_Aires",
	"America/Argentina/Catamarca",
	"America/Argentina/ComodRivadavia",
	"America/Argentina/Cordoba",
	"America/Argentina/Jujuy",
	"America/Argentina/La_Rioja",
	"America/Argentina/Mendoza",
	"America/Argentina/Rio_Gallegos",
	"America/Argentina/Salta",
	"America/Argentina/San_Juan",
	"America/Argentina/San_Luis",
	"America/Argentina/Tucuman",
	"America/Argentina/Ushuaia",
	"America/Buenos_Aires",
	"America/Catamarca",
	"America/Cordoba",
	"America/Jujuy",
	"America/Mendoza",
	"America/Rosario",
	"America/Anguilla",
	"America/Antigua",
	"America/Aruba",
	"America/Barbados",
	"America/Blanc-Sablon",
	"America/Curacao",
	"America/Dominica",
	"America/Glace_Bay",
	"America/Goose_Bay",
	"America/Grand_Turk",
	"America/Grenada",
	"America/Guadeloupe",
	"America/Halifax",
	"America/Kralendijk",
	"America/Lower_Princes",
	"America/Marigot",
	"America/Martinique",
	"America/Moncton",
	"America/Montserrat",
	"America/Port_of_Spain",
	"America/Puerto_Rico",
	"America/Santo_Domingo",
	"America/St_Barthelemy",
	"America/St_Kitts",
	"America/St_Lucia",
	"America/St_Thomas",
	"America/St_Vincent",
	"America/Thule",
	"America/Tortola",
	"America/Virgin",
	"Asia/Aden",
	"Asia/Baghdad",
	"Asia/Bahrain",
	"Asia/Kuwait",
	"Asia/Qatar",
	"Asia/Riyadh",
	"Atlantic/Bermuda",
	"Canada/Atlantic",
	"Antarctica/Casey",
	"Australia/Perth",
	"Australia/West",
	"Atlantic/Azores",
	"Asia/Baku",
	"Asia/Dacca",
	"Asia/Dhaka",
	"Asia/Brunei",
	"America/La_Paz",
	"America/Araguaina",
	"America/Bahia",
	"America/Belem",
	"America/Fortaleza",
	"America/Maceio",
	"America/Recife",
	"America/Santarem",
	"America/Sao_Paulo",
	"Brazil/East",
	"Pacific/Bougainville",
	"Asia/Thimbu",
	"Asia/Thimphu",
	"Africa/Blantyre",
	"Africa/Bujumbura",
	"Africa/Gaborone",
	"Africa/Harare",
	"Africa/Kigali",
	"Africa/Lubumbashi",
	"Africa/Lusaka",
	"Africa/Maputo",
	"Indian/Cocos",
	"Africa/Algiers",
	"Africa/Ceuta",
	"Africa/Tunis",
	"Arctic/Longyearbyen",
	"Atlantic/Jan_Mayen",
	"CET",
	"Europe/Amsterdam",
	"Europe/Andorra",
	"Europe/Belgrade",
	"Europe/Berlin",
	"Europe/Bratislava",
	"Europe/Brussels",
	"Europe/Budapest",
	"Europe/Busingen",
	"Europe/Copenhagen",
	"Europe/Gibraltar",
	"Europe/Ljubljana",
	"Europe/Luxembourg",
	"Europe/Madrid",
	"Europe/Malta",
	"Europe/Monaco",
	"Europe/Oslo",
	"Europe/Paris",
	"Europe/Podgorica",
	"Europe/Prague",
	"Europe/Rome",
	"Europe/San_Marino",
	"Europe/Sarajevo",
	"Europe/Skopje",
	"Europe/Stockholm",
	"Europe/Tirane",
	"Europe/Vaduz",
	"Europe/Vatican",
	"Europe/Vienna",
	"Europe/Warsaw",
	"Europe/Zagreb",
	"Europe/Zurich",
	"Poland",
	"NZ-CHAT",
	"Pacific/Chatham",
	"Asia/Choibalsan",
	"Pacific/Chuuk",
	"Pacific/Truk",
	"Pacific/Yap",
	"Pacific/Rarotonga",
	"America/Santiago",
	"Antarctica/Palmer",
	"Chile/Continental",
	"America/Bogota",
	"America/Bahia_Banderas",
	"America/Belize",
	"America/Chicago",
	"America/Costa_Rica",
	"America/El_Salvador",
	"America/Guatemala",
	"America/Havana",
	"America/Indiana/Knox",
	"America/Indiana/Tell_City",
	"America/Knox_IN",
	"America/Managua",
	"America/Matamoros",
	"America/Menominee",
	"America/Merida",
	"America/Mexico_City",
	"America/Monterrey",
	"America/North_Dakota/Beulah",
	"America/North_Dakota/Center",
	"America/North_Dakota/New_Salem",
	"America/Rainy_River",
	"America/Rankin_Inlet",
	"America/Regina",
	"America/Resolute",
	"America/Swift_Current",
	"America/Tegucigalpa",
	"America/Winnipeg",
	"Asia/Chongqing",
	"Asia/Chungking",
	"Asia/Harbin",
	"Asia/Macao",
	"Asia/Macau",
	"Asia/Shanghai",
	"Asia/Taipei",
	"CST6CDT",
	"Canada/Central",
	"Canada/East-Saskatchewan",
	"Canada/Saskatchewan",
	"Cuba",
	"Mexico/General",
	"PRC",
	"ROC",
	"US/Central",
	"US/Indiana-Starke",
	"Atlantic/Cape_Verde",
	"Indian/Christmas",
	"Pacific/Guam",
	"Pacific/Saipan",
	"Antarctica/Davis",
	"Antarctica/DumontDUrville",
	"Chile/EasterIsland",
	"Pacific/Easter",
	"Africa/Asmara",
	"Africa/Asmera",
	"Africa/Dar_es_Salaam",
	"Africa/Djibouti",
	"Africa/Juba",
	"Africa/Kampala",
	"Africa/Khartoum",
	"Africa/Mogadishu",
	"Africa/Nairobi",
	"Indian/Antananarivo",
	"Indian/Comoro",
	"Indian/Mayotte",
	"America/Guayaquil",
	"Africa/Cairo",
	"Africa/Tripoli",
	"Asia/Amman",
	"Asia/Beirut",
	"Asia/Damascus",
	"Asia/Gaza",
	"Asia/Hebron",
	"Asia/Istanbul",
	"Asia/Nicosia",
	"EET",
	"Egypt",
	"Europe/Athens",
	"Europe/Bucharest",
	"Europe/Chisinau",
	"Europe/Helsinki",
	"Europe/Istanbul",
	"Europe/Kaliningrad",
	"Europe/Kiev",
	"Europe/Mariehamn",
	"Europe/Nicosia",
	"Europe/Riga",
	"Europe/Sofia",
	"Europe/Tallinn",
	"Europe/Tiraspol",
	"Europe/Uzhgorod",
	"Europe/Vilnius",
	"Europe/Zaporozhye",
	"Libya",
	"Turkey",
	"America/Scoresbysund",
	"America/Atikokan",
	"America/Cancun",
	"America/Cayman",
	"America/Coral_Harbour",
	"America/Detroit",
	"America/Fort_Wayne",
	"America/Indiana/Indianapolis",
	"America/Indiana/Marengo",
	"America/Indiana/Petersburg",
	"America/Indiana/Vevay",
	"America/Indiana/Vincennes",
	"America/Indiana/Winamac",
	"America/Indianapolis",
	"America/Iqaluit",
	"America/Jamaica",
	"America/Kentucky/Louisville",
	"America/Kentucky/Monticello",
	"America/Louisville",
	"America/Montreal",
	"America/Nassau",
	"America/New_York",
	"America/Nipigon",
	"America/Panama",
	"America/Pangnirtung",
	"America/Port-au-Prince",
	"America/Thunder_Bay",
	"America/Toronto",
	"Canada/Eastern",
	"EST",
	"EST5EDT",
	"Jamaica",
	"US/East-Indiana",
	"US/Eastern",
	"US/Michigan",
	"Pacific/Fiji",
	"Atlantic/Stanley",
	"America/Noronha",
	"Brazil/DeNoronha",
	"Pacific/Galapagos",
	"Pacific/Gambier",
	"Asia/Tbilisi",
	"America/Cayenne",
	"Pacific/Tarawa",
	"Africa/Abidjan",
	"Africa/Accra",
	"Africa/Bamako",
	"Africa/Banjul",
	"Africa/Bissau",
	"Africa/Conakry",
	"Africa/Dakar",
	"Africa/Freetown",
	"Africa/Lome",
	"Africa/Monrovia",
	"Africa/Nouakchott",
	"Africa/Ouagadougou",
	"Africa/Sao_Tome",
	"Africa/Timbuktu",
	"America/Danmarkshavn",
	"Atlantic/Reykjavik",
	"Atlantic/St_Helena",
	"Eire",
	"Etc/GMT",
	"Etc/GMT+0",
	"Etc/GMT-0",
	"Etc/GMT0",
	"Etc/Greenwich",
	"Europe/Belfast",
	"Europe/Dublin",
	"Europe/Guernsey",
	"Europe/Isle_of_Man",
	"Europe/Jersey",
	"Europe/London",
	"GB",
	"GB-Eire",
	"GMT",
	"GMT+0",
	"GMT-0",
	"GMT0",
	"Greenwich",
	"Iceland",
	"Etc/GMT+1",
	"Etc/GMT+10",
	"Etc/GMT+11",
	"Etc/GMT+12",
	"Etc/GMT+2",
	"Etc/GMT+3",
	"Etc/GMT+4",
	"Etc/GMT+5",
	"Etc/GMT+6",
	"Etc/GMT+7",
	"Etc/GMT+8",
	"Etc/GMT+9",
	"Etc/GMT-1",
	"Etc/GMT-10",
	"Etc/GMT-11",
	"Etc/GMT-12",
	"Etc/GMT-13",
	"Etc/GMT-14",
	"Etc/GMT-2",
	"Etc/GMT-3",
	"Etc/GMT-4",
	"Etc/GMT-5",
	"Etc/GMT-6",
	"Etc/GMT-7",
	"Etc/GMT-8",
	"Etc/GMT-9",
	"Asia/Dubai",
	"Asia/Muscat",
	"Atlantic/South_Georgia",
	"America/Guyana",
	"Asia/Hong_Kong",
	"Hongkong",
	"Asia/Hovd",
	"America/Adak",
	"America/Atka",
	"HST",
	"Pacific/Honolulu",
	"Pacific/Johnston",
	"US/Aleutian",
	"US/Hawaii",
	"Asia/Bangkok",
	"Asia/Ho_Chi_Minh",
	"Asia/Phnom_Penh",
	"Asia/Saigon",
	"Asia/Vientiane",
	"Indian/Chagos",
	"Asia/Chita",
	"Asia/Irkutsk",
	"Asia/Tehran",
	"Iran",
	"Asia/Calcutta",
	"Asia/Colombo",
	"Asia/Jerusalem",
	"Asia/Kolkata",
	"Asia/Tel_Aviv",
	"Israel",
	"Asia/Tokyo",
	"Japan",
	"Asia/Bishkek",
	"Pacific/Kosrae",
	"Asia/Krasnoyarsk",
	"Asia/Novokuznetsk",
	"Asia/Pyongyang",
	"Asia/Seoul",
	"ROK",
	"Australia/LHI",
	"Australia/Lord_Howe",
	"Pacific/Kiritimati",
	"Asia/Magadan",
	"Pacific/Marquesas",
	"Antarctica/Mawson",
	"MET",
	"Kwajalein",
	"Pacific/Kwajalein",
	"Pacific/Majuro",
	"Antarctica/Macquarie",
	"Asia/Rangoon",
	"Europe/Minsk",
	"Europe/Moscow",
	"Europe/Simferopol",
	"Europe/Volgograd",
	"W-SU",
	"America/Boise",
	"America/Cambridge_Bay",
	"America/Chihuahua",
	"America/Creston",
	"America/Dawson_Creek",
	"America/Denver",
	"America/Edmonton",
	"America/Fort_Nelson",
	"America/Hermosillo",
	"America/Inuvik",
	"America/Mazatlan",
	"America/Ojinaga",
	"America/Phoenix",
	"America/Shiprock",
	"America/Yellowknife",
	"Canada/Mountain",
	"MST",
	"MST7MDT",
	"Mexico/BajaSur",
	"Navajo",
	"US/Arizona",
	"US/Mountain",
	"Indian/Mauritius",
	"Indian/Maldives",
	"Asia/Kuala_Lumpur",
	"Asia/Kuching",
	"Pacific/Noumea",
	"Pacific/Norfolk",
	"Asia/Novosibirsk",
	"Asia/Kathmandu",
	"Asia/Katmandu",
	"Pacific/Nauru",
	"America/St_Johns",
	"Canada/Newfoundland",
	"Pacific/Niue",
	"Antarctica/McMurdo",
	"Antarctica/South_Pole",
	"NZ",
	"Pacific/Auckland",
	"Asia/Omsk",
	"Asia/Oral",
	"America/Lima",
	"Asia/Kamchatka",
	"Pacific/Port_Moresby",
	"Pacific/Enderbury",
	"Asia/Manila",
	"Asia/Karachi",
	"America/Miquelon",
	"Pacific/Pohnpei",
	"Pacific/Ponape",
	"America/Dawson",
	"America/Ensenada",
	"America/Los_Angeles",
	"America/Metlakatla",
	"America/Santa_Isabel",
	"America/Tijuana",
	"America/Vancouver",
	"America/Whitehorse",
	"Canada/Pacific",
	"Canada/Yukon",
	"Mexico/BajaNorte",
	"PST8PDT",
	"Pacific/Pitcairn",
	"US/Pacific",
	"US/Pacific-New",
	"Pacific/Palau",
	"America/Asuncion",
	"Asia/Qyzylorda",
	"Indian/Reunion",
	"Antarctica/Rothera",
	"Asia/Sakhalin",
	"Europe/Samara",
	"Africa/Johannesburg",
	"Africa/Maseru",
	"Africa/Mbabane",
	"Pacific/Guadalcanal",
	"Indian/Mahe",
	"Asia/Singapore",
	"Singapore",
	"Asia/Srednekolymsk",
	"America/Paramaribo",
	"Pacific/Midway",
	"Pacific/Pago_Pago",
	"Pacific/Samoa",
	"US/Samoa",
	"Antarctica/Syowa",
	"Pacific/Tahiti",
	"Indian/Kerguelen",
	"Asia/Dushanbe",
	"Pacific/Fakaofo",
	"Asia/Dili",
	"Asia/Ashgabat",
	"Asia/Ashkhabad",
	"Pacific/Tongatapu",
	"Pacific/Funafuti",
	"Etc/UCT",
	"UCT",
	"Asia/Ulaanbaatar",
	"Asia/Ulan_Bator",
	"Antarctica/Troll",
	"Etc/UTC",
	"Etc/Universal",
	"Etc/Zulu",
	"UTC",
	"Universal",
	"Zulu",
	"America/Montevideo",
	"Asia/Samarkand",
	"Asia/Tashkent",
	"America/Caracas",
	"Asia/Ust-Nera",
	"Asia/Vladivostok",
	"Antarctica/Vostok",
	"Pacific/Efate",
	"Pacific/Wake",
	"Africa/Windhoek",
	"Africa/Bangui",
	"Africa/Brazzaville",
	"Africa/Douala",
	"Africa/Kinshasa",
	"Africa/Lagos",
	"Africa/Libreville",
	"Africa/Luanda",
	"Africa/Malabo",
	"Africa/Ndjamena",
	"Africa/Niamey",
	"Africa/Porto-Novo",
	"Africa/Casablanca",
	"Africa/El_Aaiun",
	"Atlantic/Canary",
	"Atlantic/Faeroe",
	"Atlantic/Faroe",
	"Atlantic/Madeira",
	"Europe/Lisbon",
	"Portugal",
	"WET",
	"Pacific/Wallis",
	"America/Godthab",
	"Asia/Jakarta",
	"Asia/Pontianak",
	"Asia/Jayapura",
	"Asia/Makassar",
	"Asia/Ujung_Pandang",
	"Pacific/Apia",
	"Asia/Kashgar",
	"Asia/Urumqi",
	"Asia/Khandyga",
	"Asia/Yakutsk",
	"Asia/Yekaterinburg",
}

// These example values must use the reference time "Mon Jan 2 15:04:05 MST 2006"
// as described at https://gobyexample.com/time-formatting-parsing
const (
	BaseDateFormat   = "2006-01-02"
	TimeFormat       = "15:04:05"
	MonthFormat      = "January"
	YearFormat       = "2006"
	DayFormat        = "Monday"
	DayOfMonthFormat = "_2"
	TimePeriodFormat = "PM"
)

// A DateTimer contains random Time generators, returning time string in certain particular format
type DateTimer interface {
	UnixTime(v reflect.Value) (interface{}, error)
	Date(v reflect.Value) (interface{}, error)
	Time(v reflect.Value) (interface{}, error)
	MonthName(v reflect.Value) (interface{}, error)
	Year(v reflect.Value) (interface{}, error)
	DayOfWeek(v reflect.Value) (interface{}, error)
	DayOfMonth(v reflect.Value) (interface{}, error)
	Timestamp(v reflect.Value) (interface{}, error)
	Century(v reflect.Value) (interface{}, error)
	TimeZone(v reflect.Value) (interface{}, error)
	TimePeriod(v reflect.Value) (interface{}, error)
}

var date DateTimer

// GetDateTimer returns a new DateTimer interface of DateTime
func GetDateTimer() DateTimer {
	mu.Lock()
	defer mu.Unlock()

	if date == nil {
		date = &DateTime{}
	}
	return date
}

// SetDateTimer sets custom date time
func SetDateTimer(d DateTimer) {
	date = d
}

// DateTime struct
type DateTime struct {
}

func (d DateTime) unixtime() int64 {
	return RandomUnixTime()
}

// UnixTime get unix time
func (d DateTime) UnixTime(v reflect.Value) (interface{}, error) {
	kind := v.Kind()
	var val int64
	if kind == reflect.Int64 {
		val = d.unixtime()
	} else {
		val = 0
	}

	return val, nil
}

// UnixTime get unix time randomly
func UnixTime() int64 {
	return singleFakeData(UnixTimeTag, func() interface{} {
		datetime := DateTime{}
		return datetime.unixtime()
	}).(int64)
}

func (d DateTime) date() string {
	return time.Unix(RandomUnixTime(), 0).Format(BaseDateFormat)
}

// Date formats DateTime using example BaseDateFormat const
func (d DateTime) Date(v reflect.Value) (interface{}, error) {
	return d.date(), nil
}

// Date get fake date in string randomly
func Date() string {
	return singleFakeData(DATE, func() interface{} {
		datetime := DateTime{}
		return datetime.date()
	}).(string)
}

func (d DateTime) time() string {
	return time.Unix(RandomUnixTime(), 0).Format(TimeFormat)
}

// Time formats DateTime using example Time const
func (d DateTime) Time(v reflect.Value) (interface{}, error) {
	return d.time(), nil
}

// TimeString get time randomly in string format
func TimeString() string {
	return singleFakeData(TIME, func() interface{} {
		datetime := DateTime{}
		return datetime.time()
	}).(string)
}

func (d DateTime) monthName() string {
	return time.Unix(RandomUnixTime(), 0).Format(MonthFormat)
}

// MonthName formats DateTime using example Month const
func (d DateTime) MonthName(v reflect.Value) (interface{}, error) {
	return d.monthName(), nil
}

// MonthName get month name randomly in string format
func MonthName() string {
	return singleFakeData(MonthNameTag, func() interface{} {
		datetime := DateTime{}
		return datetime.monthName()
	}).(string)
}

func (d DateTime) year() string {
	return time.Unix(RandomUnixTime(), 0).Format(YearFormat)
}

// Year formats DateTime using example Year const
func (d DateTime) Year(v reflect.Value) (interface{}, error) {
	return d.year(), nil
}

// YearString get year randomly in string format
func YearString() string {
	return singleFakeData(YEAR, func() interface{} {
		datetime := DateTime{}
		return datetime.year()
	}).(string)
}

func (d DateTime) dayOfWeek() string {
	return time.Unix(RandomUnixTime(), 0).Format(DayFormat)
}

// DayOfWeek formats DateTime using example Day const
func (d DateTime) DayOfWeek(v reflect.Value) (interface{}, error) {
	return d.dayOfWeek(), nil
}

// DayOfWeek get day of week randomly in string format
func DayOfWeek() string {
	return singleFakeData(DayOfWeekTag, func() interface{} {
		datetime := DateTime{}
		return datetime.dayOfWeek()
	}).(string)
}

func (d DateTime) dayOfMonth() string {
	return time.Unix(RandomUnixTime(), 0).Format(DayOfMonthFormat)
}

// DayOfMonth formats DateTime using example DayOfMonth const
func (d DateTime) DayOfMonth(v reflect.Value) (interface{}, error) {
	return d.dayOfMonth(), nil
}

// DayOfMonth get month randomly in string format
func DayOfMonth() string {
	return singleFakeData(DayOfMonthTag, func() interface{} {
		datetime := DateTime{}
		return datetime.dayOfMonth()
	}).(string)
}

func (d DateTime) timestamp() string {
	return time.Unix(RandomUnixTime(), 0).Format(fmt.Sprintf("%s %s", BaseDateFormat, TimeFormat))
}

// Timestamp formats DateTime using example Timestamp const
func (d DateTime) Timestamp(v reflect.Value) (interface{}, error) {
	return d.timestamp(), nil
}

// Timestamp get timestamp randomly in string format: 2006-01-02 15:04:05
func Timestamp() string {
	return singleFakeData(TIMESTAMP, func() interface{} {
		datetime := DateTime{}
		return datetime.timestamp()
	}).(string)
}

func (d DateTime) century() string {
	return randomElementFromSliceString(century)
}

// Century returns a random century
func (d DateTime) Century(v reflect.Value) (interface{}, error) {
	return d.century(), nil
}

// Century get century randomly in string
func Century() string {
	return singleFakeData(CENTURY, func() interface{} {
		datetime := DateTime{}
		return datetime.century()
	}).(string)
}

func (d DateTime) timezone() string {
	return randomElementFromSliceString(timezones)
}

// TimeZone returns a random timezone
func (d DateTime) TimeZone(v reflect.Value) (interface{}, error) {
	return d.timezone(), nil
}

// Timezone get timezone randomly in string
func Timezone() string {
	return singleFakeData(TIMEZONE, func() interface{} {
		datetime := DateTime{}
		return datetime.timezone()
	}).(string)
}

func (d DateTime) period() string {
	return time.Unix(RandomUnixTime(), 0).Format(TimePeriodFormat)
}

// TimePeriod formats DateTime using example TimePeriod const
func (d DateTime) TimePeriod(v reflect.Value) (interface{}, error) {
	return d.period(), nil
}

// Timeperiod get timeperiod randomly in string (AM/PM)
func Timeperiod() string {
	return singleFakeData(TimePeriodTag, func() interface{} {
		datetime := DateTime{}
		return datetime.period()
	}).(string)
}

// RandomUnixTime is a helper function returning random Unix time
func RandomUnixTime() int64 {
	return rand.Int63n(time.Now().Unix())
}
