## [1.2.18](github.com/uptrace/bun/compare/v1.2.17...v1.2.18) (2026-02-28)


### Bug Fixes

* handle []byte and [N]byte in Tuple, separate List from Tuple imp… ([#1340](/github.com/uptrace/bun/issues/1340)) ([bec98b9](github.com/uptrace/bun/commits/bec98b975ad093cd34c0a7821120350e475b901b))
* validate parenthesized content in ReadIdentifier to prevent ?(?, ?) misparse ([#1338](/github.com/uptrace/bun/issues/1338)) ([b8da15b](github.com/uptrace/bun/commits/b8da15b391fd3b78f543125dc574ad32a240780c)), closes [#1337](github.com/uptrace/bun/issues/1337)



## [1.2.17](github.com/uptrace/bun/compare/v1.2.16...v1.2.17) (2026-02-21)


### Bug Fixes

* migrator exec error propagation ([#1320](/github.com/uptrace/bun/issues/1320)) ([b40f603](github.com/uptrace/bun/commits/b40f603ffc4595894c2178ab1841122abaead50d))
* OrderAscNullsFirst mapping (fixes [#1305](/github.com/uptrace/bun/issues/1305)) ([43b6af2](github.com/uptrace/bun/commits/43b6af233feebed72a911b03dbccd63247c262de))
* panic in indirectAsKey when loading complex models. TypeOf(v) returns nil ([2788c5b](github.com/uptrace/bun/commits/2788c5b2e052b21cb1abf9af2a6da8a6ad43993c))
* RunMigration marks migration as applied after running ([#1330](/github.com/uptrace/bun/issues/1330)) ([990c2eb](github.com/uptrace/bun/commits/990c2ebf34e3e2c59979943fabdda2892c2c3dcc))


### Features

* add Tuple and List ([#1331](/github.com/uptrace/bun/issues/1331)) ([5c2b3d1](github.com/uptrace/bun/commits/5c2b3d150688ca6861d20fd590ea20cd415749a3))
* create unique index on migration name column in Migrator.Init ([#1332](/github.com/uptrace/bun/issues/1332)) ([44ac056](github.com/uptrace/bun/commits/44ac056ac35dc396dcbb09252ce69f1ccd9b0250))
* **update:** use DEFAULT instead of NULL on databases that support it ([#1315](/github.com/uptrace/bun/issues/1315)) ([cabcffd](github.com/uptrace/bun/commits/cabcffd47e35141064e4e6e15ea97675393098a0))



## [1.2.16](github.com/uptrace/bun/compare/v1.2.15...v1.2.16) (2025-11-20)


### Bug Fixes

* data race in db clone stats ([e92d910](github.com/uptrace/bun/commits/e92d91041144037c270db1f87d7b5750d52d88ab))
* **db:** data race in db clone stats ([a78f382](github.com/uptrace/bun/commits/a78f38251b0ca990e55868953cd85e3db7cadcc5))
* **db:** move DBStats to noCopyState ([c646241](github.com/uptrace/bun/commits/c64624188efb7b049a6e70d064566a6613fb9974))
* return "custom" for unknown dialects instead of "invalid" ([#1280](/github.com/uptrace/bun/issues/1280)) ([106cc08](github.com/uptrace/bun/commits/106cc081023efe0662375fc9e6b9de9b0b06f6b0)), closes [#1276](github.com/uptrace/bun/issues/1276)
* revert CreateChannel rename ([#1248](/github.com/uptrace/bun/issues/1248)) ([a5b2ac6](github.com/uptrace/bun/commits/a5b2ac63e478da9011640df2612e33094b5fc90b))
* sql injection [#1228](/github.com/uptrace/bun/issues/1228) ([#1263](/github.com/uptrace/bun/issues/1263)) ([c12edf0](github.com/uptrace/bun/commits/c12edf090a9f7b17ea04f121c8b25603590e845b))
* update `SelectQuery.Clone` to properly handle non-nil empty arg slices ([#1299](/github.com/uptrace/bun/issues/1299)) ([b499cce](github.com/uptrace/bun/commits/b499cce68bff4c2c51d8647dd1ca17c0451f468e)), closes [#1298](github.com/uptrace/bun/issues/1298)


### Features

* add Context to ConnResolver.ResolveConn ([#1275](/github.com/uptrace/bun/issues/1275)) ([d9f273f](github.com/uptrace/bun/commits/d9f273fc119ff2097883fffeb34d2e2c6074accd))
* add materialize cte support  ([#1260](/github.com/uptrace/bun/issues/1260)) ([16ebb09](github.com/uptrace/bun/commits/16ebb0916c2bffbec6bdb2af4c227d83fefdb4ae))
* add SetValues ([#1252](/github.com/uptrace/bun/issues/1252)) ([9556d3c](github.com/uptrace/bun/commits/9556d3c4fb5fb79d9295a5937d46052a0e07fb2f))
* add SortDir type to safely build order queries ([#1284](/github.com/uptrace/bun/issues/1284)) ([2ad0521](github.com/uptrace/bun/commits/2ad05214e57548e0b6dccf487401335cbd4e4ed9))
* add WithQueryHook and deprecated AddQueryHook ([#1272](/github.com/uptrace/bun/issues/1272)) ([f662c1e](github.com/uptrace/bun/commits/f662c1ee6f49306eea50ceceb23d2da5b83d28cc))
* **migrate:** add BeforeMigration and AfterMigration ([#1273](/github.com/uptrace/bun/issues/1273)) ([042b10a](github.com/uptrace/bun/commits/042b10aa50f6b0f532a59de4ce81fbddb6a3b739))
* **pgdialect:** add support for RETURNING clause in MERGE statements ([#1258](/github.com/uptrace/bun/issues/1258)) ([6f4c688](github.com/uptrace/bun/commits/6f4c68874cd3324c863253d10ba4989365837b61))



## [1.2.15](https://github.com/uptrace/bun/compare/v1.2.14...v1.2.15) (2025-07-17)


### Bug Fixes

* **pgdriver:** add mandatory space before negative numbers to resolve CVE-2024-34359 ([8067a8f](https://github.com/uptrace/bun/commit/8067a8f13f8d22fb57b76d6800f7aefc12b044cd))


### Features

* **db:** rename CleanQueryHook to ResetQueryHooks ([cb17679](https://github.com/uptrace/bun/commit/cb176796f5fbae8b3ea44e67875dd00ecf689425))
* **db:** support clean query hooks ([a5f19a7](https://github.com/uptrace/bun/commit/a5f19a7c0d68fd44eaff99ebaaeb88ca089d7538)), closes [#1226](https://github.com/uptrace/bun/issues/1226)
* **dialect:** return default on update/delete when create table ([d347b48](https://github.com/uptrace/bun/commit/d347b48c7764a23000a28ca3ad40368b8b89e298)), closes [#1212](https://github.com/uptrace/bun/issues/1212)



## [1.2.14](https://github.com/uptrace/bun/compare/v1.2.13...v1.2.14) (2025-06-16)


### Bug Fixes

* restore q.limit check ([07d32c1](https://github.com/uptrace/bun/commit/07d32c1662015a398322fdbc0dc34c5f0d10ce44))



## [1.2.13](https://github.com/uptrace/bun/compare/v1.2.12...v1.2.13) (2025-06-11)


### Bug Fixes

* **query:** scanAndCount without model ([07fb7ec](https://github.com/uptrace/bun/commit/07fb7ec540979d0625cfeb771a0679c5982c6e2a)), closes [#1209](https://github.com/uptrace/bun/issues/1209)
* sort fk constraints before appending ([c87fa90](https://github.com/uptrace/bun/commit/c87fa903c56743e24a2cb677e8e96fd5c802fba5))
* use slices sort ([8555900](https://github.com/uptrace/bun/commit/8555900ad840d9b6e73c8655af4f1b6766bc943b))



## [1.2.12](https://github.com/uptrace/bun/compare/v1.2.11...v1.2.12) (2025-06-05)


### Bug Fixes

* **automigrate:** append SQL to separate []byte slices ([f44a349](https://github.com/uptrace/bun/commit/f44a349ec61b09f9f0240a923e121cbaa3ab1d14))
* **gh-1160:** add WithExcludeForeignKeys option ([63141cb](https://github.com/uptrace/bun/commit/63141cb6c9a6d0d2abf4b41eac5b1c6078884326)), closes [#1160](https://github.com/uptrace/bun/issues/1160)
* iss-824 to allow mssql to support non unicode strings ([0565763](https://github.com/uptrace/bun/commit/056576355a0a7ff75f616cedb5d81144f6657a6a))
* **migrations:** skip template rendering if no data + fix tests ([4055827](https://github.com/uptrace/bun/commit/4055827e1af4f0b7e13879d393c1131ab497d962))
* **pgdriver:** rename channelOverflowHandler to ChannelOverflowHandler for public API ([65760a9](https://github.com/uptrace/bun/commit/65760a9e648a1ae379982e5d8737d6d864f6a8e3))
* relation join data race ([37971d7](https://github.com/uptrace/bun/commit/37971d7f83042ab83e52be1c122083f8a98a1efa))
* report BIGSERIAL ~ BIGINT in pgdialect ([ad7356a](https://github.com/uptrace/bun/commit/ad7356a772324950cf866b86d23771fc53f83505))
* skip automigrator test early ([5b22710](https://github.com/uptrace/bun/commit/5b22710f0b4d980ebec38fcd306bf459dc1eb615))
* start sequence with last+1 ([7fbf34a](https://github.com/uptrace/bun/commit/7fbf34a69ff249c72af522331a4f6116f240630a))


### Features

* add support for netip.Addr and netip.Prefix ([63ccc8f](https://github.com/uptrace/bun/commit/63ccc8f530092c3dfc71179b94a43db452fa54ec))
* exclude tables using LIKE pattern ([5351f7e](https://github.com/uptrace/bun/commit/5351f7ed4fe53662386e697cc551ba54487da018))
* **migrations:** support Go templates in SQL migrations ([d92e29e](https://github.com/uptrace/bun/commit/d92e29e459ae2804ad48e1b4f6a8147211a47a57))
* **pg:** allow user config buffer size of pg's connect ([e2f2650](https://github.com/uptrace/bun/commit/e2f2650950d13442d45694b7cd186b77b4e8e0bb)), closes [#1201](https://github.com/uptrace/bun/issues/1201)
* **pgdriver:** add option for tracing ([80c5e3c](https://github.com/uptrace/bun/commit/80c5e3c684c410dfc02170cfb8671bb8b1db2e35)), closes [#1150](https://github.com/uptrace/bun/issues/1150)
* **pgdriver:** add overflow handler to listener channel ([6f0e3a1](https://github.com/uptrace/bun/commit/6f0e3a1d33de5a61625d22ba6464bfe5da404a11))
* set notnull=true for autoincrement columns ([1bd5dd7](https://github.com/uptrace/bun/commit/1bd5dd73ce943235a403c5896b6e70401b194093))
* support changing column type to SERIAL ([136b480](https://github.com/uptrace/bun/commit/136b480e6835dd9a12b4925f57225fb73d0aa7ae))



## [1.2.11](https://github.com/uptrace/bun/compare/v1.2.10...v1.2.11) (2025-03-05)


### Bug Fixes

* always use the value returned by implemented driver.Valuer ([0c29af6](https://github.com/uptrace/bun/commit/0c29af65f17891d15019e60f64704e9c45204062))
* handle driver.Valuer in getRealValue ([fa37c7b](https://github.com/uptrace/bun/commit/fa37c7b91e570ca032d01d7311245a07b52dbed8))
* only handle pointer-based driver.Valuer implementations ([40b20cd](https://github.com/uptrace/bun/commit/40b20cd207a22b8b8f86ec36c62385f6293c192a))
* **schema:** determine whether a field is ambiguous with prefix ([83f6f99](https://github.com/uptrace/bun/commit/83f6f992bf38a654207b27fcc3bd4ea1984c9acb))
* **schema:** process embed with struct ([a06003d](https://github.com/uptrace/bun/commit/a06003d867168a663b1ad223bbed85b3d94fd920)), closes [#1136](https://github.com/uptrace/bun/issues/1136)
* **test:** define uuid type for pointer primary keys ([3b72bd4](https://github.com/uptrace/bun/commit/3b72bd4cd045aa8061b7ca8b1cb00eae6c4016f0))
* **test:** use varchar to be compatible with multiple databases ([287b0e3](https://github.com/uptrace/bun/commit/287b0e386feeab7391b749723c32377e5315a870))
* **typo:** minor typo fix in `migrate/auto.go` ([368ed3f](https://github.com/uptrace/bun/commit/368ed3f2e2a65fbad50b26080efb33366b793e83))



## [1.2.10](https://github.com/uptrace/bun/compare/v1.2.9...v1.2.10) (2025-02-18)


### Bug Fixes

* clone query in scanAndCountConcurrently to avoid data race ([66fdc39](https://github.com/uptrace/bun/commit/66fdc39b33a482534920578ed8c7f88c3f142a3d)), closes [#1117](https://github.com/uptrace/bun/issues/1117)
* **create_table:** avoid creating unintended foreign keys ([#1130](https://github.com/uptrace/bun/issues/1130)) ([187743b](https://github.com/uptrace/bun/commit/187743b1e743755cd57a9cc11e7f2f9cea0a7dcd))
* **pgdialect:** handle []*time.Time arrays ([4c4e12a](https://github.com/uptrace/bun/commit/4c4e12aa7f27cf49189427da5104afb436af4348))
* **pgdialect:** handle nill array on jsonb column ([0dc4e3e](https://github.com/uptrace/bun/commit/0dc4e3edb3f9021b02ed6f80d54cb88d2ef9b025))
* **pgdialect:** postgres syntax errors for slices of pointers and json arrays [#877](https://github.com/uptrace/bun/issues/877) ([1422b77](https://github.com/uptrace/bun/commit/1422b7726a24ac55ee6ca0e15ec084c34f7b1bd6))
* process embedded's struct field for table ([b410e42](https://github.com/uptrace/bun/commit/b410e420ab888d87d2b6ebb014f13baae8fdc2b7)), closes [#1125](https://github.com/uptrace/bun/issues/1125)


### Features

* add DBReplica and use it in ReadWriteConnResolver ([95c825e](https://github.com/uptrace/bun/commit/95c825e1215b26456caeebf1893d3b6183202bae))
* allow setting a query comment through a context value ([9f5ccfe](https://github.com/uptrace/bun/commit/9f5ccfea7144c7ced877e3ce6972c589c5c6c3e6))
* **bunotel:** ability to override span names ([04e2125](https://github.com/uptrace/bun/commit/04e21253298ee495179754fbbfccc047468a034d))
* **bunotel:** always record affected rows ([960a304](https://github.com/uptrace/bun/commit/960a3046ad0cc8ea548dc448380549f610cb5da4))



## [1.2.9](https://github.com/uptrace/bun/compare/v1.2.8...v1.2.9) (2025-01-26)


### Bug Fixes

* apply join condition to select with count ([e77b9e7](https://github.com/uptrace/bun/commit/e77b9e72fa5ae8e173d506a4e154ba64214c4aff)), closes [#597](https://github.com/uptrace/bun/issues/597)
* build ([702e525](https://github.com/uptrace/bun/commit/702e525e30ec93b6d4611359518e1008b67744af))
* individual replica timeout ([9f5e8b1](https://github.com/uptrace/bun/commit/9f5e8b1c46673bd1779bd4309a28db33dcd695bf))
* test ([dfc4059](https://github.com/uptrace/bun/commit/dfc405901907419d043bb6ced3ad20c131c1b972))


### Features

* add feature flag AlterColumnExists ([fc35e12](https://github.com/uptrace/bun/commit/fc35e1222242b3d581f0b7496a9021aadfc50b07)), closes [#704](https://github.com/uptrace/bun/issues/704)
* add Options ([815e11a](https://github.com/uptrace/bun/commit/815e11a023d2babf65d528a20ddffc7628636e7e))
* allow to specify read-only replica for SELECTs ([cbbe1e9](https://github.com/uptrace/bun/commit/cbbe1e94fd0c72d1870395a663c8053d7e8c6ace))
* downgrade to use the field in has-many-relation ([91e0d27](https://github.com/uptrace/bun/commit/91e0d2719a5a20b3208cea0232e2dbcb452d6c23)), closes [#1107](https://github.com/uptrace/bun/issues/1107)
* make WithReadOnlyReplica variadic ([4cbb15a](https://github.com/uptrace/bun/commit/4cbb15a53e566e03284253aa46be372338968954))
* **pgdialect:** allow to convert uint to int ([7d22ddd](https://github.com/uptrace/bun/commit/7d22ddd263b28b9fd6e172e0208c124b7c56f111))
* **pgdriver:** improve otel instrumentation ([c40e4f3](https://github.com/uptrace/bun/commit/c40e4f3c50c710903236dc89b56a843a0351a21a))



## [1.2.8](https://github.com/uptrace/bun/compare/v1.2.7...v1.2.8) (2025-01-06)


### Bug Fixes

* comment string zero bytes filtering ([34dfd68](https://github.com/uptrace/bun/commit/34dfd684e371c24b9f59e9b13ef57660931f0bde))
* get m2m table's structKey with driver.Valuer ([f107314](https://github.com/uptrace/bun/commit/f1073147dc73d01dcf8a6ee9252d354ff06a1062)), closes [#1100](https://github.com/uptrace/bun/issues/1100)
* return error when use dest with has-many/many-to-many ([8296774](https://github.com/uptrace/bun/commit/829677486b502e6d5d2ae37814488ae9f2c7386e)), closes [#606](https://github.com/uptrace/bun/issues/606)
* support scan float32 to float32/float64 ([a52e733](https://github.com/uptrace/bun/commit/a52e7339a93f84468878dcaffc42536faa44efae)), closes [#1087](https://github.com/uptrace/bun/issues/1087)


### Features

* add RelationWithOpts method to SelectQuery ([dd3ef52](https://github.com/uptrace/bun/commit/dd3ef522c8a9c656958b73ee5d546854fb7c6edf))
* enhance debugging by adding query comments in headers ([1376d18](https://github.com/uptrace/bun/commit/1376d1870bfe3d89e3630203787f1e87c503d5df))
* sort fields by struct ([5edb672](https://github.com/uptrace/bun/commit/5edb672e320be9b210f06d25c4f4b9e761c1c526)), closes [#1095](https://github.com/uptrace/bun/issues/1095)



## [1.2.7](https://github.com/uptrace/bun/compare/v1.2.6...v1.2.7) (2025-01-01)


### Bug Fixes

* do not create new migrations if nothing to migrate ([5cc961d](https://github.com/uptrace/bun/commit/5cc961d6cc461ad3534728fc4d3cae12bf8b736e))
* has many relation with driver.Valuer ([cb8c42c](https://github.com/uptrace/bun/commit/cb8c42cd3f65d95865c76a594abad815eea1df3c))
* improve range type to support driver.Valuer and sql.Scanner ([856e12b](https://github.com/uptrace/bun/commit/856e12b0d37275a6aa247370f6a8231fd89ca3e7))
* pass correct 'transactional' parameter ([ebdef1b](https://github.com/uptrace/bun/commit/ebdef1b0e9d33a5ca475ab4c2ec2fb44d11d4595))
* **pgdialect:** remove unsigned integer conversion ([ab3c679](https://github.com/uptrace/bun/commit/ab3c679d529dd20d44e789dc6f1d89f9510bde0b)), closes [uptrace/bun#624](https://github.com/uptrace/bun/issues/624)
* remove unused param on table.go and tables.go: canAddr ([d563e2d](https://github.com/uptrace/bun/commit/d563e2dbe95caeb0e00ad1b3e82283431747fe7b))
* replace the link to docs repo in CONTRIBUTING.md ([e120096](https://github.com/uptrace/bun/commit/e12009662ae1ddefcc1337cc5e32e73d77c7def0))
* trim surrounding '' in string literal in DEFAULT clause ([a0dff72](https://github.com/uptrace/bun/commit/a0dff72b6ab0ca24d00c96c923046200dd6112ed))


### Features

* add an ordered map to remove unnecessary dependencies ([9fea143](https://github.com/uptrace/bun/commit/9fea1437d8344d836670e802fd12d3476e8cad86))
* support disable dialect's feature ([5343bd7](https://github.com/uptrace/bun/commit/5343bd7fc4ceda866a7d607388ebb7a89f7f5823))



## [1.2.6](https://github.com/uptrace/bun/compare/v1.2.5...v1.2.6) (2024-11-20)


### Bug Fixes

* append IDENTITY to ADD COLUMN statement if needed ([694f873](https://github.com/uptrace/bun/commit/694f873d61ed8d2f09032ae0c0dbec4b71c3719e))
* **ci:** prune stale should be executed at 3 AM every day ([0cedcb0](https://github.com/uptrace/bun/commit/0cedcb068229b63041a4f48de12bb767c8454048))
* cleanup after testUniqueRenamedTable ([b1ae32e](https://github.com/uptrace/bun/commit/b1ae32e9e9f45ff2a66e50bfd13bedcf6653d874))
* fix go.mod of oracledialect ([89e21ea](https://github.com/uptrace/bun/commit/89e21eab362c60511cca00890ae29551a2ba7c46))
* has many relationship with multiple columns ([1664b2c](https://github.com/uptrace/bun/commit/1664b2c07a5f6cfd3b6730e5005373686e9830a6))
* ignore case for type equivalence ([c3253a5](https://github.com/uptrace/bun/commit/c3253a5c59b078607db9e216ddc11afdef546e05))
* implement DefaultSchema for Oracle dialect ([d08fa40](https://github.com/uptrace/bun/commit/d08fa40cc87d67296a83a77448ea511531fc8cdd))
* **oracledialect:** add go.mod file so the dialect is released properly ([#1043](https://github.com/uptrace/bun/issues/1043)) ([1bb5597](https://github.com/uptrace/bun/commit/1bb5597f1a32f5d693101ef4a62e25d99f5b9db5))
* **oracledialect:** update go.mod by go mod tidy to fix tests ([7f90a15](https://github.com/uptrace/bun/commit/7f90a15c51a2482dda94226dd13b913d6b470a29))
* **pgdialect:** array value quoting ([892c416](https://github.com/uptrace/bun/commit/892c416272a8428c592896d65d3ad51a6f2356d8))
* remove schema name from t.Name during bun-schema inspection ([31ed582](https://github.com/uptrace/bun/commit/31ed58254ad08143d88684672acd33ce044ea5a9))
* rename column only if the name does not exist in 'target' ([fed6012](https://github.com/uptrace/bun/commit/fed6012d177e55b8320b31ef37fc02a0cbf0b9f5))
* support embed with tag Unique ([3acd6dd](https://github.com/uptrace/bun/commit/3acd6dd8546118d7b867ca796a5e56311edad070))
* update oracledialect/version.go in release.sh ([bcd070f](https://github.com/uptrace/bun/commit/bcd070f48a75d0092a5620261658c9c5994f0bf6))
* update schema.Field names ([9b810de](https://github.com/uptrace/bun/commit/9b810dee4b1a721efb82c913099f39f52c44eb57))


### Features

* add and drop columns ([3fdd5b8](https://github.com/uptrace/bun/commit/3fdd5b8f635f849a74e78c665274609f75245b19))
* add and drop IDENTITY ([dd83779](https://github.com/uptrace/bun/commit/dd837795c31490fd8816eec0e9833e79fafdda32))
* add support type for net/netip.addr and net/netip.prefix ([#1028](https://github.com/uptrace/bun/issues/1028)) ([95c4a8e](https://github.com/uptrace/bun/commit/95c4a8ebd634e1e99114727a7b157eeeb9297ee9))
* **automigrate:** detect renamed tables ([c03938f](https://github.com/uptrace/bun/commit/c03938ff5e9fa2f653e4c60668b1368357d2de10))
* change column type ([3cfd8c6](https://github.com/uptrace/bun/commit/3cfd8c62125786aaf6f493acc5b39f4d3db3d628))
* **ci:** support release on osx ([435510b](https://github.com/uptrace/bun/commit/435510b0a73b0d9e6d06e3e3c3f0fa4379e9ed8c))
* create sql migrations and apply them ([1bf7cfd](https://github.com/uptrace/bun/commit/1bf7cfd067e0e26ae212b0f7421e5abc6f67fb4f))
* create transactional migration files ([c3320f6](https://github.com/uptrace/bun/commit/c3320f624830dc2fe99af2c7cbe492b2a83f9e4a))
* detect Create/Drop table ([408859f](https://github.com/uptrace/bun/commit/408859f07be38236b39a00909cdce55d49f6f824))
* detect modified relations ([a918dc4](https://github.com/uptrace/bun/commit/a918dc472a33dd24c5fffd4d048bcf49f2e07a42))
* detect renamed columns ([886d0a5](https://github.com/uptrace/bun/commit/886d0a5b18aba272f1c86af2a2cf68ce4c8879f2))
* detect renamed tables ([8857bab](https://github.com/uptrace/bun/commit/8857bab54b94170d218633f3b210d379e4e51a21))
* enhance Apply method to accept multiple functions ([7823f2f](https://github.com/uptrace/bun/commit/7823f2f24c814e104dc59475156255c7b3b26144))
* implement fmt.Stringer queries ([5060e47](https://github.com/uptrace/bun/commit/5060e47db13451a982e48d0f14055a58ba60b472))
* improve FK handling ([a822fc5](https://github.com/uptrace/bun/commit/a822fc5f8ae547b7cd41e1ca35609d519d78598b))
* include target schema name in migration name ([ac8d221](https://github.com/uptrace/bun/commit/ac8d221e6443b469e794314c5fc189250fa542d5))
* **mariadb:** support RETURNING clause in DELETE statement ([b8dec9d](https://github.com/uptrace/bun/commit/b8dec9d9a06124696bd5ee2abbf33f19087174b6))
* migrate FKs ([4c1dfdb](https://github.com/uptrace/bun/commit/4c1dfdbe99c73d0c0f2d7b1f8b11adf30c6a41f7))
* **mysql:** support ORDER BY and LIMIT clauses in UPDATE and DELETE statements ([de71bed](https://github.com/uptrace/bun/commit/de71bed9252980648269af85b7a51cbc464ce710))
* support modifying primary keys ([a734629](https://github.com/uptrace/bun/commit/a734629fa285406038cbe4a50798626b5ac08539))
* support UNIQUE constraints ([3c4d5d2](https://github.com/uptrace/bun/commit/3c4d5d2c47be4652fb9b5cf1c6bd7b6c0a437287))
* use *bun.DB in MigratorDialect ([a8788bf](https://github.com/uptrace/bun/commit/a8788bf62cbcc954a08532c299c774262de7a81d))



## [1.2.5](https://github.com/uptrace/bun/compare/v1.2.3...v1.2.5) (2024-10-26)


### Bug Fixes

* allow Limit() without Order() with MSSQL ([#1009](https://github.com/uptrace/bun/issues/1009)) ([1a46ddc](https://github.com/uptrace/bun/commit/1a46ddc0d3ca0bdc60ca8be5ad1886799d14c8b0))
* copy bytes in mapModel.Scan ([#1030](https://github.com/uptrace/bun/issues/1030)) ([#1032](https://github.com/uptrace/bun/issues/1032)) ([39fda4e](https://github.com/uptrace/bun/commit/39fda4e3d341e59e4955f751cb354a939e57c1b1))
* fix issue with has-many join and pointer fields ([#950](https://github.com/uptrace/bun/issues/950)) ([#983](https://github.com/uptrace/bun/issues/983)) ([cbc5177](https://github.com/uptrace/bun/commit/cbc517792ba6cdcef1828f3699d3d4dfe3c5e0eb))
* restore explicit column: name override ([#984](https://github.com/uptrace/bun/issues/984)) ([169f258](https://github.com/uptrace/bun/commit/169f258a9460cad451f3025d2ef8df1bbd42a003))
* return column option back ([#1036](https://github.com/uptrace/bun/issues/1036)) ([a3ccbea](https://github.com/uptrace/bun/commit/a3ccbeab39151d3eed6cb245fe15cfb5d71ba557))
* sql.NullString mistaken as custom struct ([#1019](https://github.com/uptrace/bun/issues/1019)) ([87c77b8](https://github.com/uptrace/bun/commit/87c77b8911f2035b0ee8ea96356a2c7600b5b94d))
* typos ([#1026](https://github.com/uptrace/bun/issues/1026)) ([760de7d](https://github.com/uptrace/bun/commit/760de7d0fad15dc761475670a4dde056aef9210d))


### Features

* add transaction isolation level support to pgdriver ([#1034](https://github.com/uptrace/bun/issues/1034)) ([3ef44ce](https://github.com/uptrace/bun/commit/3ef44ce1cdd969a21b76d6c803119cf12c375cb0))


### Performance Improvements

* refactor SelectQuery.ScanAndCount to optimize performance when there is no limit and offset ([#1035](https://github.com/uptrace/bun/issues/1035)) ([8638613](https://github.com/uptrace/bun/commit/86386135897485bbada6c50ec9a2743626111433))



## [1.2.4](https://github.com/uptrace/bun/compare/v1.2.3...v1.2.4) (2024-10-26)


### Bug Fixes

* allow Limit() without Order() with MSSQL ([#1009](https://github.com/uptrace/bun/issues/1009)) ([1a46ddc](https://github.com/uptrace/bun/commit/1a46ddc0d3ca0bdc60ca8be5ad1886799d14c8b0))
* copy bytes in mapModel.Scan ([#1030](https://github.com/uptrace/bun/issues/1030)) ([#1032](https://github.com/uptrace/bun/issues/1032)) ([39fda4e](https://github.com/uptrace/bun/commit/39fda4e3d341e59e4955f751cb354a939e57c1b1))
* return column option back ([#1036](https://github.com/uptrace/bun/issues/1036)) ([a3ccbea](https://github.com/uptrace/bun/commit/a3ccbeab39151d3eed6cb245fe15cfb5d71ba557))
* sql.NullString mistaken as custom struct ([#1019](https://github.com/uptrace/bun/issues/1019)) ([87c77b8](https://github.com/uptrace/bun/commit/87c77b8911f2035b0ee8ea96356a2c7600b5b94d))
* typos ([#1026](https://github.com/uptrace/bun/issues/1026)) ([760de7d](https://github.com/uptrace/bun/commit/760de7d0fad15dc761475670a4dde056aef9210d))


### Features

* add transaction isolation level support to pgdriver ([#1034](https://github.com/uptrace/bun/issues/1034)) ([3ef44ce](https://github.com/uptrace/bun/commit/3ef44ce1cdd969a21b76d6c803119cf12c375cb0))


### Performance Improvements

* refactor SelectQuery.ScanAndCount to optimize performance when there is no limit and offset ([#1035](https://github.com/uptrace/bun/issues/1035)) ([8638613](https://github.com/uptrace/bun/commit/86386135897485bbada6c50ec9a2743626111433))



## [1.2.3](https://github.com/uptrace/bun/compare/v1.2.2...v1.2.3) (2024-08-31)



## [1.2.2](https://github.com/uptrace/bun/compare/v1.2.1...v1.2.2) (2024-08-29)


### Bug Fixes

* gracefully handle empty hstore in pgdialect ([#1010](https://github.com/uptrace/bun/issues/1010)) ([2f73d8a](https://github.com/uptrace/bun/commit/2f73d8a8e16c8718ebfc956036d9c9a01a0888bc))
* number each unit test ([#974](https://github.com/uptrace/bun/issues/974)) ([b005dc2](https://github.com/uptrace/bun/commit/b005dc2a9034715c6f59dcfc8e76aa3b85df38ab))


### Features

* add ModelTableExpr to TruncateTableQuery ([#969](https://github.com/uptrace/bun/issues/969)) ([7bc330f](https://github.com/uptrace/bun/commit/7bc330f152cf0d9dc30956478e2731ea5816f012))



## [1.2.1](https://github.com/uptrace/bun/compare/v1.2.0...v1.2.1) (2024-04-02)



# [1.2.0](https://github.com/uptrace/bun/compare/v1.1.17...v1.2.0) (2024-04-02)


### Bug Fixes

* embedding of scanonly fields ([ed6ed74](https://github.com/uptrace/bun/commit/ed6ed74d5379ea6badb09cc37709211a51f5792b))
* **table:** allow alt annotation ([#956](https://github.com/uptrace/bun/issues/956)) ([8a0397b](https://github.com/uptrace/bun/commit/8a0397b6e2219909d6b00d258eb7934170058edd))
* transactional migration file extension ([#959](https://github.com/uptrace/bun/issues/959)) ([921b15b](https://github.com/uptrace/bun/commit/921b15b80110d28251a9210c77397d29924ffbc5))


### Features

* Allow overriding of Warn and Deprecated loggers ([#952](https://github.com/uptrace/bun/issues/952)) ([0e9d737](https://github.com/uptrace/bun/commit/0e9d737e4ca2deb86930237ee32a39cf3f7e8157))
* enable SNI ([#953](https://github.com/uptrace/bun/issues/953)) ([4071ffb](https://github.com/uptrace/bun/commit/4071ffb5bcb1b233cda239c92504d8139dcf1d2f))
* **idb:** add NewMerge method to IDB ([#966](https://github.com/uptrace/bun/issues/966)) ([664e2f1](https://github.com/uptrace/bun/commit/664e2f154f1153d2a80cd062a5074f1692edaee7))



## [1.1.17](https://github.com/uptrace/bun/compare/v1.1.16...v1.1.17) (2024-01-11)


### Features

* add CreateTxSQLMigrations function ([#916](https://github.com/uptrace/bun/issues/916)) ([c68ec7c](https://github.com/uptrace/bun/commit/c68ec7cfc418959eb7c79028be7ac91f97d462ef))
* add Join to UpdateQuery ([#908](https://github.com/uptrace/bun/issues/908)) ([8c4d8be](https://github.com/uptrace/bun/commit/8c4d8be3aa4e64582698b37fd21434b8960dddc0))
* bunslog.QueryHook for Bun logging using `log/slog` ([#904](https://github.com/uptrace/bun/issues/904)) ([4953367](https://github.com/uptrace/bun/commit/495336731da0a995aa28c7bc84345c7825408e48))
* dbfixture.New to accept IDB interface ([#900](https://github.com/uptrace/bun/issues/900)) ([2dee174](https://github.com/uptrace/bun/commit/2dee174bc4d09a45caeeede2885306e5fd10002d))



## [1.1.16](https://github.com/uptrace/bun/compare/v1.1.15...v1.1.16) (2023-09-16)


### Reverts

* Revert "fix: "model does not have column" error (#850)" ([387228e](https://github.com/uptrace/bun/commit/387228e85d22dfcf3659f4631dfa87106d7ef45f)), closes [#850](https://github.com/uptrace/bun/issues/850)



## [1.1.15](https://github.com/uptrace/bun/compare/v1.1.14...v1.1.15) (2023-09-10)


### Bug Fixes

* "model does not have column" error ([#850](https://github.com/uptrace/bun/issues/850)) ([16367aa](https://github.com/uptrace/bun/commit/16367aabb34b98766d28e0678f9d47710f451fae))
* alloc when mounting ([#891](https://github.com/uptrace/bun/issues/891)) ([f2256f1](https://github.com/uptrace/bun/commit/f2256f10a1d328fb924ca79cde76e77641398573))
* index hints have to be specified following a table name ([4a2ae85](https://github.com/uptrace/bun/commit/4a2ae853a1509bb300bc2d96471505caee799e43))
* make Rows.Close to drain messages ([5ceba07](https://github.com/uptrace/bun/commit/5ceba076668eb7aaddb1d8a56202256d5e6c1ead))
* run hooks on Rows ([#892](https://github.com/uptrace/bun/issues/892)) ([f652b3d](https://github.com/uptrace/bun/commit/f652b3d399a3dc46c856eb8c0f10140a12ea4310))
* scan error [#709](https://github.com/uptrace/bun/issues/709) ([#837](https://github.com/uptrace/bun/issues/837)) ([b82afa5](https://github.com/uptrace/bun/commit/b82afa52633b2a1b352db6de4ff0d369d5468a07))


### Features

* add bun.NullZero ([786bb6b](https://github.com/uptrace/bun/commit/786bb6bfeba3c12f8b28579d61e4794d9fb3e373))
* **bunotel:** add options for set otel providers ([#836](https://github.com/uptrace/bun/issues/836)) ([806e632](https://github.com/uptrace/bun/commit/806e6323f60b4703b03a71c113c263d0afc95b35))



## [1.1.14](https://github.com/uptrace/bun/compare/v1.1.13...v1.1.14) (2023-05-24)


### Bug Fixes

* enable CompositeIn for MySQL ([9f377b5](https://github.com/uptrace/bun/commit/9f377b5e744cb38ef4aadd61213855c009e47354))



## [1.1.13](https://github.com/uptrace/bun/compare/v1.1.12...v1.1.13) (2023-05-06)


### Bug Fixes

* bunbig.Int.Scan typo ([7ddabb8](https://github.com/uptrace/bun/commit/7ddabb8c667f50032bc0bb2523a287efbe0851e7))
* compare full MySQL version ([096fabe](https://github.com/uptrace/bun/commit/096fabefa114202d3601ad8e456f5e491a4e3787))
* enable delete table alias for MySQL >= 8.0.16 ([77a600b](https://github.com/uptrace/bun/commit/77a600bc060154fb91aa68e68ba6a8875e5b10fb))
* incorrect table relationship panic message [#791](https://github.com/uptrace/bun/issues/791) ([ad41888](https://github.com/uptrace/bun/commit/ad4188862eeaab30fc7c48d3224b5a786557aec5))
* should rollback if migrate using transaction and got an err (thanks [@bayshark](https://github.com/bayshark)) ([e7a119b](https://github.com/uptrace/bun/commit/e7a119b1b8911d8bf059bb271c90ad4a5f5f02be))


### Features

* add option to customize Go migration template ([f31bf73](https://github.com/uptrace/bun/commit/f31bf739b9c7a0383411b9e67cba96c858795c68))
* expose Exec(…) method for RawQuery ([11192c8](https://github.com/uptrace/bun/commit/11192c83f932eb7421ef09e06859a7f171de7803))
* prefix migration files with 1 upto 14 digits ([b74b671](https://github.com/uptrace/bun/commit/b74b6714bb6a83e470e21801c97cc40e20acfb50))
* rename option ([9353a3f](https://github.com/uptrace/bun/commit/9353a3f921c038fdf4a90665f1b0a9d0d03dc182))



## [1.1.12](https://github.com/uptrace/bun/compare/v1.1.11...v1.1.12) (2023-02-20)



## [1.1.11](https://github.com/uptrace/bun/compare/v1.1.10...v1.1.11) (2023-02-01)


### Bug Fixes

* add support for inserting values with Unicode encoding for mssql dialect ([e98c6c0](https://github.com/uptrace/bun/commit/e98c6c0f033b553bea3bbc783aa56c2eaa17718f))
* fix relation tag ([a3eedff](https://github.com/uptrace/bun/commit/a3eedff49700490d4998dcdcdc04f554d8f17166))



## [1.1.10](https://github.com/uptrace/bun/compare/v1.1.9...v1.1.10) (2023-01-16)


### Bug Fixes

* allow QueryEvent to better detect operations in raw queries ([8e44735](https://github.com/uptrace/bun/commit/8e4473538364bae6562055d35e94c3e9c0b77691))
* append default VARCHAR length instead of hardcoding it in the type definition ([e5079c7](https://github.com/uptrace/bun/commit/e5079c70343ba8c8b410aed23ac1d1ae5a2c9ff6))
* prevent panic when use pg array with custom database type ([67e4412](https://github.com/uptrace/bun/commit/67e4412a972a9ed5f3a1d07c66957beedbc8a8a3))
* properly return sql.ErrNoRows when scanning []byte ([996fead](https://github.com/uptrace/bun/commit/996fead2595fbcaff4878b77befe6709a54b3a4d))


### Features

*  mssql output support for update or delete query ([#718](https://github.com/uptrace/bun/issues/718)) ([08876b4](https://github.com/uptrace/bun/commit/08876b4d420e761cbfa658aa6bb89b3f7c62c240))
* add Err method to query builder ([c722c90](https://github.com/uptrace/bun/commit/c722c90f3dce2642ca4f4c2ab3f9a35cd496b557))
* add support for time.Time array in Postgres ([3dd6f3b](https://github.com/uptrace/bun/commit/3dd6f3b2ac1bfbcda08240dc1676647b61715a9c))
* mssql and pg merge query ([#723](https://github.com/uptrace/bun/issues/723)) ([deea764](https://github.com/uptrace/bun/commit/deea764d9380b16aad34228aa32717d10f2a4bab))
* setError on attempt to set non-positive .Varchar() ([3335e0b](https://github.com/uptrace/bun/commit/3335e0b9d6d3f424145e1f715223a0fffe773d9a))


### Reverts

* go 1.18 ([67a4488](https://github.com/uptrace/bun/commit/67a448897eaaf1ebc54d629dfd3b2509b35da352))



## [1.1.9](https://github.com/uptrace/bun/compare/v1.1.8...v1.1.9) (2022-11-23)


### Bug Fixes

* adding dialect override for append-bool ([#695](https://github.com/uptrace/bun/issues/695)) ([338f2f0](https://github.com/uptrace/bun/commit/338f2f04105ad89e64530db86aeb387e2ad4789e))
* don't call hooks twice for whereExists ([9057857](https://github.com/uptrace/bun/commit/90578578e717f248e4b6eb114c5b495fd8d4ed41))
* don't lock migrations when running Migrate and Rollback ([69a7354](https://github.com/uptrace/bun/commit/69a7354d987ff2ed5338c9ef5f4ce320724299ab))
* **query:** make WhereDeleted compatible with ForceDelete ([299c3fd](https://github.com/uptrace/bun/commit/299c3fd57866aaecd127a8f219c95332898475db)), closes [#673](https://github.com/uptrace/bun/issues/673)
* relation join soft delete SQL generate ([a98f4e9](https://github.com/uptrace/bun/commit/a98f4e9f2bbdbc2b81cd13aa228a1a91eb905ba2))


### Features

* add migrate.Exec ([d368bbe](https://github.com/uptrace/bun/commit/d368bbe52bb1ee3dabf0aada190bf967eec10255))
* **update:** "skipupdate" while bulk ([1a32b2f](https://github.com/uptrace/bun/commit/1a32b2ffbd5bc9a8d8b5978dd0f16c9fb79242ee))
* **zerolog:** added zerolog hook ([9d2267d](https://github.com/uptrace/bun/commit/9d2267d414b47164ab6ceada55bf311ad548a6b0))



## [1.1.8](https://github.com/uptrace/bun/compare/v1.1.7...v1.1.8) (2022-08-29)


### Bug Fixes

* **bunotel:** handle option attributes ([#656](https://github.com/uptrace/bun/issues/656)) ([9f1e0bd](https://github.com/uptrace/bun/commit/9f1e0bd19fc0300f12996b3e6595f093024e06b6))
* driver.Valuer returns itself causes stackoverflow ([c9f51d3](https://github.com/uptrace/bun/commit/c9f51d3e2dabed0c29c26a4221abbc426a7206f3)), closes [#657](https://github.com/uptrace/bun/issues/657)
* **pgdriver:** return FATAL and PANIC errors immediately ([4595e38](https://github.com/uptrace/bun/commit/4595e385d3706116e47bf9dc295186ec7a2ab0f9))
* quote m2m table name fixes [#649](https://github.com/uptrace/bun/issues/649) ([61a634e](https://github.com/uptrace/bun/commit/61a634e4cd5c18df4b75f756d4b0f06ea94bc3c8))
* support multi-level embed column ([177ec4c](https://github.com/uptrace/bun/commit/177ec4c6e04f92957614ad4724bc82c422649a4b)), closes [#643](https://github.com/uptrace/bun/issues/643)


### Features

* conditions not supporting composite in ([e5d78d4](https://github.com/uptrace/bun/commit/e5d78d464b94b78438cf275b4c35f713d129961d))
* **idb:** support raw query ([be4e688](https://github.com/uptrace/bun/commit/be4e6886ad94b4b6ca42f24f73d79a15b1ac3188))
* **migrate:** add MissingMigrations ([42567d0](https://github.com/uptrace/bun/commit/42567d052280f2c412d4796df7178915e537e6d9))
* **pgdriver:** implement database/sql/driver.SessionResetter ([bda298a](https://github.com/uptrace/bun/commit/bda298ac66305e5b00ba67d72d3973625930c6b9))
* **pgdriver:** provide access to the underlying net.Conn ([d07ea0e](https://github.com/uptrace/bun/commit/d07ea0ed1541225b5f08e59a4c87383811f7f051))



## [1.1.7](https://github.com/uptrace/bun/compare/v1.1.6...v1.1.7) (2022-07-29)


### Bug Fixes

* change ScanAndCount without a limit to select all rows ([de5c570](https://github.com/uptrace/bun/commit/de5c5704166563aea41a82f7863f2db88ff108e2))



## [1.1.6](https://github.com/uptrace/bun/compare/v1.1.5...v1.1.6) (2022-07-10)


### Bug Fixes

* bunotel add set attributes to query metrics ([dae82cc](https://github.com/uptrace/bun/commit/dae82cc0e3af49be1e474027b55c34364676985d))
* **db.ScanRows:** ensure rows.Close is called ([9ffbc6a](https://github.com/uptrace/bun/commit/9ffbc6a46e24b908742b6973f33ef8e5b17cc12b))
* merge apply ([3081849](https://github.com/uptrace/bun/commit/30818499eacddd3b1a3e749091ba6a1468125641))
* **migrate:** close conn/tx on error ([7b168ea](https://github.com/uptrace/bun/commit/7b168eabfe0f844bcbf8dc89629d04c385b9f58c))
* **migrate:** type Migration should be used as a value rather than a pointer ([fb43935](https://github.com/uptrace/bun/commit/fb4393582b49fe528800a66aac5fb1c9a6033048))
* **migrate:** type MigrationGroup should be used as a value rather than a pointer ([649da1b](https://github.com/uptrace/bun/commit/649da1b3c158060add9b61b32c289260daafa65a))
* mssql cursor pagination ([#589](https://github.com/uptrace/bun/issues/589)) ([b34ec97](https://github.com/uptrace/bun/commit/b34ec97ddda95629f73762721d60fd3e00e7e99f))


### Features

* "skipupdate" model field tag ([#565](https://github.com/uptrace/bun/issues/565)) ([9288294](https://github.com/uptrace/bun/commit/928829482c718a0c215aa4f4adfa6f3fb3ed4302))
* add pgdriver write error to log ([5ddda3d](https://github.com/uptrace/bun/commit/5ddda3de31cd08ceee4bdea64ceae8d15eace07b))
* add query string representation ([520da7e](https://github.com/uptrace/bun/commit/520da7e1d6dbf7b06846f6b39a7f99e8753c1466))
* add relation condition with tag ([fe5bbf6](https://github.com/uptrace/bun/commit/fe5bbf64f33d25b310e5510ece7d705b9eb3bfea))
* add support for ON UPDATE and ON DELETE rules on belongs-to relationships from struct tags ([#533](https://github.com/uptrace/bun/issues/533)) ([a327b2a](https://github.com/uptrace/bun/commit/a327b2ae216abb55a705626296c0cdbf8d648697))
* add tx methods to IDB ([#587](https://github.com/uptrace/bun/issues/587)) ([feab313](https://github.com/uptrace/bun/commit/feab313c0358200b6e270ac70f4551b011ab5276))
* added raw query calls ([#596](https://github.com/uptrace/bun/issues/596)) ([127644d](https://github.com/uptrace/bun/commit/127644d2eea443736fbd6bed3417595d439e4639))
* **bunotel:** add option to enable formatting of queries ([#547](https://github.com/uptrace/bun/issues/547)) ([b9c768c](https://github.com/uptrace/bun/commit/b9c768cec3b5dea36c3c9c344d1e76e0ffad1369))
* **config.go:** add sslrootcert support to DSN parameters ([3bd5d69](https://github.com/uptrace/bun/commit/3bd5d692d7df4f30d07b835d6a46fc7af382489a))
* create an extra module for newrelic ([#599](https://github.com/uptrace/bun/issues/599)) ([6c676ce](https://github.com/uptrace/bun/commit/6c676ce13f05fe763471fbec2d5a2db48bc88650))
* **migrate:** add WithMarkAppliedOnSuccess ([31b2cc4](https://github.com/uptrace/bun/commit/31b2cc4f5ccd794a436d081073d4974835d3780d))
* **pgdialect:** add hstore support ([66b44f7](https://github.com/uptrace/bun/commit/66b44f7c0edc205927fb8be96aaf263b31828fa1))
* **pgdialect:** add identity support ([646251e](https://github.com/uptrace/bun/commit/646251ec02a1e2ec717e907e6f128d8b51f17c6d))
* **pgdriver:** expose pgdriver.ParseTime ([405a7d7](https://github.com/uptrace/bun/commit/405a7d78d8f60cf27e8f175deaf95db5877d84be))



## [1.1.5](https://github.com/uptrace/bun/compare/v1.1.4...v1.1.5) (2022-05-12)


### Bug Fixes

* **driver/sqliteshim:** make it work with recent version of modernc sqlite ([2360584](https://github.com/uptrace/bun/commit/23605846c20684e39bf1eaac50a2147a1b68a729))



## [1.1.4](https://github.com/uptrace/bun/compare/v1.1.3...v1.1.4) (2022-04-20)


### Bug Fixes

* automatically set nullzero when there is default:value option ([72c44ae](https://github.com/uptrace/bun/commit/72c44aebbeec3a83ed97ea25a3262174d744df65))
* fix ForceDelete on live/undeleted rows ([1a33250](https://github.com/uptrace/bun/commit/1a33250f27f00e752a735ce10311ac95dcb0c968))
* fix OmitZero and value overriding ([087ea07](https://github.com/uptrace/bun/commit/087ea0730551f1e841bacb6ad2fa3afd512a1df8))
* rename Query to QueryBuilder ([98d111b](https://github.com/uptrace/bun/commit/98d111b7cc00fa61b6b2cec147f43285f4baadb4))


### Features

* add ApplyQueryBuilder ([582eca0](https://github.com/uptrace/bun/commit/582eca09cf2b59e67c2e4a2ad24f1a74cb53addd))
* **config.go:** add connect_timeout to DSN parsable params ([998b04d](https://github.com/uptrace/bun/commit/998b04d51a9a4f182ac3458f90db8dbf9185c4ba)), closes [#505](https://github.com/uptrace/bun/issues/505)



# [1.1.3](https://github.com/uptrace/bun/compare/v1.1.2...v) (2022-03-29)

### Bug Fixes

- fix panic message when has-many encounter an error
  ([cfd2747](https://github.com/uptrace/bun/commit/cfd27475fac89a1c8cf798bfa64898bd77bbba79))
- **migrate:** change rollback to match migrate behavior
  ([df5af9c](https://github.com/uptrace/bun/commit/df5af9c9cbdf54ce243e037bbb2c7b154f8422b3))

### Features

- added QueryBuilder interface for SelectQuery, UpdateQuery, DeleteQuery
  ([#499](https://github.com/uptrace/bun/issues/499))
  ([59fef48](https://github.com/uptrace/bun/commit/59fef48f6b3ec7f32bdda779b6693c333ff1dfdb))

# [1.1.2](https://github.com/uptrace/bun/compare/v1.1.2...v) (2022-03-22)

### Bug Fixes

- correctly handle bun.In([][]byte{...})
  ([800616e](https://github.com/uptrace/bun/commit/800616ed28ca600ad676319a10adb970b2b4daf6))

### Features

- accept extend option to allow extending existing models
  ([48b80e4](https://github.com/uptrace/bun/commit/48b80e4f7e3ed8a28fd305f7853ebe7ab984a497))

# [1.1.0](https://github.com/uptrace/bun/compare/v1.1.0-beta.1...v1.1.0) (2022-02-28)

### Features

- Added [MSSQL](https://bun.uptrace.dev/guide/drivers.html#mssql) support as a 4th fully supported
  DBMS.
- Added `SetColumn("col_name", "upper(?)", "hello")` in addition to
  `Set("col_name = upper(?)", "hello")` which works for all 4 supported DBMS.

* improve nil ptr values handling
  ([b398e6b](https://github.com/uptrace/bun/commit/b398e6bea840ea2fd3e001b7879c0b00b6dcd6f7))

### Breaking changes

- Bun no longer automatically marks some fields like `ID int64` as `pk` and `autoincrement`. You
  need to manually add those options:

```diff
type Model struct {
-	 ID int64
+	 ID int64 `bun:",pk,autoincrement"`
}
```

Bun [v1.0.25](#1024-2022-02-22) prints warnings for models with missing options so you are
recommended to upgrade to v1.0.24 before upgrading to v1.1.x.

- Also, Bun no longer adds `nullzero` option to `soft_delete` fields.

- Removed `nopk` and `allowzero` options.

### Bug Fixes

- append slice values
  ([4a65129](https://github.com/uptrace/bun/commit/4a651294fb0f1e73079553024810c3ead9777311))
- check for nils when appending driver.Value
  ([7bb1640](https://github.com/uptrace/bun/commit/7bb1640a00fceca1e1075fe6544b9a4842ab2b26))
- cleanup soft deletes for mssql
  ([e72e2c5](https://github.com/uptrace/bun/commit/e72e2c5d0a85f3d26c3fa22c7284c2de1dcfda8e))
- **dbfixture:** apply cascade option. Fixes [#447](https://github.com/uptrace/bun/issues/447)
  ([d32d988](https://github.com/uptrace/bun/commit/d32d98840bc23e74c836f8192cb4bc9529aa9233))
- create table WithForeignKey() and has-many relation
  ([3cf5649](https://github.com/uptrace/bun/commit/3cf56491706b5652c383dbe007ff2389ad64922e))
- do not emit m2m relations in WithForeignKeys()
  ([56c8c5e](https://github.com/uptrace/bun/commit/56c8c5ed44c0d6d734c3d3161c642ce8437e2248))
- accept dest in select queries
  ([33b5b6f](https://github.com/uptrace/bun/commit/33b5b6ff660b77238a737a543ca12675c7f0c284))

## [1.0.25](https://github.com/uptrace/bun/compare/v1.0.23...v1.0.25) (2022-02-22)

### Bug Fixes

### Deprecated

In the coming v1.1.x release, Bun will stop automatically adding `,pk,autoincrement` options on
`ID int64/int32` fields. This version (v1.0.23) only prints a warning when it encounters such
fields, but the code will continue working as before.

To fix warnings, add missing options:

```diff
type Model struct {
-	 ID int64
+	 ID int64 `bun:",pk,autoincrement"`
}
```

To silence warnings:

```go
bun.SetWarnLogger(log.New(ioutil.Discard, "", log.LstdFlags))
```

Bun will also print a warning on [soft delete](https://bun.uptrace.dev/guide/soft-deletes.html)
fields without a `,nullzero` option. You can fix the warning by adding missing `,nullzero` or
`,allowzero` options.

In v1.1.x, such options as `,nopk` and `,allowzero` will not be necessary and will be removed.

### Bug Fixes

- fix missing autoincrement warning
  ([3bc9c72](https://github.com/uptrace/bun/commit/3bc9c721e1c1c5104c256a0c01c4525df6ecefc2))

* append slice values
  ([4a65129](https://github.com/uptrace/bun/commit/4a651294fb0f1e73079553024810c3ead9777311))
* don't automatically set pk, nullzero, and autoincrement options
  ([519a0df](https://github.com/uptrace/bun/commit/519a0df9707de01a418aba0d6b7482cfe4c9a532))

### Features

- add CreateTableQuery.DetectForeignKeys
  ([a958fcb](https://github.com/uptrace/bun/commit/a958fcbab680b0c5ad7980f369c7b73f7673db87))

## [1.0.22](https://github.com/uptrace/bun/compare/v1.0.21...v1.0.22) (2022-01-28)

### Bug Fixes

- improve scan error message
  ([54048b2](https://github.com/uptrace/bun/commit/54048b296b9648fd62107ce6fa6fd7e6e2a648c7))
- properly discover json.Marshaler on ptr field
  ([3b321b0](https://github.com/uptrace/bun/commit/3b321b08601c4b8dc6bcaa24adea20875883ac14))

### Breaking (MySQL, MariaDB)

- **insert:** get last insert id only with pk support auto increment
  ([79e7c79](https://github.com/uptrace/bun/commit/79e7c797beea54bfc9dc1cb0141a7520ff941b4d)). Make
  sure your MySQL models have `bun:",pk,autoincrement"` options if you are using autoincrements.

### Features

- refuse to start when version check does not pass
  ([ff8d767](https://github.com/uptrace/bun/commit/ff8d76794894eeaebede840e5199720f3f5cf531))
- support Column in ValuesQuery
  ([0707679](https://github.com/uptrace/bun/commit/0707679b075cac57efa8e6fe9019b57b2da4bcc7))

## [1.0.21](https://github.com/uptrace/bun/compare/v1.0.20...v1.0.21) (2022-01-06)

### Bug Fixes

- append where to index create
  ([1de6cea](https://github.com/uptrace/bun/commit/1de6ceaa8bba59b69fbe0cc6916d1b27da5586d8))
- check if slice is nil when calling BeforeAppendModel
  ([938d9da](https://github.com/uptrace/bun/commit/938d9dadb72ceeeb906064d9575278929d20cbbe))
- **dbfixture:** directly set matching types via reflect
  ([780504c](https://github.com/uptrace/bun/commit/780504cf1da687fc51a22d002ea66e2ccc41e1a3))
- properly handle driver.Valuer and type:json
  ([a17454a](https://github.com/uptrace/bun/commit/a17454ac6b95b2a2e927d0c4e4aee96494108389))
- support scanning string into uint64
  ([73cc117](https://github.com/uptrace/bun/commit/73cc117a9f7a623ced1fdaedb4546e8e7470e4d3))
- unique module name for opentelemetry example
  ([f2054fe](https://github.com/uptrace/bun/commit/f2054fe1d11cea3b21d69dab6f6d6d7d97ba06bb))

### Features

- add anonymous fields with type name
  ([508375b](https://github.com/uptrace/bun/commit/508375b8f2396cb088fd4399a9259584353eb7e5))
- add baseQuery.GetConn()
  ([81a9bee](https://github.com/uptrace/bun/commit/81a9beecb74fed7ec3574a1d42acdf10a74e0b00))
- create new queries from baseQuery
  ([ae1dd61](https://github.com/uptrace/bun/commit/ae1dd611a91c2b7c79bc2bc12e9a53e857791e71))
- support INSERT ... RETURNING for MariaDB >= 10.5.0
  ([b6531c0](https://github.com/uptrace/bun/commit/b6531c00ecbd4c7ec56b4131fab213f9313edc1b))

## [1.0.20](https://github.com/uptrace/bun/compare/v1.0.19...v1.0.20) (2021-12-19)

### Bug Fixes

- add Event.QueryTemplate and change Event.Query to be always formatted
  ([52b1ccd](https://github.com/uptrace/bun/commit/52b1ccdf3578418aa427adef9dcf942d90ae4fdd))
- change GetTableName to return formatted table name in case ModelTableExpr
  ([95144dd](https://github.com/uptrace/bun/commit/95144dde937b4ac88b36b0bd8b01372421069b44))
- change ScanAndCount to work with transactions
  ([5b3f2c0](https://github.com/uptrace/bun/commit/5b3f2c021c424da366caffd33589e8adde821403))
- **dbfixture:** directly call funcs bypassing template eval
  ([a61974b](https://github.com/uptrace/bun/commit/a61974ba2d24361c5357fb9bda1f3eceec5a45cd))
- don't append CASCADE by default in drop table/column queries
  ([26457ea](https://github.com/uptrace/bun/commit/26457ea5cb20862d232e6e5fa4dbdeac5d444bf1))
- **migrate:** mark migrations as applied on error so the migration can be rolled back
  ([8ce33fb](https://github.com/uptrace/bun/commit/8ce33fbbac8e33077c20daf19a14c5ff2291bcae))
- respect nullzero when appending struct fields. Fixes
  [#339](https://github.com/uptrace/bun/issues/339)
  ([ffd02f3](https://github.com/uptrace/bun/commit/ffd02f3170b3cccdd670a48d563cfb41094c05d6))
- reuse tx for relation join ([#366](https://github.com/uptrace/bun/issues/366))
  ([60bdb1a](https://github.com/uptrace/bun/commit/60bdb1ac84c0a699429eead3b7fdfbf14fe69ac6))

### Features

- add `Dialect()` to Transaction and IDB interface
  ([693f1e1](https://github.com/uptrace/bun/commit/693f1e135999fc31cf83b99a2530a695b20f4e1b))
- add model embedding via embed:prefix\_
  ([9a2cedc](https://github.com/uptrace/bun/commit/9a2cedc8b08fa8585d4bfced338bd0a40d736b1d))
- change the default log output to stderr
  ([4bf5773](https://github.com/uptrace/bun/commit/4bf577382f19c64457cbf0d64490401450954654)),
  closes [#349](https://github.com/uptrace/bun/issues/349)

## [1.0.19](https://github.com/uptrace/bun/compare/v1.0.18...v1.0.19) (2021-11-30)

### Features

- add support for column:name to specify column name
  ([e37b460](https://github.com/uptrace/bun/commit/e37b4602823babc8221970e086cfed90c6ad4cf4))

## [1.0.18](https://github.com/uptrace/bun/compare/v1.0.17...v1.0.18) (2021-11-24)

### Bug Fixes

- use correct operation for UpdateQuery
  ([687a004](https://github.com/uptrace/bun/commit/687a004ef7ec6fe1ef06c394965dd2c2d822fc82))

### Features

- add pgdriver.Notify
  ([7ee443d](https://github.com/uptrace/bun/commit/7ee443d1b869d8ddc4746850f7425d0a9ccd012b))
- CreateTableQuery.PartitionBy and CreateTableQuery.TableSpace
  ([cd3ab4d](https://github.com/uptrace/bun/commit/cd3ab4d8f3682f5a30b87c2ebc2d7e551d739078))
- **pgdriver:** add CopyFrom and CopyTo
  ([0b97703](https://github.com/uptrace/bun/commit/0b977030b5c05f509e11d13550b5f99dfd62358d))
- support InsertQuery.Ignore on PostgreSQL
  ([1aa9d14](https://github.com/uptrace/bun/commit/1aa9d149da8e46e63ff79192e394fde4d18d9b60))

## [1.0.17](https://github.com/uptrace/bun/compare/v1.0.16...v1.0.17) (2021-11-11)

### Bug Fixes

- don't call rollback when tx is already done
  ([8246c2a](https://github.com/uptrace/bun/commit/8246c2a63e2e6eba314201c6ba87f094edf098b9))
- **mysql:** escape backslash char in strings
  ([fb32029](https://github.com/uptrace/bun/commit/fb32029ea7604d066800b16df21f239b71bf121d))

## [1.0.16](https://github.com/uptrace/bun/compare/v1.0.15...v1.0.16) (2021-11-07)

### Bug Fixes

- call query hook when tx is started, committed, or rolled back
  ([30e85b5](https://github.com/uptrace/bun/commit/30e85b5366b2e51951ef17a0cf362b58f708dab1))
- **pgdialect:** auto-enable array support if the sql type is an array
  ([62c1012](https://github.com/uptrace/bun/commit/62c1012b2482e83969e5c6f5faf89e655ce78138))

### Features

- support multiple tag options join:left_col1=right_col1,join:left_col2=right_col2
  ([78cd5aa](https://github.com/uptrace/bun/commit/78cd5aa60a5c7d1323bb89081db2b2b811113052))
- **tag:** log with bad tag name
  ([4e82d75](https://github.com/uptrace/bun/commit/4e82d75be2dabdba1a510df4e1fbb86092f92f4c))

## [1.0.15](https://github.com/uptrace/bun/compare/v1.0.14...v1.0.15) (2021-10-29)

### Bug Fixes

- fixed bug creating table when model has no columns
  ([042c50b](https://github.com/uptrace/bun/commit/042c50bfe41caaa6e279e02c887c3a84a3acd84f))
- init table with dialect once
  ([9a1ce1e](https://github.com/uptrace/bun/commit/9a1ce1e492602742bb2f587e9ed24e50d7d07cad))

### Features

- accept columns in WherePK
  ([b3e7035](https://github.com/uptrace/bun/commit/b3e70356db1aa4891115a10902316090fccbc8bf))
- support ADD COLUMN IF NOT EXISTS
  ([ca7357c](https://github.com/uptrace/bun/commit/ca7357cdfe283e2f0b94eb638372e18401c486e9))

## [1.0.14](https://github.com/uptrace/bun/compare/v1.0.13...v1.0.14) (2021-10-24)

### Bug Fixes

- correct binary serialization for mysql ([#259](https://github.com/uptrace/bun/issues/259))
  ([e899f50](https://github.com/uptrace/bun/commit/e899f50b22ef6759ef8c029a6cd3f25f2bde17ef))
- correctly escape single quotes in pg arrays
  ([3010847](https://github.com/uptrace/bun/commit/3010847f5c2c50bce1969689a0b77fd8a6fb7e55))
- use BLOB sql type to encode []byte in MySQL and SQLite
  ([725ec88](https://github.com/uptrace/bun/commit/725ec8843824a7fc8f4058ead75ab0e62a78192a))

### Features

- warn when there are args but no placeholders
  ([06dde21](https://github.com/uptrace/bun/commit/06dde215c8d0bde2b2364597190729a160e536a1))

## [1.0.13](https://github.com/uptrace/bun/compare/v1.0.12...v1.0.13) (2021-10-17)

### Breaking Change

- **pgdriver:** enable TLS by default with InsecureSkipVerify=true
  ([15ec635](https://github.com/uptrace/bun/commit/15ec6356a04d5cf62d2efbeb189610532dc5eb31))

### Features

- add BeforeAppendModelHook
  ([0b55de7](https://github.com/uptrace/bun/commit/0b55de77aaffc1ed0894ef16f45df77bca7d93c1))
- **pgdriver:** add support for unix socket DSN
  ([f398cec](https://github.com/uptrace/bun/commit/f398cec1c3873efdf61ac0b94ebe06c657f0cf91))

## [1.0.12](https://github.com/uptrace/bun/compare/v1.0.11...v1.0.12) (2021-10-14)

### Bug Fixes

- add InsertQuery.ColumnExpr to specify columns
  ([60ffe29](https://github.com/uptrace/bun/commit/60ffe293b37912d95f28e69734ff51edf4b27da7))
- **bundebug:** change WithVerbose to accept a bool flag
  ([b2f8b91](https://github.com/uptrace/bun/commit/b2f8b912de1dc29f40c79066de1e9d6379db666c))
- **pgdialect:** fix bytea[] handling
  ([a5ca013](https://github.com/uptrace/bun/commit/a5ca013742c5a2e947b43d13f9c2fc0cf6a65d9c))
- **pgdriver:** rename DriverOption to Option
  ([51c1702](https://github.com/uptrace/bun/commit/51c1702431787d7369904b2624e346bf3e59c330))
- support allowzero on the soft delete field
  ([d0abec7](https://github.com/uptrace/bun/commit/d0abec71a9a546472a83bd70ed4e6a7357659a9b))

### Features

- **bundebug:** allow to configure the hook using env var, for example, BUNDEBUG={0,1,2}
  ([ce92852](https://github.com/uptrace/bun/commit/ce928524cab9a83395f3772ae9dd5d7732af281d))
- **bunotel:** report DBStats metrics
  ([b9b1575](https://github.com/uptrace/bun/commit/b9b15750f405cdbd345b776f5a56c6f742bc7361))
- **pgdriver:** add Error.StatementTimeout
  ([8a7934d](https://github.com/uptrace/bun/commit/8a7934dd788057828bb2b0983732b4394b74e960))
- **pgdriver:** allow setting Network in config
  ([b24b5d8](https://github.com/uptrace/bun/commit/b24b5d8014195a56ad7a4c634c10681038e6044d))

## [1.0.11](https://github.com/uptrace/bun/compare/v1.0.10...v1.0.11) (2021-10-05)

### Bug Fixes

- **mysqldialect:** remove duplicate AppendTime
  ([8d42090](https://github.com/uptrace/bun/commit/8d42090af34a1760004482c7fc0923b114d79937))

## [1.0.10](https://github.com/uptrace/bun/compare/v1.0.9...v1.0.10) (2021-10-05)

### Bug Fixes

- add UpdateQuery.OmitZero
  ([2294db6](https://github.com/uptrace/bun/commit/2294db61d228711435fff1075409a30086b37555))
- make ExcludeColumn work with many-to-many queries
  ([300e12b](https://github.com/uptrace/bun/commit/300e12b993554ff839ec4fa6bbea97e16aca1b55))
- **mysqldialect:** append time in local timezone
  ([e763cc8](https://github.com/uptrace/bun/commit/e763cc81eac4b11fff4e074ad3ff6cd970a71697))
- **tagparser:** improve parsing options with brackets
  ([0daa61e](https://github.com/uptrace/bun/commit/0daa61edc3c4d927ed260332b99ee09f4bb6b42f))

### Features

- add timetz parsing
  ([6e415c4](https://github.com/uptrace/bun/commit/6e415c4c5fa2c8caf4bb4aed4e5897fe5676f5a5))

## [1.0.9](https://github.com/uptrace/bun/compare/v1.0.8...v1.0.9) (2021-09-27)

### Bug Fixes

- change DBStats to use uint32 instead of uint64 to make it work on i386
  ([caca2a7](https://github.com/uptrace/bun/commit/caca2a7130288dec49fa26b49c8550140ee52f4c))

### Features

- add IQuery and QueryEvent.IQuery
  ([b762942](https://github.com/uptrace/bun/commit/b762942fa3b1d8686d0a559f93f2a6847b83d9c1))
- add QueryEvent.Model
  ([7688201](https://github.com/uptrace/bun/commit/7688201b485d14d3e393956f09a3200ea4d4e31d))
- **bunotel:** add experimental bun.query.timing metric
  ([2cdb384](https://github.com/uptrace/bun/commit/2cdb384678631ccadac0fb75f524bd5e91e96ee2))
- **pgdriver:** add Config.ConnParams to session config params
  ([408caf0](https://github.com/uptrace/bun/commit/408caf0bb579e23e26fc6149efd6851814c22517))
- **pgdriver:** allow specifying timeout in DSN
  ([7dbc71b](https://github.com/uptrace/bun/commit/7dbc71b3494caddc2e97d113f00067071b9e19da))

## [1.0.8](https://github.com/uptrace/bun/compare/v1.0.7...v1.0.8) (2021-09-18)

### Bug Fixes

- don't append soft delete where for insert queries with on conflict clause
  ([27c477c](https://github.com/uptrace/bun/commit/27c477ce071d4c49c99a2531d638ed9f20e33461))
- improve bun.NullTime to accept string
  ([73ad6f5](https://github.com/uptrace/bun/commit/73ad6f5640a0a9b09f8df2bc4ab9cb510021c50c))
- make allowzero work with auto-detected primary keys
  ([82ca87c](https://github.com/uptrace/bun/commit/82ca87c7c49797d507b31fdaacf8343716d4feff))
- support soft deletes on nil model
  ([0556e3c](https://github.com/uptrace/bun/commit/0556e3c63692a7f4e48659d52b55ffd9cca0202a))

## [1.0.7](https://github.com/uptrace/bun/compare/v1.0.6...v1.0.7) (2021-09-15)

### Bug Fixes

- don't append zero time as NULL without nullzero tag
  ([3b8d9cb](https://github.com/uptrace/bun/commit/3b8d9cb4e39eb17f79a618396bbbe0adbc66b07b))
- **pgdriver:** return PostgreSQL DATE as a string
  ([40be0e8](https://github.com/uptrace/bun/commit/40be0e8ea85f8932b7a410a6fc2dd3acd2d18ebc))
- specify table alias for soft delete where
  ([5fff1dc](https://github.com/uptrace/bun/commit/5fff1dc1dd74fa48623a24fa79e358a544dfac0b))

### Features

- add SelectQuery.Exists helper
  ([c3e59c1](https://github.com/uptrace/bun/commit/c3e59c1bc58b43c4b8e33e7d170ad33a08fbc3c7))

## [1.0.6](https://github.com/uptrace/bun/compare/v1.0.5...v1.0.6) (2021-09-11)

### Bug Fixes

- change unique tag to create a separate unique constraint
  ([8401615](https://github.com/uptrace/bun/commit/84016155a77ca77613cc054277fefadae3098757))
- improve zero checker for ptr values
  ([2b3623d](https://github.com/uptrace/bun/commit/2b3623dd665d873911fd20ca707016929921e862))

## v1.0.5 - Sep 09 2021

- chore: tweak bundebug colors
- fix: check if table is present when appending columns
- fix: copy []byte when scanning

## v1.0.4 - Sep 08 2021

- Added support for MariaDB.
- Restored default `SET` for `ON CONFLICT DO UPDATE` queries.

## v1.0.3 - Sep 06 2021

- Fixed bulk soft deletes.
- pgdialect: fixed scanning into an array pointer.

## v1.0.2 - Sep 04 2021

- Changed to completely ignore fields marked with `bun:"-"`. If you want to be able to scan into
  such columns, use `bun:",scanonly"`.
- pgdriver: fixed SASL authentication handling.

## v1.0.1 - Sep 02 2021

- pgdriver: added erroneous zero writes retry.
- Improved column handling in Relation callback.

## v1.0.0 - Sep 01 2021

- First stable release.

## v0.4.1 - Aug 18 2021

- Fixed migrate package to properly rollback migrations.
- Added `allowzero` tag option that undoes `nullzero` option.

## v0.4.0 - Aug 11 2021

- Changed `WhereGroup` function to accept `*SelectQuery`.
- Fixed query hooks for count queries.

## v0.3.4 - Jul 19 2021

- Renamed `migrate.CreateGo` to `CreateGoMigration`.
- Added `migrate.WithPackageName` to customize the Go package name in generated migrations.
- Renamed `migrate.CreateSQL` to `CreateSQLMigrations` and changed `CreateSQLMigrations` to create
  both up and down migration files.

## v0.3.1 - Jul 12 2021

- Renamed `alias` field struct tag to `alt` so it is not confused with column alias.
- Reworked migrate package API. See
  [migrate](https://github.com/uptrace/bun/tree/master/example/migrate) example for details.

## v0.3.0 - Jul 09 2021

- Changed migrate package to return structured data instead of logging the progress. See
  [migrate](https://github.com/uptrace/bun/tree/master/example/migrate) example for details.

## v0.2.14 - Jul 01 2021

- Added [sqliteshim](https://pkg.go.dev/github.com/uptrace/bun/driver/sqliteshim) by
  [Ivan Trubach](https://github.com/tie).
- Added support for MySQL 5.7 in addition to MySQL 8.

## v0.2.12 - Jun 29 2021

- Fixed scanners for net.IP and net.IPNet.

## v0.2.10 - Jun 29 2021

- Fixed pgdriver to format passed query args.

## v0.2.9 - Jun 27 2021

- Added support for prepared statements in pgdriver.

## v0.2.7 - Jun 26 2021

- Added `UpdateQuery.Bulk` helper to generate bulk-update queries.

  Before:

  ```go
  models := []Model{
  	{42, "hello"},
  	{43, "world"},
  }
  return db.NewUpdate().
  	With("_data", db.NewValues(&models)).
  	Model(&models).
  	Table("_data").
  	Set("model.str = _data.str").
  	Where("model.id = _data.id")
  ```

  Now:

  ```go
  db.NewUpdate().
  	Model(&models).
  	Bulk()
  ```

## v0.2.5 - Jun 25 2021

- Changed time.Time to always append zero time as `NULL`.
- Added `db.RunInTx` helper.

## v0.2.4 - Jun 21 2021

- Added SSL support to pgdriver.

## v0.2.3 - Jun 20 2021

- Replaced `ForceDelete(ctx)` with `ForceDelete().Exec(ctx)` for soft deletes.

## v0.2.1 - Jun 17 2021

- Renamed `DBI` to `IConn`. `IConn` is a common interface for `*sql.DB`, `*sql.Conn`, and `*sql.Tx`.
- Added `IDB`. `IDB` is a common interface for `*bun.DB`, `bun.Conn`, and `bun.Tx`.

## v0.2.0 - Jun 16 2021

- Changed [model hooks](https://bun.uptrace.dev/guide/hooks.html#model-hooks). See
  [model-hooks](example/model-hooks) example.
- Renamed `has-one` to `belongs-to`. Renamed `belongs-to` to `has-one`. Previously Bun used
  incorrect names for these relations.
