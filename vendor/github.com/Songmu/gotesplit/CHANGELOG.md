# Changelog

## [v0.4.0](https://github.com/Songmu/gotesplit/compare/v0.3.1...v0.4.0) - 2024-09-22
- docs: add the installation guide with aqua by @suzuki-shunsuke in https://github.com/Songmu/gotesplit/pull/29
- merge coverprofiles instead of overwriting them by @CubicrootXYZ in https://github.com/Songmu/gotesplit/pull/31

## [v0.3.1](https://github.com/Songmu/gotesplit/compare/v0.3.0...v0.3.1) - 2023-09-27
- Add -race to list when it is specified for test options by @shibayu36 in https://github.com/Songmu/gotesplit/pull/26

## [v0.3.0](https://github.com/Songmu/gotesplit/compare/v0.2.1...v0.3.0) - 2023-08-20
- Use jstemmer/go-junit-report/v2 to correctly parse the output of go test by @shibayu36 in https://github.com/Songmu/gotesplit/pull/22
- introduce tagpr GitHub Action by @Songmu in https://github.com/Songmu/gotesplit/pull/23
- Go 1.21 and update deps by @Songmu in https://github.com/Songmu/gotesplit/pull/25

## [v0.2.1](https://github.com/Songmu/gotesplit/compare/v0.2.0...v0.2.1) (2022-06-09)

* perf: tags option always placed at testOpts [#21](https://github.com/Songmu/gotesplit/pull/21) ([Warashi](https://github.com/Warashi))
* fix: listing tests with tags [#20](https://github.com/Songmu/gotesplit/pull/20) ([Warashi](https://github.com/Warashi))

## [v0.2.0](https://github.com/Songmu/gotesplit/compare/v0.1.2...v0.2.0) (2022-05-22)

* care -tags flag to list test cases [#19](https://github.com/Songmu/gotesplit/pull/19) ([Songmu](https://github.com/Songmu))
* update deps and CI settings [#18](https://github.com/Songmu/gotesplit/pull/18) ([Songmu](https://github.com/Songmu))

## [v0.1.2](https://github.com/Songmu/gotesplit/compare/v0.1.1...v0.1.2) (2021-08-11)

* update deps [#15](https://github.com/Songmu/gotesplit/pull/15) ([Songmu](https://github.com/Songmu))
* Output stdout of the executed command to os.Stdout when failed. [#14](https://github.com/Songmu/gotesplit/pull/14) ([fujiwara](https://github.com/fujiwara))

## [v0.1.1](https://github.com/Songmu/gotesplit/compare/v0.1.0...v0.1.1) (2020-12-25)

* ignore os.ErrClosed [#12](https://github.com/Songmu/gotesplit/pull/12) ([Songmu](https://github.com/Songmu))

## [v0.1.0](https://github.com/Songmu/gotesplit/compare/v0.0.5...v0.1.0) (2020-11-09)

* implement -junit-dir option to store test resultJunit [#10](https://github.com/Songmu/gotesplit/pull/10) ([Songmu](https://github.com/Songmu))
* udpate README [#9](https://github.com/Songmu/gotesplit/pull/9) ([Songmu](https://github.com/Songmu))

## [v0.0.5](https://github.com/Songmu/gotesplit/compare/v0.0.4...v0.0.5) (2020-10-17)

* default to main branch [#8](https://github.com/Songmu/gotesplit/pull/8) ([Songmu](https://github.com/Songmu))
* mv cmd_run.go run.go [#7](https://github.com/Songmu/gotesplit/pull/7) ([Songmu](https://github.com/Songmu))

## [v0.0.4](https://github.com/Songmu/gotesplit/compare/v0.0.3...v0.0.4) (2020-10-17)

* CGO_ENABLED=0 [#6](https://github.com/Songmu/gotesplit/pull/6) ([Songmu](https://github.com/Songmu))

## [v0.0.3](https://github.com/Songmu/gotesplit/compare/v0.0.2...v0.0.3) (2020-10-17)

* support windows [#5](https://github.com/Songmu/gotesplit/pull/5) ([Songmu](https://github.com/Songmu))

## [v0.0.2](https://github.com/Songmu/gotesplit/compare/v0.0.1...v0.0.2) (2020-10-17)

* fix install.sh [#4](https://github.com/Songmu/gotesplit/pull/4) ([Songmu](https://github.com/Songmu))

## [v0.0.1](https://github.com/Songmu/gotesplit/compare/4a8f56789b5b...v0.0.1) (2020-10-17)

* enhance documents [#3](https://github.com/Songmu/gotesplit/pull/3) ([Songmu](https://github.com/Songmu))
* remove subcommand [#2](https://github.com/Songmu/gotesplit/pull/2) ([Songmu](https://github.com/Songmu))
* add CircleCI subcommand [#1](https://github.com/Songmu/gotesplit/pull/1) ([Songmu](https://github.com/Songmu))
