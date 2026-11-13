# substrait-go

Experimental Go bindings for [substrait](https://substrait.io)

[![release status](https://github.com/substrait-io/substrait-go/actions/workflows/release.yml/badge.svg)](https://github.com/substrait-io/substrait-go/actions/workflows/release.yml)
[![codecov](https://codecov.io/gh/substrait-io/substrait-go/branch/main/graph/badge.svg?token=7YXPNM3AMJ)](https://codecov.io/gh/substrait-io/substrait-go)
## Note:

This is work in progress still, things still to do:

- [ ] Expression parsing
- [x] Reading in extension yamls
- [x] CI building and testing the implementation
- [ ] Serialization/Deserialization of some expression types:
  - [x] IfThen
  - [x] SwitchExpression
  - [x] SingularOrList
  - [x] MultiOrList
  - [x] Cast
  - [x] Nested
  - [x] Subquery
- [ ] Serialization/Deserialization of Plan and Relations
  - [x] Plan
  - [x] PlanRel
  - [x] Rel
    - [x] ReadRel
    - [x] FilterRel
    - [x] FetchRel
    - [x] AggregateRel
    - [x] SortRel
    - [x] JoinRel
    - [x] ProjectRel
    - [x] SetRel
    - [x] ExtensionSingleRel
    - [x] ExtensionMultiRel
    - [x] ExtensionLeafRel
    - [x] CrossRel
    - [x] HashJoinRel
    - [x] MergeJoinRel
  - [ ] DdlRel
  - [ ] WriteRel
  - [ ] ExchangeRel
- [x] Plan Building helpers
  - [ ] ReadRel
    - [x] NamedScanReadRel
    - [x] VirtualTableReadRel
    - [ ] ExtensionTableReadRel
    - [ ] LocalFileReadRel
  - [x] FilterRel
  - [x] FetchRel
  - [x] AggregateRel
  - [x] SortRel
  - [x] JoinRel
  - [x] ProjectRel
  - [x] SetRel
  - [x] CrossRel
  - [ ] HashJoinRel
  - [ ] MergeJoinRel
  - [ ] DdlRel
  - [ ] WriteRel
  - [ ] ExchangeRel

As this is built out, you can expect refactors and other changes to the
structure of the package for the time being. **The API should not yet be
considered stable.**

Protobuf bindings come from [`substrait-protobuf`](https://github.com/substrait-io/substrait-protobuf). 

