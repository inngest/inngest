Release Notes
---

## [0.87.0](https://github.com/substrait-io/substrait/compare/v0.86.0...v0.87.0) (2026-03-29)

### Features

* **extensions:** add std_dev and variance with distribution enum arg  ([#1011](https://github.com/substrait-io/substrait/issues/1011)) ([00bc3c2](https://github.com/substrait-io/substrait/commit/00bc3c24078443a23b3b4c92228fae8e2aba3778))
* introduce ExecutionContextVariable and ExecutionBehaviour [#945](https://github.com/substrait-io/substrait/issues/945)) ([6b98c27](https://github.com/substrait-io/substrait/commit/6b98c27f10c9cefb221cd50555877597e83407aa))
* **tests:** add enum argument support to FuncTestCase grammar ([#1010](https://github.com/substrait-io/substrait/issues/1010)) ([7b39d4c](https://github.com/substrait-io/substrait/commit/7b39d4ca8651e0ac0613efd9e72b36095dfbea99))

## [0.86.0](https://github.com/substrait-io/substrait/compare/v0.85.0...v0.86.0) (2026-03-22)

### ⚠ BREAKING CHANGES

* **extensions:** random extraneous argument for repeat varchar (#1015)
* **extensions:** corrects the return type of the `add:date_iyear`

### Features

* add optional description field to function implementations ([#1013](https://github.com/substrait-io/substrait/issues/1013)) ([0f4c774](https://github.com/substrait-io/substrait/commit/0f4c774f0bca5d6476cd36aaaf1390fbc323fc13))
* **extensions:** support deprecation info in extensions ([#1014](https://github.com/substrait-io/substrait/issues/1014)) ([25d87ef](https://github.com/substrait-io/substrait/commit/25d87ef4b9d8803bda2644eff48dfe966a6d7464))

### Bug Fixes

* enforce nullable types for null literals in test cases ([#989](https://github.com/substrait-io/substrait/issues/989)) ([ba9b0ff](https://github.com/substrait-io/substrait/commit/ba9b0ff98e1caf0b4f5c965a7bbc02f30021ea95))
* **extensions:** correct return type for `add:date_iyear` operation ([#1007](https://github.com/substrait-io/substrait/issues/1007)) ([62147cf](https://github.com/substrait-io/substrait/commit/62147cf26b4020ee8433f506809f410cc40d5806))
* **extensions:** random extraneous argument for repeat varchar ([#1015](https://github.com/substrait-io/substrait/issues/1015)) ([271d7bb](https://github.com/substrait-io/substrait/commit/271d7bbbae2dcdbdc125ff0d7e9637f318f70b59))

## [0.85.0](https://github.com/substrait-io/substrait/compare/v0.84.0...v0.85.0) (2026-03-06)

### ⚠ BREAKING CHANGES

* drops uri fields from protobufs

### Features

* only use urns for referencing simple extensions ([#971](https://github.com/substrait-io/substrait/issues/971)) ([ada68a8](https://github.com/substrait-io/substrait/commit/ada68a84698d269040a0d365c9e6c141fc3b2c29))

## [0.84.0](https://github.com/substrait-io/substrait/compare/v0.83.0...v0.84.0) (2026-03-06)

### Features

* add common list functions to core extensions ([#969](https://github.com/substrait-io/substrait/issues/969)) ([609cad7](https://github.com/substrait-io/substrait/commit/609cad7330ea08c851e1069ba70fb56ebca19008))
* add metadata fields to dialect schema ([#966](https://github.com/substrait-io/substrait/issues/966)) ([0fbf52e](https://github.com/substrait-io/substrait/commit/0fbf52e4623f72a7d85625a7f8e48b098447385a))
* **proto:** deprecate time type and literal ([#985](https://github.com/substrait-io/substrait/issues/985)) ([dc27654](https://github.com/substrait-io/substrait/commit/dc276549da20d92aa83269219ae267682d55dba4))
* support null list types and nested list literals in tests ([#991](https://github.com/substrait-io/substrait/issues/991)) ([e2dca29](https://github.com/substrait-io/substrait/commit/e2dca291fb9061a38bbd03574f11a85b56f96c28))

### Bug Fixes

* **grammar:** remove redundant import of SubstraitLexer in FuncTestCaseParser ([#982](https://github.com/substrait-io/substrait/issues/982)) ([9dd1f05](https://github.com/substrait-io/substrait/commit/9dd1f057392181208573edf2485a466c13a874d3))

## [0.83.0](https://github.com/substrait-io/substrait/compare/v0.82.0...v0.83.0) (2026-03-01)

### Features

* **dialect:** support specifying maximum supported subsecond precision ([#961](https://github.com/substrait-io/substrait/issues/961)) ([e559368](https://github.com/substrait-io/substrait/commit/e5593685d5c7036fe67542bf7b46e6a3ee5e4a1d))

### Bug Fixes

* **grammar:** add IntervalCompound to ANTLR grammar ([#963](https://github.com/substrait-io/substrait/issues/963)) ([2705258](https://github.com/substrait-io/substrait/commit/2705258137b7272830cf00680e6f3a36b8d3ed4b))

## [0.82.0](https://github.com/substrait-io/substrait/compare/v0.81.0...v0.82.0) (2026-02-22)

### ⚠ BREAKING CHANGES

* removes capabilities.proto and corresponding
documentation

Signed-off-by: Niels Pardon <par@zurich.ibm.com>

### Features

* remove capabilities.proto ([#952](https://github.com/substrait-io/substrait/issues/952)) ([23ac2f3](https://github.com/substrait-io/substrait/commit/23ac2f30e03a8d28f76e445bcd439064f3d688fd))

## [0.81.0](https://github.com/substrait-io/substrait/compare/v0.80.0...v0.81.0) (2026-02-08)

### ⚠ BREAKING CHANGES

* I would propose to remove these proto definitions if
they are not used / no longer part of the spec.

### Features

* remove unused function.proto, parameterized_types.proto, type_expressions.proto ([#940](https://github.com/substrait-io/substrait/issues/940)) ([788faab](https://github.com/substrait-io/substrait/commit/788faab7fdcc42742f5f2cb83294dd6966804109))

### Bug Fixes

* add missing visit methods in TestCaseVisitor ([#958](https://github.com/substrait-io/substrait/issues/958)) ([09e7c18](https://github.com/substrait-io/substrait/commit/09e7c180b32ebbd153a66e8fd78eec11d5aad386))

## [0.80.0](https://github.com/substrait-io/substrait/compare/v0.79.0...v0.80.0) (2026-01-18)

### Features

* add optional metadata to simple extension yaml files ([#922](https://github.com/substrait-io/substrait/issues/922)) ([0ef31ac](https://github.com/substrait-io/substrait/commit/0ef31ac4572b3e03fd8ad86346db451f08ce17cd))

## [0.79.0](https://github.com/substrait-io/substrait/compare/v0.78.2...v0.79.0) (2025-12-21)

### Features

* introduce lambda expressions ([#889](https://github.com/substrait-io/substrait/issues/889)) ([e47197c](https://github.com/substrait-io/substrait/commit/e47197c8de012ac6db4329ad7c68caeb746c5951))

### Bug Fixes

* remove redundant nullability markers in test case grammar ([#910](https://github.com/substrait-io/substrait/issues/910)) ([6ad9251](https://github.com/substrait-io/substrait/commit/6ad9251f27f4760936b9fb84edba8378be593ae4))

## [0.78.2](https://github.com/substrait-io/substrait/compare/v0.78.1...v0.78.2) (2025-12-14)

### Bug Fixes

* **extensions:** nullif output should always be nullable ([#913](https://github.com/substrait-io/substrait/issues/913)) ([7e0404b](https://github.com/substrait-io/substrait/commit/7e0404ba8540bcd910bb652d67baeab86a69ccab))

## [0.78.1](https://github.com/substrait-io/substrait/compare/v0.78.0...v0.78.1) (2025-11-23)

### Bug Fixes

* remove invalid equal function test ([#893](https://github.com/substrait-io/substrait/issues/893)) ([9533cfe](https://github.com/substrait-io/substrait/commit/9533cfe79449cb4504d934ccb76a2d6026123f43))

## [0.78.0](https://github.com/substrait-io/substrait/compare/v0.77.0...v0.78.0) (2025-11-16)

### ⚠ BREAKING CHANGES

* `type_reference` with value **0** is encoded
differently on wire. However, the usage of `type_reference` in generated
consumer code does not break in practice. **New consumer should consider
`type_reference` is set to zero when no oneof is set.** See [Wire

### Features

* support type alias in user defined literal ([#868](https://github.com/substrait-io/substrait/issues/868)) ([cd99f03](https://github.com/substrait-io/substrait/commit/cd99f033ec3cf8170d5e76affe31f3262f60d4e7)), closes [#Wire-Encoding-Change](https://github.com/substrait-io/substrait/issues/Wire-Encoding-Change) [#857](https://github.com/substrait-io/substrait/issues/857)

## [0.77.0](https://github.com/substrait-io/substrait/compare/v0.76.0...v0.77.0) (2025-10-05)

### Features

* per plan type aliases ([#857](https://github.com/substrait-io/substrait/issues/857)) ([dfd04c1](https://github.com/substrait-io/substrait/commit/dfd04c15caa12845f77db50688fdc16f59fdc9d0))

## [0.76.0](https://github.com/substrait-io/substrait/compare/v0.75.0...v0.76.0) (2025-09-21)

### Features

* codify Substrait dialects ([#816](https://github.com/substrait-io/substrait/issues/816)) ([b8090b2](https://github.com/substrait-io/substrait/commit/b8090b247eb8f9f05dcf46048f5e8145ac719318))

## [0.75.0](https://github.com/substrait-io/substrait/compare/v0.74.1...v0.75.0) (2025-09-14)

### Features

* add AdvancedExtension to SavedComputation and LoadedComputation ([#858](https://github.com/substrait-io/substrait/issues/858)) ([9ea478a](https://github.com/substrait-io/substrait/commit/9ea478af81a6971e378afb1ba28c1151ba276bc4))
* canonicalize extension URNs ([#859](https://github.com/substrait-io/substrait/issues/859)) ([06cadc8](https://github.com/substrait-io/substrait/commit/06cadc8bc6dc5d031083b6094b0ece479219966e))

## [0.74.1](https://github.com/substrait-io/substrait/compare/v0.74.0...v0.74.1) (2025-08-10)

### Bug Fixes

* make simple extensions schema accessible at specified URL ([#848](https://github.com/substrait-io/substrait/issues/848)) ([c0c74ea](https://github.com/substrait-io/substrait/commit/c0c74ea7f17c2d8f5d96e24d2c0dbb664f8126f9))
* typo on window functions site page ([#825](https://github.com/substrait-io/substrait/issues/825)) ([36c1fd8](https://github.com/substrait-io/substrait/commit/36c1fd86e8c2dbfef879e58d42e91f303e429314))

## [0.74.0](https://github.com/substrait-io/substrait/compare/v0.73.0...v0.74.0) (2025-06-22)

### Features

* bitwise shift functions ([#799](https://github.com/substrait-io/substrait/issues/799)) ([fa53460](https://github.com/substrait-io/substrait/commit/fa534605c49a74ac82dcda7a5cdb11a3ec15dab8))

## [0.73.0](https://github.com/substrait-io/substrait/compare/v0.72.0...v0.73.0) (2025-06-01)

### Features

* specify build input of hash join operator ([#810](https://github.com/substrait-io/substrait/issues/810)) ([0c4b658](https://github.com/substrait-io/substrait/commit/0c4b658c0ff474dfdab345ed751b94080a0cc715))

## [0.72.0](https://github.com/substrait-io/substrait/compare/v0.71.0...v0.72.0) (2025-05-04)

### ⚠ BREAKING CHANGES

* direct output order of semi/anti/mark joins no longer
includes invalid side (i.e., right of lefty joins, and left of righty
joins).
* single join raises runtime error if there more than one
matching rows to adhere to the original proposed usage (unnesting scalar
subqueries).

# Problems

1. Semi-joins and anti-joins are one-sided and fields from the other
side of joins are not valid. The `Direct Output Order` in the document
is okay for other types of joins but not in one-side joins.
2. Single joins are proposed to be used unnesting scalar subqueries
where exactly 1 row is expected. The current documentation citing the
paper and behavior although relaxed the behavior so that the
implementation can silently produce wrong result.

# What this PR do?

1. Clarify the direct output order by explicitly stating the semi, anti,
and mark joins. Introducing `Input Order` (the previous `Direct Output
Order`) so that all the properties referencing `Input Order` to reduce
ambiguity.
2. Single joins expecting at most one row for each join key. Otherwise,
runtime error. This behavior can be extended in the future if such
generalization is justified with correct use cases.

### Features

* add description field to types definition in schema ([#811](https://github.com/substrait-io/substrait/issues/811)) ([a4e3a82](https://github.com/substrait-io/substrait/commit/a4e3a82c54a153100ac6c3c9f88add78dc8cd3a8))
* clarify behavior and direct output order of joins ([#803](https://github.com/substrait-io/substrait/issues/803)) ([fe3f1c6](https://github.com/substrait-io/substrait/commit/fe3f1c673ebdb54107dd384073de5a5816d94a44))

## [0.71.0](https://github.com/substrait-io/substrait/compare/v0.70.0...v0.71.0) (2025-04-20)

### ⚠ BREAKING CHANGES

* go_package now points to substrait-protobuf

### Build System

* update go_package of .proto files to substrait-protobuf ([#804](https://github.com/substrait-io/substrait/issues/804)) ([f081cb4](https://github.com/substrait-io/substrait/commit/f081cb4115d84c9516abafe2c84caa2957142f18))

## [0.70.0](https://github.com/substrait-io/substrait/compare/v0.69.0...v0.70.0) (2025-04-13)

### ⚠ BREAKING CHANGES

* Hash Equijoin no longer preserves ordering for inner
joins

The original `Property Maintenance` of hash join operator is following.

> Orderedness of the left set is maintained in INNER join cases,
otherwise it is eliminated.

This holds ONLY very specific implementation of a hash join, for
instance, when build side input completely fits in memory, and probe
side input is streamed in single thread. It is also strange why INNER
JOIN is specifically called out because other joins can preserve order
of probe (LEFT) when build (RIGHT) fits in memory.

Nonetheless, if you throw some kicks and chops, this order preserving
claim quickly falls apart unless implementation does some non-trivial
work under following scenarios.

* build does not fit in memory (i.e., spill to storage)
* parallel probe

So in general, we should not say hash join preserves order of probe. It
*may* assuming a specific implementation under particular conditions,
which is more of optimization or hint territory.

### Features

* remove ordering guarantees from Hash Equijoin operator ([#796](https://github.com/substrait-io/substrait/issues/796)) ([7408bbd](https://github.com/substrait-io/substrait/commit/7408bbdd51f2e52bedd323a0e44184986fe7f519))

## [0.69.0](https://github.com/substrait-io/substrait/compare/v0.68.0...v0.69.0) (2025-03-16)

### Features

* add decimal argument support to round function ([#713](https://github.com/substrait-io/substrait/issues/713)) ([eb696b5](https://github.com/substrait-io/substrait/commit/eb696b5f4aebeef9cf9a98fc828f0cd67143aa16))
* implement PrecisionTime ([#788](https://github.com/substrait-io/substrait/issues/788)) ([1f67065](https://github.com/substrait-io/substrait/commit/1f670654189565565a649ff6249089ae4750ab92))

## [0.68.0](https://github.com/substrait-io/substrait/compare/v0.67.0...v0.68.0) (2025-03-09)

### Features

* add max:pts variant to functions_datetime ([#763](https://github.com/substrait-io/substrait/issues/763)) ([d387335](https://github.com/substrait-io/substrait/commit/d387335a412a1539867f488ecae07f0433ccb4ca))

### Bug Fixes

* minor docs build issues and warnings ([#792](https://github.com/substrait-io/substrait/issues/792)) ([10d53ca](https://github.com/substrait-io/substrait/commit/10d53ca376cde9342a74d3480fcd2e77109d65cd))
* update github actions workflows for building website ([#791](https://github.com/substrait-io/substrait/issues/791)) ([996144c](https://github.com/substrait-io/substrait/commit/996144cfde3c2c9f88678a573c6739d8eeb84607))

## [0.67.0](https://github.com/substrait-io/substrait/compare/v0.66.1...v0.67.0) (2025-02-16)

### Features

* add dynamic parameter expression ([#780](https://github.com/substrait-io/substrait/issues/780)) ([fdf1b38](https://github.com/substrait-io/substrait/commit/fdf1b38050e8683f52a115b7f14953956e84b972))
* support picoseconds in precisionTimestamp and precistionTimestampTZ ([#777](https://github.com/substrait-io/substrait/issues/777)) ([dbce0bd](https://github.com/substrait-io/substrait/commit/dbce0bd9c08acfe9dff11304037b2f57909b8e27))

## [0.66.1](https://github.com/substrait-io/substrait/compare/v0.66.0...v0.66.1) (2025-02-09)

### Bug Fixes

* add is_null for scalar types in testcase grammar ([#779](https://github.com/substrait-io/substrait/issues/779)) ([a30b3e2](https://github.com/substrait-io/substrait/commit/a30b3e2d7ec667a6da8fee083d7823b11768bd2c))
* test coverage miscounting window functions ([#784](https://github.com/substrait-io/substrait/issues/784)) ([5932eb9](https://github.com/substrait-io/substrait/commit/5932eb90edeee7efbdb6ced090ba81dd4b7553a8))

## [0.66.0](https://github.com/substrait-io/substrait/compare/v0.65.0...v0.66.0) (2025-02-02)

### Features

* add advanced extension field to DdlRel, WriteRel, and UpdateRel ([#766](https://github.com/substrait-io/substrait/issues/766)) ([a428f96](https://github.com/substrait-io/substrait/commit/a428f9663043178b121a0354c923ec293edad332))

## [0.65.0](https://github.com/substrait-io/substrait/compare/v0.64.0...v0.65.0) (2025-01-26)

### Features

* add the test coverage baseline as a json file ([#774](https://github.com/substrait-io/substrait/issues/774)) ([dfbe734](https://github.com/substrait-io/substrait/commit/dfbe73418a9de5042ec82608e6e983d96d69e635))

## [0.64.0](https://github.com/substrait-io/substrait/compare/v0.63.1...v0.64.0) (2025-01-12)

### Features

* additional boolean comparison functions ([#764](https://github.com/substrait-io/substrait/issues/764)) ([2d8b1b6](https://github.com/substrait-io/substrait/commit/2d8b1b673718df3deb9deae73de1f53dedba75f1))
* introduce Iceberg table type using metadata file ([#758](https://github.com/substrait-io/substrait/issues/758)) ([7434e2f](https://github.com/substrait-io/substrait/commit/7434e2fa91325c064ae418ded6329c75decb866a))
* run pytest in pr workflow to check function test coverage ([#765](https://github.com/substrait-io/substrait/issues/765)) ([7bfc37c](https://github.com/substrait-io/substrait/commit/7bfc37ca818f7dcb6789b3ad19db41b920584b1e))

### Bug Fixes

* bump flake8 version to 7.0.0 ([#768](https://github.com/substrait-io/substrait/issues/768)) ([57770b6](https://github.com/substrait-io/substrait/commit/57770b6c51dc57be4537630dbd0c23c465557afd))
* update the doc to clarify that function names are case-sensitive ([#757](https://github.com/substrait-io/substrait/issues/757)) ([203e6e4](https://github.com/substrait-io/substrait/commit/203e6e4cf05b5e9d0c08e3ce44655b763a7f00fc))

## [0.63.1](https://github.com/substrait-io/substrait/compare/v0.63.0...v0.63.1) (2024-12-22)

### Bug Fixes

* change the type of the bool_and testcase to aggregate ([#756](https://github.com/substrait-io/substrait/issues/756)) ([50f1a2e](https://github.com/substrait-io/substrait/commit/50f1a2edb017660cc52617336ece4ce692ff546e))

## [0.63.0](https://github.com/substrait-io/substrait/compare/v0.62.0...v0.63.0) (2024-12-15)

### ⚠ BREAKING CHANGES

* The encoding of FetchRel has changed in a strictly
backwards incompatible way. The change involves transitioning offset and
count from a standalone int64 field to a oneof structure, where the
original int64 field is marked as deprecated, and a new field of
Expression type is introduced. Using a oneof may cause ambiguity between
unset and set-to-zero states in older messages. However, the fields are
defined such that their logical meaning remains indistinguishable,
ensuring consistency across encodings.

### Features

* add expression support for count and offset in the fetch operator ([#748](https://github.com/substrait-io/substrait/issues/748)) ([bd4b431](https://github.com/substrait-io/substrait/commit/bd4b431c8c900580f944807f654bc829cdb5dc8d))
* add simple linking to the examples ([#702](https://github.com/substrait-io/substrait/issues/702)) ([4c00b1c](https://github.com/substrait-io/substrait/commit/4c00b1cd549d9deeb29f08f506810464d723cd5a))
* support missing variants for regexp string functions ([#750](https://github.com/substrait-io/substrait/issues/750)) ([3410a3e](https://github.com/substrait-io/substrait/commit/3410a3e1a2c70b04216f8baab500b3dd217b65d2))

## [0.62.0](https://github.com/substrait-io/substrait/compare/v0.61.0...v0.62.0) (2024-11-24)

### Features

* add readme for testcase file format ([#746](https://github.com/substrait-io/substrait/issues/746)) ([708a7b8](https://github.com/substrait-io/substrait/commit/708a7b811641c3384f826f69b9e8247d973c49b9))
* port function testcases from bft ([#738](https://github.com/substrait-io/substrait/issues/738)) ([d84ccd1](https://github.com/substrait-io/substrait/commit/d84ccd1262019fc05dbb32646cfbd71f9800d78a))

### Bug Fixes

* fix function lookup in coverage tool ([#744](https://github.com/substrait-io/substrait/issues/744)) ([3d2ff77](https://github.com/substrait-io/substrait/commit/3d2ff77575a7177f82a4d5b53408a059e9818922))

## [0.61.0](https://github.com/substrait-io/substrait/compare/v0.60.0...v0.61.0) (2024-11-17)

### Features

* add substrait test files to go embedded fs ([#740](https://github.com/substrait-io/substrait/issues/740)) ([e3a7773](https://github.com/substrait-io/substrait/commit/e3a7773895f4121deb6904b79efbcd76c656c9e8))
* handle parsing of list arguments in func testcases ([#737](https://github.com/substrait-io/substrait/issues/737)) ([1f9c710](https://github.com/substrait-io/substrait/commit/1f9c710cb45203bbabaaccd83b792b0b6739ae3a))
* update operator to update a table ([#734](https://github.com/substrait-io/substrait/issues/734)) ([adb1079](https://github.com/substrait-io/substrait/commit/adb1079ab60480d3b7293840bcb923a0f0612211))

### Bug Fixes

* misc fixes and cleanup to func testcase grammar ([#742](https://github.com/substrait-io/substrait/issues/742)) ([ea994c2](https://github.com/substrait-io/substrait/commit/ea994c2ead91eececb770cca1e7f0268e0ebb2ad))

## [0.60.0](https://github.com/substrait-io/substrait/compare/v0.59.0...v0.60.0) (2024-11-10)

### Features

* add antlr grammar for test file format ([#728](https://github.com/substrait-io/substrait/issues/728)) ([752aa63](https://github.com/substrait-io/substrait/commit/752aa636c01647ef15fd59edfd675533e881e692))
* add CreateMode for CTAS in WriteRel ([#715](https://github.com/substrait-io/substrait/issues/715)) ([2e13d0b](https://github.com/substrait-io/substrait/commit/2e13d0b7b4a2b5ac8221e7f02cc8f04456c8f11c))
* update test file format to support aggregate functions ([#736](https://github.com/substrait-io/substrait/issues/736)) ([c18c0c1](https://github.com/substrait-io/substrait/commit/c18c0c1347376efa4dfab503bb4db9f820df3cf3))

### Bug Fixes

* typo in site/docs/tutorial/sql_to_substrait.md ([#735](https://github.com/substrait-io/substrait/issues/735)) ([9cccb04](https://github.com/substrait-io/substrait/commit/9cccb04fba336489b70ed42b71f73a0a1e34f9f5))

## [0.59.0](https://github.com/substrait-io/substrait/compare/v0.58.0...v0.59.0) (2024-11-03)

### ⚠ BREAKING CHANGES

* changes the message type for Expressions field in
VirtualTable

### Features

* add antlr grammar for types ([#730](https://github.com/substrait-io/substrait/issues/730)) ([820085f](https://github.com/substrait-io/substrait/commit/820085fc913692147d0c8fdfcbf289fb8b348835))

### Bug Fixes

* virtualTable expression should represent a row of expression ([#727](https://github.com/substrait-io/substrait/issues/727)) ([a2df42c](https://github.com/substrait-io/substrait/commit/a2df42c76282854d8674aa664d69dacd43630551))

## [0.58.0](https://github.com/substrait-io/substrait/compare/v0.57.1...v0.58.0) (2024-10-13)

### Features

* define sideband optimization hints ([#705](https://github.com/substrait-io/substrait/issues/705)) ([e386a29](https://github.com/substrait-io/substrait/commit/e386a29377c4138a6a2aee87750501b35edae86d))
* enhance VirtualTable to have expression as value ([#711](https://github.com/substrait-io/substrait/issues/711)) ([954bcbc](https://github.com/substrait-io/substrait/commit/954bcbc1a3eeabb696e7dc10721b85e1f475ecfd))
* specify row_number start ([#722](https://github.com/substrait-io/substrait/issues/722)) ([#723](https://github.com/substrait-io/substrait/issues/723)) ([a0388ff](https://github.com/substrait-io/substrait/commit/a0388ff69a83aa4addf51236b77bf275b3647590))

## [0.57.1](https://github.com/substrait-io/substrait/compare/v0.57.0...v0.57.1) (2024-10-06)

### Bug Fixes

* add missing udt identifier in functions_geometry ([#716](https://github.com/substrait-io/substrait/issues/716)) ([f1cedd2](https://github.com/substrait-io/substrait/commit/f1cedd2394bde76f173875a960af061d6b7244ab))

## [0.57.0](https://github.com/substrait-io/substrait/compare/v0.56.0...v0.57.0) (2024-10-02)

### ⚠ BREAKING CHANGES

* This PR changes the definition of grouping sets in
`AggregateRel` to consist of references into a list of grouping
expressions instead of consisting of expressions directly.

With the previous definition, consumers had to deduplicate the
expressions in the grouping sets in order to execute the query or even
derive the output schema (which is problematic, as explained below).
With this change, the responsibility of deduplicating expressions is now
on the producer. Concretely, consumers are now expected to be simpler:
The list of grouping expressions immediately provides the information
needed to derive the output schema and the list of grouping sets
explicitly and unambiguously provides the equality of grouping
expressions. Producers now have to specify the grouping sets explicitly.
If their internal representation of grouping sets consists of full
grouping expressions (rather than references), then they must
deduplicate these expressions according to their internal notion of
expression equality in order to produce grouping sets consisting of
references to these deduplicated expressions.

If the previous format is desired, it can be obtained from the new
format by (1) deduplicating the grouping expressions (according to the
previously applicable definition of expression equality), (2)
re-establishing the duplicates using the emit clause, and (3)
"dereferencing" the references in the grouping sets, i.e., by replacing
each reference in the grouping sets with the expression it refers to.

The previous version was problematic because it required the *consumers*
to deduplicate the expressions from the grouping sets. This, in turn,
requires to parse and understand 100% of these expression even in cases
where that understanding is otherwise optional, which is in opposition
to the general philosophy of allowing for simple-minded consumers. The
new version avoids that problem and, thus, allows consumers to be

### Features

* change grouping expressions in AggregateRel to references ([#706](https://github.com/substrait-io/substrait/issues/706)) ([65a7d38](https://github.com/substrait-io/substrait/commit/65a7d38146f513c82bf2ab27c8597a8c09427a05)), closes [#700](https://github.com/substrait-io/substrait/issues/700)
* clarify behaviour of SetRel operations ([#708](https://github.com/substrait-io/substrait/issues/708)) ([f796521](https://github.com/substrait-io/substrait/commit/f796521d64144a74bf0ac602c46fb66336afe74f))
* make substrait repo a go module ([#712](https://github.com/substrait-io/substrait/issues/712)) ([3dca9b5](https://github.com/substrait-io/substrait/commit/3dca9b505d8122542de311a7d21973b74b58b761))

## [0.56.0](https://github.com/substrait-io/substrait/compare/v0.55.0...v0.56.0) (2024-09-15)

### Features

* add optional metadata containing field names to RelCommon ([#696](https://github.com/substrait-io/substrait/issues/696)) ([5a73281](https://github.com/substrait-io/substrait/commit/5a73281e14448a69d5cb38515cb1a79050eb54eb))
* define mark join ([#682](https://github.com/substrait-io/substrait/issues/682)) ([bc1b93f](https://github.com/substrait-io/substrait/commit/bc1b93f2bf5d2485f417f022c11500a341354ce5))

### Bug Fixes

* correct format for nullable interval_day parameters ([#687](https://github.com/substrait-io/substrait/issues/687)) ([8ae1084](https://github.com/substrait-io/substrait/commit/8ae1084761c0c384c9c54bd4d7af62c4a58ea1cc)), closes [#679](https://github.com/substrait-io/substrait/issues/679)

## [0.55.0](https://github.com/substrait-io/substrait/compare/v0.54.0...v0.55.0) (2024-08-18)

### Features

* update interval_day function extensions to include precision param ([#679](https://github.com/substrait-io/substrait/issues/679)) ([28025cb](https://github.com/substrait-io/substrait/commit/28025cbaa8dc3c65b736d8a68fa7070c465fb494))

## [0.54.0](https://github.com/substrait-io/substrait/compare/v0.53.0...v0.54.0) (2024-08-11)

### ⚠ BREAKING CHANGES

* The encoding of IntervalDay literals has changed in a
strictly backwards incompatible way. However, the logical meaning across
encoding is maintained using a oneof. Moving a field into a oneof makes
unset/set to zero unclear with older messages but the fields are defined
such that the logical meaning of the two is indistinct. If neither
microseconds nor precision is set, the value can be considered a
precision 6 value. If you aren't using IntervalDay type, you will not
need to make any changes.
* TypeExpression and Parameterized type protobufs (used
to serialize output derivation) are updated to match the now compound
nature of IntervalDay. If you use protobuf to serialize output
derivation that refer to IntervalDay type, you will need to rework that
logic.
* JoinRel's type enum now has LEFT_SINGLE
instead of SINGLE.  Similarly there is now LEFT_ANTI and LEFT_SEMI.
Other values are available in all join type enums. This affects JSON and
text formats only (binary plans -- the interoperable part of Substrait --
will still be compatible before and after this change).

### Features

* add arithmetic function "power" with decimal type ([#660](https://github.com/substrait-io/substrait/issues/660)) ([9af2d66](https://github.com/substrait-io/substrait/commit/9af2d66addc30ef49ed8b570a2bf9c2e1c21bad2))
* add CSV (text) file support ([#646](https://github.com/substrait-io/substrait/issues/646)) ([5d49e04](https://github.com/substrait-io/substrait/commit/5d49e04325fcdd7c9632bda9e869a71a9d8fa8dc))
* add precision to IntervalDay and new IntervalCompound type ([#665](https://github.com/substrait-io/substrait/issues/665)) ([e41eff2](https://github.com/substrait-io/substrait/commit/e41eff2cfed5ae6f20d0fde9b6b86da91f9d6542)), closes [#664](https://github.com/substrait-io/substrait/issues/664)
* normalize the join types ([#662](https://github.com/substrait-io/substrait/issues/662)) ([bed84ec](https://github.com/substrait-io/substrait/commit/bed84ecb6193c22bf2ff83dc3a391ec5a9a3aa68))

## [0.53.0](https://github.com/substrait-io/substrait/compare/v0.52.0...v0.53.0) (2024-08-04)

### ⚠ BREAKING CHANGES

* PrecisionTimestamp(Tz) literal's value is now int64
instead of uint64

### Features

* add aggregate count functions with decimal return type ([#670](https://github.com/substrait-io/substrait/issues/670)) ([2aa516b](https://github.com/substrait-io/substrait/commit/2aa516bff3b2cc3e5ad262152c98f1d9b15c6765))
* add arithmetic function "sqrt" and "factorial" with decimal type ([#674](https://github.com/substrait-io/substrait/issues/674)) ([e4f5b68](https://github.com/substrait-io/substrait/commit/e4f5b68981953d3546835572ce566e9586d497be))
* add arithmetic function for bitwise(AND/OR/XOR) operation with decimal arguments ([#675](https://github.com/substrait-io/substrait/issues/675)) ([a70cf72](https://github.com/substrait-io/substrait/commit/a70cf72425c3a0eed432238c2a8afedab1cc025b))
* add logarithmic functions with decimal type args ([#669](https://github.com/substrait-io/substrait/issues/669)) ([d9fb1e3](https://github.com/substrait-io/substrait/commit/d9fb1e355e0b378e1b6460f256d724a3aae931d3))
* add precision timestamp datetime fn variants ([#666](https://github.com/substrait-io/substrait/issues/666)) ([60c93d2](https://github.com/substrait-io/substrait/commit/60c93d28c8e4df3174ba6b3f687a30d256acdcae))
* clarify the meaning of plans ([#616](https://github.com/substrait-io/substrait/issues/616)) ([c1553df](https://github.com/substrait-io/substrait/commit/c1553dfafa09de1b2441cdb1d22a251a675419a7)), closes [#612](https://github.com/substrait-io/substrait/issues/612) [#613](https://github.com/substrait-io/substrait/issues/613)

### Bug Fixes

* use int64 instead of uint64 for PrecisionTimestamp(Tz) literal value ([#668](https://github.com/substrait-io/substrait/issues/668)) ([da3c74e](https://github.com/substrait-io/substrait/commit/da3c74eccc4978bdaeca4760e98a77aff560e19b))

## [0.52.0](https://github.com/substrait-io/substrait/compare/v0.51.0...v0.52.0) (2024-07-14)

### ⚠ BREAKING CHANGES

* changes the message type for Literal PrecisionTimestamp
and PrecisionTimestampTZ

The PrecisionTimestamp and PrecisionTimestampTZ literals were introduced

### Bug Fixes

* include precision information in PrecisionTimestamp and PrecisionTimestampTZ literals ([#659](https://github.com/substrait-io/substrait/issues/659)) ([f9e5f9c](https://github.com/substrait-io/substrait/commit/f9e5f9c515d4b8be079bc7d9dfcd89a6fa5e6c7e)), closes [#594](https://github.com/substrait-io/substrait/issues/594) [/github.com/substrait-io/substrait/pull/594#discussion_r1471844566](https://github.com/substrait-io//github.com/substrait-io/substrait/pull/594/issues/discussion_r1471844566)

## [0.51.0](https://github.com/substrait-io/substrait/compare/v0.50.0...v0.51.0) (2024-07-07)

### Features

* add "initcap" function ([#656](https://github.com/substrait-io/substrait/issues/656)) ([95bc6ba](https://github.com/substrait-io/substrait/commit/95bc6ba0ca5056274ccc81608919de22032084ad)), closes [/github.com/Blizzara/substrait/blob/70d1eb71623ca0754157dd5d87348bae51d420c4/extensions/functions_string.yaml#L1023](https://github.com/substrait-io//github.com/Blizzara/substrait/blob/70d1eb71623ca0754157dd5d87348bae51d420c4/extensions/functions_string.yaml/issues/L1023)
* add null input handling options for `any_value` ([#652](https://github.com/substrait-io/substrait/issues/652)) ([1890e6a](https://github.com/substrait-io/substrait/commit/1890e6a7814c5161f38a31aba3e284dde1bc79d4))
* allow naming/aliasing relations ([#649](https://github.com/substrait-io/substrait/issues/649)) ([4cf8108](https://github.com/substrait-io/substrait/commit/4cf8108e0746bbe6d6cf5ea95a6a5276580e0dde)), closes [#648](https://github.com/substrait-io/substrait/issues/648) [#571](https://github.com/substrait-io/substrait/issues/571)
* define SetRel output nullability derivation ([#558](https://github.com/substrait-io/substrait/issues/558)) ([#654](https://github.com/substrait-io/substrait/issues/654)) ([612123a](https://github.com/substrait-io/substrait/commit/612123a4a84cf9554e0b8f92671ea5159c6deb21))

## [0.50.0](https://github.com/substrait-io/substrait/compare/v0.49.0...v0.50.0) (2024-06-30)

### ⚠ BREAKING CHANGES

* consumers must now check for multiple optimization
messages within an AdvancedExtension

### Features

* make optimization a repeated field ([#653](https://github.com/substrait-io/substrait/issues/653)) ([e523d5d](https://github.com/substrait-io/substrait/commit/e523d5d9fa25cf432bd07cd418a3d7f829f01037))

## [0.49.0](https://github.com/substrait-io/substrait/compare/v0.48.0...v0.49.0) (2024-05-23)


### Features

* abs add decimal type ([#637](https://github.com/substrait-io/substrait/issues/637)) ([beff1f0](https://github.com/substrait-io/substrait/commit/beff1f039b618f8f14b3c699e139234964c0b2f7))
* add is distinct from function ([#638](https://github.com/substrait-io/substrait/issues/638)) ([de4fcbc](https://github.com/substrait-io/substrait/commit/de4fcbc066315ca874ca163275affdc4156e570e))


### Bug Fixes

* **ci:** pin `conventional-changelog-conventionalcommits` to `7.0.2` ([#644](https://github.com/substrait-io/substrait/issues/644)) ([9528bd2](https://github.com/substrait-io/substrait/commit/9528bd28c9c403f00d2f018fa50b572c9aa93a89))
* specify a minimum length for the options of enum args ([#642](https://github.com/substrait-io/substrait/issues/642)) ([8e65af5](https://github.com/substrait-io/substrait/commit/8e65af5363da41fd73c131c2a465d5186c52c403)), closes [/github.com/substrait-io/substrait-rs/pull/185#discussion_r1603513149](https://github.com/substrait-io//github.com/substrait-io/substrait-rs/pull/185/issues/discussion_r1603513149)

## [0.48.0](https://github.com/substrait-io/substrait/compare/v0.47.0...v0.48.0) (2024-04-25)


### ⚠ BREAKING CHANGES

* min:ts has been moved to functions_datetime
* max:ts has been moved to functions_datetime

### Bug Fixes

* duplicate declaration of min:ts and max:ts ([#631](https://github.com/substrait-io/substrait/issues/631)) ([7fc86f8](https://github.com/substrait-io/substrait/commit/7fc86f85e468ab404825ca093e718bd3996d8241))

## [0.47.0](https://github.com/substrait-io/substrait/compare/v0.46.0...v0.47.0) (2024-04-18)


### Features

* add i64 variant for exp, ln, log10, log2 and logb functions ([#628](https://github.com/substrait-io/substrait/issues/628)) ([fef2253](https://github.com/substrait-io/substrait/commit/fef225343d8b686c3150deea7436792252057fb9))
* allow FetchRel to specify a return of ALL results ([#622](https://github.com/substrait-io/substrait/issues/622)) ([#627](https://github.com/substrait-io/substrait/issues/627)) ([37f43b4](https://github.com/substrait-io/substrait/commit/37f43b4f8f74ad36e8d2d9dc7c5fecb740ac7ca1))


### Bug Fixes

* index_in has wrong return type ([#632](https://github.com/substrait-io/substrait/issues/632)) ([4cd2089](https://github.com/substrait-io/substrait/commit/4cd2089fe85ba53eb22718b314b3bc1132dc4265))
* use any1 instead of T in function extensions ([#629](https://github.com/substrait-io/substrait/issues/629)) ([0bddf68](https://github.com/substrait-io/substrait/commit/0bddf681feb8176bb111cd5139c884f9137c2e0b))

## [0.46.0](https://github.com/substrait-io/substrait/compare/v0.45.0...v0.46.0) (2024-04-14)


### Features

* expand division function options ([#615](https://github.com/substrait-io/substrait/issues/615)) ([7b79437](https://github.com/substrait-io/substrait/commit/7b794379c51436620515b5ebad7720c502aab991))


### Bug Fixes

* remove implicit casts in trig extension functions ([#620](https://github.com/substrait-io/substrait/issues/620)) ([b883120](https://github.com/substrait-io/substrait/commit/b8831200909d6494e1c3bc1ba1157e4741f60377))

## [0.45.0](https://github.com/substrait-io/substrait/compare/v0.44.0...v0.45.0) (2024-03-24)


### Features

* add decimal type support for sum0 function ([#610](https://github.com/substrait-io/substrait/issues/610)) ([6bd0c7b](https://github.com/substrait-io/substrait/commit/6bd0c7be43149a5b6e2f2ec8decdbb47f64577b7))

## [0.44.0](https://github.com/substrait-io/substrait/compare/v0.43.0...v0.44.0) (2024-03-03)


### ⚠ BREAKING CHANGES

* Adding a NULL option to the on_domain_errors.

SQLite returns null for some inputs such as negative infinity

### Features

* add extra option for on domain errors in log functions ([#536](https://github.com/substrait-io/substrait/issues/536)) ([cbec079](https://github.com/substrait-io/substrait/commit/cbec079ea03bec65cc063daa15e42807c4039707))
* add ignore nulls options to concat function ([#605](https://github.com/substrait-io/substrait/issues/605)) ([55db05b](https://github.com/substrait-io/substrait/commit/55db05b4cf8cbb1e2bf565e4f5f0c6def6f0e6ed))

## [0.43.0](https://github.com/substrait-io/substrait/compare/v0.42.1...v0.43.0) (2024-02-25)


### Features

* include precision parameter in timestamp types ([#594](https://github.com/substrait-io/substrait/issues/594)) ([087f87c](https://github.com/substrait-io/substrait/commit/087f87c0307572cf2e9a7d1db7fdd673662699c3))


### Bug Fixes

* remove function definitions w/ invalid return types ([#599](https://github.com/substrait-io/substrait/issues/599)) ([a3b1f32](https://github.com/substrait-io/substrait/commit/a3b1f32b0e6aac08bf0ee7437a5ae1c10100a859))

## [0.42.1](https://github.com/substrait-io/substrait/compare/v0.42.0...v0.42.1) (2024-01-28)


### Bug Fixes

* add missing RelCommon field to WriteRel and DdlRel ([#591](https://github.com/substrait-io/substrait/issues/591)) ([d55703a](https://github.com/substrait-io/substrait/commit/d55703a18a7a8f2ecf695f9367ca33fab6b1ef33))

## [0.42.0](https://github.com/substrait-io/substrait/compare/v0.41.0...v0.42.0) (2024-01-21)


### Features

* add custom equality behavior to the hash/merge join ([#585](https://github.com/substrait-io/substrait/issues/585)) ([daeac31](https://github.com/substrait-io/substrait/commit/daeac314e9efb6c385306c7f14b95ded2da226ac))
* add interval multiplication ([#580](https://github.com/substrait-io/substrait/issues/580)) ([c1254ac](https://github.com/substrait-io/substrait/commit/c1254ac5c5f1105478d26b7d715bab8d21dd31d1))
* add min/max for datetime types ([#584](https://github.com/substrait-io/substrait/issues/584)) ([5c8fa04](https://github.com/substrait-io/substrait/commit/5c8fa047993835b2bba60b196af0855316e5efdb))

## [0.41.0](https://github.com/substrait-io/substrait/compare/v0.40.0...v0.41.0) (2023-12-24)


### ⚠ BREAKING CHANGES

* Renamed modulus to modulo. 

Added options and documentation for the modulo operator as defined in
math and comp sci.

### Bug Fixes

* renamed modulus to modulo; updated modulo operator defintion ([#583](https://github.com/substrait-io/substrait/issues/583)) ([aba1bc7](https://github.com/substrait-io/substrait/commit/aba1bc79acc5bf40a719b23276bfa6f7546e7ed5)), closes [#353](https://github.com/substrait-io/substrait/issues/353)

## [0.40.0](https://github.com/substrait-io/substrait/compare/v0.39.0...v0.40.0) (2023-12-17)


### ⚠ BREAKING CHANGES

* The enum `WriteRel::OutputMode` had an option change
from
`OUTPUT_MODE_MODIFIED_TUPLES` to `OUTPUT_MODE_MODIFIED_RECORDS`
* The message `AggregateFunction.ReferenceRel` has moved
to `ReferenceRel`.

### Features

* add missing rels to rel message ([#582](https://github.com/substrait-io/substrait/issues/582)) ([d952b45](https://github.com/substrait-io/substrait/commit/d952b4566e806b5d759fa365c605eb7c4e2629c3)), closes [#288](https://github.com/substrait-io/substrait/issues/288)

## [0.39.0](https://github.com/substrait-io/substrait/compare/v0.38.0...v0.39.0) (2023-11-26)


### ⚠ BREAKING CHANGES

*   * Map keys may be repeated.
   * Map keys must not be NULL.
   * The map key type may be nullable.

This is based on the current restrictions found in the wild.

DuckDB, Velox, Spark, and Acero all reject attempts to provide NULL as a
key.

Despite DuckDB specifically calling out that keys must be unique in its
implementation other implementations such as Velox and Acero do not
require the key to be unique so we cannot require the map key to be 1:1
with map values.

### Features

* support for simple extensions dependencies ([#265](https://github.com/substrait-io/substrait/issues/265)) ([f0ecf54](https://github.com/substrait-io/substrait/commit/f0ecf54e271f060687d87707e58c0354b02fd769))


### Documentation

* clarify map key behavior ([#521](https://github.com/substrait-io/substrait/issues/521)) ([e3860f5](https://github.com/substrait-io/substrait/commit/e3860f56a262a41582503c61dd5095188e96f644))

## [0.38.0](https://github.com/substrait-io/substrait/compare/v0.37.0...v0.38.0) (2023-11-05)


### Features

* add least and greatest functions to functions_comparison.yml ([#247](https://github.com/substrait-io/substrait/issues/247)) ([b3071bc](https://github.com/substrait-io/substrait/commit/b3071bc9cd77cf916568641c83056a285f8123be))

## [0.37.0](https://github.com/substrait-io/substrait/compare/v0.36.0...v0.37.0) (2023-10-22)


### Features

* add NestedLoopJoinRel definition ([#561](https://github.com/substrait-io/substrait/issues/561)) ([cf32750](https://github.com/substrait-io/substrait/commit/cf327502bdb187ae06d9210e9de460193027679e))

## [0.36.0](https://github.com/substrait-io/substrait/compare/v0.35.0...v0.36.0) (2023-10-08)


### Features

* geometry processing functions ([#556](https://github.com/substrait-io/substrait/issues/556)) ([8406cf6](https://github.com/substrait-io/substrait/commit/8406cf6753b97829b2b5211344822d6f2f840eab))

## [0.35.0](https://github.com/substrait-io/substrait/compare/v0.34.0...v0.35.0) (2023-10-01)


### ⚠ BREAKING CHANGES

* nullability of is_not_distinct_from has changed
* The minimum precision for floating point numbers is
now mandated.

### Features

* add approval guidelines for documentation updates ([#553](https://github.com/substrait-io/substrait/issues/553)) ([da4b32a](https://github.com/substrait-io/substrait/commit/da4b32ac41827ae8b53a2833ec34872670904e57))
* add geometric data types and functions ([#543](https://github.com/substrait-io/substrait/issues/543)) ([db52bbd](https://github.com/substrait-io/substrait/commit/db52bbd844f7d8db328f1b6f00758f07009ca95b))
* add geometry editor functions ([#554](https://github.com/substrait-io/substrait/issues/554)) ([727467c](https://github.com/substrait-io/substrait/commit/727467cc66f4c4984c7a8ea1205a473644f00b23))
* adding geometry accessor functions ([#552](https://github.com/substrait-io/substrait/issues/552)) ([784fa9b](https://github.com/substrait-io/substrait/commit/784fa9b1702a1df64a8286a25fce377a0aa29fd4))
* explicitly reference IEEE 754 and mandate precision as well as range ([#449](https://github.com/substrait-io/substrait/issues/449)) ([54e3d52](https://github.com/substrait-io/substrait/commit/54e3d52bc07c8952af86f57250253d10a97dadc3)), closes [#447](https://github.com/substrait-io/substrait/issues/447)


### Bug Fixes

* specify nullability for is_not_distinct_from ([#555](https://github.com/substrait-io/substrait/issues/555)) ([30773b2](https://github.com/substrait-io/substrait/commit/30773b2fcb67413625535cd1ada144dccfdcde22))

## [0.34.0](https://github.com/substrait-io/substrait/compare/v0.33.0...v0.34.0) (2023-09-17)


### Features

* add more window functions ([#534](https://github.com/substrait-io/substrait/issues/534)) ([f2bfe15](https://github.com/substrait-io/substrait/commit/f2bfe15585943a137fafa560401e0cf0266c0650))
* allow agg functions to be used in windows ([#540](https://github.com/substrait-io/substrait/issues/540)) ([565a1ef](https://github.com/substrait-io/substrait/commit/565a1ef26eccffba8f31ffe885667fab475d1da5))

## [0.33.0](https://github.com/substrait-io/substrait/compare/v0.32.0...v0.33.0) (2023-08-27)


### Features

* add radians and degrees functions ([#544](https://github.com/substrait-io/substrait/issues/544)) ([2da2afa](https://github.com/substrait-io/substrait/commit/2da2afad579a428bb8f7460a153a1799af5c6ee3))

## [0.32.0](https://github.com/substrait-io/substrait/compare/v0.31.0...v0.32.0) (2023-08-21)


### ⚠ BREAKING CHANGES

* plans referencing functions using simple
names (e.g. not vs not:bool) will no longer be valid.

### Features

* add ExchangeRel as a type in Rel ([#518](https://github.com/substrait-io/substrait/issues/518)) ([89b0c62](https://github.com/substrait-io/substrait/commit/89b0c6259a7440f760fafe32e8999d5d37cac8c7))
* add expand rel ([#368](https://github.com/substrait-io/substrait/issues/368)) ([98380b0](https://github.com/substrait-io/substrait/commit/98380b0dd1dd9eb30457800ec49d7912b5dce11f))
* add options to substring for start parameter being negative ([#508](https://github.com/substrait-io/substrait/issues/508)) ([281dc0f](https://github.com/substrait-io/substrait/commit/281dc0fba176df22fc35ff5f5acb7a05863b9d59))
* add windowrel support  in proto ([#399](https://github.com/substrait-io/substrait/issues/399)) ([bd14e0e](https://github.com/substrait-io/substrait/commit/bd14e0e40782dbd0fa49de597ec30217b48961f2))
* require compound functions names in extension references ([#537](https://github.com/substrait-io/substrait/issues/537)) ([2503beb](https://github.com/substrait-io/substrait/commit/2503beb3c872928483c05f76bf74d18188c84798))

## [0.31.0](https://github.com/substrait-io/substrait/compare/v0.30.0...v0.31.0) (2023-07-02)


### Features

* add a two-arg variant of substring ([#513](https://github.com/substrait-io/substrait/issues/513)) ([a6ead70](https://github.com/substrait-io/substrait/commit/a6ead70b1d62b79fad7ba2f9fdaf76c5b6d7696b))
* add timestamp types to max/min function ([#511](https://github.com/substrait-io/substrait/issues/511)) ([6943400](https://github.com/substrait-io/substrait/commit/694340013433b1c0408c2a1cd77b22dfb9b22ad0))

## [0.30.0](https://github.com/substrait-io/substrait/compare/v0.29.0...v0.30.0) (2023-05-14)


### ⚠ BREAKING CHANGES

* This adds an option to control indexing of components

### Features

* control indexing in temporal extraction ([#479](https://github.com/substrait-io/substrait/issues/479)) ([aacd25c](https://github.com/substrait-io/substrait/commit/aacd25c8fa5eb680c3456d2e0298ca0807eb7b87)), closes [#477](https://github.com/substrait-io/substrait/issues/477)

## [0.29.0](https://github.com/substrait-io/substrait/compare/v0.28.2...v0.29.0) (2023-04-23)


### ⚠ BREAKING CHANGES

* **text:** mark `name` and `structure` property of `type` extension item as required (#495)

### Bug Fixes

* referenced simple extension in tutorial (set instead of string) ([#494](https://github.com/substrait-io/substrait/issues/494)) ([b5d7ed2](https://github.com/substrait-io/substrait/commit/b5d7ed26a17c0a0bd6d0779d312942e5216ea9fa))
* **text:** mark `name` and `structure` property of `type` extension item as required ([#495](https://github.com/substrait-io/substrait/issues/495)) ([7246102](https://github.com/substrait-io/substrait/commit/7246102f0e1f056a3b5a13eb96fec36ff28d27a5))

## [0.28.2](https://github.com/substrait-io/substrait/compare/v0.28.1...v0.28.2) (2023-04-16)


### Bug Fixes

* separate strptime to fix spec violation ([#493](https://github.com/substrait-io/substrait/issues/493)) ([8c230af](https://github.com/substrait-io/substrait/commit/8c230af70bc98805d84d20c72f32d0ddb84f8644))

## [0.28.1](https://github.com/substrait-io/substrait/compare/v0.28.0...v0.28.1) (2023-04-09)


### Bug Fixes

* typo in the comment/docstring ([#492](https://github.com/substrait-io/substrait/issues/492)) ([9046945](https://github.com/substrait-io/substrait/commit/90469453d111ba93983b00944dd79d0ddd8a3808))

## [0.28.0](https://github.com/substrait-io/substrait/compare/v0.27.0...v0.28.0) (2023-04-02)


### Features

* adding BibTex entry to cite Substrait ([#481](https://github.com/substrait-io/substrait/issues/481)) ([425e7f8](https://github.com/substrait-io/substrait/commit/425e7f868e0f89115bc125e8dab2c04b8144ff82)), closes [#480](https://github.com/substrait-io/substrait/issues/480)
* adding SUM0 definition for aggregate functions ([#465](https://github.com/substrait-io/substrait/issues/465)) ([73228b4](https://github.com/substrait-io/substrait/commit/73228b4112d79eb1011af0ebb41753ce23ca180c)), closes [#259](https://github.com/substrait-io/substrait/issues/259)

## [0.27.0](https://github.com/substrait-io/substrait/compare/v0.26.0...v0.27.0) (2023-03-26)


### ⚠ BREAKING CHANGES

* `group` argument added to `regexp_match_substring`
function

Add regexp_match_substring_all function

Resolves https://github.com/substrait-io/substrait/issues/466

### Features

* add regexp_match_substring_all function to yaml ([#469](https://github.com/substrait-io/substrait/issues/469)) ([b4d81fb](https://github.com/substrait-io/substrait/commit/b4d81fba48990523012c7b2c6cc71d2c01650e59))


### Bug Fixes

* **ci:** fix link to conventional commits spec ([#482](https://github.com/substrait-io/substrait/issues/482)) ([45b4e48](https://github.com/substrait-io/substrait/commit/45b4e483ff1fca3c3e4d0f71e6e55436c6d7638a))
* remove duplication in simple extensions schema ([#404](https://github.com/substrait-io/substrait/issues/404)) ([b7df38d](https://github.com/substrait-io/substrait/commit/b7df38d2099cd970d1ed1783d441d828ce84253d))

## [0.26.0](https://github.com/substrait-io/substrait/compare/v0.25.0...v0.26.0) (2023-03-05)


### Features

* add script to re-namespace .proto files for internal use in public libraries ([#207](https://github.com/substrait-io/substrait/issues/207)) ([a6f24db](https://github.com/substrait-io/substrait/commit/a6f24dbdc592baf4d0d775ee2d3b296eb747e86a))
* add temporal functions ([#272](https://github.com/substrait-io/substrait/issues/272)) ([beb104b](https://github.com/substrait-io/substrait/commit/beb104b31aebe584f859f6ce27e3e3a62bc70132)), closes [#222](https://github.com/substrait-io/substrait/issues/222)

## [0.25.0](https://github.com/substrait-io/substrait/compare/v0.24.0...v0.25.0) (2023-02-26)


### ⚠ BREAKING CHANGES

* (add/subtract)ing an interval to a timestamp_tz
now requires a time zone and returns a timestamp_tz

### Bug Fixes

* correct return of temporal add and subtract and add timezone parameter ([#337](https://github.com/substrait-io/substrait/issues/337)) ([1b184cc](https://github.com/substrait-io/substrait/commit/1b184cc79197c20f510aa74e633658f5ce249e47))
* **extension:** fix typo in scalar function argument type ([#445](https://github.com/substrait-io/substrait/issues/445)) ([7d7ddf1](https://github.com/substrait-io/substrait/commit/7d7ddf11f3ce0b5f69a9d32ef10a699888f18a61))

## [0.24.0](https://github.com/substrait-io/substrait/compare/v0.23.0...v0.24.0) (2023-02-12)


### Features

* add round function ([#322](https://github.com/substrait-io/substrait/issues/322)) ([57121c8](https://github.com/substrait-io/substrait/commit/57121c8ca6f1fe815e98eda8962f8f84736c58e2))

## [0.23.0](https://github.com/substrait-io/substrait/compare/v0.22.0...v0.23.0) (2023-01-22)


### Features

* add extended expression for expression only evaluation ([#405](https://github.com/substrait-io/substrait/issues/405)) ([d35f0ed](https://github.com/substrait-io/substrait/commit/d35f0ed98ccefe31a90d53ff887402636a74bbd1))
* **spec:** add physical plans for hashJoin and mergeJoin ([#336](https://github.com/substrait-io/substrait/issues/336)) ([431651e](https://github.com/substrait-io/substrait/commit/431651efbd64958d2611b035ffdb25f589b28477))


### Bug Fixes

* update extension yaml files to match type-syntax spec ([#423](https://github.com/substrait-io/substrait/issues/423)) ([0608878](https://github.com/substrait-io/substrait/commit/0608878b25e7f9b4b3ffe33662eea9ef0f016548))

## [0.22.0](https://github.com/substrait-io/substrait/compare/v0.21.1...v0.22.0) (2022-12-18)


### Features

* add bitwise NOT, AND, OR & XOR functions ([#370](https://github.com/substrait-io/substrait/issues/370)) ([81e34d4](https://github.com/substrait-io/substrait/commit/81e34d4054ff0dbde23ac749fbb8fcc232989c5d))

## [0.21.1](https://github.com/substrait-io/substrait/compare/v0.21.0...v0.21.1) (2022-12-04)


### Bug Fixes

* rename regex_string_split to regexp_string_split ([#393](https://github.com/substrait-io/substrait/issues/393)) ([f9f4967](https://github.com/substrait-io/substrait/commit/f9f4967e6785b49eccb64a42497b5b4aaffa63ff))

## [0.21.0](https://github.com/substrait-io/substrait/compare/v0.20.0...v0.21.0) (2022-11-27)


### Features

* add nested type constructor expressions ([#351](https://github.com/substrait-io/substrait/issues/351)) ([b64d30b](https://github.com/substrait-io/substrait/commit/b64d30b28077973dd94f1f49e5016662a35bcf56))
* add title to simple extensions schema ([#387](https://github.com/substrait-io/substrait/issues/387)) ([2819ecc](https://github.com/substrait-io/substrait/commit/2819ecc69175b96eefb8a73fb4b533431890f3da))

## [0.20.0](https://github.com/substrait-io/substrait/compare/v0.19.0...v0.20.0) (2022-11-20)


### ⚠ BREAKING CHANGES

* optional arguments are no longer allowed to be specified
as a part of FunctionArgument messages.  Instead they are now specified
separately as part of the function invocation.
* optional arguments are now specified separately from
required arguments in the YAML specification.

Co-authored-by: Benjamin Kietzman <bengilgit@gmail.com>

Co-authored-by: Benjamin Kietzman <bengilgit@gmail.com>

### Features

* add best effort filter to read rel and clarify that the pre-masked schema should be used ([#271](https://github.com/substrait-io/substrait/issues/271)) ([4beff87](https://github.com/substrait-io/substrait/commit/4beff877550ac4ac10199748acbba391aca172f6))
* optional args are now specified separately from required args ([#342](https://github.com/substrait-io/substrait/issues/342)) ([bd29ea3](https://github.com/substrait-io/substrait/commit/bd29ea3b06391ae9018de851055db11075fd0758))

## [0.19.0](https://github.com/substrait-io/substrait/compare/v0.18.0...v0.19.0) (2022-11-06)


### Features

* add functions for splitting strings ([#346](https://github.com/substrait-io/substrait/issues/346)) ([20a2f14](https://github.com/substrait-io/substrait/commit/20a2f14ee0f2c3186318543d7ff264c91f714967))


### Bug Fixes

* rename version fields which conflict with sysmacros ([#362](https://github.com/substrait-io/substrait/issues/362)) ([4170bf1](https://github.com/substrait-io/substrait/commit/4170bf12c0f86032d8649a0880c684c37a5065f7))

## [0.18.0](https://github.com/substrait-io/substrait/compare/v0.17.0...v0.18.0) (2022-10-09)


### Features

* attach Substrait version number to plans ([#347](https://github.com/substrait-io/substrait/issues/347)) ([2d1bb9d](https://github.com/substrait-io/substrait/commit/2d1bb9d9472409715f1667dfeae241677c6c5ec2))

## [0.17.0](https://github.com/substrait-io/substrait/compare/v0.16.0...v0.17.0) (2022-10-02)


### Features

* support non-struct type class structure ([#328](https://github.com/substrait-io/substrait/issues/328)) ([dd7f9f0](https://github.com/substrait-io/substrait/commit/dd7f9f01bdf11f5ac9db7713c5ff3d2f82ff5a78))

## [0.16.0](https://github.com/substrait-io/substrait/compare/v0.15.0...v0.16.0) (2022-09-25)


### Features

* add any_value aggregate function ([#321](https://github.com/substrait-io/substrait/issues/321)) ([6f603d3](https://github.com/substrait-io/substrait/commit/6f603d3b61ad26a2f7da1bc74e2a60dd246def16))
* support constant function arguments ([#305](https://github.com/substrait-io/substrait/issues/305)) ([6021030](https://github.com/substrait-io/substrait/commit/6021030a599134f959ebc0f36621b27127316356))

## [0.15.0](https://github.com/substrait-io/substrait/compare/v0.14.0...v0.15.0) (2022-09-18)


### ⚠ BREAKING CHANGES

* options were added to division and logarithmic functions.

### Features

* add options for behaviour when dividing by zero or calculating log zero ([#329](https://github.com/substrait-io/substrait/issues/329)) ([1c170c8](https://github.com/substrait-io/substrait/commit/1c170c8d984ffbee759f7d7371cbb93b1fd24db9))


### Bug Fixes

* **naming:** add missing arg names in functions_aggregate_*.yaml ([#316](https://github.com/substrait-io/substrait/issues/316)) ([fb92997](https://github.com/substrait-io/substrait/commit/fb9299735f4e67cffaa7b153f4dce885c9f7f93d))

## [0.14.0](https://github.com/substrait-io/substrait/compare/v0.13.0...v0.14.0) (2022-09-11)


### ⚠ BREAKING CHANGES

* option argument added to std_dev and variance aggregate functions

### Features

* add bool_and and bool_or aggregate functions ([#314](https://github.com/substrait-io/substrait/issues/314)) ([52fa523](https://github.com/substrait-io/substrait/commit/52fa5235c6bb2f43ccc2e25c6fe548a0f0215524))
* add corr and mode aggregation functions ([#296](https://github.com/substrait-io/substrait/issues/296)) ([96b13d7](https://github.com/substrait-io/substrait/commit/96b13d7ea4e9dc95c051d02521812e6011c47e20))
* add median and count_distinct aggregation functions ([#278](https://github.com/substrait-io/substrait/issues/278)) ([9be62e5](https://github.com/substrait-io/substrait/commit/9be62e5067c13858e8c545689891937c2dced4ee))
* add population option to variance and standard deviation functions ([#295](https://github.com/substrait-io/substrait/issues/295)) ([c47fffa](https://github.com/substrait-io/substrait/commit/c47fffa83af26f7278a5d7f6501d9eadbd365d30))
* add quantile aggregate function ([#279](https://github.com/substrait-io/substrait/issues/279)) ([de6bc9f](https://github.com/substrait-io/substrait/commit/de6bc9fad440880b6b5333cb0ee129d2c19e471c))
* add string_agg aggregate function ([#297](https://github.com/substrait-io/substrait/issues/297)) ([fbe5e09](https://github.com/substrait-io/substrait/commit/fbe5e0949b863334d02b5ad9ecac55ec8fc4debb))


### Bug Fixes

* mark string_agg aggregate as being sensitive to input order ([#312](https://github.com/substrait-io/substrait/issues/312)) ([683faaa](https://github.com/substrait-io/substrait/commit/683faaa37ce8cad444a5fe703a7653dc04d02486))
* **naming:** add missing arg names in functions_arithmetic.yaml ([#315](https://github.com/substrait-io/substrait/issues/315)) ([d433a06](https://github.com/substrait-io/substrait/commit/d433a06adc77d9d71db3a3b956d82b8318d220ed))
* **naming:** add missing arg names in functions_datetime.yaml ([#318](https://github.com/substrait-io/substrait/issues/318)) ([b7347d1](https://github.com/substrait-io/substrait/commit/b7347d15c62e67fbca2cb810c008c32460263d7b))
* **naming:** add missing arg names in functions_logarithmic.yaml and functions_set.yaml ([#319](https://github.com/substrait-io/substrait/issues/319)) ([1c14d27](https://github.com/substrait-io/substrait/commit/1c14d271557addb5980123778102f844359a749e))
* **naming:** add/replace arg names in functions_boolean.yaml ([#317](https://github.com/substrait-io/substrait/issues/317)) ([809a2f4](https://github.com/substrait-io/substrait/commit/809a2f42c2f2795bc7efd64b7ff4cef3d9abc807))
* revert addition of count_distinct aggregate function ([#311](https://github.com/substrait-io/substrait/issues/311)) ([90d7c0d](https://github.com/substrait-io/substrait/commit/90d7c0df9c729a3027988aeadfd74104f7385014))

## [0.13.0](https://github.com/substrait-io/substrait/compare/v0.12.0...v0.13.0) (2022-09-04)


### ⚠ BREAKING CHANGES

* nullability behavior of is_nan, is_finite, and is_infinite has changed
* compound name for concat has changed to concat:str and
concat:vchar (one argument) to make it 1+ variadic

### Features

* add center function ([#282](https://github.com/substrait-io/substrait/issues/282)) ([7697d39](https://github.com/substrait-io/substrait/commit/7697d397aaf53999d6eca7799bb4535f30af4e45))
* add coalesce function ([#301](https://github.com/substrait-io/substrait/issues/301)) ([63c5da0](https://github.com/substrait-io/substrait/commit/63c5da0173369ce3d7667da6a30c9581057fa890))
* add dwrf file format ([#304](https://github.com/substrait-io/substrait/issues/304)) ([0f7c2ea](https://github.com/substrait-io/substrait/commit/0f7c2eae469f8bf92905230bbed0d6e88dff7f40))
* add exp function ([#299](https://github.com/substrait-io/substrait/issues/299)) ([7ed31f6](https://github.com/substrait-io/substrait/commit/7ed31f60e58aeff0b5e17af1be3fa0fba24b7ae1))
* add factorial scalar function ([#300](https://github.com/substrait-io/substrait/issues/300)) ([a4d6f35](https://github.com/substrait-io/substrait/commit/a4d6f35f3d12c50d45e15ac974f5cc366e4aa905))
* add hyperbolic functions ([#290](https://github.com/substrait-io/substrait/issues/290)) ([4252824](https://github.com/substrait-io/substrait/commit/4252824264025f69352cf16cd6f886cd4b30df48))
* add log1p function ([#273](https://github.com/substrait-io/substrait/issues/273)) ([55e8275](https://github.com/substrait-io/substrait/commit/55e827519d70b466e748e5c3fef3a568733a9076))
* add regexp_match_substring, regexp_strpos, and regexp_count_substring ([#293](https://github.com/substrait-io/substrait/issues/293)) ([6b8191f](https://github.com/substrait-io/substrait/commit/6b8191f304d28171dfc8edb5a82c1e254284cd5b))
* add regexp_replace function ([#281](https://github.com/substrait-io/substrait/issues/281)) ([433d049](https://github.com/substrait-io/substrait/commit/433d0493b66d67c048f5e41017c6fdcd521b92d9))
* add string transform functions ([#267](https://github.com/substrait-io/substrait/issues/267)) ([ff2f7f1](https://github.com/substrait-io/substrait/commit/ff2f7f1da8ea38452a3760ccc8a232cd8f59cfee))
* clarify behavior of is_null, is_not_null, is_nan, is_finite, and is_infinite for nulls ([#285](https://github.com/substrait-io/substrait/issues/285)) ([cb25124](https://github.com/substrait-io/substrait/commit/cb25124d2d12f629a2f6335bb4f2563c1745758f))

## [0.12.0](https://github.com/substrait-io/substrait/compare/v0.11.0...v0.12.0) (2022-08-28)


### Features

* add between function ([#287](https://github.com/substrait-io/substrait/issues/287)) ([aad6f63](https://github.com/substrait-io/substrait/commit/aad6f637a19c96f02effc7bd5068f4c2d11525c4))
* add case_sensitivity option to string functions ([#289](https://github.com/substrait-io/substrait/issues/289)) ([4c354de](https://github.com/substrait-io/substrait/commit/4c354de568ac5448053b1b11a6373fe0b7e7a229))

## [0.11.0](https://github.com/substrait-io/substrait/compare/v0.10.0...v0.11.0) (2022-08-21)


### Features

* add nullif function ([#291](https://github.com/substrait-io/substrait/issues/291)) ([dc677c2](https://github.com/substrait-io/substrait/commit/dc677c226623489786f0def03db2a8c1e0d0116f))
* **set:** add basic set membership operations ([#280](https://github.com/substrait-io/substrait/issues/280)) ([1bd1bd1](https://github.com/substrait-io/substrait/commit/1bd1bd1aa01e11bf769bfc68fbccb81920a46677))

## [0.10.0](https://github.com/substrait-io/substrait/compare/v0.9.0...v0.10.0) (2022-08-14)


### Features

* add and_not boolean function ([#276](https://github.com/substrait-io/substrait/issues/276)) ([8af3fe0](https://github.com/substrait-io/substrait/commit/8af3fe0e874d8006430699628adfc755d4a1a1b0))
* add is_finite and is_infinite ([#286](https://github.com/substrait-io/substrait/issues/286)) ([01d5428](https://github.com/substrait-io/substrait/commit/01d54287f69635b463832c8b84a75a8fa90f9f3f))
* add support for DDL and INSERT/DELETE/UPDATE operations ([#252](https://github.com/substrait-io/substrait/issues/252)) ([cbb6c26](https://github.com/substrait-io/substrait/commit/cbb6c26e16bced5187c779eaa7027c90461e3e2e))

## [0.9.0](https://github.com/substrait-io/substrait/compare/v0.8.0...v0.9.0) (2022-07-31)


### ⚠ BREAKING CHANGES

* **arithmetic:** Options SILENT, SATURATE, ERROR are no longer valid for use with floating point
arguments to add, subtract, multiply or divide
* function argument bindings were open to interpretation
before, and were often produced incorrectly; therefore, this change
semantically shifts some responsibilities from the consumers to the
producers.
* the grouping set index column now only exists if there is more
than one grouping set.
* Existing plans that are modeling `cast` with the `cast`
function (as opposed to the `cast` expression) will no longer be valid. All
producers/consumers should use the `cast` expression type.

### Features

* add functions for arithmetic, rounding, logarithmic, and string transformations ([#245](https://github.com/substrait-io/substrait/issues/245)) ([f7c5da5](https://github.com/substrait-io/substrait/commit/f7c5da5625f50514ba70b9e8a32cb2e7085c24f1))
* add standard deviation functions ([#257](https://github.com/substrait-io/substrait/issues/257)) ([1339534](https://github.com/substrait-io/substrait/commit/13395340f6971f705e43f304005ea540d04780ce))
* add string containment functions ([#256](https://github.com/substrait-io/substrait/issues/256)) ([d6b9b34](https://github.com/substrait-io/substrait/commit/d6b9b344f0f0865573a79feb6ec818c146b47f62))
* add string trimming and padding functions ([#248](https://github.com/substrait-io/substrait/issues/248)) ([8a8f65d](https://github.com/substrait-io/substrait/commit/8a8f65d3860ce8fc09424947b4fb45b8dd21cef0))
* add trigonometry functions ([#241](https://github.com/substrait-io/substrait/issues/241)) ([d83d566](https://github.com/substrait-io/substrait/commit/d83d566851a0fb5d35c2b23ed8aa549b88d6a63b))
* add variance function ([#263](https://github.com/substrait-io/substrait/issues/263)) ([b6c3772](https://github.com/substrait-io/substrait/commit/b6c377216687a6e253d4b7ec77b48a886cfb501a))
* **arithmetic:** add abs and sign to scalar function extensions ([#244](https://github.com/substrait-io/substrait/issues/244)) ([1b9a45f](https://github.com/substrait-io/substrait/commit/1b9a45fd4f4ea9f9db3d3e7132c5db4d06c05e77))
* support window functions ([#224](https://github.com/substrait-io/substrait/issues/224)) ([4b2072a](https://github.com/substrait-io/substrait/commit/4b2072a40447a4f1a3f6875fa0476cc57145ba30))


### Bug Fixes

* **message:** commit lint issue ([#250](https://github.com/substrait-io/substrait/issues/250)) ([34ec8f5](https://github.com/substrait-io/substrait/commit/34ec8f570b7782c1d26bc6d237d461f211dd8078))
* removes cast function definition ([#253](https://github.com/substrait-io/substrait/issues/253)) ([66a3476](https://github.com/substrait-io/substrait/commit/66a347603bd0a2cba27d749864a9bdb1164eb1dd)), closes [#88](https://github.com/substrait-io/substrait/issues/88) [#152](https://github.com/substrait-io/substrait/issues/152)
* specify how function arguments are to be bound ([#231](https://github.com/substrait-io/substrait/issues/231)) ([d4cfbe0](https://github.com/substrait-io/substrait/commit/d4cfbe014e9c126ac008094323a2baca9f47c42d))


### Documentation

* better explain aggregate relations ([#260](https://github.com/substrait-io/substrait/issues/260)) ([42d9ca3](https://github.com/substrait-io/substrait/commit/42d9ca31a032e1fac0248a998501241eaa27b56f))


### Code Refactoring

* **arithmetic:** specify FP overflow and domain options for remaining ops ([#269](https://github.com/substrait-io/substrait/issues/269)) ([de64a3c](https://github.com/substrait-io/substrait/commit/de64a3c8879c6e0219dd405ce18659219ead926a))

## [0.8.0](https://github.com/substrait-io/substrait/compare/v0.7.0...v0.8.0) (2022-07-17)


### ⚠ BREAKING CHANGES

* The signature of divide functions for multiple types now specify an enumeration prior to specifying operands.

### Bug Fixes

* add overflow behavior to integer division ([#223](https://github.com/substrait-io/substrait/issues/223)) ([cf552d7](https://github.com/substrait-io/substrait/commit/cf552d7c76da9a91bce992391356c6ffb5a969ac))

## [0.7.0](https://github.com/substrait-io/substrait/compare/v0.6.0...v0.7.0) (2022-07-11)


### Features

* introduce compound (parameterizable) extension types and variations ([#196](https://github.com/substrait-io/substrait/issues/196)) ([a79eb07](https://github.com/substrait-io/substrait/commit/a79eb07a15cfa157e795f028a83f746967c98805))

## [0.6.0](https://github.com/substrait-io/substrait/compare/v0.5.0...v0.6.0) (2022-06-26)


### Features

* add contains, starts_with and ends_with functions definitions ([#228](https://github.com/substrait-io/substrait/issues/228)) ([a5fa851](https://github.com/substrait-io/substrait/commit/a5fa85153ffbf7005b9039e06f502a9cc8a732f0))


### Bug Fixes

* fix binary serialization idl link ([#229](https://github.com/substrait-io/substrait/issues/229)) ([af0b452](https://github.com/substrait-io/substrait/commit/af0b45247692dc4bb8fbd25c7f8ec59ff49dbc36))

## [0.5.0](https://github.com/substrait-io/substrait/compare/v0.4.0...v0.5.0) (2022-06-12)


### ⚠ BREAKING CHANGES

* The `substrait/ReadRel/LocalFiles/format` field is deprecated. This will cause a hard break in compatibility. Newer consumers will not be able to read older files. Older consumers will not be able to read newer files. One should now express format concepts using the file_format oneof field.

Co-authored-by: Jacques Nadeau <jacques@apache.org>

### Features

* add aggregate function min/max support ([#219](https://github.com/substrait-io/substrait/issues/219)) ([48b6b12](https://github.com/substrait-io/substrait/commit/48b6b12ebf74c3cc38d4381b950e2caaeb4eef78))
* add Arrow and Orc file formats ([#169](https://github.com/substrait-io/substrait/issues/169)) ([43be00a](https://github.com/substrait-io/substrait/commit/43be00a73abd90fe8f0cafef2b8da9b078d1f243))
* support nullable and non-default variation user-defined types ([#217](https://github.com/substrait-io/substrait/issues/217)) ([5851b02](https://github.com/substrait-io/substrait/commit/5851b02d29aafe44cd804f4248b95b0593878c0a))

## [0.4.0](https://github.com/substrait-io/substrait/compare/v0.3.0...v0.4.0) (2022-06-05)


### ⚠ BREAKING CHANGES

* there was an accidental inclusion of a binary `not` function with unspecified behavior. This function was removed. Use the unary `not` function to return the compliment of an input argument.

### Bug Fixes

* remove not function that expects two arguments ([#182](https://github.com/substrait-io/substrait/issues/182)) ([e06067c](https://github.com/substrait-io/substrait/commit/e06067c991ddc34b2720408ed7e1ca5152774a29))

## [0.3.0](https://github.com/substrait-io/substrait/compare/v0.2.0...v0.3.0) (2022-05-22)


### Features

* support type function arguments in protobuf ([#161](https://github.com/substrait-io/substrait/issues/161)) ([df98816](https://github.com/substrait-io/substrait/commit/df988163a5afcebe8823b9e466c3e1923c3b9e79))
* define APPROX_COUNT_DISTINCT in new yaml for approximate aggregate functions ([#204](https://github.com/substrait-io/substrait/issues/204)) ([8e206b9](https://github.com/substrait-io/substrait/commit/8e206b9594880886c513c8437663fac15e0dfe59))
* literals for extension types ([#197](https://github.com/substrait-io/substrait/issues/197)) ([296c266](https://github.com/substrait-io/substrait/commit/296c2661de007a2d8f41d3fe242a1f4b6e60c9e1))
* support fractional seconds for interval_day literals ([#199](https://github.com/substrait-io/substrait/issues/199)) ([129e52f](https://github.com/substrait-io/substrait/commit/129e52f2519db00d6cef35f3faa3bc9e1ff1e890))

## [0.2.0](https://github.com/substrait-io/substrait/compare/v0.1.2...v0.2.0) (2022-05-15)


### Features

* add flag FailureBehavior in Cast expression ([#186](https://github.com/substrait-io/substrait/issues/186)) ([a3d3b2f](https://github.com/substrait-io/substrait/commit/a3d3b2f5ccc6e8375a950290eda09489c7fb30e7))
* add invocation property to AggregateFunction message for specifying distinct vs all ([#191](https://github.com/substrait-io/substrait/issues/191)) ([373b33f](https://github.com/substrait-io/substrait/commit/373b33f62b1e8f026718bc3b55cbe267421a1abb))

### [0.1.2](https://github.com/substrait-io/substrait/compare/v0.1.1...v0.1.2) (2022-05-01)


### Bug Fixes

* **docs:** use conventionalcommits to show breaking changes first ([#181](https://github.com/substrait-io/substrait/issues/181)) ([b7f2587](https://github.com/substrait-io/substrait/commit/b7f2587f492071bed2250eb6f04c0b8123e715e1))

## [0.1.1](https://github.com/substrait-io/substrait/compare/v0.1.0...v0.1.1) (2022-04-28)


### Bug Fixes

* **ci:** cd into buf-configured proto directory ([#180](https://github.com/substrait-io/substrait/issues/180)) ([78c0781](https://github.com/substrait-io/substrait/commit/78c0781f72cae2f4445a708ae3ccf0c2c3eb9725))

# [0.1.0](https://github.com/substrait-io/substrait/compare/v0.0.0...v0.1.0) (2022-04-28)


### Bug Fixes

* add missing switch expression ([#160](https://github.com/substrait-io/substrait/issues/160)) ([4db2a9f](https://github.com/substrait-io/substrait/commit/4db2a9fb7e7849c73adcd21d1b06fb7e8df73fae))


### Features

* add subquery representation ([#134](https://github.com/substrait-io/substrait/issues/134)) ([3670518](https://github.com/substrait-io/substrait/commit/3670518d37c53660d496860f81c761ccb0afbce0))
