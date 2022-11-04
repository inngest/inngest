[![API Documentation](https://godoc.org/github.com/pascaldekloe/name?status.svg)](https://godoc.org/github.com/pascaldekloe/name)
[![Build Status](https://travis-ci.org/pascaldekloe/name.svg?branch=master)](https://travis-ci.org/pascaldekloe/name)

Naming convention library for the Go programming language.
The functions offer flexible parsing and strict formatting for label
techniques such as snake_case, Lisp-case, CamelCase and (Java) property keys.


This is free and unencumbered software released into the
[public domain](http://creativecommons.org/publicdomain/zero/1.0).


### Inspiration

* `name.CamelCase("pascal case", true)` returns *PascalCase*
* `name.CamelCase("snake_to_camel AND CamelToCamel?", false)` returns *snakeToCamelANDCamelToCamel*
* `name.Delimit("* All Hype is aGoodThing (TM)", '-')` returns *all-hype-is-a-good-thing-TM*
* `name.DotSeparated("WebCrawler#socketTimeout")` returns *web.crawler.socket.timeout*
