# Changelog

This project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

The structure and content of this file follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [1.24.1] - 2024-09-15
### Fixed
- Fixed reflection map key matches when keys are string derivitives.

## [1.24.0] - 2024-08-09
### Added
- Added the `jp.PathMatch` function that compares a normalized JSONPath with a target JSONPath.
- Added `jp.MatchHandler` a TokenHandler that can be used to
  build a path and data while processing a JSON document.
- Added `oj.Match` and `sen.Match` functions.

## [1.23.0] - 2024-07-07
### Added
- New script functions can now be added with `jp.RegisterUnaryFunction()` and `jp.RegisterBinaryFunction()`.

## [1.22.1] - 2024-06-23
### Added
- Added the missing support of Keyed and Indexed in jp.Modify.

## [1.22.0] - 2024-04-22
### Added
- Added support for C style comment /* */ in the SEN parser.
### Fixed
- Comments are the start of a SEN document now parses without error.

## [1.21.5] - 2024-04-11
### Added
- Makefiles

## [1.21.4] - 2024-02-29
### Fixed
- Fixed regexp parsing to allow regexp escape sequences.

## [1.21.3] - 2024-02-20
### Fixed
- Evaluation of a not group such as `!(@.x == 2)` is now correct.

## [1.21.2] - 2024-02-14
### Fixed
- Reworked the jp equation parser to eliminate some parsing issues.
- Fixed wildcards in filters so that if any value is true a match is
  returned. As an example, a path of `a[?(@.b[*].c == 2)].b[0]` might
  match multiple values of `c` in array `b`. If any of the values is 2
  then it is considered a match.

## [1.21.1] - 2024-02-02
### Fixed
- Fixed script parsing when padded with spaces.
- Fixed script parsing for negation without parenthesis.

## [1.21.0] - 2023-12-18
### Added
- Added the Expr function `BracketString` to force the use of bracket
  notation to for normalized paths as described by the draft IETF
  JSONPath document in section 2.7.
- Added `jp.Expr.Locate()` function that returns normalized paths for JSONPath expression.
### Fixed
- Unmarshal now supports arrays such as `[4]int`.

## [1.20.3] - 2023-11-09
### Added
- Added an option to jp.Walk to just callback on leaves making the function more useable.

## [1.20.2] - 2023-10-30
### Fixed
- A script of `@.x` is now read correctly as `@.x exists true`.

## [1.20.1] - 2023-10-20
### Fixed
- Calling jp.Set on a map or struct with a nil value no longer panics
  if nil is a valid value for the element being set.

## [1.20.0] - 2023-10-17
### Added
- Added ojg.Options.FloatFormat to allow float output format to be specified.
### Fixed
- A single quote character can now be escaped in strings when using SEN for parsing.
- Descent on a struct for get and first fixed.

## [1.19.4] - 2023-10-03
### Added
- Updated cmd/oj version and updated notes to reference the brew formula.

### Fixed
## [1.19.3] - 2023-09-11
### Fixed
- asm cond condition is now evaluated correctly.

## [1.19.2] - 2023-08-07
### Fixed
- The test tool (tt) package Panic() function should return the recovered panic and now does.

## [1.19.1] - 2023-07-05
### Fixed
- Pooled parsers now reset the number conversion method.

## [1.19.0] - 2023-07-05
### Added
- Added NumConvMethod to convert json.Number to either float64 or a
  string on parse and recompose.

## [1.18.7] - 2023-06-02
### Fixed
- Fixed alt.Diff to not skip non-matching map entries.

## [1.18.6] - 2023-05-24
### Added
- Thanks to @thiagodpf allowing JSONPath get on structs to use JSON annotation.

## [1.18.5] - 2023-04-20
### Fixed
- alt.Diff now handles slice indexes correctly.

## [1.18.4] - 2023-04-04
### Fixed
- JSONPath with a child selector containing `'` is not escaped properly.

## [1.18.3] - 2023-04-01
### Fixed
- A comma after a number at the top level now errors out as expected.

## [1.18.2] - 2023-03-29
### Fixed
- Strings in bracketed JSONPaths with escaped characters are now handled correctly.
### Added
- Added support for the Keyed and Indexed interface in the JSONPath (jp) evaluations.

## [1.18.1] - 2023-03-11
### Fixed
- Fixed "has" and "exists" issue where comparisons were broken with the introduction of `Nothing`.

## [1.18.0] - 2023-03-07
### Added
- Added support for root fragments in filters such as `$.data[?(@.id == $.key)]`.
- "exists" is now an alias for the "has" filter operation.
- Added length, count, match, and search functions.
- Added `Nothing` as a value for comparison to return values where nothing is found.
- Added support no parenthesis around a filter so `[?@.x == 3]` is now valid.
- `alt.String()` now converts `[]byte`.
### Fixed
- Fix order of union with when final elements are not an `[]any`.

## [1.17.5] - 2023-02-19
### Added
- Added alt.Filter, a variation on alt.Match.
- Added the OmitEmpty option to oj, sen, pretty, and alt packages.
- Added the -o option for omit nil and empty to the oj command.

## [1.17.4] - 2023-02-02
### Fixed
- Fixed (preserve) order of JSONPath wildcard reflect elements.

## [1.17.3] - 2023-01-27
### Fixed
- Fixed (preserve) order of JSONPath filtered elements.

## [1.17.2] - 2023-01-15
### Fixed
- Fixed big number parsing.

## [1.17.1] - 2023-01-09
### Fixed
- Fixed the descent fragment use in the Modify() functions of the jp package.

## [1.17.0] - 2023-01-05
### Added
- Modify() functions added to the jp package.
- Added the `has` operator to the jp package scripts.

## [1.16.0] - 2023-01-02
### Added
- Remove() functions added to the jp package.
- jp.Set() operations now allow a union as the last fragment in an expression.

## [1.15.0] - 2022-12-16
### Added
- Added `jp.Script.Inspect()` to be able to get the details of a script.
- The parser callback function now allows `func(any)` in addition to `func(any) bool`.

## [1.14.5] - 2022-10-12
### Fixed
- alt.Builder Pop fixed for nested objects.

## [1.14.4] - 2022-08-11
### Fixed
- Private members that match a JSON element no longer cause a panic.

## [1.14.3] - 2022-06-12
### Fixed
- Returned `[]byte` from oj.Marshal and pretty.Marshal now copy the
  internal buffer instead of just returing it.

## [1.14.2] - 2022-06-03
### Added
- Added SameType test tool.

## [1.14.1] - 2022-05-31
### Fixed
- Removed dependency on external packages.

## [1.14.0] - 2022-04-08
### Added
- Added the JSONPath filter operation `in`.
- Added the JSONPath filter operation `empty`.
- Added the JSONPath filter operation `=~` for regex.

## [1.13.1] - 2022-03-19
### Fixed
- Fixed a case where a un-terminated JSON did not return an error.

## [1.13.0] - 2022-03-05
### Added
- Added jp.Expr.Has() function.
- Added jp.Walk to walk data and provide a the path and value for each
  element.

## [1.12.14] - 2022-02-28
### Fixed
- `[]byte` are encoded according to the ojg.Options.

## [1.12.13] - 2022-02-23
### Fixed
- For JSONPath (jp) reflection Get returns `has` value correctly for zero field values.

## [1.12.12] - 2021-12-27
### Fixed
- JSONPath scripts (jp.Script or [?(@.foo == 123)]) is now thread safe.

## [1.12.11] - 2021-12-10
### Fixed
- Parser reuse was no resetting callback and channels. It does now.

## [1.12.10] - 2021-12-07
### Added
- Added a delete option to the oj application.

## [1.12.9] - 2021-10-31
### Fixed
- Stuttering extracted elements when using the `-x` options has been fixed.

## [1.12.8] - 2021-09-21
### Fixed
- Correct unicode character is now included in error messages.

## [1.12.7] - 2021-09-14
### Fixed
- Typo in maxEnd for 32 bit architecture fixed.
- json.Unmarshaler fields in a struct correctly unmarshal.

## [1.12.6] - 2021-09-12
### Fixed
- Due to limitation (a bug most likely) in the stardard math package
  math.MaxInt64 can not be used on 32 bit architectures. Changes were
  made to work around this limitation.

- Embedded (Anonymous) pointers types now encode correctly.

### Added
- Support for json.Unmarshaler interface added.

## [1.12.5] - 2021-08-17
# Changed
- Updated to use go 1.17.

## [1.12.4] - 2021-08-06
### Fixed
- Setting an element in an array that does not exist now creates the array is the Nth value is not negative.

## [1.12.3] - 2021-08-01
### Fixed
- Error message on failed recompose was fixed to display the correct error message.
- Marshal of a non-pointer that contains a json.Marshaller that is not a pointer no longer fails.

## [1.12.2] - 2021-07-28
### Fixed
- Structs with recursive lists no longer fail.

## [1.12.1] - 2021-07-23
### Fixed
- Applying filters to a non-simple list such as `[]*Sample` now supported as expected.

## [1.12.0] - 2021-07-03
### Added
- SEN format parsing now allows string to be delimited with the single quote character.
- SEN format parsing now allows strings to be concatenated with syntax like `["abc" + "def"]`.
- SEN format parsing now allows functions such as `ISODate("2021-06-28T10:11:12Z")` in SEN data.
### Changed
- When Pretty Align is true map members are now aligned.

## [1.11.1] - 2021-05-29
### Fixed
- Missing support for json.Marshaler and encoding.TextMarshaler added.

## [1.11.0] - 2021-05-23
### Fixed
- Struct with pointers to base types such as *float64 are fixed.
- Stack overflow when converting values to JSON which are a type alias
  of a builtin.
### Added
- Added `[]byte` converation option for decompose.
- Added MustXxx versions of multiple functions to allow a panic and recover code pattern.
### Changed
- oj.Unmarshal now emits float64 for all numbers instead of int64 for
  integers. The parse functions remain unchanged.

## [1.10.0] - 2021-04-22
### Fixed
- Multiple part json tags are now parsed correctly and the string
  options is supported in both decompose and compose.
### Added
- Tokenize callback parser added.

## [1.9.5] - 2021-04-04
### Fixed
- OmitNil now catches nil maps and slices more consistently.

## [1.9.4] - 2021-04-04
### Fixed
- Number parsing in the form of 2e-7 has been fixed.

## [1.9.3] - 2021-03-30
### Fixed
- Writer functions now decompose structs if possible instead of resorting to %v too quickly.

## [1.9.2] - 2021-03-24
### Fixed
- When parsing SEN format `\r` is now allowed in strings to support
  Windows line termination as it works in Linux and macOS.

## [1.9.1] - 2021-03-21
### Fixed
- oj.Unmarshal now supports the optional alt.Recomposer as documented.
- Recomposer handles time.Time recomposing like any other struct.
- Write writes time.Time to conform to other struct encoding.

## [1.9.0] - 2021-03-13
### Added
- The Recomposer is now more flexibly in regard to input types. It now
  allows json.Unmarshal() targets as well as the type create key
  approach.
- Added flag to alt.Options to determine whether embedded anonymous
  types whould be output as nested elements or flattened.
- Added oj.Unmarshal and sen.Unmarshal.

## [1.8.0] - 2021-03-05
### Added
- Added alignment option for pretty printing.
- Added alt.Diff() and alt.Compare().
- Added color option for encoded time.
- Add alt.Converter along with some built in converter for time and mongodb export maps.

## [1.7.1] - 2021-02-25
### Added
- Added HTMLUnsafe option to oj JSON writing to not encode &, <, and > to provide consistency
- Added HTMLSafe option to sen options to encode &, <, and > to provide consistency
### Fixed
- Fixed panic for `{"""":0}`. Now an error is returned.

## [1.7.0] - 2021-02-21
### Added
- Added support for a configuration file.
- Added ability to set colors when using the -c and -b option.
- Added ability to set HTML colors when using the -html option.

## [1.6.0] - 2021-02-19
### Added
- Added assembly plan package and cmd/oj option that allows assembling a new JSON from parsed data.
- Added sen.Parse() and sen.ParseReader() that use a new sen.DefaultParser
- Added the pretty package for prettier JSON layout.
- Added HTMLOptions for generating HTML color styled text.

## [1.5.0] - 2021-02-09
### Fixed
- Fixed reflection bug that occurred when a struct did not have the requested field.
### Added
- Added tab option for indentation.
### Changed
- Write operations now use panic and recovery internally for more
  robust error handling and for a very slight performance improvement.

## [1.4.1] - 2021-02-02
### Fixed
- The SEN parser and writer did not allow `\n` or `\t` in strings. It
  now does as would be expected from a friendly format.

## [1.4.0] - 2020-01-03
### Fixed
- JSONPath Slice end is now exclusive as called for in the Goessner description and the consensus.
- Nested array parsing bug fixed.

## [1.3.0] - 2020-10-28
### Added
- oj.Marshal added. The function fails if an un-encodeable value is encountered.
- UseTags option added for write and decompose Options.

## [1.2.1] - 2020-09-13
### Fixed
- Order is preserved when using JSONPath to follow wildcards, unions, and slices.

## [1.2.0] - 2020-07-20
### Added
- Parse Resuse option added to allow reusing maps on subsequent parses.
- In addition to callbacks, parsing multi-json documents can place elements on a `chan interface{}`.
### Changed
- A code refactoring resulting in a performance boost to Parsing and Validation.

## [1.1.4] - 2020-07-13
### Changed
- Validation speedup using a one switch statement and character maps.

## [1.1.3] - 2020-07-09
### Fixed
- Validator bug introduced in the speedup fixed.

## [1.1.2] - 2020-07-08
### Changed
- Performance improvement on validation and parsing.

## [1.1.1] - 2020-07-05
### Fixed
- Write bug that incorrectly wrote some UTF-8 sequences.

## [1.1.0] - 2020-07-04
### Added
- [Simple Encoding Notation](sen.md)
- Lazy input and out options to the `cmd/oj` command.

## [1.0.2] - 2020-07-01
### Added
- Filters will now iterate over Object members as well as Array members.

## [1.0.1] - 2020-06-23
### Added
- `cmd/oj` now correctly allows JSON as an argument in addition to reading from a file.

## [1.0.0] - 2020-06-22
### Added
- Initial release.
