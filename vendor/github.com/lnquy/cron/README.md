# cron
<p align="left">
  <a href="https://godoc.org/github.com/lnquy/cron" title="GoDoc Reference" rel="nofollow"><img src="https://img.shields.io/badge/go-documentation-blue.svg?style=flat" alt="GoDoc Reference"></a>
  <a href="https://github.com/github.com/lnquy/cron/releases/tag/v1.0.0" title="1.0.0 Release" rel="nofollow"><img src="https://img.shields.io/badge/version-1.0.0-blue.svg?style=flat" alt="1.0.0 release"></a>
  <a href="https://goreportcard.com/report/github.com/lnquy/cron"><img src="https://goreportcard.com/badge/github.com/lnquy/cron" alt="Code Status" /></a>
  <a href="https://travis-ci.org/lnquy/cron"><img src="https://travis-ci.org/lnquy/cron.svg?branch=master" alt="Build Status" /></a>
  <a href='https://coveralls.io/github/lnquy/cron?branch=master'><img src='https://coveralls.io/repos/github/lnquy/cron/badge.svg?branch=master' alt='Coverage Status' /></a>
  <br />
</p>

cron is a Go library that parses a cron expression and outputs a human readable description of the cron schedule.  
For example, given the expression `*/5 * * * *` it will output `Every 5 minutes`.  

Translated to Go from [cron-expression-descriptor](https://github.com/bradymholt/cron-expression-descriptor) (C#) via [cRonstrue](https://github.com/bradymholt/cRonstrue) (Javascript).  
Original Author & Credit: Brady Holt (http://www.geekytidbits.com).

## Features
- Zero dependencies
- Supports all cron expression special characters including `* / , - ? L W #`
- Supports 5, 6 (w/ seconds or year), or 7 (w/ seconds and year) part cron expressions
- Supports [Quartz Job Scheduler](http://www.quartz-scheduler.org/) cron expressions
- i18n support with 26 locales.

## Installation
`cron` module can be used with both Go module (>= 1.11) and earlier Go versions.
```
go get -u -v github.com/lnquy/cron
```

## Usage

```go
// Init with default EN locale
exprDesc, _ := cron.NewDescriptor()

desc, _ := exprDesc.ToDescription("* * * * *", cron.Locale_en)
// "Every minute" 

desc, _ := exprDesc.ToDescription("0 23 ? * MON-FRI", cron.Locale_en)
// "At 11:00 PM, Monday through Friday" 

desc, _ := exprDesc.ToDescription("23 14 * * SUN#2", cron.Locale_en)
// "At 02:23 PM, on the second Sunday of the month"

// Init with custom configs
exprDesc, _ := cron.NewDescriptor(
    cron.Use24HourTimeFormat(true),
    cron.DayOfWeekStartsAtOne(true),
    cron.Verbose(true),
    cron.SetLogger(log.New(os.Stdout, "cron: ", 0)),
    cron.SetLocales(cron.Locale_en, cron.Locale_fr),
)
```

For more usage examples, including a demonstration of how cron can handle some very complex cron expressions, you can reference [the unit tests](https://github.com/lnquy/cron/blob/develop/locale_en_test.go) or [the example codes](https://github.com/lnquy/cron/tree/develop/examples).

## i18n

To use the i18n support, you must configure the locales when create a new `ExpressionDescriptor` via `SetLocales()` option.
```go
exprDesc, _ := cron.NewDescriptor(
    cron.SetLocales(cron.Locale_en, cron.Locale_es, cron.Locale_fr),
)
// or load all cron.LocaleAll
exprDesc, _ := cron.NewDescriptor(cron.SetLocales(cron.LocaleAll))

desc, _ := exprDesc.ToDescription("* * * * *", cron.Locale_fr)
// Toutes les minutes
```

By default, `ExpressionDescriptor` always load the `Locale_en`. If you pass an unregistered locale into `ToDescription()` function, the result will be returned in English.

### Supported Locales

| Locale Code | Language             | Contributors                                               |
| ----------- | -------------------- | ---------------------------------------------------------- |
|  cs         | Czech                | [hanbar](https://github.com/hanbar)                        |
|  da         | Danish               | [Rasmus Melchior Jacobsen](https://github.com/rmja)        |
|  de         | German               | [Michael Schuler](https://github.com/mschuler)             |
|  en         | English              | [Brady Holt](https://github.com/bradymholt)                |
|  es         | Spanish              | [Ivan Santos](https://github.com/ivansg)                   |
|  fa         | Farsi                | [A. Bahrami](https://github.com/alirezakoo)                |
|  fi         | Finnish              | [Mikael Rosenberg](https://github.com/MR77FI)              |
|  fr         | French               | [Arnaud TAMAILLON](https://github.com/Greybird)            |
|  he         | Hebrew               | [Ilan Firsov](https://github.com/IlanF)                    |
|  it         | Italian              | [rinaldihno](https://github.com/rinaldihno)                |
|  ja         | Japanese             | [Alin Sarivan](https://github.com/asarivan)                |
|  ko         | Korean               | [Ion Mincu](https://github.com/ionmincu)                   |
|  nb         | Norwegian            | [Siarhei Khalipski](https://github.com/KhalipskiSiarhei)   |
|  nl         | Dutch                | [TotalMace](https://github.com/TotalMace)                  |
|  pl         | Polish               | [foka](https://github.com/foka)                            |
|  pt_BR      | Portuguese (Brazil)  | [Renato Lima](https://github.com/natenho)                  |
|  ro         | Romanian             | [Illegitimis](https://github.com/illegitimis)              |
|  ru         | Russian              | [LbISS](https://github.com/LbISS)                          |
|  sk         | Slovakian            | [hanbar](https://github.com/hanbar)                        |
|  sl         | Slovenian            | [Jani Bevk](https://github.com/jenzy)                      |
|  sv         | Swedish              | [roobin](https://github.com/roobin)                        |
|  sw         | Swahili              | [Leylow Lujuo](https://github.com/leyluj)                  |
|  tr         | Turkish              | [Mustafa SADEDİL](https://github.com/sadedil)              |
|  uk         | Ukrainian            | [Taras](https://github.com/tbudurovych)                    |
|  zh_CN      | Chinese (Simplified) | [Star Peng](https://github.com/starpeng)                   |
|  zh_TW      | Chinese (Traditional)| [Ricky Chiang](https://github.com/metavige)                |



## hcron

`hcron` is the CLI tool to convert the CRON expression to human readable string.  
You can pass the CRON expressions as the program argument, piped `hcron` with stdin or given the path to crontab file.

### Install

You can find the pre-built binaries for Linux, MacOS, FreeBSD and Windows from the [Release](https://github.com/lnquy/cron/releases).  

For other OS or architecture, you can build the code using Go as below:

```shell
$ go get -u -v github.com/lnquy/cron/cmd/hcron

# or

$ git clone https://github.com/lnquy/cron
$ cd cron/cmd/hcron
$ go build
```

### Usage

```shell
$ hcron -h
hcron converts the CRON expression to human readable description.

Usage:
  hcron [flags] [cron expression]

Flags:
  -24-hour
        Output description in 24 hour time format
  -dow-starts-at-one
        Is day of the week starts at 1 (Monday-Sunday: 1-7)
  -file string
        Path to crontab file
  -h    Print help then exit
  -locale string
        Output description in which locale (default "en")
  -v    Print app version then exit
  -verbose
        Output description in verbose format

Examples:
  $ hcron "0 15 * * 1-5"
  $ hcron "0 */10 9 * * 1-5 2020"
  $ hcron -locale fr "0 */10 9 * * 1-5 2020"
  $ hcron -file /var/spool/cron/crontabs/mycronfile
  $ another-app | hcron 
  $ another-app | hcron --dow-starts-at-one --24-hour -locale es
```



## Project status

- [x] Port 1-1 code from cRonstrue Javascript
- [X] Port and pass all test cases from cRonstrue
- [X] i18n for 25 languages
- [X] Test cases i18n
- [x] Fix i18n issues of FA, HE, RO, RU, UK, ZH_CN and ZH_TW
- [x] hcron CLI tool
- [x] Performance improvement
- [x] Release v1.0.0

## License

This project is under the MIT License. See the [LICENSE](https://github.com/lnquy/cron/blob/master/LICENSE) file for the full license text.
