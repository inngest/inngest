[![API Documentation](https://godoc.org/github.com/pascaldekloe/name?status.svg)](https://godoc.org/github.com/pascaldekloe/name)
[![Build Status](https://travis-ci.org/pascaldekloe/name.svg?branch=master)](https://travis-ci.org/pascaldekloe/name)

## About

… a naming-convention library for the Go programming language.
The two categories are delimiter-separated and letter case-separated words.
Each of the formatting functions support both techniques for input, without
any context.

This is free and unencumbered software released into the
[public domain](http://creativecommons.org/publicdomain/zero/1.0).


### Inspiration

* `name.CamelCase("pascal case", true)` returns “PascalCase”
* `name.CamelCase("snake_to_camel AND CamelToCamel?", false)` returns “snakeToCamelANDCamelToCamel”
* `name.Delimit("* All Hype is aGoodThing (TM)", '-')` returns “all-hype-is-a-good-thing-TM”
* `name.DotSeparated("WebCrawler#socketTimeout")` returns “web.crawler.socket.timeout”


### Performance

The following results were measured with Go 1.15 on an Intel i5-7500.

```
name                                                            time/op
Cases/a2B/CamelCase-4                                           38.9ns ± 5%
Cases/a2B/snake_case-4                                          41.1ns ± 1%
Cases/foo-bar/CamelCase-4                                       58.0ns ± 6%
Cases/foo-bar/snake_case-4                                      67.0ns ± 1%
Cases/ProcessHelperFactoryConfig#defaultIDBuilder/CamelCase-4    272ns ± 6%
Cases/ProcessHelperFactoryConfig#defaultIDBuilder/snake_case-4   324ns ± 1%

name                                                            alloc/op
Cases/a2B/CamelCase-4                                            3.00B ± 0%
Cases/a2B/snake_case-4                                           4.00B ± 0%
Cases/foo-bar/CamelCase-4                                        8.00B ± 0%
Cases/foo-bar/snake_case-4                                       16.0B ± 0%
Cases/ProcessHelperFactoryConfig#defaultIDBuilder/CamelCase-4    48.0B ± 0%
Cases/ProcessHelperFactoryConfig#defaultIDBuilder/snake_case-4   64.0B ± 0%

name                                                            allocs/op
Cases/a2B/CamelCase-4                                             1.00 ± 0%
Cases/a2B/snake_case-4                                            1.00 ± 0%
Cases/foo-bar/CamelCase-4                                         1.00 ± 0%
Cases/foo-bar/snake_case-4                                        1.00 ± 0%
Cases/ProcessHelperFactoryConfig#defaultIDBuilder/CamelCase-4     1.00 ± 0%
Cases/ProcessHelperFactoryConfig#defaultIDBuilder/snake_case-4    1.00 ± 0%
```
