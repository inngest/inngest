// Code generated DO NOT EDIT

package cmds

import "strconv"

type FtAggregate Completed

func (b Builder) FtAggregate() (c FtAggregate) {
	c = FtAggregate{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FT.AGGREGATE")
	return c
}

func (c FtAggregate) Index(index string) FtAggregateIndex {
	c.cs.s = append(c.cs.s, index)
	return (FtAggregateIndex)(c)
}

type FtAggregateCursorCount Completed

func (c FtAggregateCursorCount) Maxidle(idleTime int64) FtAggregateCursorMaxidle {
	c.cs.s = append(c.cs.s, "MAXIDLE", strconv.FormatInt(idleTime, 10))
	return (FtAggregateCursorMaxidle)(c)
}

func (c FtAggregateCursorCount) Params() FtAggregateParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtAggregateParamsParams)(c)
}

func (c FtAggregateCursorCount) Dialect(dialect int64) FtAggregateDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtAggregateDialect)(c)
}

func (c FtAggregateCursorCount) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAggregateCursorMaxidle Completed

func (c FtAggregateCursorMaxidle) Params() FtAggregateParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtAggregateParamsParams)(c)
}

func (c FtAggregateCursorMaxidle) Dialect(dialect int64) FtAggregateDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtAggregateDialect)(c)
}

func (c FtAggregateCursorMaxidle) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAggregateCursorWithcursor Completed

func (c FtAggregateCursorWithcursor) Count(readSize int64) FtAggregateCursorCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(readSize, 10))
	return (FtAggregateCursorCount)(c)
}

func (c FtAggregateCursorWithcursor) Maxidle(idleTime int64) FtAggregateCursorMaxidle {
	c.cs.s = append(c.cs.s, "MAXIDLE", strconv.FormatInt(idleTime, 10))
	return (FtAggregateCursorMaxidle)(c)
}

func (c FtAggregateCursorWithcursor) Params() FtAggregateParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtAggregateParamsParams)(c)
}

func (c FtAggregateCursorWithcursor) Dialect(dialect int64) FtAggregateDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtAggregateDialect)(c)
}

func (c FtAggregateCursorWithcursor) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAggregateDialect Completed

func (c FtAggregateDialect) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAggregateIndex Completed

func (c FtAggregateIndex) Query(query string) FtAggregateQuery {
	c.cs.s = append(c.cs.s, query)
	return (FtAggregateQuery)(c)
}

type FtAggregateOpApplyApply Completed

func (c FtAggregateOpApplyApply) As(name string) FtAggregateOpApplyAs {
	c.cs.s = append(c.cs.s, "AS", name)
	return (FtAggregateOpApplyAs)(c)
}

type FtAggregateOpApplyAs Completed

func (c FtAggregateOpApplyAs) Apply(expression string) FtAggregateOpApplyApply {
	c.cs.s = append(c.cs.s, "APPLY", expression)
	return (FtAggregateOpApplyApply)(c)
}

func (c FtAggregateOpApplyAs) Limit() FtAggregateOpLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtAggregateOpLimitLimit)(c)
}

func (c FtAggregateOpApplyAs) Filter(filter string) FtAggregateOpFilter {
	c.cs.s = append(c.cs.s, "FILTER", filter)
	return (FtAggregateOpFilter)(c)
}

func (c FtAggregateOpApplyAs) LoadAll() FtAggregateOpLoadallLoadAll {
	c.cs.s = append(c.cs.s, "LOAD", "*")
	return (FtAggregateOpLoadallLoadAll)(c)
}

func (c FtAggregateOpApplyAs) Load(count int64) FtAggregateOpLoadLoad {
	c.cs.s = append(c.cs.s, "LOAD", strconv.FormatInt(count, 10))
	return (FtAggregateOpLoadLoad)(c)
}

func (c FtAggregateOpApplyAs) Groupby(nargs int64) FtAggregateOpGroupbyGroupby {
	c.cs.s = append(c.cs.s, "GROUPBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpGroupbyGroupby)(c)
}

func (c FtAggregateOpApplyAs) Sortby(nargs int64) FtAggregateOpSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpSortbySortby)(c)
}

func (c FtAggregateOpApplyAs) Withcursor() FtAggregateCursorWithcursor {
	c.cs.s = append(c.cs.s, "WITHCURSOR")
	return (FtAggregateCursorWithcursor)(c)
}

func (c FtAggregateOpApplyAs) Params() FtAggregateParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtAggregateParamsParams)(c)
}

func (c FtAggregateOpApplyAs) Dialect(dialect int64) FtAggregateDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtAggregateDialect)(c)
}

func (c FtAggregateOpApplyAs) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAggregateOpFilter Completed

func (c FtAggregateOpFilter) LoadAll() FtAggregateOpLoadallLoadAll {
	c.cs.s = append(c.cs.s, "LOAD", "*")
	return (FtAggregateOpLoadallLoadAll)(c)
}

func (c FtAggregateOpFilter) Load(count int64) FtAggregateOpLoadLoad {
	c.cs.s = append(c.cs.s, "LOAD", strconv.FormatInt(count, 10))
	return (FtAggregateOpLoadLoad)(c)
}

func (c FtAggregateOpFilter) Groupby(nargs int64) FtAggregateOpGroupbyGroupby {
	c.cs.s = append(c.cs.s, "GROUPBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpGroupbyGroupby)(c)
}

func (c FtAggregateOpFilter) Sortby(nargs int64) FtAggregateOpSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpSortbySortby)(c)
}

func (c FtAggregateOpFilter) Apply(expression string) FtAggregateOpApplyApply {
	c.cs.s = append(c.cs.s, "APPLY", expression)
	return (FtAggregateOpApplyApply)(c)
}

func (c FtAggregateOpFilter) Limit() FtAggregateOpLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtAggregateOpLimitLimit)(c)
}

func (c FtAggregateOpFilter) Filter(filter string) FtAggregateOpFilter {
	c.cs.s = append(c.cs.s, "FILTER", filter)
	return c
}

func (c FtAggregateOpFilter) Withcursor() FtAggregateCursorWithcursor {
	c.cs.s = append(c.cs.s, "WITHCURSOR")
	return (FtAggregateCursorWithcursor)(c)
}

func (c FtAggregateOpFilter) Params() FtAggregateParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtAggregateParamsParams)(c)
}

func (c FtAggregateOpFilter) Dialect(dialect int64) FtAggregateDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtAggregateDialect)(c)
}

func (c FtAggregateOpFilter) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAggregateOpGroupbyGroupby Completed

func (c FtAggregateOpGroupbyGroupby) Property(property ...string) FtAggregateOpGroupbyProperty {
	c.cs.s = append(c.cs.s, property...)
	return (FtAggregateOpGroupbyProperty)(c)
}

func (c FtAggregateOpGroupbyGroupby) Reduce(function string) FtAggregateOpGroupbyReduceReduce {
	c.cs.s = append(c.cs.s, "REDUCE", function)
	return (FtAggregateOpGroupbyReduceReduce)(c)
}

func (c FtAggregateOpGroupbyGroupby) Groupby(nargs int64) FtAggregateOpGroupbyGroupby {
	c.cs.s = append(c.cs.s, "GROUPBY", strconv.FormatInt(nargs, 10))
	return c
}

func (c FtAggregateOpGroupbyGroupby) Sortby(nargs int64) FtAggregateOpSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpSortbySortby)(c)
}

func (c FtAggregateOpGroupbyGroupby) Apply(expression string) FtAggregateOpApplyApply {
	c.cs.s = append(c.cs.s, "APPLY", expression)
	return (FtAggregateOpApplyApply)(c)
}

func (c FtAggregateOpGroupbyGroupby) Limit() FtAggregateOpLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtAggregateOpLimitLimit)(c)
}

func (c FtAggregateOpGroupbyGroupby) Filter(filter string) FtAggregateOpFilter {
	c.cs.s = append(c.cs.s, "FILTER", filter)
	return (FtAggregateOpFilter)(c)
}

func (c FtAggregateOpGroupbyGroupby) LoadAll() FtAggregateOpLoadallLoadAll {
	c.cs.s = append(c.cs.s, "LOAD", "*")
	return (FtAggregateOpLoadallLoadAll)(c)
}

func (c FtAggregateOpGroupbyGroupby) Load(count int64) FtAggregateOpLoadLoad {
	c.cs.s = append(c.cs.s, "LOAD", strconv.FormatInt(count, 10))
	return (FtAggregateOpLoadLoad)(c)
}

func (c FtAggregateOpGroupbyGroupby) Withcursor() FtAggregateCursorWithcursor {
	c.cs.s = append(c.cs.s, "WITHCURSOR")
	return (FtAggregateCursorWithcursor)(c)
}

func (c FtAggregateOpGroupbyGroupby) Params() FtAggregateParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtAggregateParamsParams)(c)
}

func (c FtAggregateOpGroupbyGroupby) Dialect(dialect int64) FtAggregateDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtAggregateDialect)(c)
}

func (c FtAggregateOpGroupbyGroupby) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAggregateOpGroupbyProperty Completed

func (c FtAggregateOpGroupbyProperty) Property(property ...string) FtAggregateOpGroupbyProperty {
	c.cs.s = append(c.cs.s, property...)
	return c
}

func (c FtAggregateOpGroupbyProperty) Reduce(function string) FtAggregateOpGroupbyReduceReduce {
	c.cs.s = append(c.cs.s, "REDUCE", function)
	return (FtAggregateOpGroupbyReduceReduce)(c)
}

func (c FtAggregateOpGroupbyProperty) Groupby(nargs int64) FtAggregateOpGroupbyGroupby {
	c.cs.s = append(c.cs.s, "GROUPBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpGroupbyGroupby)(c)
}

func (c FtAggregateOpGroupbyProperty) Sortby(nargs int64) FtAggregateOpSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpSortbySortby)(c)
}

func (c FtAggregateOpGroupbyProperty) Apply(expression string) FtAggregateOpApplyApply {
	c.cs.s = append(c.cs.s, "APPLY", expression)
	return (FtAggregateOpApplyApply)(c)
}

func (c FtAggregateOpGroupbyProperty) Limit() FtAggregateOpLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtAggregateOpLimitLimit)(c)
}

func (c FtAggregateOpGroupbyProperty) Filter(filter string) FtAggregateOpFilter {
	c.cs.s = append(c.cs.s, "FILTER", filter)
	return (FtAggregateOpFilter)(c)
}

func (c FtAggregateOpGroupbyProperty) LoadAll() FtAggregateOpLoadallLoadAll {
	c.cs.s = append(c.cs.s, "LOAD", "*")
	return (FtAggregateOpLoadallLoadAll)(c)
}

func (c FtAggregateOpGroupbyProperty) Load(count int64) FtAggregateOpLoadLoad {
	c.cs.s = append(c.cs.s, "LOAD", strconv.FormatInt(count, 10))
	return (FtAggregateOpLoadLoad)(c)
}

func (c FtAggregateOpGroupbyProperty) Withcursor() FtAggregateCursorWithcursor {
	c.cs.s = append(c.cs.s, "WITHCURSOR")
	return (FtAggregateCursorWithcursor)(c)
}

func (c FtAggregateOpGroupbyProperty) Params() FtAggregateParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtAggregateParamsParams)(c)
}

func (c FtAggregateOpGroupbyProperty) Dialect(dialect int64) FtAggregateDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtAggregateDialect)(c)
}

func (c FtAggregateOpGroupbyProperty) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAggregateOpGroupbyReduceArg Completed

func (c FtAggregateOpGroupbyReduceArg) Arg(arg ...string) FtAggregateOpGroupbyReduceArg {
	c.cs.s = append(c.cs.s, arg...)
	return c
}

func (c FtAggregateOpGroupbyReduceArg) As(name string) FtAggregateOpGroupbyReduceAs {
	c.cs.s = append(c.cs.s, "AS", name)
	return (FtAggregateOpGroupbyReduceAs)(c)
}

func (c FtAggregateOpGroupbyReduceArg) By(by string) FtAggregateOpGroupbyReduceBy {
	c.cs.s = append(c.cs.s, "BY", by)
	return (FtAggregateOpGroupbyReduceBy)(c)
}

func (c FtAggregateOpGroupbyReduceArg) Asc() FtAggregateOpGroupbyReduceOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (FtAggregateOpGroupbyReduceOrderAsc)(c)
}

func (c FtAggregateOpGroupbyReduceArg) Desc() FtAggregateOpGroupbyReduceOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (FtAggregateOpGroupbyReduceOrderDesc)(c)
}

func (c FtAggregateOpGroupbyReduceArg) Reduce(function string) FtAggregateOpGroupbyReduceReduce {
	c.cs.s = append(c.cs.s, "REDUCE", function)
	return (FtAggregateOpGroupbyReduceReduce)(c)
}

func (c FtAggregateOpGroupbyReduceArg) Groupby(nargs int64) FtAggregateOpGroupbyGroupby {
	c.cs.s = append(c.cs.s, "GROUPBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpGroupbyGroupby)(c)
}

func (c FtAggregateOpGroupbyReduceArg) Sortby(nargs int64) FtAggregateOpSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpSortbySortby)(c)
}

func (c FtAggregateOpGroupbyReduceArg) Apply(expression string) FtAggregateOpApplyApply {
	c.cs.s = append(c.cs.s, "APPLY", expression)
	return (FtAggregateOpApplyApply)(c)
}

func (c FtAggregateOpGroupbyReduceArg) Limit() FtAggregateOpLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtAggregateOpLimitLimit)(c)
}

func (c FtAggregateOpGroupbyReduceArg) Filter(filter string) FtAggregateOpFilter {
	c.cs.s = append(c.cs.s, "FILTER", filter)
	return (FtAggregateOpFilter)(c)
}

func (c FtAggregateOpGroupbyReduceArg) LoadAll() FtAggregateOpLoadallLoadAll {
	c.cs.s = append(c.cs.s, "LOAD", "*")
	return (FtAggregateOpLoadallLoadAll)(c)
}

func (c FtAggregateOpGroupbyReduceArg) Load(count int64) FtAggregateOpLoadLoad {
	c.cs.s = append(c.cs.s, "LOAD", strconv.FormatInt(count, 10))
	return (FtAggregateOpLoadLoad)(c)
}

func (c FtAggregateOpGroupbyReduceArg) Withcursor() FtAggregateCursorWithcursor {
	c.cs.s = append(c.cs.s, "WITHCURSOR")
	return (FtAggregateCursorWithcursor)(c)
}

func (c FtAggregateOpGroupbyReduceArg) Params() FtAggregateParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtAggregateParamsParams)(c)
}

func (c FtAggregateOpGroupbyReduceArg) Dialect(dialect int64) FtAggregateDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtAggregateDialect)(c)
}

func (c FtAggregateOpGroupbyReduceArg) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAggregateOpGroupbyReduceAs Completed

func (c FtAggregateOpGroupbyReduceAs) By(by string) FtAggregateOpGroupbyReduceBy {
	c.cs.s = append(c.cs.s, "BY", by)
	return (FtAggregateOpGroupbyReduceBy)(c)
}

func (c FtAggregateOpGroupbyReduceAs) Asc() FtAggregateOpGroupbyReduceOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (FtAggregateOpGroupbyReduceOrderAsc)(c)
}

func (c FtAggregateOpGroupbyReduceAs) Desc() FtAggregateOpGroupbyReduceOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (FtAggregateOpGroupbyReduceOrderDesc)(c)
}

func (c FtAggregateOpGroupbyReduceAs) Reduce(function string) FtAggregateOpGroupbyReduceReduce {
	c.cs.s = append(c.cs.s, "REDUCE", function)
	return (FtAggregateOpGroupbyReduceReduce)(c)
}

func (c FtAggregateOpGroupbyReduceAs) Groupby(nargs int64) FtAggregateOpGroupbyGroupby {
	c.cs.s = append(c.cs.s, "GROUPBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpGroupbyGroupby)(c)
}

func (c FtAggregateOpGroupbyReduceAs) Sortby(nargs int64) FtAggregateOpSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpSortbySortby)(c)
}

func (c FtAggregateOpGroupbyReduceAs) Apply(expression string) FtAggregateOpApplyApply {
	c.cs.s = append(c.cs.s, "APPLY", expression)
	return (FtAggregateOpApplyApply)(c)
}

func (c FtAggregateOpGroupbyReduceAs) Limit() FtAggregateOpLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtAggregateOpLimitLimit)(c)
}

func (c FtAggregateOpGroupbyReduceAs) Filter(filter string) FtAggregateOpFilter {
	c.cs.s = append(c.cs.s, "FILTER", filter)
	return (FtAggregateOpFilter)(c)
}

func (c FtAggregateOpGroupbyReduceAs) LoadAll() FtAggregateOpLoadallLoadAll {
	c.cs.s = append(c.cs.s, "LOAD", "*")
	return (FtAggregateOpLoadallLoadAll)(c)
}

func (c FtAggregateOpGroupbyReduceAs) Load(count int64) FtAggregateOpLoadLoad {
	c.cs.s = append(c.cs.s, "LOAD", strconv.FormatInt(count, 10))
	return (FtAggregateOpLoadLoad)(c)
}

func (c FtAggregateOpGroupbyReduceAs) Withcursor() FtAggregateCursorWithcursor {
	c.cs.s = append(c.cs.s, "WITHCURSOR")
	return (FtAggregateCursorWithcursor)(c)
}

func (c FtAggregateOpGroupbyReduceAs) Params() FtAggregateParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtAggregateParamsParams)(c)
}

func (c FtAggregateOpGroupbyReduceAs) Dialect(dialect int64) FtAggregateDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtAggregateDialect)(c)
}

func (c FtAggregateOpGroupbyReduceAs) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAggregateOpGroupbyReduceBy Completed

func (c FtAggregateOpGroupbyReduceBy) Asc() FtAggregateOpGroupbyReduceOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (FtAggregateOpGroupbyReduceOrderAsc)(c)
}

func (c FtAggregateOpGroupbyReduceBy) Desc() FtAggregateOpGroupbyReduceOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (FtAggregateOpGroupbyReduceOrderDesc)(c)
}

func (c FtAggregateOpGroupbyReduceBy) Reduce(function string) FtAggregateOpGroupbyReduceReduce {
	c.cs.s = append(c.cs.s, "REDUCE", function)
	return (FtAggregateOpGroupbyReduceReduce)(c)
}

func (c FtAggregateOpGroupbyReduceBy) Groupby(nargs int64) FtAggregateOpGroupbyGroupby {
	c.cs.s = append(c.cs.s, "GROUPBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpGroupbyGroupby)(c)
}

func (c FtAggregateOpGroupbyReduceBy) Sortby(nargs int64) FtAggregateOpSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpSortbySortby)(c)
}

func (c FtAggregateOpGroupbyReduceBy) Apply(expression string) FtAggregateOpApplyApply {
	c.cs.s = append(c.cs.s, "APPLY", expression)
	return (FtAggregateOpApplyApply)(c)
}

func (c FtAggregateOpGroupbyReduceBy) Limit() FtAggregateOpLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtAggregateOpLimitLimit)(c)
}

func (c FtAggregateOpGroupbyReduceBy) Filter(filter string) FtAggregateOpFilter {
	c.cs.s = append(c.cs.s, "FILTER", filter)
	return (FtAggregateOpFilter)(c)
}

func (c FtAggregateOpGroupbyReduceBy) LoadAll() FtAggregateOpLoadallLoadAll {
	c.cs.s = append(c.cs.s, "LOAD", "*")
	return (FtAggregateOpLoadallLoadAll)(c)
}

func (c FtAggregateOpGroupbyReduceBy) Load(count int64) FtAggregateOpLoadLoad {
	c.cs.s = append(c.cs.s, "LOAD", strconv.FormatInt(count, 10))
	return (FtAggregateOpLoadLoad)(c)
}

func (c FtAggregateOpGroupbyReduceBy) Withcursor() FtAggregateCursorWithcursor {
	c.cs.s = append(c.cs.s, "WITHCURSOR")
	return (FtAggregateCursorWithcursor)(c)
}

func (c FtAggregateOpGroupbyReduceBy) Params() FtAggregateParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtAggregateParamsParams)(c)
}

func (c FtAggregateOpGroupbyReduceBy) Dialect(dialect int64) FtAggregateDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtAggregateDialect)(c)
}

func (c FtAggregateOpGroupbyReduceBy) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAggregateOpGroupbyReduceNargs Completed

func (c FtAggregateOpGroupbyReduceNargs) Arg(arg ...string) FtAggregateOpGroupbyReduceArg {
	c.cs.s = append(c.cs.s, arg...)
	return (FtAggregateOpGroupbyReduceArg)(c)
}

func (c FtAggregateOpGroupbyReduceNargs) As(name string) FtAggregateOpGroupbyReduceAs {
	c.cs.s = append(c.cs.s, "AS", name)
	return (FtAggregateOpGroupbyReduceAs)(c)
}

func (c FtAggregateOpGroupbyReduceNargs) By(by string) FtAggregateOpGroupbyReduceBy {
	c.cs.s = append(c.cs.s, "BY", by)
	return (FtAggregateOpGroupbyReduceBy)(c)
}

func (c FtAggregateOpGroupbyReduceNargs) Asc() FtAggregateOpGroupbyReduceOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (FtAggregateOpGroupbyReduceOrderAsc)(c)
}

func (c FtAggregateOpGroupbyReduceNargs) Desc() FtAggregateOpGroupbyReduceOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (FtAggregateOpGroupbyReduceOrderDesc)(c)
}

func (c FtAggregateOpGroupbyReduceNargs) Reduce(function string) FtAggregateOpGroupbyReduceReduce {
	c.cs.s = append(c.cs.s, "REDUCE", function)
	return (FtAggregateOpGroupbyReduceReduce)(c)
}

func (c FtAggregateOpGroupbyReduceNargs) Groupby(nargs int64) FtAggregateOpGroupbyGroupby {
	c.cs.s = append(c.cs.s, "GROUPBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpGroupbyGroupby)(c)
}

func (c FtAggregateOpGroupbyReduceNargs) Sortby(nargs int64) FtAggregateOpSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpSortbySortby)(c)
}

func (c FtAggregateOpGroupbyReduceNargs) Apply(expression string) FtAggregateOpApplyApply {
	c.cs.s = append(c.cs.s, "APPLY", expression)
	return (FtAggregateOpApplyApply)(c)
}

func (c FtAggregateOpGroupbyReduceNargs) Limit() FtAggregateOpLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtAggregateOpLimitLimit)(c)
}

func (c FtAggregateOpGroupbyReduceNargs) Filter(filter string) FtAggregateOpFilter {
	c.cs.s = append(c.cs.s, "FILTER", filter)
	return (FtAggregateOpFilter)(c)
}

func (c FtAggregateOpGroupbyReduceNargs) LoadAll() FtAggregateOpLoadallLoadAll {
	c.cs.s = append(c.cs.s, "LOAD", "*")
	return (FtAggregateOpLoadallLoadAll)(c)
}

func (c FtAggregateOpGroupbyReduceNargs) Load(count int64) FtAggregateOpLoadLoad {
	c.cs.s = append(c.cs.s, "LOAD", strconv.FormatInt(count, 10))
	return (FtAggregateOpLoadLoad)(c)
}

func (c FtAggregateOpGroupbyReduceNargs) Withcursor() FtAggregateCursorWithcursor {
	c.cs.s = append(c.cs.s, "WITHCURSOR")
	return (FtAggregateCursorWithcursor)(c)
}

func (c FtAggregateOpGroupbyReduceNargs) Params() FtAggregateParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtAggregateParamsParams)(c)
}

func (c FtAggregateOpGroupbyReduceNargs) Dialect(dialect int64) FtAggregateDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtAggregateDialect)(c)
}

func (c FtAggregateOpGroupbyReduceNargs) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAggregateOpGroupbyReduceOrderAsc Completed

func (c FtAggregateOpGroupbyReduceOrderAsc) Reduce(function string) FtAggregateOpGroupbyReduceReduce {
	c.cs.s = append(c.cs.s, "REDUCE", function)
	return (FtAggregateOpGroupbyReduceReduce)(c)
}

func (c FtAggregateOpGroupbyReduceOrderAsc) Groupby(nargs int64) FtAggregateOpGroupbyGroupby {
	c.cs.s = append(c.cs.s, "GROUPBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpGroupbyGroupby)(c)
}

func (c FtAggregateOpGroupbyReduceOrderAsc) Sortby(nargs int64) FtAggregateOpSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpSortbySortby)(c)
}

func (c FtAggregateOpGroupbyReduceOrderAsc) Apply(expression string) FtAggregateOpApplyApply {
	c.cs.s = append(c.cs.s, "APPLY", expression)
	return (FtAggregateOpApplyApply)(c)
}

func (c FtAggregateOpGroupbyReduceOrderAsc) Limit() FtAggregateOpLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtAggregateOpLimitLimit)(c)
}

func (c FtAggregateOpGroupbyReduceOrderAsc) Filter(filter string) FtAggregateOpFilter {
	c.cs.s = append(c.cs.s, "FILTER", filter)
	return (FtAggregateOpFilter)(c)
}

func (c FtAggregateOpGroupbyReduceOrderAsc) LoadAll() FtAggregateOpLoadallLoadAll {
	c.cs.s = append(c.cs.s, "LOAD", "*")
	return (FtAggregateOpLoadallLoadAll)(c)
}

func (c FtAggregateOpGroupbyReduceOrderAsc) Load(count int64) FtAggregateOpLoadLoad {
	c.cs.s = append(c.cs.s, "LOAD", strconv.FormatInt(count, 10))
	return (FtAggregateOpLoadLoad)(c)
}

func (c FtAggregateOpGroupbyReduceOrderAsc) Withcursor() FtAggregateCursorWithcursor {
	c.cs.s = append(c.cs.s, "WITHCURSOR")
	return (FtAggregateCursorWithcursor)(c)
}

func (c FtAggregateOpGroupbyReduceOrderAsc) Params() FtAggregateParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtAggregateParamsParams)(c)
}

func (c FtAggregateOpGroupbyReduceOrderAsc) Dialect(dialect int64) FtAggregateDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtAggregateDialect)(c)
}

func (c FtAggregateOpGroupbyReduceOrderAsc) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAggregateOpGroupbyReduceOrderDesc Completed

func (c FtAggregateOpGroupbyReduceOrderDesc) Reduce(function string) FtAggregateOpGroupbyReduceReduce {
	c.cs.s = append(c.cs.s, "REDUCE", function)
	return (FtAggregateOpGroupbyReduceReduce)(c)
}

func (c FtAggregateOpGroupbyReduceOrderDesc) Groupby(nargs int64) FtAggregateOpGroupbyGroupby {
	c.cs.s = append(c.cs.s, "GROUPBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpGroupbyGroupby)(c)
}

func (c FtAggregateOpGroupbyReduceOrderDesc) Sortby(nargs int64) FtAggregateOpSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpSortbySortby)(c)
}

func (c FtAggregateOpGroupbyReduceOrderDesc) Apply(expression string) FtAggregateOpApplyApply {
	c.cs.s = append(c.cs.s, "APPLY", expression)
	return (FtAggregateOpApplyApply)(c)
}

func (c FtAggregateOpGroupbyReduceOrderDesc) Limit() FtAggregateOpLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtAggregateOpLimitLimit)(c)
}

func (c FtAggregateOpGroupbyReduceOrderDesc) Filter(filter string) FtAggregateOpFilter {
	c.cs.s = append(c.cs.s, "FILTER", filter)
	return (FtAggregateOpFilter)(c)
}

func (c FtAggregateOpGroupbyReduceOrderDesc) LoadAll() FtAggregateOpLoadallLoadAll {
	c.cs.s = append(c.cs.s, "LOAD", "*")
	return (FtAggregateOpLoadallLoadAll)(c)
}

func (c FtAggregateOpGroupbyReduceOrderDesc) Load(count int64) FtAggregateOpLoadLoad {
	c.cs.s = append(c.cs.s, "LOAD", strconv.FormatInt(count, 10))
	return (FtAggregateOpLoadLoad)(c)
}

func (c FtAggregateOpGroupbyReduceOrderDesc) Withcursor() FtAggregateCursorWithcursor {
	c.cs.s = append(c.cs.s, "WITHCURSOR")
	return (FtAggregateCursorWithcursor)(c)
}

func (c FtAggregateOpGroupbyReduceOrderDesc) Params() FtAggregateParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtAggregateParamsParams)(c)
}

func (c FtAggregateOpGroupbyReduceOrderDesc) Dialect(dialect int64) FtAggregateDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtAggregateDialect)(c)
}

func (c FtAggregateOpGroupbyReduceOrderDesc) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAggregateOpGroupbyReduceReduce Completed

func (c FtAggregateOpGroupbyReduceReduce) Nargs(nargs int64) FtAggregateOpGroupbyReduceNargs {
	c.cs.s = append(c.cs.s, strconv.FormatInt(nargs, 10))
	return (FtAggregateOpGroupbyReduceNargs)(c)
}

type FtAggregateOpLimitLimit Completed

func (c FtAggregateOpLimitLimit) OffsetNum(offset int64, num int64) FtAggregateOpLimitOffsetNum {
	c.cs.s = append(c.cs.s, strconv.FormatInt(offset, 10), strconv.FormatInt(num, 10))
	return (FtAggregateOpLimitOffsetNum)(c)
}

type FtAggregateOpLimitOffsetNum Completed

func (c FtAggregateOpLimitOffsetNum) Filter(filter string) FtAggregateOpFilter {
	c.cs.s = append(c.cs.s, "FILTER", filter)
	return (FtAggregateOpFilter)(c)
}

func (c FtAggregateOpLimitOffsetNum) LoadAll() FtAggregateOpLoadallLoadAll {
	c.cs.s = append(c.cs.s, "LOAD", "*")
	return (FtAggregateOpLoadallLoadAll)(c)
}

func (c FtAggregateOpLimitOffsetNum) Load(count int64) FtAggregateOpLoadLoad {
	c.cs.s = append(c.cs.s, "LOAD", strconv.FormatInt(count, 10))
	return (FtAggregateOpLoadLoad)(c)
}

func (c FtAggregateOpLimitOffsetNum) Groupby(nargs int64) FtAggregateOpGroupbyGroupby {
	c.cs.s = append(c.cs.s, "GROUPBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpGroupbyGroupby)(c)
}

func (c FtAggregateOpLimitOffsetNum) Sortby(nargs int64) FtAggregateOpSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpSortbySortby)(c)
}

func (c FtAggregateOpLimitOffsetNum) Apply(expression string) FtAggregateOpApplyApply {
	c.cs.s = append(c.cs.s, "APPLY", expression)
	return (FtAggregateOpApplyApply)(c)
}

func (c FtAggregateOpLimitOffsetNum) Limit() FtAggregateOpLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtAggregateOpLimitLimit)(c)
}

func (c FtAggregateOpLimitOffsetNum) Withcursor() FtAggregateCursorWithcursor {
	c.cs.s = append(c.cs.s, "WITHCURSOR")
	return (FtAggregateCursorWithcursor)(c)
}

func (c FtAggregateOpLimitOffsetNum) Params() FtAggregateParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtAggregateParamsParams)(c)
}

func (c FtAggregateOpLimitOffsetNum) Dialect(dialect int64) FtAggregateDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtAggregateDialect)(c)
}

func (c FtAggregateOpLimitOffsetNum) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAggregateOpLoadField Completed

func (c FtAggregateOpLoadField) Field(field ...string) FtAggregateOpLoadField {
	c.cs.s = append(c.cs.s, field...)
	return c
}

func (c FtAggregateOpLoadField) Groupby(nargs int64) FtAggregateOpGroupbyGroupby {
	c.cs.s = append(c.cs.s, "GROUPBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpGroupbyGroupby)(c)
}

func (c FtAggregateOpLoadField) Sortby(nargs int64) FtAggregateOpSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpSortbySortby)(c)
}

func (c FtAggregateOpLoadField) Apply(expression string) FtAggregateOpApplyApply {
	c.cs.s = append(c.cs.s, "APPLY", expression)
	return (FtAggregateOpApplyApply)(c)
}

func (c FtAggregateOpLoadField) Limit() FtAggregateOpLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtAggregateOpLimitLimit)(c)
}

func (c FtAggregateOpLoadField) Filter(filter string) FtAggregateOpFilter {
	c.cs.s = append(c.cs.s, "FILTER", filter)
	return (FtAggregateOpFilter)(c)
}

func (c FtAggregateOpLoadField) LoadAll() FtAggregateOpLoadallLoadAll {
	c.cs.s = append(c.cs.s, "LOAD", "*")
	return (FtAggregateOpLoadallLoadAll)(c)
}

func (c FtAggregateOpLoadField) Load(count int64) FtAggregateOpLoadLoad {
	c.cs.s = append(c.cs.s, "LOAD", strconv.FormatInt(count, 10))
	return (FtAggregateOpLoadLoad)(c)
}

func (c FtAggregateOpLoadField) Withcursor() FtAggregateCursorWithcursor {
	c.cs.s = append(c.cs.s, "WITHCURSOR")
	return (FtAggregateCursorWithcursor)(c)
}

func (c FtAggregateOpLoadField) Params() FtAggregateParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtAggregateParamsParams)(c)
}

func (c FtAggregateOpLoadField) Dialect(dialect int64) FtAggregateDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtAggregateDialect)(c)
}

func (c FtAggregateOpLoadField) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAggregateOpLoadLoad Completed

func (c FtAggregateOpLoadLoad) Field(field ...string) FtAggregateOpLoadField {
	c.cs.s = append(c.cs.s, field...)
	return (FtAggregateOpLoadField)(c)
}

type FtAggregateOpLoadallLoadAll Completed

func (c FtAggregateOpLoadallLoadAll) Load(count int64) FtAggregateOpLoadLoad {
	c.cs.s = append(c.cs.s, "LOAD", strconv.FormatInt(count, 10))
	return (FtAggregateOpLoadLoad)(c)
}

func (c FtAggregateOpLoadallLoadAll) Groupby(nargs int64) FtAggregateOpGroupbyGroupby {
	c.cs.s = append(c.cs.s, "GROUPBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpGroupbyGroupby)(c)
}

func (c FtAggregateOpLoadallLoadAll) Sortby(nargs int64) FtAggregateOpSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpSortbySortby)(c)
}

func (c FtAggregateOpLoadallLoadAll) Apply(expression string) FtAggregateOpApplyApply {
	c.cs.s = append(c.cs.s, "APPLY", expression)
	return (FtAggregateOpApplyApply)(c)
}

func (c FtAggregateOpLoadallLoadAll) Limit() FtAggregateOpLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtAggregateOpLimitLimit)(c)
}

func (c FtAggregateOpLoadallLoadAll) Filter(filter string) FtAggregateOpFilter {
	c.cs.s = append(c.cs.s, "FILTER", filter)
	return (FtAggregateOpFilter)(c)
}

func (c FtAggregateOpLoadallLoadAll) LoadAll() FtAggregateOpLoadallLoadAll {
	c.cs.s = append(c.cs.s, "LOAD", "*")
	return c
}

func (c FtAggregateOpLoadallLoadAll) Withcursor() FtAggregateCursorWithcursor {
	c.cs.s = append(c.cs.s, "WITHCURSOR")
	return (FtAggregateCursorWithcursor)(c)
}

func (c FtAggregateOpLoadallLoadAll) Params() FtAggregateParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtAggregateParamsParams)(c)
}

func (c FtAggregateOpLoadallLoadAll) Dialect(dialect int64) FtAggregateDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtAggregateDialect)(c)
}

func (c FtAggregateOpLoadallLoadAll) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAggregateOpSortbyFieldsOrderAsc Completed

func (c FtAggregateOpSortbyFieldsOrderAsc) Property(property string) FtAggregateOpSortbyFieldsProperty {
	c.cs.s = append(c.cs.s, property)
	return (FtAggregateOpSortbyFieldsProperty)(c)
}

func (c FtAggregateOpSortbyFieldsOrderAsc) Max(num int64) FtAggregateOpSortbyMax {
	c.cs.s = append(c.cs.s, "MAX", strconv.FormatInt(num, 10))
	return (FtAggregateOpSortbyMax)(c)
}

func (c FtAggregateOpSortbyFieldsOrderAsc) Apply(expression string) FtAggregateOpApplyApply {
	c.cs.s = append(c.cs.s, "APPLY", expression)
	return (FtAggregateOpApplyApply)(c)
}

func (c FtAggregateOpSortbyFieldsOrderAsc) Limit() FtAggregateOpLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtAggregateOpLimitLimit)(c)
}

func (c FtAggregateOpSortbyFieldsOrderAsc) Filter(filter string) FtAggregateOpFilter {
	c.cs.s = append(c.cs.s, "FILTER", filter)
	return (FtAggregateOpFilter)(c)
}

func (c FtAggregateOpSortbyFieldsOrderAsc) LoadAll() FtAggregateOpLoadallLoadAll {
	c.cs.s = append(c.cs.s, "LOAD", "*")
	return (FtAggregateOpLoadallLoadAll)(c)
}

func (c FtAggregateOpSortbyFieldsOrderAsc) Load(count int64) FtAggregateOpLoadLoad {
	c.cs.s = append(c.cs.s, "LOAD", strconv.FormatInt(count, 10))
	return (FtAggregateOpLoadLoad)(c)
}

func (c FtAggregateOpSortbyFieldsOrderAsc) Groupby(nargs int64) FtAggregateOpGroupbyGroupby {
	c.cs.s = append(c.cs.s, "GROUPBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpGroupbyGroupby)(c)
}

func (c FtAggregateOpSortbyFieldsOrderAsc) Sortby(nargs int64) FtAggregateOpSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpSortbySortby)(c)
}

func (c FtAggregateOpSortbyFieldsOrderAsc) Withcursor() FtAggregateCursorWithcursor {
	c.cs.s = append(c.cs.s, "WITHCURSOR")
	return (FtAggregateCursorWithcursor)(c)
}

func (c FtAggregateOpSortbyFieldsOrderAsc) Params() FtAggregateParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtAggregateParamsParams)(c)
}

func (c FtAggregateOpSortbyFieldsOrderAsc) Dialect(dialect int64) FtAggregateDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtAggregateDialect)(c)
}

func (c FtAggregateOpSortbyFieldsOrderAsc) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAggregateOpSortbyFieldsOrderDesc Completed

func (c FtAggregateOpSortbyFieldsOrderDesc) Property(property string) FtAggregateOpSortbyFieldsProperty {
	c.cs.s = append(c.cs.s, property)
	return (FtAggregateOpSortbyFieldsProperty)(c)
}

func (c FtAggregateOpSortbyFieldsOrderDesc) Max(num int64) FtAggregateOpSortbyMax {
	c.cs.s = append(c.cs.s, "MAX", strconv.FormatInt(num, 10))
	return (FtAggregateOpSortbyMax)(c)
}

func (c FtAggregateOpSortbyFieldsOrderDesc) Apply(expression string) FtAggregateOpApplyApply {
	c.cs.s = append(c.cs.s, "APPLY", expression)
	return (FtAggregateOpApplyApply)(c)
}

func (c FtAggregateOpSortbyFieldsOrderDesc) Limit() FtAggregateOpLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtAggregateOpLimitLimit)(c)
}

func (c FtAggregateOpSortbyFieldsOrderDesc) Filter(filter string) FtAggregateOpFilter {
	c.cs.s = append(c.cs.s, "FILTER", filter)
	return (FtAggregateOpFilter)(c)
}

func (c FtAggregateOpSortbyFieldsOrderDesc) LoadAll() FtAggregateOpLoadallLoadAll {
	c.cs.s = append(c.cs.s, "LOAD", "*")
	return (FtAggregateOpLoadallLoadAll)(c)
}

func (c FtAggregateOpSortbyFieldsOrderDesc) Load(count int64) FtAggregateOpLoadLoad {
	c.cs.s = append(c.cs.s, "LOAD", strconv.FormatInt(count, 10))
	return (FtAggregateOpLoadLoad)(c)
}

func (c FtAggregateOpSortbyFieldsOrderDesc) Groupby(nargs int64) FtAggregateOpGroupbyGroupby {
	c.cs.s = append(c.cs.s, "GROUPBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpGroupbyGroupby)(c)
}

func (c FtAggregateOpSortbyFieldsOrderDesc) Sortby(nargs int64) FtAggregateOpSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpSortbySortby)(c)
}

func (c FtAggregateOpSortbyFieldsOrderDesc) Withcursor() FtAggregateCursorWithcursor {
	c.cs.s = append(c.cs.s, "WITHCURSOR")
	return (FtAggregateCursorWithcursor)(c)
}

func (c FtAggregateOpSortbyFieldsOrderDesc) Params() FtAggregateParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtAggregateParamsParams)(c)
}

func (c FtAggregateOpSortbyFieldsOrderDesc) Dialect(dialect int64) FtAggregateDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtAggregateDialect)(c)
}

func (c FtAggregateOpSortbyFieldsOrderDesc) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAggregateOpSortbyFieldsProperty Completed

func (c FtAggregateOpSortbyFieldsProperty) Asc() FtAggregateOpSortbyFieldsOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (FtAggregateOpSortbyFieldsOrderAsc)(c)
}

func (c FtAggregateOpSortbyFieldsProperty) Desc() FtAggregateOpSortbyFieldsOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (FtAggregateOpSortbyFieldsOrderDesc)(c)
}

func (c FtAggregateOpSortbyFieldsProperty) Property(property string) FtAggregateOpSortbyFieldsProperty {
	c.cs.s = append(c.cs.s, property)
	return c
}

func (c FtAggregateOpSortbyFieldsProperty) Max(num int64) FtAggregateOpSortbyMax {
	c.cs.s = append(c.cs.s, "MAX", strconv.FormatInt(num, 10))
	return (FtAggregateOpSortbyMax)(c)
}

func (c FtAggregateOpSortbyFieldsProperty) Apply(expression string) FtAggregateOpApplyApply {
	c.cs.s = append(c.cs.s, "APPLY", expression)
	return (FtAggregateOpApplyApply)(c)
}

func (c FtAggregateOpSortbyFieldsProperty) Limit() FtAggregateOpLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtAggregateOpLimitLimit)(c)
}

func (c FtAggregateOpSortbyFieldsProperty) Filter(filter string) FtAggregateOpFilter {
	c.cs.s = append(c.cs.s, "FILTER", filter)
	return (FtAggregateOpFilter)(c)
}

func (c FtAggregateOpSortbyFieldsProperty) LoadAll() FtAggregateOpLoadallLoadAll {
	c.cs.s = append(c.cs.s, "LOAD", "*")
	return (FtAggregateOpLoadallLoadAll)(c)
}

func (c FtAggregateOpSortbyFieldsProperty) Load(count int64) FtAggregateOpLoadLoad {
	c.cs.s = append(c.cs.s, "LOAD", strconv.FormatInt(count, 10))
	return (FtAggregateOpLoadLoad)(c)
}

func (c FtAggregateOpSortbyFieldsProperty) Groupby(nargs int64) FtAggregateOpGroupbyGroupby {
	c.cs.s = append(c.cs.s, "GROUPBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpGroupbyGroupby)(c)
}

func (c FtAggregateOpSortbyFieldsProperty) Sortby(nargs int64) FtAggregateOpSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpSortbySortby)(c)
}

func (c FtAggregateOpSortbyFieldsProperty) Withcursor() FtAggregateCursorWithcursor {
	c.cs.s = append(c.cs.s, "WITHCURSOR")
	return (FtAggregateCursorWithcursor)(c)
}

func (c FtAggregateOpSortbyFieldsProperty) Params() FtAggregateParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtAggregateParamsParams)(c)
}

func (c FtAggregateOpSortbyFieldsProperty) Dialect(dialect int64) FtAggregateDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtAggregateDialect)(c)
}

func (c FtAggregateOpSortbyFieldsProperty) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAggregateOpSortbyMax Completed

func (c FtAggregateOpSortbyMax) Apply(expression string) FtAggregateOpApplyApply {
	c.cs.s = append(c.cs.s, "APPLY", expression)
	return (FtAggregateOpApplyApply)(c)
}

func (c FtAggregateOpSortbyMax) Limit() FtAggregateOpLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtAggregateOpLimitLimit)(c)
}

func (c FtAggregateOpSortbyMax) Filter(filter string) FtAggregateOpFilter {
	c.cs.s = append(c.cs.s, "FILTER", filter)
	return (FtAggregateOpFilter)(c)
}

func (c FtAggregateOpSortbyMax) LoadAll() FtAggregateOpLoadallLoadAll {
	c.cs.s = append(c.cs.s, "LOAD", "*")
	return (FtAggregateOpLoadallLoadAll)(c)
}

func (c FtAggregateOpSortbyMax) Load(count int64) FtAggregateOpLoadLoad {
	c.cs.s = append(c.cs.s, "LOAD", strconv.FormatInt(count, 10))
	return (FtAggregateOpLoadLoad)(c)
}

func (c FtAggregateOpSortbyMax) Groupby(nargs int64) FtAggregateOpGroupbyGroupby {
	c.cs.s = append(c.cs.s, "GROUPBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpGroupbyGroupby)(c)
}

func (c FtAggregateOpSortbyMax) Sortby(nargs int64) FtAggregateOpSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpSortbySortby)(c)
}

func (c FtAggregateOpSortbyMax) Withcursor() FtAggregateCursorWithcursor {
	c.cs.s = append(c.cs.s, "WITHCURSOR")
	return (FtAggregateCursorWithcursor)(c)
}

func (c FtAggregateOpSortbyMax) Params() FtAggregateParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtAggregateParamsParams)(c)
}

func (c FtAggregateOpSortbyMax) Dialect(dialect int64) FtAggregateDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtAggregateDialect)(c)
}

func (c FtAggregateOpSortbyMax) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAggregateOpSortbySortby Completed

func (c FtAggregateOpSortbySortby) Property(property string) FtAggregateOpSortbyFieldsProperty {
	c.cs.s = append(c.cs.s, property)
	return (FtAggregateOpSortbyFieldsProperty)(c)
}

func (c FtAggregateOpSortbySortby) Max(num int64) FtAggregateOpSortbyMax {
	c.cs.s = append(c.cs.s, "MAX", strconv.FormatInt(num, 10))
	return (FtAggregateOpSortbyMax)(c)
}

func (c FtAggregateOpSortbySortby) Apply(expression string) FtAggregateOpApplyApply {
	c.cs.s = append(c.cs.s, "APPLY", expression)
	return (FtAggregateOpApplyApply)(c)
}

func (c FtAggregateOpSortbySortby) Limit() FtAggregateOpLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtAggregateOpLimitLimit)(c)
}

func (c FtAggregateOpSortbySortby) Filter(filter string) FtAggregateOpFilter {
	c.cs.s = append(c.cs.s, "FILTER", filter)
	return (FtAggregateOpFilter)(c)
}

func (c FtAggregateOpSortbySortby) LoadAll() FtAggregateOpLoadallLoadAll {
	c.cs.s = append(c.cs.s, "LOAD", "*")
	return (FtAggregateOpLoadallLoadAll)(c)
}

func (c FtAggregateOpSortbySortby) Load(count int64) FtAggregateOpLoadLoad {
	c.cs.s = append(c.cs.s, "LOAD", strconv.FormatInt(count, 10))
	return (FtAggregateOpLoadLoad)(c)
}

func (c FtAggregateOpSortbySortby) Groupby(nargs int64) FtAggregateOpGroupbyGroupby {
	c.cs.s = append(c.cs.s, "GROUPBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpGroupbyGroupby)(c)
}

func (c FtAggregateOpSortbySortby) Sortby(nargs int64) FtAggregateOpSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", strconv.FormatInt(nargs, 10))
	return c
}

func (c FtAggregateOpSortbySortby) Withcursor() FtAggregateCursorWithcursor {
	c.cs.s = append(c.cs.s, "WITHCURSOR")
	return (FtAggregateCursorWithcursor)(c)
}

func (c FtAggregateOpSortbySortby) Params() FtAggregateParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtAggregateParamsParams)(c)
}

func (c FtAggregateOpSortbySortby) Dialect(dialect int64) FtAggregateDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtAggregateDialect)(c)
}

func (c FtAggregateOpSortbySortby) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAggregateParamsNameValue Completed

func (c FtAggregateParamsNameValue) NameValue(name string, value string) FtAggregateParamsNameValue {
	c.cs.s = append(c.cs.s, name, value)
	return c
}

func (c FtAggregateParamsNameValue) Dialect(dialect int64) FtAggregateDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtAggregateDialect)(c)
}

func (c FtAggregateParamsNameValue) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAggregateParamsNargs Completed

func (c FtAggregateParamsNargs) NameValue() FtAggregateParamsNameValue {
	return (FtAggregateParamsNameValue)(c)
}

type FtAggregateParamsParams Completed

func (c FtAggregateParamsParams) Nargs(nargs int64) FtAggregateParamsNargs {
	c.cs.s = append(c.cs.s, strconv.FormatInt(nargs, 10))
	return (FtAggregateParamsNargs)(c)
}

type FtAggregateQuery Completed

func (c FtAggregateQuery) Verbatim() FtAggregateVerbatim {
	c.cs.s = append(c.cs.s, "VERBATIM")
	return (FtAggregateVerbatim)(c)
}

func (c FtAggregateQuery) Timeout(timeout int64) FtAggregateTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (FtAggregateTimeout)(c)
}

func (c FtAggregateQuery) LoadAll() FtAggregateOpLoadallLoadAll {
	c.cs.s = append(c.cs.s, "LOAD", "*")
	return (FtAggregateOpLoadallLoadAll)(c)
}

func (c FtAggregateQuery) Load(count int64) FtAggregateOpLoadLoad {
	c.cs.s = append(c.cs.s, "LOAD", strconv.FormatInt(count, 10))
	return (FtAggregateOpLoadLoad)(c)
}

func (c FtAggregateQuery) Groupby(nargs int64) FtAggregateOpGroupbyGroupby {
	c.cs.s = append(c.cs.s, "GROUPBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpGroupbyGroupby)(c)
}

func (c FtAggregateQuery) Sortby(nargs int64) FtAggregateOpSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpSortbySortby)(c)
}

func (c FtAggregateQuery) Apply(expression string) FtAggregateOpApplyApply {
	c.cs.s = append(c.cs.s, "APPLY", expression)
	return (FtAggregateOpApplyApply)(c)
}

func (c FtAggregateQuery) Limit() FtAggregateOpLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtAggregateOpLimitLimit)(c)
}

func (c FtAggregateQuery) Filter(filter string) FtAggregateOpFilter {
	c.cs.s = append(c.cs.s, "FILTER", filter)
	return (FtAggregateOpFilter)(c)
}

func (c FtAggregateQuery) Withcursor() FtAggregateCursorWithcursor {
	c.cs.s = append(c.cs.s, "WITHCURSOR")
	return (FtAggregateCursorWithcursor)(c)
}

func (c FtAggregateQuery) Params() FtAggregateParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtAggregateParamsParams)(c)
}

func (c FtAggregateQuery) Dialect(dialect int64) FtAggregateDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtAggregateDialect)(c)
}

func (c FtAggregateQuery) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAggregateTimeout Completed

func (c FtAggregateTimeout) LoadAll() FtAggregateOpLoadallLoadAll {
	c.cs.s = append(c.cs.s, "LOAD", "*")
	return (FtAggregateOpLoadallLoadAll)(c)
}

func (c FtAggregateTimeout) Load(count int64) FtAggregateOpLoadLoad {
	c.cs.s = append(c.cs.s, "LOAD", strconv.FormatInt(count, 10))
	return (FtAggregateOpLoadLoad)(c)
}

func (c FtAggregateTimeout) Groupby(nargs int64) FtAggregateOpGroupbyGroupby {
	c.cs.s = append(c.cs.s, "GROUPBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpGroupbyGroupby)(c)
}

func (c FtAggregateTimeout) Sortby(nargs int64) FtAggregateOpSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpSortbySortby)(c)
}

func (c FtAggregateTimeout) Apply(expression string) FtAggregateOpApplyApply {
	c.cs.s = append(c.cs.s, "APPLY", expression)
	return (FtAggregateOpApplyApply)(c)
}

func (c FtAggregateTimeout) Limit() FtAggregateOpLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtAggregateOpLimitLimit)(c)
}

func (c FtAggregateTimeout) Filter(filter string) FtAggregateOpFilter {
	c.cs.s = append(c.cs.s, "FILTER", filter)
	return (FtAggregateOpFilter)(c)
}

func (c FtAggregateTimeout) Withcursor() FtAggregateCursorWithcursor {
	c.cs.s = append(c.cs.s, "WITHCURSOR")
	return (FtAggregateCursorWithcursor)(c)
}

func (c FtAggregateTimeout) Params() FtAggregateParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtAggregateParamsParams)(c)
}

func (c FtAggregateTimeout) Dialect(dialect int64) FtAggregateDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtAggregateDialect)(c)
}

func (c FtAggregateTimeout) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAggregateVerbatim Completed

func (c FtAggregateVerbatim) Timeout(timeout int64) FtAggregateTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (FtAggregateTimeout)(c)
}

func (c FtAggregateVerbatim) LoadAll() FtAggregateOpLoadallLoadAll {
	c.cs.s = append(c.cs.s, "LOAD", "*")
	return (FtAggregateOpLoadallLoadAll)(c)
}

func (c FtAggregateVerbatim) Load(count int64) FtAggregateOpLoadLoad {
	c.cs.s = append(c.cs.s, "LOAD", strconv.FormatInt(count, 10))
	return (FtAggregateOpLoadLoad)(c)
}

func (c FtAggregateVerbatim) Groupby(nargs int64) FtAggregateOpGroupbyGroupby {
	c.cs.s = append(c.cs.s, "GROUPBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpGroupbyGroupby)(c)
}

func (c FtAggregateVerbatim) Sortby(nargs int64) FtAggregateOpSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpSortbySortby)(c)
}

func (c FtAggregateVerbatim) Apply(expression string) FtAggregateOpApplyApply {
	c.cs.s = append(c.cs.s, "APPLY", expression)
	return (FtAggregateOpApplyApply)(c)
}

func (c FtAggregateVerbatim) Limit() FtAggregateOpLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtAggregateOpLimitLimit)(c)
}

func (c FtAggregateVerbatim) Filter(filter string) FtAggregateOpFilter {
	c.cs.s = append(c.cs.s, "FILTER", filter)
	return (FtAggregateOpFilter)(c)
}

func (c FtAggregateVerbatim) Withcursor() FtAggregateCursorWithcursor {
	c.cs.s = append(c.cs.s, "WITHCURSOR")
	return (FtAggregateCursorWithcursor)(c)
}

func (c FtAggregateVerbatim) Params() FtAggregateParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtAggregateParamsParams)(c)
}

func (c FtAggregateVerbatim) Dialect(dialect int64) FtAggregateDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtAggregateDialect)(c)
}

func (c FtAggregateVerbatim) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAliasadd Completed

func (b Builder) FtAliasadd() (c FtAliasadd) {
	c = FtAliasadd{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FT.ALIASADD")
	return c
}

func (c FtAliasadd) Alias(alias string) FtAliasaddAlias {
	c.cs.s = append(c.cs.s, alias)
	return (FtAliasaddAlias)(c)
}

type FtAliasaddAlias Completed

func (c FtAliasaddAlias) Index(index string) FtAliasaddIndex {
	c.cs.s = append(c.cs.s, index)
	return (FtAliasaddIndex)(c)
}

type FtAliasaddIndex Completed

func (c FtAliasaddIndex) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAliasdel Completed

func (b Builder) FtAliasdel() (c FtAliasdel) {
	c = FtAliasdel{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FT.ALIASDEL")
	return c
}

func (c FtAliasdel) Alias(alias string) FtAliasdelAlias {
	c.cs.s = append(c.cs.s, alias)
	return (FtAliasdelAlias)(c)
}

type FtAliasdelAlias Completed

func (c FtAliasdelAlias) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAliasupdate Completed

func (b Builder) FtAliasupdate() (c FtAliasupdate) {
	c = FtAliasupdate{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FT.ALIASUPDATE")
	return c
}

func (c FtAliasupdate) Alias(alias string) FtAliasupdateAlias {
	c.cs.s = append(c.cs.s, alias)
	return (FtAliasupdateAlias)(c)
}

type FtAliasupdateAlias Completed

func (c FtAliasupdateAlias) Index(index string) FtAliasupdateIndex {
	c.cs.s = append(c.cs.s, index)
	return (FtAliasupdateIndex)(c)
}

type FtAliasupdateIndex Completed

func (c FtAliasupdateIndex) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAlter Completed

func (b Builder) FtAlter() (c FtAlter) {
	c = FtAlter{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FT.ALTER")
	return c
}

func (c FtAlter) Index(index string) FtAlterIndex {
	c.cs.s = append(c.cs.s, index)
	return (FtAlterIndex)(c)
}

type FtAlterAdd Completed

func (c FtAlterAdd) Field(field string) FtAlterField {
	c.cs.s = append(c.cs.s, field)
	return (FtAlterField)(c)
}

type FtAlterField Completed

func (c FtAlterField) Options(options string) FtAlterOptions {
	c.cs.s = append(c.cs.s, options)
	return (FtAlterOptions)(c)
}

type FtAlterIndex Completed

func (c FtAlterIndex) Skipinitialscan() FtAlterSkipinitialscan {
	c.cs.s = append(c.cs.s, "SKIPINITIALSCAN")
	return (FtAlterSkipinitialscan)(c)
}

func (c FtAlterIndex) Schema() FtAlterSchema {
	c.cs.s = append(c.cs.s, "SCHEMA")
	return (FtAlterSchema)(c)
}

type FtAlterOptions Completed

func (c FtAlterOptions) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAlterSchema Completed

func (c FtAlterSchema) Add() FtAlterAdd {
	c.cs.s = append(c.cs.s, "ADD")
	return (FtAlterAdd)(c)
}

type FtAlterSkipinitialscan Completed

func (c FtAlterSkipinitialscan) Schema() FtAlterSchema {
	c.cs.s = append(c.cs.s, "SCHEMA")
	return (FtAlterSchema)(c)
}

type FtConfigGet Completed

func (b Builder) FtConfigGet() (c FtConfigGet) {
	c = FtConfigGet{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FT.CONFIG", "GET")
	return c
}

func (c FtConfigGet) Option(option string) FtConfigGetOption {
	c.cs.s = append(c.cs.s, option)
	return (FtConfigGetOption)(c)
}

type FtConfigGetOption Completed

func (c FtConfigGetOption) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtConfigHelp Completed

func (b Builder) FtConfigHelp() (c FtConfigHelp) {
	c = FtConfigHelp{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FT.CONFIG", "HELP")
	return c
}

func (c FtConfigHelp) Option(option string) FtConfigHelpOption {
	c.cs.s = append(c.cs.s, option)
	return (FtConfigHelpOption)(c)
}

type FtConfigHelpOption Completed

func (c FtConfigHelpOption) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtConfigSet Completed

func (b Builder) FtConfigSet() (c FtConfigSet) {
	c = FtConfigSet{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FT.CONFIG", "SET")
	return c
}

func (c FtConfigSet) Option(option string) FtConfigSetOption {
	c.cs.s = append(c.cs.s, option)
	return (FtConfigSetOption)(c)
}

type FtConfigSetOption Completed

func (c FtConfigSetOption) Value(value string) FtConfigSetValue {
	c.cs.s = append(c.cs.s, value)
	return (FtConfigSetValue)(c)
}

type FtConfigSetValue Completed

func (c FtConfigSetValue) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtCreate Completed

func (b Builder) FtCreate() (c FtCreate) {
	c = FtCreate{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FT.CREATE")
	return c
}

func (c FtCreate) Index(index string) FtCreateIndex {
	c.cs.s = append(c.cs.s, index)
	return (FtCreateIndex)(c)
}

type FtCreateFieldAs Completed

func (c FtCreateFieldAs) Text() FtCreateFieldFieldTypeText {
	c.cs.s = append(c.cs.s, "TEXT")
	return (FtCreateFieldFieldTypeText)(c)
}

func (c FtCreateFieldAs) Tag() FtCreateFieldFieldTypeTag {
	c.cs.s = append(c.cs.s, "TAG")
	return (FtCreateFieldFieldTypeTag)(c)
}

func (c FtCreateFieldAs) Numeric() FtCreateFieldFieldTypeNumeric {
	c.cs.s = append(c.cs.s, "NUMERIC")
	return (FtCreateFieldFieldTypeNumeric)(c)
}

func (c FtCreateFieldAs) Geo() FtCreateFieldFieldTypeGeo {
	c.cs.s = append(c.cs.s, "GEO")
	return (FtCreateFieldFieldTypeGeo)(c)
}

func (c FtCreateFieldAs) Vector(algo string, nargs int64, args ...string) FtCreateFieldFieldTypeVector {
	c.cs.s = append(c.cs.s, "VECTOR", algo, strconv.FormatInt(nargs, 10))
	c.cs.s = append(c.cs.s, args...)
	return (FtCreateFieldFieldTypeVector)(c)
}

type FtCreateFieldFieldName Completed

func (c FtCreateFieldFieldName) As(alias string) FtCreateFieldAs {
	c.cs.s = append(c.cs.s, "AS", alias)
	return (FtCreateFieldAs)(c)
}

func (c FtCreateFieldFieldName) Text() FtCreateFieldFieldTypeText {
	c.cs.s = append(c.cs.s, "TEXT")
	return (FtCreateFieldFieldTypeText)(c)
}

func (c FtCreateFieldFieldName) Tag() FtCreateFieldFieldTypeTag {
	c.cs.s = append(c.cs.s, "TAG")
	return (FtCreateFieldFieldTypeTag)(c)
}

func (c FtCreateFieldFieldName) Numeric() FtCreateFieldFieldTypeNumeric {
	c.cs.s = append(c.cs.s, "NUMERIC")
	return (FtCreateFieldFieldTypeNumeric)(c)
}

func (c FtCreateFieldFieldName) Geo() FtCreateFieldFieldTypeGeo {
	c.cs.s = append(c.cs.s, "GEO")
	return (FtCreateFieldFieldTypeGeo)(c)
}

func (c FtCreateFieldFieldName) Vector(algo string, nargs int64, args ...string) FtCreateFieldFieldTypeVector {
	c.cs.s = append(c.cs.s, "VECTOR", algo, strconv.FormatInt(nargs, 10))
	c.cs.s = append(c.cs.s, args...)
	return (FtCreateFieldFieldTypeVector)(c)
}

type FtCreateFieldFieldTypeGeo Completed

func (c FtCreateFieldFieldTypeGeo) Withsuffixtrie() FtCreateFieldOptionWithsuffixtrie {
	c.cs.s = append(c.cs.s, "WITHSUFFIXTRIE")
	return (FtCreateFieldOptionWithsuffixtrie)(c)
}

func (c FtCreateFieldFieldTypeGeo) Sortable() FtCreateFieldOptionSortableSortable {
	c.cs.s = append(c.cs.s, "SORTABLE")
	return (FtCreateFieldOptionSortableSortable)(c)
}

func (c FtCreateFieldFieldTypeGeo) Noindex() FtCreateFieldOptionNoindex {
	c.cs.s = append(c.cs.s, "NOINDEX")
	return (FtCreateFieldOptionNoindex)(c)
}

func (c FtCreateFieldFieldTypeGeo) Nostem() FtCreateFieldOptionNostem {
	c.cs.s = append(c.cs.s, "NOSTEM")
	return (FtCreateFieldOptionNostem)(c)
}

func (c FtCreateFieldFieldTypeGeo) Phonetic(phonetic string) FtCreateFieldOptionPhonetic {
	c.cs.s = append(c.cs.s, "PHONETIC", phonetic)
	return (FtCreateFieldOptionPhonetic)(c)
}

func (c FtCreateFieldFieldTypeGeo) Weight(weight float64) FtCreateFieldOptionWeight {
	c.cs.s = append(c.cs.s, "WEIGHT", strconv.FormatFloat(weight, 'f', -1, 64))
	return (FtCreateFieldOptionWeight)(c)
}

func (c FtCreateFieldFieldTypeGeo) Separator(separator string) FtCreateFieldOptionSeparator {
	c.cs.s = append(c.cs.s, "SEPARATOR", separator)
	return (FtCreateFieldOptionSeparator)(c)
}

func (c FtCreateFieldFieldTypeGeo) Casesensitive() FtCreateFieldOptionCasesensitive {
	c.cs.s = append(c.cs.s, "CASESENSITIVE")
	return (FtCreateFieldOptionCasesensitive)(c)
}

func (c FtCreateFieldFieldTypeGeo) FieldName(fieldName string) FtCreateFieldFieldName {
	c.cs.s = append(c.cs.s, fieldName)
	return (FtCreateFieldFieldName)(c)
}

func (c FtCreateFieldFieldTypeGeo) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtCreateFieldFieldTypeNumeric Completed

func (c FtCreateFieldFieldTypeNumeric) Withsuffixtrie() FtCreateFieldOptionWithsuffixtrie {
	c.cs.s = append(c.cs.s, "WITHSUFFIXTRIE")
	return (FtCreateFieldOptionWithsuffixtrie)(c)
}

func (c FtCreateFieldFieldTypeNumeric) Sortable() FtCreateFieldOptionSortableSortable {
	c.cs.s = append(c.cs.s, "SORTABLE")
	return (FtCreateFieldOptionSortableSortable)(c)
}

func (c FtCreateFieldFieldTypeNumeric) Noindex() FtCreateFieldOptionNoindex {
	c.cs.s = append(c.cs.s, "NOINDEX")
	return (FtCreateFieldOptionNoindex)(c)
}

func (c FtCreateFieldFieldTypeNumeric) Nostem() FtCreateFieldOptionNostem {
	c.cs.s = append(c.cs.s, "NOSTEM")
	return (FtCreateFieldOptionNostem)(c)
}

func (c FtCreateFieldFieldTypeNumeric) Phonetic(phonetic string) FtCreateFieldOptionPhonetic {
	c.cs.s = append(c.cs.s, "PHONETIC", phonetic)
	return (FtCreateFieldOptionPhonetic)(c)
}

func (c FtCreateFieldFieldTypeNumeric) Weight(weight float64) FtCreateFieldOptionWeight {
	c.cs.s = append(c.cs.s, "WEIGHT", strconv.FormatFloat(weight, 'f', -1, 64))
	return (FtCreateFieldOptionWeight)(c)
}

func (c FtCreateFieldFieldTypeNumeric) Separator(separator string) FtCreateFieldOptionSeparator {
	c.cs.s = append(c.cs.s, "SEPARATOR", separator)
	return (FtCreateFieldOptionSeparator)(c)
}

func (c FtCreateFieldFieldTypeNumeric) Casesensitive() FtCreateFieldOptionCasesensitive {
	c.cs.s = append(c.cs.s, "CASESENSITIVE")
	return (FtCreateFieldOptionCasesensitive)(c)
}

func (c FtCreateFieldFieldTypeNumeric) FieldName(fieldName string) FtCreateFieldFieldName {
	c.cs.s = append(c.cs.s, fieldName)
	return (FtCreateFieldFieldName)(c)
}

func (c FtCreateFieldFieldTypeNumeric) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtCreateFieldFieldTypeTag Completed

func (c FtCreateFieldFieldTypeTag) Withsuffixtrie() FtCreateFieldOptionWithsuffixtrie {
	c.cs.s = append(c.cs.s, "WITHSUFFIXTRIE")
	return (FtCreateFieldOptionWithsuffixtrie)(c)
}

func (c FtCreateFieldFieldTypeTag) Sortable() FtCreateFieldOptionSortableSortable {
	c.cs.s = append(c.cs.s, "SORTABLE")
	return (FtCreateFieldOptionSortableSortable)(c)
}

func (c FtCreateFieldFieldTypeTag) Noindex() FtCreateFieldOptionNoindex {
	c.cs.s = append(c.cs.s, "NOINDEX")
	return (FtCreateFieldOptionNoindex)(c)
}

func (c FtCreateFieldFieldTypeTag) Nostem() FtCreateFieldOptionNostem {
	c.cs.s = append(c.cs.s, "NOSTEM")
	return (FtCreateFieldOptionNostem)(c)
}

func (c FtCreateFieldFieldTypeTag) Phonetic(phonetic string) FtCreateFieldOptionPhonetic {
	c.cs.s = append(c.cs.s, "PHONETIC", phonetic)
	return (FtCreateFieldOptionPhonetic)(c)
}

func (c FtCreateFieldFieldTypeTag) Weight(weight float64) FtCreateFieldOptionWeight {
	c.cs.s = append(c.cs.s, "WEIGHT", strconv.FormatFloat(weight, 'f', -1, 64))
	return (FtCreateFieldOptionWeight)(c)
}

func (c FtCreateFieldFieldTypeTag) Separator(separator string) FtCreateFieldOptionSeparator {
	c.cs.s = append(c.cs.s, "SEPARATOR", separator)
	return (FtCreateFieldOptionSeparator)(c)
}

func (c FtCreateFieldFieldTypeTag) Casesensitive() FtCreateFieldOptionCasesensitive {
	c.cs.s = append(c.cs.s, "CASESENSITIVE")
	return (FtCreateFieldOptionCasesensitive)(c)
}

func (c FtCreateFieldFieldTypeTag) FieldName(fieldName string) FtCreateFieldFieldName {
	c.cs.s = append(c.cs.s, fieldName)
	return (FtCreateFieldFieldName)(c)
}

func (c FtCreateFieldFieldTypeTag) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtCreateFieldFieldTypeText Completed

func (c FtCreateFieldFieldTypeText) Withsuffixtrie() FtCreateFieldOptionWithsuffixtrie {
	c.cs.s = append(c.cs.s, "WITHSUFFIXTRIE")
	return (FtCreateFieldOptionWithsuffixtrie)(c)
}

func (c FtCreateFieldFieldTypeText) Sortable() FtCreateFieldOptionSortableSortable {
	c.cs.s = append(c.cs.s, "SORTABLE")
	return (FtCreateFieldOptionSortableSortable)(c)
}

func (c FtCreateFieldFieldTypeText) Noindex() FtCreateFieldOptionNoindex {
	c.cs.s = append(c.cs.s, "NOINDEX")
	return (FtCreateFieldOptionNoindex)(c)
}

func (c FtCreateFieldFieldTypeText) Nostem() FtCreateFieldOptionNostem {
	c.cs.s = append(c.cs.s, "NOSTEM")
	return (FtCreateFieldOptionNostem)(c)
}

func (c FtCreateFieldFieldTypeText) Phonetic(phonetic string) FtCreateFieldOptionPhonetic {
	c.cs.s = append(c.cs.s, "PHONETIC", phonetic)
	return (FtCreateFieldOptionPhonetic)(c)
}

func (c FtCreateFieldFieldTypeText) Weight(weight float64) FtCreateFieldOptionWeight {
	c.cs.s = append(c.cs.s, "WEIGHT", strconv.FormatFloat(weight, 'f', -1, 64))
	return (FtCreateFieldOptionWeight)(c)
}

func (c FtCreateFieldFieldTypeText) Separator(separator string) FtCreateFieldOptionSeparator {
	c.cs.s = append(c.cs.s, "SEPARATOR", separator)
	return (FtCreateFieldOptionSeparator)(c)
}

func (c FtCreateFieldFieldTypeText) Casesensitive() FtCreateFieldOptionCasesensitive {
	c.cs.s = append(c.cs.s, "CASESENSITIVE")
	return (FtCreateFieldOptionCasesensitive)(c)
}

func (c FtCreateFieldFieldTypeText) FieldName(fieldName string) FtCreateFieldFieldName {
	c.cs.s = append(c.cs.s, fieldName)
	return (FtCreateFieldFieldName)(c)
}

func (c FtCreateFieldFieldTypeText) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtCreateFieldFieldTypeVector Completed

func (c FtCreateFieldFieldTypeVector) Withsuffixtrie() FtCreateFieldOptionWithsuffixtrie {
	c.cs.s = append(c.cs.s, "WITHSUFFIXTRIE")
	return (FtCreateFieldOptionWithsuffixtrie)(c)
}

func (c FtCreateFieldFieldTypeVector) Sortable() FtCreateFieldOptionSortableSortable {
	c.cs.s = append(c.cs.s, "SORTABLE")
	return (FtCreateFieldOptionSortableSortable)(c)
}

func (c FtCreateFieldFieldTypeVector) Noindex() FtCreateFieldOptionNoindex {
	c.cs.s = append(c.cs.s, "NOINDEX")
	return (FtCreateFieldOptionNoindex)(c)
}

func (c FtCreateFieldFieldTypeVector) Nostem() FtCreateFieldOptionNostem {
	c.cs.s = append(c.cs.s, "NOSTEM")
	return (FtCreateFieldOptionNostem)(c)
}

func (c FtCreateFieldFieldTypeVector) Phonetic(phonetic string) FtCreateFieldOptionPhonetic {
	c.cs.s = append(c.cs.s, "PHONETIC", phonetic)
	return (FtCreateFieldOptionPhonetic)(c)
}

func (c FtCreateFieldFieldTypeVector) Weight(weight float64) FtCreateFieldOptionWeight {
	c.cs.s = append(c.cs.s, "WEIGHT", strconv.FormatFloat(weight, 'f', -1, 64))
	return (FtCreateFieldOptionWeight)(c)
}

func (c FtCreateFieldFieldTypeVector) Separator(separator string) FtCreateFieldOptionSeparator {
	c.cs.s = append(c.cs.s, "SEPARATOR", separator)
	return (FtCreateFieldOptionSeparator)(c)
}

func (c FtCreateFieldFieldTypeVector) Casesensitive() FtCreateFieldOptionCasesensitive {
	c.cs.s = append(c.cs.s, "CASESENSITIVE")
	return (FtCreateFieldOptionCasesensitive)(c)
}

func (c FtCreateFieldFieldTypeVector) FieldName(fieldName string) FtCreateFieldFieldName {
	c.cs.s = append(c.cs.s, fieldName)
	return (FtCreateFieldFieldName)(c)
}

func (c FtCreateFieldFieldTypeVector) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtCreateFieldOptionCasesensitive Completed

func (c FtCreateFieldOptionCasesensitive) Withsuffixtrie() FtCreateFieldOptionWithsuffixtrie {
	c.cs.s = append(c.cs.s, "WITHSUFFIXTRIE")
	return (FtCreateFieldOptionWithsuffixtrie)(c)
}

func (c FtCreateFieldOptionCasesensitive) Sortable() FtCreateFieldOptionSortableSortable {
	c.cs.s = append(c.cs.s, "SORTABLE")
	return (FtCreateFieldOptionSortableSortable)(c)
}

func (c FtCreateFieldOptionCasesensitive) Noindex() FtCreateFieldOptionNoindex {
	c.cs.s = append(c.cs.s, "NOINDEX")
	return (FtCreateFieldOptionNoindex)(c)
}

func (c FtCreateFieldOptionCasesensitive) Nostem() FtCreateFieldOptionNostem {
	c.cs.s = append(c.cs.s, "NOSTEM")
	return (FtCreateFieldOptionNostem)(c)
}

func (c FtCreateFieldOptionCasesensitive) Phonetic(phonetic string) FtCreateFieldOptionPhonetic {
	c.cs.s = append(c.cs.s, "PHONETIC", phonetic)
	return (FtCreateFieldOptionPhonetic)(c)
}

func (c FtCreateFieldOptionCasesensitive) Weight(weight float64) FtCreateFieldOptionWeight {
	c.cs.s = append(c.cs.s, "WEIGHT", strconv.FormatFloat(weight, 'f', -1, 64))
	return (FtCreateFieldOptionWeight)(c)
}

func (c FtCreateFieldOptionCasesensitive) Separator(separator string) FtCreateFieldOptionSeparator {
	c.cs.s = append(c.cs.s, "SEPARATOR", separator)
	return (FtCreateFieldOptionSeparator)(c)
}

func (c FtCreateFieldOptionCasesensitive) Casesensitive() FtCreateFieldOptionCasesensitive {
	c.cs.s = append(c.cs.s, "CASESENSITIVE")
	return c
}

func (c FtCreateFieldOptionCasesensitive) FieldName(fieldName string) FtCreateFieldFieldName {
	c.cs.s = append(c.cs.s, fieldName)
	return (FtCreateFieldFieldName)(c)
}

func (c FtCreateFieldOptionCasesensitive) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtCreateFieldOptionNoindex Completed

func (c FtCreateFieldOptionNoindex) Nostem() FtCreateFieldOptionNostem {
	c.cs.s = append(c.cs.s, "NOSTEM")
	return (FtCreateFieldOptionNostem)(c)
}

func (c FtCreateFieldOptionNoindex) Phonetic(phonetic string) FtCreateFieldOptionPhonetic {
	c.cs.s = append(c.cs.s, "PHONETIC", phonetic)
	return (FtCreateFieldOptionPhonetic)(c)
}

func (c FtCreateFieldOptionNoindex) Weight(weight float64) FtCreateFieldOptionWeight {
	c.cs.s = append(c.cs.s, "WEIGHT", strconv.FormatFloat(weight, 'f', -1, 64))
	return (FtCreateFieldOptionWeight)(c)
}

func (c FtCreateFieldOptionNoindex) Separator(separator string) FtCreateFieldOptionSeparator {
	c.cs.s = append(c.cs.s, "SEPARATOR", separator)
	return (FtCreateFieldOptionSeparator)(c)
}

func (c FtCreateFieldOptionNoindex) Casesensitive() FtCreateFieldOptionCasesensitive {
	c.cs.s = append(c.cs.s, "CASESENSITIVE")
	return (FtCreateFieldOptionCasesensitive)(c)
}

func (c FtCreateFieldOptionNoindex) Withsuffixtrie() FtCreateFieldOptionWithsuffixtrie {
	c.cs.s = append(c.cs.s, "WITHSUFFIXTRIE")
	return (FtCreateFieldOptionWithsuffixtrie)(c)
}

func (c FtCreateFieldOptionNoindex) Sortable() FtCreateFieldOptionSortableSortable {
	c.cs.s = append(c.cs.s, "SORTABLE")
	return (FtCreateFieldOptionSortableSortable)(c)
}

func (c FtCreateFieldOptionNoindex) Noindex() FtCreateFieldOptionNoindex {
	c.cs.s = append(c.cs.s, "NOINDEX")
	return c
}

func (c FtCreateFieldOptionNoindex) FieldName(fieldName string) FtCreateFieldFieldName {
	c.cs.s = append(c.cs.s, fieldName)
	return (FtCreateFieldFieldName)(c)
}

func (c FtCreateFieldOptionNoindex) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtCreateFieldOptionNostem Completed

func (c FtCreateFieldOptionNostem) Phonetic(phonetic string) FtCreateFieldOptionPhonetic {
	c.cs.s = append(c.cs.s, "PHONETIC", phonetic)
	return (FtCreateFieldOptionPhonetic)(c)
}

func (c FtCreateFieldOptionNostem) Weight(weight float64) FtCreateFieldOptionWeight {
	c.cs.s = append(c.cs.s, "WEIGHT", strconv.FormatFloat(weight, 'f', -1, 64))
	return (FtCreateFieldOptionWeight)(c)
}

func (c FtCreateFieldOptionNostem) Separator(separator string) FtCreateFieldOptionSeparator {
	c.cs.s = append(c.cs.s, "SEPARATOR", separator)
	return (FtCreateFieldOptionSeparator)(c)
}

func (c FtCreateFieldOptionNostem) Casesensitive() FtCreateFieldOptionCasesensitive {
	c.cs.s = append(c.cs.s, "CASESENSITIVE")
	return (FtCreateFieldOptionCasesensitive)(c)
}

func (c FtCreateFieldOptionNostem) Withsuffixtrie() FtCreateFieldOptionWithsuffixtrie {
	c.cs.s = append(c.cs.s, "WITHSUFFIXTRIE")
	return (FtCreateFieldOptionWithsuffixtrie)(c)
}

func (c FtCreateFieldOptionNostem) Sortable() FtCreateFieldOptionSortableSortable {
	c.cs.s = append(c.cs.s, "SORTABLE")
	return (FtCreateFieldOptionSortableSortable)(c)
}

func (c FtCreateFieldOptionNostem) Noindex() FtCreateFieldOptionNoindex {
	c.cs.s = append(c.cs.s, "NOINDEX")
	return (FtCreateFieldOptionNoindex)(c)
}

func (c FtCreateFieldOptionNostem) Nostem() FtCreateFieldOptionNostem {
	c.cs.s = append(c.cs.s, "NOSTEM")
	return c
}

func (c FtCreateFieldOptionNostem) FieldName(fieldName string) FtCreateFieldFieldName {
	c.cs.s = append(c.cs.s, fieldName)
	return (FtCreateFieldFieldName)(c)
}

func (c FtCreateFieldOptionNostem) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtCreateFieldOptionPhonetic Completed

func (c FtCreateFieldOptionPhonetic) Weight(weight float64) FtCreateFieldOptionWeight {
	c.cs.s = append(c.cs.s, "WEIGHT", strconv.FormatFloat(weight, 'f', -1, 64))
	return (FtCreateFieldOptionWeight)(c)
}

func (c FtCreateFieldOptionPhonetic) Separator(separator string) FtCreateFieldOptionSeparator {
	c.cs.s = append(c.cs.s, "SEPARATOR", separator)
	return (FtCreateFieldOptionSeparator)(c)
}

func (c FtCreateFieldOptionPhonetic) Casesensitive() FtCreateFieldOptionCasesensitive {
	c.cs.s = append(c.cs.s, "CASESENSITIVE")
	return (FtCreateFieldOptionCasesensitive)(c)
}

func (c FtCreateFieldOptionPhonetic) Withsuffixtrie() FtCreateFieldOptionWithsuffixtrie {
	c.cs.s = append(c.cs.s, "WITHSUFFIXTRIE")
	return (FtCreateFieldOptionWithsuffixtrie)(c)
}

func (c FtCreateFieldOptionPhonetic) Sortable() FtCreateFieldOptionSortableSortable {
	c.cs.s = append(c.cs.s, "SORTABLE")
	return (FtCreateFieldOptionSortableSortable)(c)
}

func (c FtCreateFieldOptionPhonetic) Noindex() FtCreateFieldOptionNoindex {
	c.cs.s = append(c.cs.s, "NOINDEX")
	return (FtCreateFieldOptionNoindex)(c)
}

func (c FtCreateFieldOptionPhonetic) Nostem() FtCreateFieldOptionNostem {
	c.cs.s = append(c.cs.s, "NOSTEM")
	return (FtCreateFieldOptionNostem)(c)
}

func (c FtCreateFieldOptionPhonetic) Phonetic(phonetic string) FtCreateFieldOptionPhonetic {
	c.cs.s = append(c.cs.s, "PHONETIC", phonetic)
	return c
}

func (c FtCreateFieldOptionPhonetic) FieldName(fieldName string) FtCreateFieldFieldName {
	c.cs.s = append(c.cs.s, fieldName)
	return (FtCreateFieldFieldName)(c)
}

func (c FtCreateFieldOptionPhonetic) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtCreateFieldOptionSeparator Completed

func (c FtCreateFieldOptionSeparator) Casesensitive() FtCreateFieldOptionCasesensitive {
	c.cs.s = append(c.cs.s, "CASESENSITIVE")
	return (FtCreateFieldOptionCasesensitive)(c)
}

func (c FtCreateFieldOptionSeparator) Withsuffixtrie() FtCreateFieldOptionWithsuffixtrie {
	c.cs.s = append(c.cs.s, "WITHSUFFIXTRIE")
	return (FtCreateFieldOptionWithsuffixtrie)(c)
}

func (c FtCreateFieldOptionSeparator) Sortable() FtCreateFieldOptionSortableSortable {
	c.cs.s = append(c.cs.s, "SORTABLE")
	return (FtCreateFieldOptionSortableSortable)(c)
}

func (c FtCreateFieldOptionSeparator) Noindex() FtCreateFieldOptionNoindex {
	c.cs.s = append(c.cs.s, "NOINDEX")
	return (FtCreateFieldOptionNoindex)(c)
}

func (c FtCreateFieldOptionSeparator) Nostem() FtCreateFieldOptionNostem {
	c.cs.s = append(c.cs.s, "NOSTEM")
	return (FtCreateFieldOptionNostem)(c)
}

func (c FtCreateFieldOptionSeparator) Phonetic(phonetic string) FtCreateFieldOptionPhonetic {
	c.cs.s = append(c.cs.s, "PHONETIC", phonetic)
	return (FtCreateFieldOptionPhonetic)(c)
}

func (c FtCreateFieldOptionSeparator) Weight(weight float64) FtCreateFieldOptionWeight {
	c.cs.s = append(c.cs.s, "WEIGHT", strconv.FormatFloat(weight, 'f', -1, 64))
	return (FtCreateFieldOptionWeight)(c)
}

func (c FtCreateFieldOptionSeparator) Separator(separator string) FtCreateFieldOptionSeparator {
	c.cs.s = append(c.cs.s, "SEPARATOR", separator)
	return c
}

func (c FtCreateFieldOptionSeparator) FieldName(fieldName string) FtCreateFieldFieldName {
	c.cs.s = append(c.cs.s, fieldName)
	return (FtCreateFieldFieldName)(c)
}

func (c FtCreateFieldOptionSeparator) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtCreateFieldOptionSortableSortable Completed

func (c FtCreateFieldOptionSortableSortable) Unf() FtCreateFieldOptionSortableUnf {
	c.cs.s = append(c.cs.s, "UNF")
	return (FtCreateFieldOptionSortableUnf)(c)
}

func (c FtCreateFieldOptionSortableSortable) Noindex() FtCreateFieldOptionNoindex {
	c.cs.s = append(c.cs.s, "NOINDEX")
	return (FtCreateFieldOptionNoindex)(c)
}

func (c FtCreateFieldOptionSortableSortable) Nostem() FtCreateFieldOptionNostem {
	c.cs.s = append(c.cs.s, "NOSTEM")
	return (FtCreateFieldOptionNostem)(c)
}

func (c FtCreateFieldOptionSortableSortable) Phonetic(phonetic string) FtCreateFieldOptionPhonetic {
	c.cs.s = append(c.cs.s, "PHONETIC", phonetic)
	return (FtCreateFieldOptionPhonetic)(c)
}

func (c FtCreateFieldOptionSortableSortable) Weight(weight float64) FtCreateFieldOptionWeight {
	c.cs.s = append(c.cs.s, "WEIGHT", strconv.FormatFloat(weight, 'f', -1, 64))
	return (FtCreateFieldOptionWeight)(c)
}

func (c FtCreateFieldOptionSortableSortable) Separator(separator string) FtCreateFieldOptionSeparator {
	c.cs.s = append(c.cs.s, "SEPARATOR", separator)
	return (FtCreateFieldOptionSeparator)(c)
}

func (c FtCreateFieldOptionSortableSortable) Casesensitive() FtCreateFieldOptionCasesensitive {
	c.cs.s = append(c.cs.s, "CASESENSITIVE")
	return (FtCreateFieldOptionCasesensitive)(c)
}

func (c FtCreateFieldOptionSortableSortable) Withsuffixtrie() FtCreateFieldOptionWithsuffixtrie {
	c.cs.s = append(c.cs.s, "WITHSUFFIXTRIE")
	return (FtCreateFieldOptionWithsuffixtrie)(c)
}

func (c FtCreateFieldOptionSortableSortable) Sortable() FtCreateFieldOptionSortableSortable {
	c.cs.s = append(c.cs.s, "SORTABLE")
	return c
}

func (c FtCreateFieldOptionSortableSortable) FieldName(fieldName string) FtCreateFieldFieldName {
	c.cs.s = append(c.cs.s, fieldName)
	return (FtCreateFieldFieldName)(c)
}

func (c FtCreateFieldOptionSortableSortable) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtCreateFieldOptionSortableUnf Completed

func (c FtCreateFieldOptionSortableUnf) Noindex() FtCreateFieldOptionNoindex {
	c.cs.s = append(c.cs.s, "NOINDEX")
	return (FtCreateFieldOptionNoindex)(c)
}

func (c FtCreateFieldOptionSortableUnf) Nostem() FtCreateFieldOptionNostem {
	c.cs.s = append(c.cs.s, "NOSTEM")
	return (FtCreateFieldOptionNostem)(c)
}

func (c FtCreateFieldOptionSortableUnf) Phonetic(phonetic string) FtCreateFieldOptionPhonetic {
	c.cs.s = append(c.cs.s, "PHONETIC", phonetic)
	return (FtCreateFieldOptionPhonetic)(c)
}

func (c FtCreateFieldOptionSortableUnf) Weight(weight float64) FtCreateFieldOptionWeight {
	c.cs.s = append(c.cs.s, "WEIGHT", strconv.FormatFloat(weight, 'f', -1, 64))
	return (FtCreateFieldOptionWeight)(c)
}

func (c FtCreateFieldOptionSortableUnf) Separator(separator string) FtCreateFieldOptionSeparator {
	c.cs.s = append(c.cs.s, "SEPARATOR", separator)
	return (FtCreateFieldOptionSeparator)(c)
}

func (c FtCreateFieldOptionSortableUnf) Casesensitive() FtCreateFieldOptionCasesensitive {
	c.cs.s = append(c.cs.s, "CASESENSITIVE")
	return (FtCreateFieldOptionCasesensitive)(c)
}

func (c FtCreateFieldOptionSortableUnf) Withsuffixtrie() FtCreateFieldOptionWithsuffixtrie {
	c.cs.s = append(c.cs.s, "WITHSUFFIXTRIE")
	return (FtCreateFieldOptionWithsuffixtrie)(c)
}

func (c FtCreateFieldOptionSortableUnf) Sortable() FtCreateFieldOptionSortableSortable {
	c.cs.s = append(c.cs.s, "SORTABLE")
	return (FtCreateFieldOptionSortableSortable)(c)
}

func (c FtCreateFieldOptionSortableUnf) FieldName(fieldName string) FtCreateFieldFieldName {
	c.cs.s = append(c.cs.s, fieldName)
	return (FtCreateFieldFieldName)(c)
}

func (c FtCreateFieldOptionSortableUnf) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtCreateFieldOptionWeight Completed

func (c FtCreateFieldOptionWeight) Separator(separator string) FtCreateFieldOptionSeparator {
	c.cs.s = append(c.cs.s, "SEPARATOR", separator)
	return (FtCreateFieldOptionSeparator)(c)
}

func (c FtCreateFieldOptionWeight) Casesensitive() FtCreateFieldOptionCasesensitive {
	c.cs.s = append(c.cs.s, "CASESENSITIVE")
	return (FtCreateFieldOptionCasesensitive)(c)
}

func (c FtCreateFieldOptionWeight) Withsuffixtrie() FtCreateFieldOptionWithsuffixtrie {
	c.cs.s = append(c.cs.s, "WITHSUFFIXTRIE")
	return (FtCreateFieldOptionWithsuffixtrie)(c)
}

func (c FtCreateFieldOptionWeight) Sortable() FtCreateFieldOptionSortableSortable {
	c.cs.s = append(c.cs.s, "SORTABLE")
	return (FtCreateFieldOptionSortableSortable)(c)
}

func (c FtCreateFieldOptionWeight) Noindex() FtCreateFieldOptionNoindex {
	c.cs.s = append(c.cs.s, "NOINDEX")
	return (FtCreateFieldOptionNoindex)(c)
}

func (c FtCreateFieldOptionWeight) Nostem() FtCreateFieldOptionNostem {
	c.cs.s = append(c.cs.s, "NOSTEM")
	return (FtCreateFieldOptionNostem)(c)
}

func (c FtCreateFieldOptionWeight) Phonetic(phonetic string) FtCreateFieldOptionPhonetic {
	c.cs.s = append(c.cs.s, "PHONETIC", phonetic)
	return (FtCreateFieldOptionPhonetic)(c)
}

func (c FtCreateFieldOptionWeight) Weight(weight float64) FtCreateFieldOptionWeight {
	c.cs.s = append(c.cs.s, "WEIGHT", strconv.FormatFloat(weight, 'f', -1, 64))
	return c
}

func (c FtCreateFieldOptionWeight) FieldName(fieldName string) FtCreateFieldFieldName {
	c.cs.s = append(c.cs.s, fieldName)
	return (FtCreateFieldFieldName)(c)
}

func (c FtCreateFieldOptionWeight) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtCreateFieldOptionWithsuffixtrie Completed

func (c FtCreateFieldOptionWithsuffixtrie) Sortable() FtCreateFieldOptionSortableSortable {
	c.cs.s = append(c.cs.s, "SORTABLE")
	return (FtCreateFieldOptionSortableSortable)(c)
}

func (c FtCreateFieldOptionWithsuffixtrie) Noindex() FtCreateFieldOptionNoindex {
	c.cs.s = append(c.cs.s, "NOINDEX")
	return (FtCreateFieldOptionNoindex)(c)
}

func (c FtCreateFieldOptionWithsuffixtrie) Nostem() FtCreateFieldOptionNostem {
	c.cs.s = append(c.cs.s, "NOSTEM")
	return (FtCreateFieldOptionNostem)(c)
}

func (c FtCreateFieldOptionWithsuffixtrie) Phonetic(phonetic string) FtCreateFieldOptionPhonetic {
	c.cs.s = append(c.cs.s, "PHONETIC", phonetic)
	return (FtCreateFieldOptionPhonetic)(c)
}

func (c FtCreateFieldOptionWithsuffixtrie) Weight(weight float64) FtCreateFieldOptionWeight {
	c.cs.s = append(c.cs.s, "WEIGHT", strconv.FormatFloat(weight, 'f', -1, 64))
	return (FtCreateFieldOptionWeight)(c)
}

func (c FtCreateFieldOptionWithsuffixtrie) Separator(separator string) FtCreateFieldOptionSeparator {
	c.cs.s = append(c.cs.s, "SEPARATOR", separator)
	return (FtCreateFieldOptionSeparator)(c)
}

func (c FtCreateFieldOptionWithsuffixtrie) Casesensitive() FtCreateFieldOptionCasesensitive {
	c.cs.s = append(c.cs.s, "CASESENSITIVE")
	return (FtCreateFieldOptionCasesensitive)(c)
}

func (c FtCreateFieldOptionWithsuffixtrie) Withsuffixtrie() FtCreateFieldOptionWithsuffixtrie {
	c.cs.s = append(c.cs.s, "WITHSUFFIXTRIE")
	return c
}

func (c FtCreateFieldOptionWithsuffixtrie) FieldName(fieldName string) FtCreateFieldFieldName {
	c.cs.s = append(c.cs.s, fieldName)
	return (FtCreateFieldFieldName)(c)
}

func (c FtCreateFieldOptionWithsuffixtrie) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtCreateFilter Completed

func (c FtCreateFilter) Language(defaultLang string) FtCreateLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", defaultLang)
	return (FtCreateLanguage)(c)
}

func (c FtCreateFilter) LanguageField(langAttribute string) FtCreateLanguageField {
	c.cs.s = append(c.cs.s, "LANGUAGE_FIELD", langAttribute)
	return (FtCreateLanguageField)(c)
}

func (c FtCreateFilter) Score(defaultScore float64) FtCreateScore {
	c.cs.s = append(c.cs.s, "SCORE", strconv.FormatFloat(defaultScore, 'f', -1, 64))
	return (FtCreateScore)(c)
}

func (c FtCreateFilter) ScoreField(scoreAttribute string) FtCreateScoreField {
	c.cs.s = append(c.cs.s, "SCORE_FIELD", scoreAttribute)
	return (FtCreateScoreField)(c)
}

func (c FtCreateFilter) PayloadField(payloadAttribute string) FtCreatePayloadField {
	c.cs.s = append(c.cs.s, "PAYLOAD_FIELD", payloadAttribute)
	return (FtCreatePayloadField)(c)
}

func (c FtCreateFilter) Maxtextfields() FtCreateMaxtextfields {
	c.cs.s = append(c.cs.s, "MAXTEXTFIELDS")
	return (FtCreateMaxtextfields)(c)
}

func (c FtCreateFilter) Temporary(seconds float64) FtCreateTemporary {
	c.cs.s = append(c.cs.s, "TEMPORARY", strconv.FormatFloat(seconds, 'f', -1, 64))
	return (FtCreateTemporary)(c)
}

func (c FtCreateFilter) Nooffsets() FtCreateNooffsets {
	c.cs.s = append(c.cs.s, "NOOFFSETS")
	return (FtCreateNooffsets)(c)
}

func (c FtCreateFilter) Nohl() FtCreateNohl {
	c.cs.s = append(c.cs.s, "NOHL")
	return (FtCreateNohl)(c)
}

func (c FtCreateFilter) Nofields() FtCreateNofields {
	c.cs.s = append(c.cs.s, "NOFIELDS")
	return (FtCreateNofields)(c)
}

func (c FtCreateFilter) Nofreqs() FtCreateNofreqs {
	c.cs.s = append(c.cs.s, "NOFREQS")
	return (FtCreateNofreqs)(c)
}

func (c FtCreateFilter) Stopwords(count int64) FtCreateStopwordsStopwords {
	c.cs.s = append(c.cs.s, "STOPWORDS", strconv.FormatInt(count, 10))
	return (FtCreateStopwordsStopwords)(c)
}

func (c FtCreateFilter) Skipinitialscan() FtCreateSkipinitialscan {
	c.cs.s = append(c.cs.s, "SKIPINITIALSCAN")
	return (FtCreateSkipinitialscan)(c)
}

func (c FtCreateFilter) Schema() FtCreateSchema {
	c.cs.s = append(c.cs.s, "SCHEMA")
	return (FtCreateSchema)(c)
}

type FtCreateIndex Completed

func (c FtCreateIndex) OnHash() FtCreateOnHash {
	c.cs.s = append(c.cs.s, "ON", "HASH")
	return (FtCreateOnHash)(c)
}

func (c FtCreateIndex) OnJson() FtCreateOnJson {
	c.cs.s = append(c.cs.s, "ON", "JSON")
	return (FtCreateOnJson)(c)
}

func (c FtCreateIndex) Prefix(count int64) FtCreatePrefixCount {
	c.cs.s = append(c.cs.s, "PREFIX", strconv.FormatInt(count, 10))
	return (FtCreatePrefixCount)(c)
}

func (c FtCreateIndex) Filter(filter string) FtCreateFilter {
	c.cs.s = append(c.cs.s, "FILTER", filter)
	return (FtCreateFilter)(c)
}

func (c FtCreateIndex) Language(defaultLang string) FtCreateLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", defaultLang)
	return (FtCreateLanguage)(c)
}

func (c FtCreateIndex) LanguageField(langAttribute string) FtCreateLanguageField {
	c.cs.s = append(c.cs.s, "LANGUAGE_FIELD", langAttribute)
	return (FtCreateLanguageField)(c)
}

func (c FtCreateIndex) Score(defaultScore float64) FtCreateScore {
	c.cs.s = append(c.cs.s, "SCORE", strconv.FormatFloat(defaultScore, 'f', -1, 64))
	return (FtCreateScore)(c)
}

func (c FtCreateIndex) ScoreField(scoreAttribute string) FtCreateScoreField {
	c.cs.s = append(c.cs.s, "SCORE_FIELD", scoreAttribute)
	return (FtCreateScoreField)(c)
}

func (c FtCreateIndex) PayloadField(payloadAttribute string) FtCreatePayloadField {
	c.cs.s = append(c.cs.s, "PAYLOAD_FIELD", payloadAttribute)
	return (FtCreatePayloadField)(c)
}

func (c FtCreateIndex) Maxtextfields() FtCreateMaxtextfields {
	c.cs.s = append(c.cs.s, "MAXTEXTFIELDS")
	return (FtCreateMaxtextfields)(c)
}

func (c FtCreateIndex) Temporary(seconds float64) FtCreateTemporary {
	c.cs.s = append(c.cs.s, "TEMPORARY", strconv.FormatFloat(seconds, 'f', -1, 64))
	return (FtCreateTemporary)(c)
}

func (c FtCreateIndex) Nooffsets() FtCreateNooffsets {
	c.cs.s = append(c.cs.s, "NOOFFSETS")
	return (FtCreateNooffsets)(c)
}

func (c FtCreateIndex) Nohl() FtCreateNohl {
	c.cs.s = append(c.cs.s, "NOHL")
	return (FtCreateNohl)(c)
}

func (c FtCreateIndex) Nofields() FtCreateNofields {
	c.cs.s = append(c.cs.s, "NOFIELDS")
	return (FtCreateNofields)(c)
}

func (c FtCreateIndex) Nofreqs() FtCreateNofreqs {
	c.cs.s = append(c.cs.s, "NOFREQS")
	return (FtCreateNofreqs)(c)
}

func (c FtCreateIndex) Stopwords(count int64) FtCreateStopwordsStopwords {
	c.cs.s = append(c.cs.s, "STOPWORDS", strconv.FormatInt(count, 10))
	return (FtCreateStopwordsStopwords)(c)
}

func (c FtCreateIndex) Skipinitialscan() FtCreateSkipinitialscan {
	c.cs.s = append(c.cs.s, "SKIPINITIALSCAN")
	return (FtCreateSkipinitialscan)(c)
}

func (c FtCreateIndex) Schema() FtCreateSchema {
	c.cs.s = append(c.cs.s, "SCHEMA")
	return (FtCreateSchema)(c)
}

type FtCreateLanguage Completed

func (c FtCreateLanguage) LanguageField(langAttribute string) FtCreateLanguageField {
	c.cs.s = append(c.cs.s, "LANGUAGE_FIELD", langAttribute)
	return (FtCreateLanguageField)(c)
}

func (c FtCreateLanguage) Score(defaultScore float64) FtCreateScore {
	c.cs.s = append(c.cs.s, "SCORE", strconv.FormatFloat(defaultScore, 'f', -1, 64))
	return (FtCreateScore)(c)
}

func (c FtCreateLanguage) ScoreField(scoreAttribute string) FtCreateScoreField {
	c.cs.s = append(c.cs.s, "SCORE_FIELD", scoreAttribute)
	return (FtCreateScoreField)(c)
}

func (c FtCreateLanguage) PayloadField(payloadAttribute string) FtCreatePayloadField {
	c.cs.s = append(c.cs.s, "PAYLOAD_FIELD", payloadAttribute)
	return (FtCreatePayloadField)(c)
}

func (c FtCreateLanguage) Maxtextfields() FtCreateMaxtextfields {
	c.cs.s = append(c.cs.s, "MAXTEXTFIELDS")
	return (FtCreateMaxtextfields)(c)
}

func (c FtCreateLanguage) Temporary(seconds float64) FtCreateTemporary {
	c.cs.s = append(c.cs.s, "TEMPORARY", strconv.FormatFloat(seconds, 'f', -1, 64))
	return (FtCreateTemporary)(c)
}

func (c FtCreateLanguage) Nooffsets() FtCreateNooffsets {
	c.cs.s = append(c.cs.s, "NOOFFSETS")
	return (FtCreateNooffsets)(c)
}

func (c FtCreateLanguage) Nohl() FtCreateNohl {
	c.cs.s = append(c.cs.s, "NOHL")
	return (FtCreateNohl)(c)
}

func (c FtCreateLanguage) Nofields() FtCreateNofields {
	c.cs.s = append(c.cs.s, "NOFIELDS")
	return (FtCreateNofields)(c)
}

func (c FtCreateLanguage) Nofreqs() FtCreateNofreqs {
	c.cs.s = append(c.cs.s, "NOFREQS")
	return (FtCreateNofreqs)(c)
}

func (c FtCreateLanguage) Stopwords(count int64) FtCreateStopwordsStopwords {
	c.cs.s = append(c.cs.s, "STOPWORDS", strconv.FormatInt(count, 10))
	return (FtCreateStopwordsStopwords)(c)
}

func (c FtCreateLanguage) Skipinitialscan() FtCreateSkipinitialscan {
	c.cs.s = append(c.cs.s, "SKIPINITIALSCAN")
	return (FtCreateSkipinitialscan)(c)
}

func (c FtCreateLanguage) Schema() FtCreateSchema {
	c.cs.s = append(c.cs.s, "SCHEMA")
	return (FtCreateSchema)(c)
}

type FtCreateLanguageField Completed

func (c FtCreateLanguageField) Score(defaultScore float64) FtCreateScore {
	c.cs.s = append(c.cs.s, "SCORE", strconv.FormatFloat(defaultScore, 'f', -1, 64))
	return (FtCreateScore)(c)
}

func (c FtCreateLanguageField) ScoreField(scoreAttribute string) FtCreateScoreField {
	c.cs.s = append(c.cs.s, "SCORE_FIELD", scoreAttribute)
	return (FtCreateScoreField)(c)
}

func (c FtCreateLanguageField) PayloadField(payloadAttribute string) FtCreatePayloadField {
	c.cs.s = append(c.cs.s, "PAYLOAD_FIELD", payloadAttribute)
	return (FtCreatePayloadField)(c)
}

func (c FtCreateLanguageField) Maxtextfields() FtCreateMaxtextfields {
	c.cs.s = append(c.cs.s, "MAXTEXTFIELDS")
	return (FtCreateMaxtextfields)(c)
}

func (c FtCreateLanguageField) Temporary(seconds float64) FtCreateTemporary {
	c.cs.s = append(c.cs.s, "TEMPORARY", strconv.FormatFloat(seconds, 'f', -1, 64))
	return (FtCreateTemporary)(c)
}

func (c FtCreateLanguageField) Nooffsets() FtCreateNooffsets {
	c.cs.s = append(c.cs.s, "NOOFFSETS")
	return (FtCreateNooffsets)(c)
}

func (c FtCreateLanguageField) Nohl() FtCreateNohl {
	c.cs.s = append(c.cs.s, "NOHL")
	return (FtCreateNohl)(c)
}

func (c FtCreateLanguageField) Nofields() FtCreateNofields {
	c.cs.s = append(c.cs.s, "NOFIELDS")
	return (FtCreateNofields)(c)
}

func (c FtCreateLanguageField) Nofreqs() FtCreateNofreqs {
	c.cs.s = append(c.cs.s, "NOFREQS")
	return (FtCreateNofreqs)(c)
}

func (c FtCreateLanguageField) Stopwords(count int64) FtCreateStopwordsStopwords {
	c.cs.s = append(c.cs.s, "STOPWORDS", strconv.FormatInt(count, 10))
	return (FtCreateStopwordsStopwords)(c)
}

func (c FtCreateLanguageField) Skipinitialscan() FtCreateSkipinitialscan {
	c.cs.s = append(c.cs.s, "SKIPINITIALSCAN")
	return (FtCreateSkipinitialscan)(c)
}

func (c FtCreateLanguageField) Schema() FtCreateSchema {
	c.cs.s = append(c.cs.s, "SCHEMA")
	return (FtCreateSchema)(c)
}

type FtCreateMaxtextfields Completed

func (c FtCreateMaxtextfields) Temporary(seconds float64) FtCreateTemporary {
	c.cs.s = append(c.cs.s, "TEMPORARY", strconv.FormatFloat(seconds, 'f', -1, 64))
	return (FtCreateTemporary)(c)
}

func (c FtCreateMaxtextfields) Nooffsets() FtCreateNooffsets {
	c.cs.s = append(c.cs.s, "NOOFFSETS")
	return (FtCreateNooffsets)(c)
}

func (c FtCreateMaxtextfields) Nohl() FtCreateNohl {
	c.cs.s = append(c.cs.s, "NOHL")
	return (FtCreateNohl)(c)
}

func (c FtCreateMaxtextfields) Nofields() FtCreateNofields {
	c.cs.s = append(c.cs.s, "NOFIELDS")
	return (FtCreateNofields)(c)
}

func (c FtCreateMaxtextfields) Nofreqs() FtCreateNofreqs {
	c.cs.s = append(c.cs.s, "NOFREQS")
	return (FtCreateNofreqs)(c)
}

func (c FtCreateMaxtextfields) Stopwords(count int64) FtCreateStopwordsStopwords {
	c.cs.s = append(c.cs.s, "STOPWORDS", strconv.FormatInt(count, 10))
	return (FtCreateStopwordsStopwords)(c)
}

func (c FtCreateMaxtextfields) Skipinitialscan() FtCreateSkipinitialscan {
	c.cs.s = append(c.cs.s, "SKIPINITIALSCAN")
	return (FtCreateSkipinitialscan)(c)
}

func (c FtCreateMaxtextfields) Schema() FtCreateSchema {
	c.cs.s = append(c.cs.s, "SCHEMA")
	return (FtCreateSchema)(c)
}

type FtCreateNofields Completed

func (c FtCreateNofields) Nofreqs() FtCreateNofreqs {
	c.cs.s = append(c.cs.s, "NOFREQS")
	return (FtCreateNofreqs)(c)
}

func (c FtCreateNofields) Stopwords(count int64) FtCreateStopwordsStopwords {
	c.cs.s = append(c.cs.s, "STOPWORDS", strconv.FormatInt(count, 10))
	return (FtCreateStopwordsStopwords)(c)
}

func (c FtCreateNofields) Skipinitialscan() FtCreateSkipinitialscan {
	c.cs.s = append(c.cs.s, "SKIPINITIALSCAN")
	return (FtCreateSkipinitialscan)(c)
}

func (c FtCreateNofields) Schema() FtCreateSchema {
	c.cs.s = append(c.cs.s, "SCHEMA")
	return (FtCreateSchema)(c)
}

type FtCreateNofreqs Completed

func (c FtCreateNofreqs) Stopwords(count int64) FtCreateStopwordsStopwords {
	c.cs.s = append(c.cs.s, "STOPWORDS", strconv.FormatInt(count, 10))
	return (FtCreateStopwordsStopwords)(c)
}

func (c FtCreateNofreqs) Skipinitialscan() FtCreateSkipinitialscan {
	c.cs.s = append(c.cs.s, "SKIPINITIALSCAN")
	return (FtCreateSkipinitialscan)(c)
}

func (c FtCreateNofreqs) Schema() FtCreateSchema {
	c.cs.s = append(c.cs.s, "SCHEMA")
	return (FtCreateSchema)(c)
}

type FtCreateNohl Completed

func (c FtCreateNohl) Nofields() FtCreateNofields {
	c.cs.s = append(c.cs.s, "NOFIELDS")
	return (FtCreateNofields)(c)
}

func (c FtCreateNohl) Nofreqs() FtCreateNofreqs {
	c.cs.s = append(c.cs.s, "NOFREQS")
	return (FtCreateNofreqs)(c)
}

func (c FtCreateNohl) Stopwords(count int64) FtCreateStopwordsStopwords {
	c.cs.s = append(c.cs.s, "STOPWORDS", strconv.FormatInt(count, 10))
	return (FtCreateStopwordsStopwords)(c)
}

func (c FtCreateNohl) Skipinitialscan() FtCreateSkipinitialscan {
	c.cs.s = append(c.cs.s, "SKIPINITIALSCAN")
	return (FtCreateSkipinitialscan)(c)
}

func (c FtCreateNohl) Schema() FtCreateSchema {
	c.cs.s = append(c.cs.s, "SCHEMA")
	return (FtCreateSchema)(c)
}

type FtCreateNooffsets Completed

func (c FtCreateNooffsets) Nohl() FtCreateNohl {
	c.cs.s = append(c.cs.s, "NOHL")
	return (FtCreateNohl)(c)
}

func (c FtCreateNooffsets) Nofields() FtCreateNofields {
	c.cs.s = append(c.cs.s, "NOFIELDS")
	return (FtCreateNofields)(c)
}

func (c FtCreateNooffsets) Nofreqs() FtCreateNofreqs {
	c.cs.s = append(c.cs.s, "NOFREQS")
	return (FtCreateNofreqs)(c)
}

func (c FtCreateNooffsets) Stopwords(count int64) FtCreateStopwordsStopwords {
	c.cs.s = append(c.cs.s, "STOPWORDS", strconv.FormatInt(count, 10))
	return (FtCreateStopwordsStopwords)(c)
}

func (c FtCreateNooffsets) Skipinitialscan() FtCreateSkipinitialscan {
	c.cs.s = append(c.cs.s, "SKIPINITIALSCAN")
	return (FtCreateSkipinitialscan)(c)
}

func (c FtCreateNooffsets) Schema() FtCreateSchema {
	c.cs.s = append(c.cs.s, "SCHEMA")
	return (FtCreateSchema)(c)
}

type FtCreateOnHash Completed

func (c FtCreateOnHash) Prefix(count int64) FtCreatePrefixCount {
	c.cs.s = append(c.cs.s, "PREFIX", strconv.FormatInt(count, 10))
	return (FtCreatePrefixCount)(c)
}

func (c FtCreateOnHash) Filter(filter string) FtCreateFilter {
	c.cs.s = append(c.cs.s, "FILTER", filter)
	return (FtCreateFilter)(c)
}

func (c FtCreateOnHash) Language(defaultLang string) FtCreateLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", defaultLang)
	return (FtCreateLanguage)(c)
}

func (c FtCreateOnHash) LanguageField(langAttribute string) FtCreateLanguageField {
	c.cs.s = append(c.cs.s, "LANGUAGE_FIELD", langAttribute)
	return (FtCreateLanguageField)(c)
}

func (c FtCreateOnHash) Score(defaultScore float64) FtCreateScore {
	c.cs.s = append(c.cs.s, "SCORE", strconv.FormatFloat(defaultScore, 'f', -1, 64))
	return (FtCreateScore)(c)
}

func (c FtCreateOnHash) ScoreField(scoreAttribute string) FtCreateScoreField {
	c.cs.s = append(c.cs.s, "SCORE_FIELD", scoreAttribute)
	return (FtCreateScoreField)(c)
}

func (c FtCreateOnHash) PayloadField(payloadAttribute string) FtCreatePayloadField {
	c.cs.s = append(c.cs.s, "PAYLOAD_FIELD", payloadAttribute)
	return (FtCreatePayloadField)(c)
}

func (c FtCreateOnHash) Maxtextfields() FtCreateMaxtextfields {
	c.cs.s = append(c.cs.s, "MAXTEXTFIELDS")
	return (FtCreateMaxtextfields)(c)
}

func (c FtCreateOnHash) Temporary(seconds float64) FtCreateTemporary {
	c.cs.s = append(c.cs.s, "TEMPORARY", strconv.FormatFloat(seconds, 'f', -1, 64))
	return (FtCreateTemporary)(c)
}

func (c FtCreateOnHash) Nooffsets() FtCreateNooffsets {
	c.cs.s = append(c.cs.s, "NOOFFSETS")
	return (FtCreateNooffsets)(c)
}

func (c FtCreateOnHash) Nohl() FtCreateNohl {
	c.cs.s = append(c.cs.s, "NOHL")
	return (FtCreateNohl)(c)
}

func (c FtCreateOnHash) Nofields() FtCreateNofields {
	c.cs.s = append(c.cs.s, "NOFIELDS")
	return (FtCreateNofields)(c)
}

func (c FtCreateOnHash) Nofreqs() FtCreateNofreqs {
	c.cs.s = append(c.cs.s, "NOFREQS")
	return (FtCreateNofreqs)(c)
}

func (c FtCreateOnHash) Stopwords(count int64) FtCreateStopwordsStopwords {
	c.cs.s = append(c.cs.s, "STOPWORDS", strconv.FormatInt(count, 10))
	return (FtCreateStopwordsStopwords)(c)
}

func (c FtCreateOnHash) Skipinitialscan() FtCreateSkipinitialscan {
	c.cs.s = append(c.cs.s, "SKIPINITIALSCAN")
	return (FtCreateSkipinitialscan)(c)
}

func (c FtCreateOnHash) Schema() FtCreateSchema {
	c.cs.s = append(c.cs.s, "SCHEMA")
	return (FtCreateSchema)(c)
}

type FtCreateOnJson Completed

func (c FtCreateOnJson) Prefix(count int64) FtCreatePrefixCount {
	c.cs.s = append(c.cs.s, "PREFIX", strconv.FormatInt(count, 10))
	return (FtCreatePrefixCount)(c)
}

func (c FtCreateOnJson) Filter(filter string) FtCreateFilter {
	c.cs.s = append(c.cs.s, "FILTER", filter)
	return (FtCreateFilter)(c)
}

func (c FtCreateOnJson) Language(defaultLang string) FtCreateLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", defaultLang)
	return (FtCreateLanguage)(c)
}

func (c FtCreateOnJson) LanguageField(langAttribute string) FtCreateLanguageField {
	c.cs.s = append(c.cs.s, "LANGUAGE_FIELD", langAttribute)
	return (FtCreateLanguageField)(c)
}

func (c FtCreateOnJson) Score(defaultScore float64) FtCreateScore {
	c.cs.s = append(c.cs.s, "SCORE", strconv.FormatFloat(defaultScore, 'f', -1, 64))
	return (FtCreateScore)(c)
}

func (c FtCreateOnJson) ScoreField(scoreAttribute string) FtCreateScoreField {
	c.cs.s = append(c.cs.s, "SCORE_FIELD", scoreAttribute)
	return (FtCreateScoreField)(c)
}

func (c FtCreateOnJson) PayloadField(payloadAttribute string) FtCreatePayloadField {
	c.cs.s = append(c.cs.s, "PAYLOAD_FIELD", payloadAttribute)
	return (FtCreatePayloadField)(c)
}

func (c FtCreateOnJson) Maxtextfields() FtCreateMaxtextfields {
	c.cs.s = append(c.cs.s, "MAXTEXTFIELDS")
	return (FtCreateMaxtextfields)(c)
}

func (c FtCreateOnJson) Temporary(seconds float64) FtCreateTemporary {
	c.cs.s = append(c.cs.s, "TEMPORARY", strconv.FormatFloat(seconds, 'f', -1, 64))
	return (FtCreateTemporary)(c)
}

func (c FtCreateOnJson) Nooffsets() FtCreateNooffsets {
	c.cs.s = append(c.cs.s, "NOOFFSETS")
	return (FtCreateNooffsets)(c)
}

func (c FtCreateOnJson) Nohl() FtCreateNohl {
	c.cs.s = append(c.cs.s, "NOHL")
	return (FtCreateNohl)(c)
}

func (c FtCreateOnJson) Nofields() FtCreateNofields {
	c.cs.s = append(c.cs.s, "NOFIELDS")
	return (FtCreateNofields)(c)
}

func (c FtCreateOnJson) Nofreqs() FtCreateNofreqs {
	c.cs.s = append(c.cs.s, "NOFREQS")
	return (FtCreateNofreqs)(c)
}

func (c FtCreateOnJson) Stopwords(count int64) FtCreateStopwordsStopwords {
	c.cs.s = append(c.cs.s, "STOPWORDS", strconv.FormatInt(count, 10))
	return (FtCreateStopwordsStopwords)(c)
}

func (c FtCreateOnJson) Skipinitialscan() FtCreateSkipinitialscan {
	c.cs.s = append(c.cs.s, "SKIPINITIALSCAN")
	return (FtCreateSkipinitialscan)(c)
}

func (c FtCreateOnJson) Schema() FtCreateSchema {
	c.cs.s = append(c.cs.s, "SCHEMA")
	return (FtCreateSchema)(c)
}

type FtCreatePayloadField Completed

func (c FtCreatePayloadField) Maxtextfields() FtCreateMaxtextfields {
	c.cs.s = append(c.cs.s, "MAXTEXTFIELDS")
	return (FtCreateMaxtextfields)(c)
}

func (c FtCreatePayloadField) Temporary(seconds float64) FtCreateTemporary {
	c.cs.s = append(c.cs.s, "TEMPORARY", strconv.FormatFloat(seconds, 'f', -1, 64))
	return (FtCreateTemporary)(c)
}

func (c FtCreatePayloadField) Nooffsets() FtCreateNooffsets {
	c.cs.s = append(c.cs.s, "NOOFFSETS")
	return (FtCreateNooffsets)(c)
}

func (c FtCreatePayloadField) Nohl() FtCreateNohl {
	c.cs.s = append(c.cs.s, "NOHL")
	return (FtCreateNohl)(c)
}

func (c FtCreatePayloadField) Nofields() FtCreateNofields {
	c.cs.s = append(c.cs.s, "NOFIELDS")
	return (FtCreateNofields)(c)
}

func (c FtCreatePayloadField) Nofreqs() FtCreateNofreqs {
	c.cs.s = append(c.cs.s, "NOFREQS")
	return (FtCreateNofreqs)(c)
}

func (c FtCreatePayloadField) Stopwords(count int64) FtCreateStopwordsStopwords {
	c.cs.s = append(c.cs.s, "STOPWORDS", strconv.FormatInt(count, 10))
	return (FtCreateStopwordsStopwords)(c)
}

func (c FtCreatePayloadField) Skipinitialscan() FtCreateSkipinitialscan {
	c.cs.s = append(c.cs.s, "SKIPINITIALSCAN")
	return (FtCreateSkipinitialscan)(c)
}

func (c FtCreatePayloadField) Schema() FtCreateSchema {
	c.cs.s = append(c.cs.s, "SCHEMA")
	return (FtCreateSchema)(c)
}

type FtCreatePrefixCount Completed

func (c FtCreatePrefixCount) Prefix(prefix ...string) FtCreatePrefixPrefix {
	c.cs.s = append(c.cs.s, prefix...)
	return (FtCreatePrefixPrefix)(c)
}

type FtCreatePrefixPrefix Completed

func (c FtCreatePrefixPrefix) Prefix(prefix ...string) FtCreatePrefixPrefix {
	c.cs.s = append(c.cs.s, prefix...)
	return c
}

func (c FtCreatePrefixPrefix) Filter(filter string) FtCreateFilter {
	c.cs.s = append(c.cs.s, "FILTER", filter)
	return (FtCreateFilter)(c)
}

func (c FtCreatePrefixPrefix) Language(defaultLang string) FtCreateLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", defaultLang)
	return (FtCreateLanguage)(c)
}

func (c FtCreatePrefixPrefix) LanguageField(langAttribute string) FtCreateLanguageField {
	c.cs.s = append(c.cs.s, "LANGUAGE_FIELD", langAttribute)
	return (FtCreateLanguageField)(c)
}

func (c FtCreatePrefixPrefix) Score(defaultScore float64) FtCreateScore {
	c.cs.s = append(c.cs.s, "SCORE", strconv.FormatFloat(defaultScore, 'f', -1, 64))
	return (FtCreateScore)(c)
}

func (c FtCreatePrefixPrefix) ScoreField(scoreAttribute string) FtCreateScoreField {
	c.cs.s = append(c.cs.s, "SCORE_FIELD", scoreAttribute)
	return (FtCreateScoreField)(c)
}

func (c FtCreatePrefixPrefix) PayloadField(payloadAttribute string) FtCreatePayloadField {
	c.cs.s = append(c.cs.s, "PAYLOAD_FIELD", payloadAttribute)
	return (FtCreatePayloadField)(c)
}

func (c FtCreatePrefixPrefix) Maxtextfields() FtCreateMaxtextfields {
	c.cs.s = append(c.cs.s, "MAXTEXTFIELDS")
	return (FtCreateMaxtextfields)(c)
}

func (c FtCreatePrefixPrefix) Temporary(seconds float64) FtCreateTemporary {
	c.cs.s = append(c.cs.s, "TEMPORARY", strconv.FormatFloat(seconds, 'f', -1, 64))
	return (FtCreateTemporary)(c)
}

func (c FtCreatePrefixPrefix) Nooffsets() FtCreateNooffsets {
	c.cs.s = append(c.cs.s, "NOOFFSETS")
	return (FtCreateNooffsets)(c)
}

func (c FtCreatePrefixPrefix) Nohl() FtCreateNohl {
	c.cs.s = append(c.cs.s, "NOHL")
	return (FtCreateNohl)(c)
}

func (c FtCreatePrefixPrefix) Nofields() FtCreateNofields {
	c.cs.s = append(c.cs.s, "NOFIELDS")
	return (FtCreateNofields)(c)
}

func (c FtCreatePrefixPrefix) Nofreqs() FtCreateNofreqs {
	c.cs.s = append(c.cs.s, "NOFREQS")
	return (FtCreateNofreqs)(c)
}

func (c FtCreatePrefixPrefix) Stopwords(count int64) FtCreateStopwordsStopwords {
	c.cs.s = append(c.cs.s, "STOPWORDS", strconv.FormatInt(count, 10))
	return (FtCreateStopwordsStopwords)(c)
}

func (c FtCreatePrefixPrefix) Skipinitialscan() FtCreateSkipinitialscan {
	c.cs.s = append(c.cs.s, "SKIPINITIALSCAN")
	return (FtCreateSkipinitialscan)(c)
}

func (c FtCreatePrefixPrefix) Schema() FtCreateSchema {
	c.cs.s = append(c.cs.s, "SCHEMA")
	return (FtCreateSchema)(c)
}

type FtCreateSchema Completed

func (c FtCreateSchema) FieldName(fieldName string) FtCreateFieldFieldName {
	c.cs.s = append(c.cs.s, fieldName)
	return (FtCreateFieldFieldName)(c)
}

type FtCreateScore Completed

func (c FtCreateScore) ScoreField(scoreAttribute string) FtCreateScoreField {
	c.cs.s = append(c.cs.s, "SCORE_FIELD", scoreAttribute)
	return (FtCreateScoreField)(c)
}

func (c FtCreateScore) PayloadField(payloadAttribute string) FtCreatePayloadField {
	c.cs.s = append(c.cs.s, "PAYLOAD_FIELD", payloadAttribute)
	return (FtCreatePayloadField)(c)
}

func (c FtCreateScore) Maxtextfields() FtCreateMaxtextfields {
	c.cs.s = append(c.cs.s, "MAXTEXTFIELDS")
	return (FtCreateMaxtextfields)(c)
}

func (c FtCreateScore) Temporary(seconds float64) FtCreateTemporary {
	c.cs.s = append(c.cs.s, "TEMPORARY", strconv.FormatFloat(seconds, 'f', -1, 64))
	return (FtCreateTemporary)(c)
}

func (c FtCreateScore) Nooffsets() FtCreateNooffsets {
	c.cs.s = append(c.cs.s, "NOOFFSETS")
	return (FtCreateNooffsets)(c)
}

func (c FtCreateScore) Nohl() FtCreateNohl {
	c.cs.s = append(c.cs.s, "NOHL")
	return (FtCreateNohl)(c)
}

func (c FtCreateScore) Nofields() FtCreateNofields {
	c.cs.s = append(c.cs.s, "NOFIELDS")
	return (FtCreateNofields)(c)
}

func (c FtCreateScore) Nofreqs() FtCreateNofreqs {
	c.cs.s = append(c.cs.s, "NOFREQS")
	return (FtCreateNofreqs)(c)
}

func (c FtCreateScore) Stopwords(count int64) FtCreateStopwordsStopwords {
	c.cs.s = append(c.cs.s, "STOPWORDS", strconv.FormatInt(count, 10))
	return (FtCreateStopwordsStopwords)(c)
}

func (c FtCreateScore) Skipinitialscan() FtCreateSkipinitialscan {
	c.cs.s = append(c.cs.s, "SKIPINITIALSCAN")
	return (FtCreateSkipinitialscan)(c)
}

func (c FtCreateScore) Schema() FtCreateSchema {
	c.cs.s = append(c.cs.s, "SCHEMA")
	return (FtCreateSchema)(c)
}

type FtCreateScoreField Completed

func (c FtCreateScoreField) PayloadField(payloadAttribute string) FtCreatePayloadField {
	c.cs.s = append(c.cs.s, "PAYLOAD_FIELD", payloadAttribute)
	return (FtCreatePayloadField)(c)
}

func (c FtCreateScoreField) Maxtextfields() FtCreateMaxtextfields {
	c.cs.s = append(c.cs.s, "MAXTEXTFIELDS")
	return (FtCreateMaxtextfields)(c)
}

func (c FtCreateScoreField) Temporary(seconds float64) FtCreateTemporary {
	c.cs.s = append(c.cs.s, "TEMPORARY", strconv.FormatFloat(seconds, 'f', -1, 64))
	return (FtCreateTemporary)(c)
}

func (c FtCreateScoreField) Nooffsets() FtCreateNooffsets {
	c.cs.s = append(c.cs.s, "NOOFFSETS")
	return (FtCreateNooffsets)(c)
}

func (c FtCreateScoreField) Nohl() FtCreateNohl {
	c.cs.s = append(c.cs.s, "NOHL")
	return (FtCreateNohl)(c)
}

func (c FtCreateScoreField) Nofields() FtCreateNofields {
	c.cs.s = append(c.cs.s, "NOFIELDS")
	return (FtCreateNofields)(c)
}

func (c FtCreateScoreField) Nofreqs() FtCreateNofreqs {
	c.cs.s = append(c.cs.s, "NOFREQS")
	return (FtCreateNofreqs)(c)
}

func (c FtCreateScoreField) Stopwords(count int64) FtCreateStopwordsStopwords {
	c.cs.s = append(c.cs.s, "STOPWORDS", strconv.FormatInt(count, 10))
	return (FtCreateStopwordsStopwords)(c)
}

func (c FtCreateScoreField) Skipinitialscan() FtCreateSkipinitialscan {
	c.cs.s = append(c.cs.s, "SKIPINITIALSCAN")
	return (FtCreateSkipinitialscan)(c)
}

func (c FtCreateScoreField) Schema() FtCreateSchema {
	c.cs.s = append(c.cs.s, "SCHEMA")
	return (FtCreateSchema)(c)
}

type FtCreateSkipinitialscan Completed

func (c FtCreateSkipinitialscan) Schema() FtCreateSchema {
	c.cs.s = append(c.cs.s, "SCHEMA")
	return (FtCreateSchema)(c)
}

type FtCreateStopwordsStopword Completed

func (c FtCreateStopwordsStopword) Stopword(stopword ...string) FtCreateStopwordsStopword {
	c.cs.s = append(c.cs.s, stopword...)
	return c
}

func (c FtCreateStopwordsStopword) Skipinitialscan() FtCreateSkipinitialscan {
	c.cs.s = append(c.cs.s, "SKIPINITIALSCAN")
	return (FtCreateSkipinitialscan)(c)
}

func (c FtCreateStopwordsStopword) Schema() FtCreateSchema {
	c.cs.s = append(c.cs.s, "SCHEMA")
	return (FtCreateSchema)(c)
}

type FtCreateStopwordsStopwords Completed

func (c FtCreateStopwordsStopwords) Stopword(stopword ...string) FtCreateStopwordsStopword {
	c.cs.s = append(c.cs.s, stopword...)
	return (FtCreateStopwordsStopword)(c)
}

func (c FtCreateStopwordsStopwords) Skipinitialscan() FtCreateSkipinitialscan {
	c.cs.s = append(c.cs.s, "SKIPINITIALSCAN")
	return (FtCreateSkipinitialscan)(c)
}

func (c FtCreateStopwordsStopwords) Schema() FtCreateSchema {
	c.cs.s = append(c.cs.s, "SCHEMA")
	return (FtCreateSchema)(c)
}

type FtCreateTemporary Completed

func (c FtCreateTemporary) Nooffsets() FtCreateNooffsets {
	c.cs.s = append(c.cs.s, "NOOFFSETS")
	return (FtCreateNooffsets)(c)
}

func (c FtCreateTemporary) Nohl() FtCreateNohl {
	c.cs.s = append(c.cs.s, "NOHL")
	return (FtCreateNohl)(c)
}

func (c FtCreateTemporary) Nofields() FtCreateNofields {
	c.cs.s = append(c.cs.s, "NOFIELDS")
	return (FtCreateNofields)(c)
}

func (c FtCreateTemporary) Nofreqs() FtCreateNofreqs {
	c.cs.s = append(c.cs.s, "NOFREQS")
	return (FtCreateNofreqs)(c)
}

func (c FtCreateTemporary) Stopwords(count int64) FtCreateStopwordsStopwords {
	c.cs.s = append(c.cs.s, "STOPWORDS", strconv.FormatInt(count, 10))
	return (FtCreateStopwordsStopwords)(c)
}

func (c FtCreateTemporary) Skipinitialscan() FtCreateSkipinitialscan {
	c.cs.s = append(c.cs.s, "SKIPINITIALSCAN")
	return (FtCreateSkipinitialscan)(c)
}

func (c FtCreateTemporary) Schema() FtCreateSchema {
	c.cs.s = append(c.cs.s, "SCHEMA")
	return (FtCreateSchema)(c)
}

type FtCursorDel Completed

func (b Builder) FtCursorDel() (c FtCursorDel) {
	c = FtCursorDel{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FT.CURSOR", "DEL")
	return c
}

func (c FtCursorDel) Index(index string) FtCursorDelIndex {
	c.cs.s = append(c.cs.s, index)
	return (FtCursorDelIndex)(c)
}

type FtCursorDelCursorId Completed

func (c FtCursorDelCursorId) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtCursorDelIndex Completed

func (c FtCursorDelIndex) CursorId(cursorId int64) FtCursorDelCursorId {
	c.cs.s = append(c.cs.s, strconv.FormatInt(cursorId, 10))
	return (FtCursorDelCursorId)(c)
}

type FtCursorRead Completed

func (b Builder) FtCursorRead() (c FtCursorRead) {
	c = FtCursorRead{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FT.CURSOR", "READ")
	return c
}

func (c FtCursorRead) Index(index string) FtCursorReadIndex {
	c.cs.s = append(c.cs.s, index)
	return (FtCursorReadIndex)(c)
}

type FtCursorReadCount Completed

func (c FtCursorReadCount) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtCursorReadCursorId Completed

func (c FtCursorReadCursorId) Count(readSize int64) FtCursorReadCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(readSize, 10))
	return (FtCursorReadCount)(c)
}

func (c FtCursorReadCursorId) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtCursorReadIndex Completed

func (c FtCursorReadIndex) CursorId(cursorId int64) FtCursorReadCursorId {
	c.cs.s = append(c.cs.s, strconv.FormatInt(cursorId, 10))
	return (FtCursorReadCursorId)(c)
}

type FtDictadd Completed

func (b Builder) FtDictadd() (c FtDictadd) {
	c = FtDictadd{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FT.DICTADD")
	return c
}

func (c FtDictadd) Dict(dict string) FtDictaddDict {
	c.cs.s = append(c.cs.s, dict)
	return (FtDictaddDict)(c)
}

type FtDictaddDict Completed

func (c FtDictaddDict) Term(term ...string) FtDictaddTerm {
	c.cs.s = append(c.cs.s, term...)
	return (FtDictaddTerm)(c)
}

type FtDictaddTerm Completed

func (c FtDictaddTerm) Term(term ...string) FtDictaddTerm {
	c.cs.s = append(c.cs.s, term...)
	return c
}

func (c FtDictaddTerm) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtDictdel Completed

func (b Builder) FtDictdel() (c FtDictdel) {
	c = FtDictdel{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FT.DICTDEL")
	return c
}

func (c FtDictdel) Dict(dict string) FtDictdelDict {
	c.cs.s = append(c.cs.s, dict)
	return (FtDictdelDict)(c)
}

type FtDictdelDict Completed

func (c FtDictdelDict) Term(term ...string) FtDictdelTerm {
	c.cs.s = append(c.cs.s, term...)
	return (FtDictdelTerm)(c)
}

type FtDictdelTerm Completed

func (c FtDictdelTerm) Term(term ...string) FtDictdelTerm {
	c.cs.s = append(c.cs.s, term...)
	return c
}

func (c FtDictdelTerm) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtDictdump Completed

func (b Builder) FtDictdump() (c FtDictdump) {
	c = FtDictdump{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FT.DICTDUMP")
	return c
}

func (c FtDictdump) Dict(dict string) FtDictdumpDict {
	c.cs.s = append(c.cs.s, dict)
	return (FtDictdumpDict)(c)
}

type FtDictdumpDict Completed

func (c FtDictdumpDict) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtDropindex Completed

func (b Builder) FtDropindex() (c FtDropindex) {
	c = FtDropindex{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FT.DROPINDEX")
	return c
}

func (c FtDropindex) Index(index string) FtDropindexIndex {
	c.cs.s = append(c.cs.s, index)
	return (FtDropindexIndex)(c)
}

type FtDropindexDeleteDocsDd Completed

func (c FtDropindexDeleteDocsDd) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtDropindexIndex Completed

func (c FtDropindexIndex) Dd() FtDropindexDeleteDocsDd {
	c.cs.s = append(c.cs.s, "DD")
	return (FtDropindexDeleteDocsDd)(c)
}

func (c FtDropindexIndex) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtExplain Completed

func (b Builder) FtExplain() (c FtExplain) {
	c = FtExplain{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FT.EXPLAIN")
	return c
}

func (c FtExplain) Index(index string) FtExplainIndex {
	c.cs.s = append(c.cs.s, index)
	return (FtExplainIndex)(c)
}

type FtExplainDialect Completed

func (c FtExplainDialect) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtExplainIndex Completed

func (c FtExplainIndex) Query(query string) FtExplainQuery {
	c.cs.s = append(c.cs.s, query)
	return (FtExplainQuery)(c)
}

type FtExplainQuery Completed

func (c FtExplainQuery) Dialect(dialect int64) FtExplainDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtExplainDialect)(c)
}

func (c FtExplainQuery) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtExplaincli Completed

func (b Builder) FtExplaincli() (c FtExplaincli) {
	c = FtExplaincli{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FT.EXPLAINCLI")
	return c
}

func (c FtExplaincli) Index(index string) FtExplaincliIndex {
	c.cs.s = append(c.cs.s, index)
	return (FtExplaincliIndex)(c)
}

type FtExplaincliDialect Completed

func (c FtExplaincliDialect) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtExplaincliIndex Completed

func (c FtExplaincliIndex) Query(query string) FtExplaincliQuery {
	c.cs.s = append(c.cs.s, query)
	return (FtExplaincliQuery)(c)
}

type FtExplaincliQuery Completed

func (c FtExplaincliQuery) Dialect(dialect int64) FtExplaincliDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtExplaincliDialect)(c)
}

func (c FtExplaincliQuery) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtInfo Completed

func (b Builder) FtInfo() (c FtInfo) {
	c = FtInfo{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FT.INFO")
	return c
}

func (c FtInfo) Index(index string) FtInfoIndex {
	c.cs.s = append(c.cs.s, index)
	return (FtInfoIndex)(c)
}

type FtInfoIndex Completed

func (c FtInfoIndex) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtList Completed

func (b Builder) FtList() (c FtList) {
	c = FtList{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FT._LIST")
	return c
}

func (c FtList) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtProfile Completed

func (b Builder) FtProfile() (c FtProfile) {
	c = FtProfile{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FT.PROFILE")
	return c
}

func (c FtProfile) Index(index string) FtProfileIndex {
	c.cs.s = append(c.cs.s, index)
	return (FtProfileIndex)(c)
}

type FtProfileIndex Completed

func (c FtProfileIndex) Search() FtProfileQuerytypeSearch {
	c.cs.s = append(c.cs.s, "SEARCH")
	return (FtProfileQuerytypeSearch)(c)
}

func (c FtProfileIndex) Aggregate() FtProfileQuerytypeAggregate {
	c.cs.s = append(c.cs.s, "AGGREGATE")
	return (FtProfileQuerytypeAggregate)(c)
}

type FtProfileLimited Completed

func (c FtProfileLimited) Query(query string) FtProfileQuery {
	c.cs.s = append(c.cs.s, "QUERY", query)
	return (FtProfileQuery)(c)
}

type FtProfileQuery Completed

func (c FtProfileQuery) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtProfileQuerytypeAggregate Completed

func (c FtProfileQuerytypeAggregate) Limited() FtProfileLimited {
	c.cs.s = append(c.cs.s, "LIMITED")
	return (FtProfileLimited)(c)
}

func (c FtProfileQuerytypeAggregate) Query(query string) FtProfileQuery {
	c.cs.s = append(c.cs.s, "QUERY", query)
	return (FtProfileQuery)(c)
}

type FtProfileQuerytypeSearch Completed

func (c FtProfileQuerytypeSearch) Limited() FtProfileLimited {
	c.cs.s = append(c.cs.s, "LIMITED")
	return (FtProfileLimited)(c)
}

func (c FtProfileQuerytypeSearch) Query(query string) FtProfileQuery {
	c.cs.s = append(c.cs.s, "QUERY", query)
	return (FtProfileQuery)(c)
}

type FtSearch Completed

func (b Builder) FtSearch() (c FtSearch) {
	c = FtSearch{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FT.SEARCH")
	return c
}

func (c FtSearch) Index(index string) FtSearchIndex {
	c.cs.s = append(c.cs.s, index)
	return (FtSearchIndex)(c)
}

type FtSearchDialect Completed

func (c FtSearchDialect) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchExpander Completed

func (c FtSearchExpander) Scorer(scorer string) FtSearchScorer {
	c.cs.s = append(c.cs.s, "SCORER", scorer)
	return (FtSearchScorer)(c)
}

func (c FtSearchExpander) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchExpander) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchExpander) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchExpander) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchExpander) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchExpander) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchExpander) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchExplainscore Completed

func (c FtSearchExplainscore) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchExplainscore) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchExplainscore) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchExplainscore) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchExplainscore) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchExplainscore) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchFilterFilter Completed

func (c FtSearchFilterFilter) Min(min float64) FtSearchFilterMin {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(min, 'f', -1, 64))
	return (FtSearchFilterMin)(c)
}

type FtSearchFilterMax Completed

func (c FtSearchFilterMax) Filter(numericField string) FtSearchFilterFilter {
	c.cs.s = append(c.cs.s, "FILTER", numericField)
	return (FtSearchFilterFilter)(c)
}

func (c FtSearchFilterMax) Geofilter(geoField string) FtSearchGeoFilterGeofilter {
	c.cs.s = append(c.cs.s, "GEOFILTER", geoField)
	return (FtSearchGeoFilterGeofilter)(c)
}

func (c FtSearchFilterMax) Inkeys(count string) FtSearchInKeysInkeys {
	c.cs.s = append(c.cs.s, "INKEYS", count)
	return (FtSearchInKeysInkeys)(c)
}

func (c FtSearchFilterMax) Infields(count string) FtSearchInFieldsInfields {
	c.cs.s = append(c.cs.s, "INFIELDS", count)
	return (FtSearchInFieldsInfields)(c)
}

func (c FtSearchFilterMax) Return(count string) FtSearchReturnReturn {
	c.cs.s = append(c.cs.s, "RETURN", count)
	return (FtSearchReturnReturn)(c)
}

func (c FtSearchFilterMax) Summarize() FtSearchSummarizeSummarize {
	c.cs.s = append(c.cs.s, "SUMMARIZE")
	return (FtSearchSummarizeSummarize)(c)
}

func (c FtSearchFilterMax) Highlight() FtSearchHighlightHighlight {
	c.cs.s = append(c.cs.s, "HIGHLIGHT")
	return (FtSearchHighlightHighlight)(c)
}

func (c FtSearchFilterMax) Slop(slop int64) FtSearchSlop {
	c.cs.s = append(c.cs.s, "SLOP", strconv.FormatInt(slop, 10))
	return (FtSearchSlop)(c)
}

func (c FtSearchFilterMax) Timeout(timeout int64) FtSearchTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (FtSearchTimeout)(c)
}

func (c FtSearchFilterMax) Inorder() FtSearchTagsInorder {
	c.cs.s = append(c.cs.s, "INORDER")
	return (FtSearchTagsInorder)(c)
}

func (c FtSearchFilterMax) Language(language string) FtSearchLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", language)
	return (FtSearchLanguage)(c)
}

func (c FtSearchFilterMax) Expander(expander string) FtSearchExpander {
	c.cs.s = append(c.cs.s, "EXPANDER", expander)
	return (FtSearchExpander)(c)
}

func (c FtSearchFilterMax) Scorer(scorer string) FtSearchScorer {
	c.cs.s = append(c.cs.s, "SCORER", scorer)
	return (FtSearchScorer)(c)
}

func (c FtSearchFilterMax) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchFilterMax) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchFilterMax) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchFilterMax) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchFilterMax) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchFilterMax) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchFilterMax) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchFilterMin Completed

func (c FtSearchFilterMin) Max(max float64) FtSearchFilterMax {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(max, 'f', -1, 64))
	return (FtSearchFilterMax)(c)
}

type FtSearchGeoFilterGeofilter Completed

func (c FtSearchGeoFilterGeofilter) Lon(lon float64) FtSearchGeoFilterLon {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(lon, 'f', -1, 64))
	return (FtSearchGeoFilterLon)(c)
}

type FtSearchGeoFilterLat Completed

func (c FtSearchGeoFilterLat) Radius(radius float64) FtSearchGeoFilterRadius {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(radius, 'f', -1, 64))
	return (FtSearchGeoFilterRadius)(c)
}

type FtSearchGeoFilterLon Completed

func (c FtSearchGeoFilterLon) Lat(lat float64) FtSearchGeoFilterLat {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(lat, 'f', -1, 64))
	return (FtSearchGeoFilterLat)(c)
}

type FtSearchGeoFilterRadius Completed

func (c FtSearchGeoFilterRadius) M() FtSearchGeoFilterRadiusTypeM {
	c.cs.s = append(c.cs.s, "m")
	return (FtSearchGeoFilterRadiusTypeM)(c)
}

func (c FtSearchGeoFilterRadius) Km() FtSearchGeoFilterRadiusTypeKm {
	c.cs.s = append(c.cs.s, "km")
	return (FtSearchGeoFilterRadiusTypeKm)(c)
}

func (c FtSearchGeoFilterRadius) Mi() FtSearchGeoFilterRadiusTypeMi {
	c.cs.s = append(c.cs.s, "mi")
	return (FtSearchGeoFilterRadiusTypeMi)(c)
}

func (c FtSearchGeoFilterRadius) Ft() FtSearchGeoFilterRadiusTypeFt {
	c.cs.s = append(c.cs.s, "ft")
	return (FtSearchGeoFilterRadiusTypeFt)(c)
}

type FtSearchGeoFilterRadiusTypeFt Completed

func (c FtSearchGeoFilterRadiusTypeFt) Geofilter(geoField string) FtSearchGeoFilterGeofilter {
	c.cs.s = append(c.cs.s, "GEOFILTER", geoField)
	return (FtSearchGeoFilterGeofilter)(c)
}

func (c FtSearchGeoFilterRadiusTypeFt) Inkeys(count string) FtSearchInKeysInkeys {
	c.cs.s = append(c.cs.s, "INKEYS", count)
	return (FtSearchInKeysInkeys)(c)
}

func (c FtSearchGeoFilterRadiusTypeFt) Infields(count string) FtSearchInFieldsInfields {
	c.cs.s = append(c.cs.s, "INFIELDS", count)
	return (FtSearchInFieldsInfields)(c)
}

func (c FtSearchGeoFilterRadiusTypeFt) Return(count string) FtSearchReturnReturn {
	c.cs.s = append(c.cs.s, "RETURN", count)
	return (FtSearchReturnReturn)(c)
}

func (c FtSearchGeoFilterRadiusTypeFt) Summarize() FtSearchSummarizeSummarize {
	c.cs.s = append(c.cs.s, "SUMMARIZE")
	return (FtSearchSummarizeSummarize)(c)
}

func (c FtSearchGeoFilterRadiusTypeFt) Highlight() FtSearchHighlightHighlight {
	c.cs.s = append(c.cs.s, "HIGHLIGHT")
	return (FtSearchHighlightHighlight)(c)
}

func (c FtSearchGeoFilterRadiusTypeFt) Slop(slop int64) FtSearchSlop {
	c.cs.s = append(c.cs.s, "SLOP", strconv.FormatInt(slop, 10))
	return (FtSearchSlop)(c)
}

func (c FtSearchGeoFilterRadiusTypeFt) Timeout(timeout int64) FtSearchTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (FtSearchTimeout)(c)
}

func (c FtSearchGeoFilterRadiusTypeFt) Inorder() FtSearchTagsInorder {
	c.cs.s = append(c.cs.s, "INORDER")
	return (FtSearchTagsInorder)(c)
}

func (c FtSearchGeoFilterRadiusTypeFt) Language(language string) FtSearchLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", language)
	return (FtSearchLanguage)(c)
}

func (c FtSearchGeoFilterRadiusTypeFt) Expander(expander string) FtSearchExpander {
	c.cs.s = append(c.cs.s, "EXPANDER", expander)
	return (FtSearchExpander)(c)
}

func (c FtSearchGeoFilterRadiusTypeFt) Scorer(scorer string) FtSearchScorer {
	c.cs.s = append(c.cs.s, "SCORER", scorer)
	return (FtSearchScorer)(c)
}

func (c FtSearchGeoFilterRadiusTypeFt) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchGeoFilterRadiusTypeFt) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchGeoFilterRadiusTypeFt) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchGeoFilterRadiusTypeFt) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchGeoFilterRadiusTypeFt) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchGeoFilterRadiusTypeFt) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchGeoFilterRadiusTypeFt) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchGeoFilterRadiusTypeKm Completed

func (c FtSearchGeoFilterRadiusTypeKm) Geofilter(geoField string) FtSearchGeoFilterGeofilter {
	c.cs.s = append(c.cs.s, "GEOFILTER", geoField)
	return (FtSearchGeoFilterGeofilter)(c)
}

func (c FtSearchGeoFilterRadiusTypeKm) Inkeys(count string) FtSearchInKeysInkeys {
	c.cs.s = append(c.cs.s, "INKEYS", count)
	return (FtSearchInKeysInkeys)(c)
}

func (c FtSearchGeoFilterRadiusTypeKm) Infields(count string) FtSearchInFieldsInfields {
	c.cs.s = append(c.cs.s, "INFIELDS", count)
	return (FtSearchInFieldsInfields)(c)
}

func (c FtSearchGeoFilterRadiusTypeKm) Return(count string) FtSearchReturnReturn {
	c.cs.s = append(c.cs.s, "RETURN", count)
	return (FtSearchReturnReturn)(c)
}

func (c FtSearchGeoFilterRadiusTypeKm) Summarize() FtSearchSummarizeSummarize {
	c.cs.s = append(c.cs.s, "SUMMARIZE")
	return (FtSearchSummarizeSummarize)(c)
}

func (c FtSearchGeoFilterRadiusTypeKm) Highlight() FtSearchHighlightHighlight {
	c.cs.s = append(c.cs.s, "HIGHLIGHT")
	return (FtSearchHighlightHighlight)(c)
}

func (c FtSearchGeoFilterRadiusTypeKm) Slop(slop int64) FtSearchSlop {
	c.cs.s = append(c.cs.s, "SLOP", strconv.FormatInt(slop, 10))
	return (FtSearchSlop)(c)
}

func (c FtSearchGeoFilterRadiusTypeKm) Timeout(timeout int64) FtSearchTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (FtSearchTimeout)(c)
}

func (c FtSearchGeoFilterRadiusTypeKm) Inorder() FtSearchTagsInorder {
	c.cs.s = append(c.cs.s, "INORDER")
	return (FtSearchTagsInorder)(c)
}

func (c FtSearchGeoFilterRadiusTypeKm) Language(language string) FtSearchLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", language)
	return (FtSearchLanguage)(c)
}

func (c FtSearchGeoFilterRadiusTypeKm) Expander(expander string) FtSearchExpander {
	c.cs.s = append(c.cs.s, "EXPANDER", expander)
	return (FtSearchExpander)(c)
}

func (c FtSearchGeoFilterRadiusTypeKm) Scorer(scorer string) FtSearchScorer {
	c.cs.s = append(c.cs.s, "SCORER", scorer)
	return (FtSearchScorer)(c)
}

func (c FtSearchGeoFilterRadiusTypeKm) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchGeoFilterRadiusTypeKm) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchGeoFilterRadiusTypeKm) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchGeoFilterRadiusTypeKm) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchGeoFilterRadiusTypeKm) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchGeoFilterRadiusTypeKm) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchGeoFilterRadiusTypeKm) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchGeoFilterRadiusTypeM Completed

func (c FtSearchGeoFilterRadiusTypeM) Geofilter(geoField string) FtSearchGeoFilterGeofilter {
	c.cs.s = append(c.cs.s, "GEOFILTER", geoField)
	return (FtSearchGeoFilterGeofilter)(c)
}

func (c FtSearchGeoFilterRadiusTypeM) Inkeys(count string) FtSearchInKeysInkeys {
	c.cs.s = append(c.cs.s, "INKEYS", count)
	return (FtSearchInKeysInkeys)(c)
}

func (c FtSearchGeoFilterRadiusTypeM) Infields(count string) FtSearchInFieldsInfields {
	c.cs.s = append(c.cs.s, "INFIELDS", count)
	return (FtSearchInFieldsInfields)(c)
}

func (c FtSearchGeoFilterRadiusTypeM) Return(count string) FtSearchReturnReturn {
	c.cs.s = append(c.cs.s, "RETURN", count)
	return (FtSearchReturnReturn)(c)
}

func (c FtSearchGeoFilterRadiusTypeM) Summarize() FtSearchSummarizeSummarize {
	c.cs.s = append(c.cs.s, "SUMMARIZE")
	return (FtSearchSummarizeSummarize)(c)
}

func (c FtSearchGeoFilterRadiusTypeM) Highlight() FtSearchHighlightHighlight {
	c.cs.s = append(c.cs.s, "HIGHLIGHT")
	return (FtSearchHighlightHighlight)(c)
}

func (c FtSearchGeoFilterRadiusTypeM) Slop(slop int64) FtSearchSlop {
	c.cs.s = append(c.cs.s, "SLOP", strconv.FormatInt(slop, 10))
	return (FtSearchSlop)(c)
}

func (c FtSearchGeoFilterRadiusTypeM) Timeout(timeout int64) FtSearchTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (FtSearchTimeout)(c)
}

func (c FtSearchGeoFilterRadiusTypeM) Inorder() FtSearchTagsInorder {
	c.cs.s = append(c.cs.s, "INORDER")
	return (FtSearchTagsInorder)(c)
}

func (c FtSearchGeoFilterRadiusTypeM) Language(language string) FtSearchLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", language)
	return (FtSearchLanguage)(c)
}

func (c FtSearchGeoFilterRadiusTypeM) Expander(expander string) FtSearchExpander {
	c.cs.s = append(c.cs.s, "EXPANDER", expander)
	return (FtSearchExpander)(c)
}

func (c FtSearchGeoFilterRadiusTypeM) Scorer(scorer string) FtSearchScorer {
	c.cs.s = append(c.cs.s, "SCORER", scorer)
	return (FtSearchScorer)(c)
}

func (c FtSearchGeoFilterRadiusTypeM) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchGeoFilterRadiusTypeM) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchGeoFilterRadiusTypeM) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchGeoFilterRadiusTypeM) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchGeoFilterRadiusTypeM) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchGeoFilterRadiusTypeM) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchGeoFilterRadiusTypeM) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchGeoFilterRadiusTypeMi Completed

func (c FtSearchGeoFilterRadiusTypeMi) Geofilter(geoField string) FtSearchGeoFilterGeofilter {
	c.cs.s = append(c.cs.s, "GEOFILTER", geoField)
	return (FtSearchGeoFilterGeofilter)(c)
}

func (c FtSearchGeoFilterRadiusTypeMi) Inkeys(count string) FtSearchInKeysInkeys {
	c.cs.s = append(c.cs.s, "INKEYS", count)
	return (FtSearchInKeysInkeys)(c)
}

func (c FtSearchGeoFilterRadiusTypeMi) Infields(count string) FtSearchInFieldsInfields {
	c.cs.s = append(c.cs.s, "INFIELDS", count)
	return (FtSearchInFieldsInfields)(c)
}

func (c FtSearchGeoFilterRadiusTypeMi) Return(count string) FtSearchReturnReturn {
	c.cs.s = append(c.cs.s, "RETURN", count)
	return (FtSearchReturnReturn)(c)
}

func (c FtSearchGeoFilterRadiusTypeMi) Summarize() FtSearchSummarizeSummarize {
	c.cs.s = append(c.cs.s, "SUMMARIZE")
	return (FtSearchSummarizeSummarize)(c)
}

func (c FtSearchGeoFilterRadiusTypeMi) Highlight() FtSearchHighlightHighlight {
	c.cs.s = append(c.cs.s, "HIGHLIGHT")
	return (FtSearchHighlightHighlight)(c)
}

func (c FtSearchGeoFilterRadiusTypeMi) Slop(slop int64) FtSearchSlop {
	c.cs.s = append(c.cs.s, "SLOP", strconv.FormatInt(slop, 10))
	return (FtSearchSlop)(c)
}

func (c FtSearchGeoFilterRadiusTypeMi) Timeout(timeout int64) FtSearchTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (FtSearchTimeout)(c)
}

func (c FtSearchGeoFilterRadiusTypeMi) Inorder() FtSearchTagsInorder {
	c.cs.s = append(c.cs.s, "INORDER")
	return (FtSearchTagsInorder)(c)
}

func (c FtSearchGeoFilterRadiusTypeMi) Language(language string) FtSearchLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", language)
	return (FtSearchLanguage)(c)
}

func (c FtSearchGeoFilterRadiusTypeMi) Expander(expander string) FtSearchExpander {
	c.cs.s = append(c.cs.s, "EXPANDER", expander)
	return (FtSearchExpander)(c)
}

func (c FtSearchGeoFilterRadiusTypeMi) Scorer(scorer string) FtSearchScorer {
	c.cs.s = append(c.cs.s, "SCORER", scorer)
	return (FtSearchScorer)(c)
}

func (c FtSearchGeoFilterRadiusTypeMi) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchGeoFilterRadiusTypeMi) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchGeoFilterRadiusTypeMi) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchGeoFilterRadiusTypeMi) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchGeoFilterRadiusTypeMi) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchGeoFilterRadiusTypeMi) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchGeoFilterRadiusTypeMi) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchHighlightFieldsField Completed

func (c FtSearchHighlightFieldsField) Field(field ...string) FtSearchHighlightFieldsField {
	c.cs.s = append(c.cs.s, field...)
	return c
}

func (c FtSearchHighlightFieldsField) Tags() FtSearchHighlightTagsTags {
	c.cs.s = append(c.cs.s, "TAGS")
	return (FtSearchHighlightTagsTags)(c)
}

func (c FtSearchHighlightFieldsField) Slop(slop int64) FtSearchSlop {
	c.cs.s = append(c.cs.s, "SLOP", strconv.FormatInt(slop, 10))
	return (FtSearchSlop)(c)
}

func (c FtSearchHighlightFieldsField) Timeout(timeout int64) FtSearchTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (FtSearchTimeout)(c)
}

func (c FtSearchHighlightFieldsField) Inorder() FtSearchTagsInorder {
	c.cs.s = append(c.cs.s, "INORDER")
	return (FtSearchTagsInorder)(c)
}

func (c FtSearchHighlightFieldsField) Language(language string) FtSearchLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", language)
	return (FtSearchLanguage)(c)
}

func (c FtSearchHighlightFieldsField) Expander(expander string) FtSearchExpander {
	c.cs.s = append(c.cs.s, "EXPANDER", expander)
	return (FtSearchExpander)(c)
}

func (c FtSearchHighlightFieldsField) Scorer(scorer string) FtSearchScorer {
	c.cs.s = append(c.cs.s, "SCORER", scorer)
	return (FtSearchScorer)(c)
}

func (c FtSearchHighlightFieldsField) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchHighlightFieldsField) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchHighlightFieldsField) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchHighlightFieldsField) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchHighlightFieldsField) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchHighlightFieldsField) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchHighlightFieldsField) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchHighlightFieldsFields Completed

func (c FtSearchHighlightFieldsFields) Field(field ...string) FtSearchHighlightFieldsField {
	c.cs.s = append(c.cs.s, field...)
	return (FtSearchHighlightFieldsField)(c)
}

type FtSearchHighlightHighlight Completed

func (c FtSearchHighlightHighlight) Fields(count string) FtSearchHighlightFieldsFields {
	c.cs.s = append(c.cs.s, "FIELDS", count)
	return (FtSearchHighlightFieldsFields)(c)
}

func (c FtSearchHighlightHighlight) Tags() FtSearchHighlightTagsTags {
	c.cs.s = append(c.cs.s, "TAGS")
	return (FtSearchHighlightTagsTags)(c)
}

func (c FtSearchHighlightHighlight) Slop(slop int64) FtSearchSlop {
	c.cs.s = append(c.cs.s, "SLOP", strconv.FormatInt(slop, 10))
	return (FtSearchSlop)(c)
}

func (c FtSearchHighlightHighlight) Timeout(timeout int64) FtSearchTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (FtSearchTimeout)(c)
}

func (c FtSearchHighlightHighlight) Inorder() FtSearchTagsInorder {
	c.cs.s = append(c.cs.s, "INORDER")
	return (FtSearchTagsInorder)(c)
}

func (c FtSearchHighlightHighlight) Language(language string) FtSearchLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", language)
	return (FtSearchLanguage)(c)
}

func (c FtSearchHighlightHighlight) Expander(expander string) FtSearchExpander {
	c.cs.s = append(c.cs.s, "EXPANDER", expander)
	return (FtSearchExpander)(c)
}

func (c FtSearchHighlightHighlight) Scorer(scorer string) FtSearchScorer {
	c.cs.s = append(c.cs.s, "SCORER", scorer)
	return (FtSearchScorer)(c)
}

func (c FtSearchHighlightHighlight) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchHighlightHighlight) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchHighlightHighlight) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchHighlightHighlight) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchHighlightHighlight) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchHighlightHighlight) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchHighlightHighlight) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchHighlightTagsOpenClose Completed

func (c FtSearchHighlightTagsOpenClose) Slop(slop int64) FtSearchSlop {
	c.cs.s = append(c.cs.s, "SLOP", strconv.FormatInt(slop, 10))
	return (FtSearchSlop)(c)
}

func (c FtSearchHighlightTagsOpenClose) Timeout(timeout int64) FtSearchTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (FtSearchTimeout)(c)
}

func (c FtSearchHighlightTagsOpenClose) Inorder() FtSearchTagsInorder {
	c.cs.s = append(c.cs.s, "INORDER")
	return (FtSearchTagsInorder)(c)
}

func (c FtSearchHighlightTagsOpenClose) Language(language string) FtSearchLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", language)
	return (FtSearchLanguage)(c)
}

func (c FtSearchHighlightTagsOpenClose) Expander(expander string) FtSearchExpander {
	c.cs.s = append(c.cs.s, "EXPANDER", expander)
	return (FtSearchExpander)(c)
}

func (c FtSearchHighlightTagsOpenClose) Scorer(scorer string) FtSearchScorer {
	c.cs.s = append(c.cs.s, "SCORER", scorer)
	return (FtSearchScorer)(c)
}

func (c FtSearchHighlightTagsOpenClose) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchHighlightTagsOpenClose) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchHighlightTagsOpenClose) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchHighlightTagsOpenClose) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchHighlightTagsOpenClose) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchHighlightTagsOpenClose) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchHighlightTagsOpenClose) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchHighlightTagsTags Completed

func (c FtSearchHighlightTagsTags) OpenClose(open string, close string) FtSearchHighlightTagsOpenClose {
	c.cs.s = append(c.cs.s, open, close)
	return (FtSearchHighlightTagsOpenClose)(c)
}

type FtSearchInFieldsField Completed

func (c FtSearchInFieldsField) Field(field ...string) FtSearchInFieldsField {
	c.cs.s = append(c.cs.s, field...)
	return c
}

func (c FtSearchInFieldsField) Return(count string) FtSearchReturnReturn {
	c.cs.s = append(c.cs.s, "RETURN", count)
	return (FtSearchReturnReturn)(c)
}

func (c FtSearchInFieldsField) Summarize() FtSearchSummarizeSummarize {
	c.cs.s = append(c.cs.s, "SUMMARIZE")
	return (FtSearchSummarizeSummarize)(c)
}

func (c FtSearchInFieldsField) Highlight() FtSearchHighlightHighlight {
	c.cs.s = append(c.cs.s, "HIGHLIGHT")
	return (FtSearchHighlightHighlight)(c)
}

func (c FtSearchInFieldsField) Slop(slop int64) FtSearchSlop {
	c.cs.s = append(c.cs.s, "SLOP", strconv.FormatInt(slop, 10))
	return (FtSearchSlop)(c)
}

func (c FtSearchInFieldsField) Timeout(timeout int64) FtSearchTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (FtSearchTimeout)(c)
}

func (c FtSearchInFieldsField) Inorder() FtSearchTagsInorder {
	c.cs.s = append(c.cs.s, "INORDER")
	return (FtSearchTagsInorder)(c)
}

func (c FtSearchInFieldsField) Language(language string) FtSearchLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", language)
	return (FtSearchLanguage)(c)
}

func (c FtSearchInFieldsField) Expander(expander string) FtSearchExpander {
	c.cs.s = append(c.cs.s, "EXPANDER", expander)
	return (FtSearchExpander)(c)
}

func (c FtSearchInFieldsField) Scorer(scorer string) FtSearchScorer {
	c.cs.s = append(c.cs.s, "SCORER", scorer)
	return (FtSearchScorer)(c)
}

func (c FtSearchInFieldsField) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchInFieldsField) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchInFieldsField) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchInFieldsField) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchInFieldsField) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchInFieldsField) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchInFieldsField) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchInFieldsInfields Completed

func (c FtSearchInFieldsInfields) Field(field ...string) FtSearchInFieldsField {
	c.cs.s = append(c.cs.s, field...)
	return (FtSearchInFieldsField)(c)
}

type FtSearchInKeysInkeys Completed

func (c FtSearchInKeysInkeys) Key(key ...string) FtSearchInKeysKey {
	c.cs.s = append(c.cs.s, key...)
	return (FtSearchInKeysKey)(c)
}

type FtSearchInKeysKey Completed

func (c FtSearchInKeysKey) Key(key ...string) FtSearchInKeysKey {
	c.cs.s = append(c.cs.s, key...)
	return c
}

func (c FtSearchInKeysKey) Infields(count string) FtSearchInFieldsInfields {
	c.cs.s = append(c.cs.s, "INFIELDS", count)
	return (FtSearchInFieldsInfields)(c)
}

func (c FtSearchInKeysKey) Return(count string) FtSearchReturnReturn {
	c.cs.s = append(c.cs.s, "RETURN", count)
	return (FtSearchReturnReturn)(c)
}

func (c FtSearchInKeysKey) Summarize() FtSearchSummarizeSummarize {
	c.cs.s = append(c.cs.s, "SUMMARIZE")
	return (FtSearchSummarizeSummarize)(c)
}

func (c FtSearchInKeysKey) Highlight() FtSearchHighlightHighlight {
	c.cs.s = append(c.cs.s, "HIGHLIGHT")
	return (FtSearchHighlightHighlight)(c)
}

func (c FtSearchInKeysKey) Slop(slop int64) FtSearchSlop {
	c.cs.s = append(c.cs.s, "SLOP", strconv.FormatInt(slop, 10))
	return (FtSearchSlop)(c)
}

func (c FtSearchInKeysKey) Timeout(timeout int64) FtSearchTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (FtSearchTimeout)(c)
}

func (c FtSearchInKeysKey) Inorder() FtSearchTagsInorder {
	c.cs.s = append(c.cs.s, "INORDER")
	return (FtSearchTagsInorder)(c)
}

func (c FtSearchInKeysKey) Language(language string) FtSearchLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", language)
	return (FtSearchLanguage)(c)
}

func (c FtSearchInKeysKey) Expander(expander string) FtSearchExpander {
	c.cs.s = append(c.cs.s, "EXPANDER", expander)
	return (FtSearchExpander)(c)
}

func (c FtSearchInKeysKey) Scorer(scorer string) FtSearchScorer {
	c.cs.s = append(c.cs.s, "SCORER", scorer)
	return (FtSearchScorer)(c)
}

func (c FtSearchInKeysKey) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchInKeysKey) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchInKeysKey) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchInKeysKey) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchInKeysKey) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchInKeysKey) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchInKeysKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchIndex Completed

func (c FtSearchIndex) Query(query string) FtSearchQuery {
	c.cs.s = append(c.cs.s, query)
	return (FtSearchQuery)(c)
}

type FtSearchLanguage Completed

func (c FtSearchLanguage) Expander(expander string) FtSearchExpander {
	c.cs.s = append(c.cs.s, "EXPANDER", expander)
	return (FtSearchExpander)(c)
}

func (c FtSearchLanguage) Scorer(scorer string) FtSearchScorer {
	c.cs.s = append(c.cs.s, "SCORER", scorer)
	return (FtSearchScorer)(c)
}

func (c FtSearchLanguage) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchLanguage) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchLanguage) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchLanguage) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchLanguage) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchLanguage) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchLanguage) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchLimitLimit Completed

func (c FtSearchLimitLimit) OffsetNum(offset int64, num int64) FtSearchLimitOffsetNum {
	c.cs.s = append(c.cs.s, strconv.FormatInt(offset, 10), strconv.FormatInt(num, 10))
	return (FtSearchLimitOffsetNum)(c)
}

type FtSearchLimitOffsetNum Completed

func (c FtSearchLimitOffsetNum) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchLimitOffsetNum) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchLimitOffsetNum) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchNocontent Completed

func (c FtSearchNocontent) Verbatim() FtSearchVerbatim {
	c.cs.s = append(c.cs.s, "VERBATIM")
	return (FtSearchVerbatim)(c)
}

func (c FtSearchNocontent) Nostopwords() FtSearchNostopwords {
	c.cs.s = append(c.cs.s, "NOSTOPWORDS")
	return (FtSearchNostopwords)(c)
}

func (c FtSearchNocontent) Withscores() FtSearchWithscores {
	c.cs.s = append(c.cs.s, "WITHSCORES")
	return (FtSearchWithscores)(c)
}

func (c FtSearchNocontent) Withpayloads() FtSearchWithpayloads {
	c.cs.s = append(c.cs.s, "WITHPAYLOADS")
	return (FtSearchWithpayloads)(c)
}

func (c FtSearchNocontent) Withsortkeys() FtSearchWithsortkeys {
	c.cs.s = append(c.cs.s, "WITHSORTKEYS")
	return (FtSearchWithsortkeys)(c)
}

func (c FtSearchNocontent) Filter(numericField string) FtSearchFilterFilter {
	c.cs.s = append(c.cs.s, "FILTER", numericField)
	return (FtSearchFilterFilter)(c)
}

func (c FtSearchNocontent) Geofilter(geoField string) FtSearchGeoFilterGeofilter {
	c.cs.s = append(c.cs.s, "GEOFILTER", geoField)
	return (FtSearchGeoFilterGeofilter)(c)
}

func (c FtSearchNocontent) Inkeys(count string) FtSearchInKeysInkeys {
	c.cs.s = append(c.cs.s, "INKEYS", count)
	return (FtSearchInKeysInkeys)(c)
}

func (c FtSearchNocontent) Infields(count string) FtSearchInFieldsInfields {
	c.cs.s = append(c.cs.s, "INFIELDS", count)
	return (FtSearchInFieldsInfields)(c)
}

func (c FtSearchNocontent) Return(count string) FtSearchReturnReturn {
	c.cs.s = append(c.cs.s, "RETURN", count)
	return (FtSearchReturnReturn)(c)
}

func (c FtSearchNocontent) Summarize() FtSearchSummarizeSummarize {
	c.cs.s = append(c.cs.s, "SUMMARIZE")
	return (FtSearchSummarizeSummarize)(c)
}

func (c FtSearchNocontent) Highlight() FtSearchHighlightHighlight {
	c.cs.s = append(c.cs.s, "HIGHLIGHT")
	return (FtSearchHighlightHighlight)(c)
}

func (c FtSearchNocontent) Slop(slop int64) FtSearchSlop {
	c.cs.s = append(c.cs.s, "SLOP", strconv.FormatInt(slop, 10))
	return (FtSearchSlop)(c)
}

func (c FtSearchNocontent) Timeout(timeout int64) FtSearchTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (FtSearchTimeout)(c)
}

func (c FtSearchNocontent) Inorder() FtSearchTagsInorder {
	c.cs.s = append(c.cs.s, "INORDER")
	return (FtSearchTagsInorder)(c)
}

func (c FtSearchNocontent) Language(language string) FtSearchLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", language)
	return (FtSearchLanguage)(c)
}

func (c FtSearchNocontent) Expander(expander string) FtSearchExpander {
	c.cs.s = append(c.cs.s, "EXPANDER", expander)
	return (FtSearchExpander)(c)
}

func (c FtSearchNocontent) Scorer(scorer string) FtSearchScorer {
	c.cs.s = append(c.cs.s, "SCORER", scorer)
	return (FtSearchScorer)(c)
}

func (c FtSearchNocontent) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchNocontent) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchNocontent) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchNocontent) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchNocontent) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchNocontent) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchNocontent) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchNostopwords Completed

func (c FtSearchNostopwords) Withscores() FtSearchWithscores {
	c.cs.s = append(c.cs.s, "WITHSCORES")
	return (FtSearchWithscores)(c)
}

func (c FtSearchNostopwords) Withpayloads() FtSearchWithpayloads {
	c.cs.s = append(c.cs.s, "WITHPAYLOADS")
	return (FtSearchWithpayloads)(c)
}

func (c FtSearchNostopwords) Withsortkeys() FtSearchWithsortkeys {
	c.cs.s = append(c.cs.s, "WITHSORTKEYS")
	return (FtSearchWithsortkeys)(c)
}

func (c FtSearchNostopwords) Filter(numericField string) FtSearchFilterFilter {
	c.cs.s = append(c.cs.s, "FILTER", numericField)
	return (FtSearchFilterFilter)(c)
}

func (c FtSearchNostopwords) Geofilter(geoField string) FtSearchGeoFilterGeofilter {
	c.cs.s = append(c.cs.s, "GEOFILTER", geoField)
	return (FtSearchGeoFilterGeofilter)(c)
}

func (c FtSearchNostopwords) Inkeys(count string) FtSearchInKeysInkeys {
	c.cs.s = append(c.cs.s, "INKEYS", count)
	return (FtSearchInKeysInkeys)(c)
}

func (c FtSearchNostopwords) Infields(count string) FtSearchInFieldsInfields {
	c.cs.s = append(c.cs.s, "INFIELDS", count)
	return (FtSearchInFieldsInfields)(c)
}

func (c FtSearchNostopwords) Return(count string) FtSearchReturnReturn {
	c.cs.s = append(c.cs.s, "RETURN", count)
	return (FtSearchReturnReturn)(c)
}

func (c FtSearchNostopwords) Summarize() FtSearchSummarizeSummarize {
	c.cs.s = append(c.cs.s, "SUMMARIZE")
	return (FtSearchSummarizeSummarize)(c)
}

func (c FtSearchNostopwords) Highlight() FtSearchHighlightHighlight {
	c.cs.s = append(c.cs.s, "HIGHLIGHT")
	return (FtSearchHighlightHighlight)(c)
}

func (c FtSearchNostopwords) Slop(slop int64) FtSearchSlop {
	c.cs.s = append(c.cs.s, "SLOP", strconv.FormatInt(slop, 10))
	return (FtSearchSlop)(c)
}

func (c FtSearchNostopwords) Timeout(timeout int64) FtSearchTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (FtSearchTimeout)(c)
}

func (c FtSearchNostopwords) Inorder() FtSearchTagsInorder {
	c.cs.s = append(c.cs.s, "INORDER")
	return (FtSearchTagsInorder)(c)
}

func (c FtSearchNostopwords) Language(language string) FtSearchLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", language)
	return (FtSearchLanguage)(c)
}

func (c FtSearchNostopwords) Expander(expander string) FtSearchExpander {
	c.cs.s = append(c.cs.s, "EXPANDER", expander)
	return (FtSearchExpander)(c)
}

func (c FtSearchNostopwords) Scorer(scorer string) FtSearchScorer {
	c.cs.s = append(c.cs.s, "SCORER", scorer)
	return (FtSearchScorer)(c)
}

func (c FtSearchNostopwords) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchNostopwords) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchNostopwords) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchNostopwords) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchNostopwords) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchNostopwords) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchNostopwords) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchParamsNameValue Completed

func (c FtSearchParamsNameValue) NameValue(name string, value string) FtSearchParamsNameValue {
	c.cs.s = append(c.cs.s, name, value)
	return c
}

func (c FtSearchParamsNameValue) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchParamsNameValue) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchParamsNargs Completed

func (c FtSearchParamsNargs) NameValue() FtSearchParamsNameValue {
	return (FtSearchParamsNameValue)(c)
}

type FtSearchParamsParams Completed

func (c FtSearchParamsParams) Nargs(nargs int64) FtSearchParamsNargs {
	c.cs.s = append(c.cs.s, strconv.FormatInt(nargs, 10))
	return (FtSearchParamsNargs)(c)
}

type FtSearchPayload Completed

func (c FtSearchPayload) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchPayload) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchPayload) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchPayload) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchPayload) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchQuery Completed

func (c FtSearchQuery) Nocontent() FtSearchNocontent {
	c.cs.s = append(c.cs.s, "NOCONTENT")
	return (FtSearchNocontent)(c)
}

func (c FtSearchQuery) Verbatim() FtSearchVerbatim {
	c.cs.s = append(c.cs.s, "VERBATIM")
	return (FtSearchVerbatim)(c)
}

func (c FtSearchQuery) Nostopwords() FtSearchNostopwords {
	c.cs.s = append(c.cs.s, "NOSTOPWORDS")
	return (FtSearchNostopwords)(c)
}

func (c FtSearchQuery) Withscores() FtSearchWithscores {
	c.cs.s = append(c.cs.s, "WITHSCORES")
	return (FtSearchWithscores)(c)
}

func (c FtSearchQuery) Withpayloads() FtSearchWithpayloads {
	c.cs.s = append(c.cs.s, "WITHPAYLOADS")
	return (FtSearchWithpayloads)(c)
}

func (c FtSearchQuery) Withsortkeys() FtSearchWithsortkeys {
	c.cs.s = append(c.cs.s, "WITHSORTKEYS")
	return (FtSearchWithsortkeys)(c)
}

func (c FtSearchQuery) Filter(numericField string) FtSearchFilterFilter {
	c.cs.s = append(c.cs.s, "FILTER", numericField)
	return (FtSearchFilterFilter)(c)
}

func (c FtSearchQuery) Geofilter(geoField string) FtSearchGeoFilterGeofilter {
	c.cs.s = append(c.cs.s, "GEOFILTER", geoField)
	return (FtSearchGeoFilterGeofilter)(c)
}

func (c FtSearchQuery) Inkeys(count string) FtSearchInKeysInkeys {
	c.cs.s = append(c.cs.s, "INKEYS", count)
	return (FtSearchInKeysInkeys)(c)
}

func (c FtSearchQuery) Infields(count string) FtSearchInFieldsInfields {
	c.cs.s = append(c.cs.s, "INFIELDS", count)
	return (FtSearchInFieldsInfields)(c)
}

func (c FtSearchQuery) Return(count string) FtSearchReturnReturn {
	c.cs.s = append(c.cs.s, "RETURN", count)
	return (FtSearchReturnReturn)(c)
}

func (c FtSearchQuery) Summarize() FtSearchSummarizeSummarize {
	c.cs.s = append(c.cs.s, "SUMMARIZE")
	return (FtSearchSummarizeSummarize)(c)
}

func (c FtSearchQuery) Highlight() FtSearchHighlightHighlight {
	c.cs.s = append(c.cs.s, "HIGHLIGHT")
	return (FtSearchHighlightHighlight)(c)
}

func (c FtSearchQuery) Slop(slop int64) FtSearchSlop {
	c.cs.s = append(c.cs.s, "SLOP", strconv.FormatInt(slop, 10))
	return (FtSearchSlop)(c)
}

func (c FtSearchQuery) Timeout(timeout int64) FtSearchTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (FtSearchTimeout)(c)
}

func (c FtSearchQuery) Inorder() FtSearchTagsInorder {
	c.cs.s = append(c.cs.s, "INORDER")
	return (FtSearchTagsInorder)(c)
}

func (c FtSearchQuery) Language(language string) FtSearchLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", language)
	return (FtSearchLanguage)(c)
}

func (c FtSearchQuery) Expander(expander string) FtSearchExpander {
	c.cs.s = append(c.cs.s, "EXPANDER", expander)
	return (FtSearchExpander)(c)
}

func (c FtSearchQuery) Scorer(scorer string) FtSearchScorer {
	c.cs.s = append(c.cs.s, "SCORER", scorer)
	return (FtSearchScorer)(c)
}

func (c FtSearchQuery) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchQuery) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchQuery) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchQuery) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchQuery) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchQuery) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchQuery) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchReturnIdentifiersAs Completed

func (c FtSearchReturnIdentifiersAs) Identifier(identifier string) FtSearchReturnIdentifiersIdentifier {
	c.cs.s = append(c.cs.s, identifier)
	return (FtSearchReturnIdentifiersIdentifier)(c)
}

func (c FtSearchReturnIdentifiersAs) Summarize() FtSearchSummarizeSummarize {
	c.cs.s = append(c.cs.s, "SUMMARIZE")
	return (FtSearchSummarizeSummarize)(c)
}

func (c FtSearchReturnIdentifiersAs) Highlight() FtSearchHighlightHighlight {
	c.cs.s = append(c.cs.s, "HIGHLIGHT")
	return (FtSearchHighlightHighlight)(c)
}

func (c FtSearchReturnIdentifiersAs) Slop(slop int64) FtSearchSlop {
	c.cs.s = append(c.cs.s, "SLOP", strconv.FormatInt(slop, 10))
	return (FtSearchSlop)(c)
}

func (c FtSearchReturnIdentifiersAs) Timeout(timeout int64) FtSearchTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (FtSearchTimeout)(c)
}

func (c FtSearchReturnIdentifiersAs) Inorder() FtSearchTagsInorder {
	c.cs.s = append(c.cs.s, "INORDER")
	return (FtSearchTagsInorder)(c)
}

func (c FtSearchReturnIdentifiersAs) Language(language string) FtSearchLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", language)
	return (FtSearchLanguage)(c)
}

func (c FtSearchReturnIdentifiersAs) Expander(expander string) FtSearchExpander {
	c.cs.s = append(c.cs.s, "EXPANDER", expander)
	return (FtSearchExpander)(c)
}

func (c FtSearchReturnIdentifiersAs) Scorer(scorer string) FtSearchScorer {
	c.cs.s = append(c.cs.s, "SCORER", scorer)
	return (FtSearchScorer)(c)
}

func (c FtSearchReturnIdentifiersAs) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchReturnIdentifiersAs) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchReturnIdentifiersAs) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchReturnIdentifiersAs) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchReturnIdentifiersAs) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchReturnIdentifiersAs) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchReturnIdentifiersAs) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchReturnIdentifiersIdentifier Completed

func (c FtSearchReturnIdentifiersIdentifier) As(property string) FtSearchReturnIdentifiersAs {
	c.cs.s = append(c.cs.s, "AS", property)
	return (FtSearchReturnIdentifiersAs)(c)
}

func (c FtSearchReturnIdentifiersIdentifier) Identifier(identifier string) FtSearchReturnIdentifiersIdentifier {
	c.cs.s = append(c.cs.s, identifier)
	return c
}

func (c FtSearchReturnIdentifiersIdentifier) Summarize() FtSearchSummarizeSummarize {
	c.cs.s = append(c.cs.s, "SUMMARIZE")
	return (FtSearchSummarizeSummarize)(c)
}

func (c FtSearchReturnIdentifiersIdentifier) Highlight() FtSearchHighlightHighlight {
	c.cs.s = append(c.cs.s, "HIGHLIGHT")
	return (FtSearchHighlightHighlight)(c)
}

func (c FtSearchReturnIdentifiersIdentifier) Slop(slop int64) FtSearchSlop {
	c.cs.s = append(c.cs.s, "SLOP", strconv.FormatInt(slop, 10))
	return (FtSearchSlop)(c)
}

func (c FtSearchReturnIdentifiersIdentifier) Timeout(timeout int64) FtSearchTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (FtSearchTimeout)(c)
}

func (c FtSearchReturnIdentifiersIdentifier) Inorder() FtSearchTagsInorder {
	c.cs.s = append(c.cs.s, "INORDER")
	return (FtSearchTagsInorder)(c)
}

func (c FtSearchReturnIdentifiersIdentifier) Language(language string) FtSearchLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", language)
	return (FtSearchLanguage)(c)
}

func (c FtSearchReturnIdentifiersIdentifier) Expander(expander string) FtSearchExpander {
	c.cs.s = append(c.cs.s, "EXPANDER", expander)
	return (FtSearchExpander)(c)
}

func (c FtSearchReturnIdentifiersIdentifier) Scorer(scorer string) FtSearchScorer {
	c.cs.s = append(c.cs.s, "SCORER", scorer)
	return (FtSearchScorer)(c)
}

func (c FtSearchReturnIdentifiersIdentifier) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchReturnIdentifiersIdentifier) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchReturnIdentifiersIdentifier) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchReturnIdentifiersIdentifier) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchReturnIdentifiersIdentifier) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchReturnIdentifiersIdentifier) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchReturnIdentifiersIdentifier) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchReturnReturn Completed

func (c FtSearchReturnReturn) Identifier(identifier string) FtSearchReturnIdentifiersIdentifier {
	c.cs.s = append(c.cs.s, identifier)
	return (FtSearchReturnIdentifiersIdentifier)(c)
}

type FtSearchScorer Completed

func (c FtSearchScorer) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchScorer) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchScorer) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchScorer) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchScorer) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchScorer) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchScorer) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchSlop Completed

func (c FtSearchSlop) Timeout(timeout int64) FtSearchTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (FtSearchTimeout)(c)
}

func (c FtSearchSlop) Inorder() FtSearchTagsInorder {
	c.cs.s = append(c.cs.s, "INORDER")
	return (FtSearchTagsInorder)(c)
}

func (c FtSearchSlop) Language(language string) FtSearchLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", language)
	return (FtSearchLanguage)(c)
}

func (c FtSearchSlop) Expander(expander string) FtSearchExpander {
	c.cs.s = append(c.cs.s, "EXPANDER", expander)
	return (FtSearchExpander)(c)
}

func (c FtSearchSlop) Scorer(scorer string) FtSearchScorer {
	c.cs.s = append(c.cs.s, "SCORER", scorer)
	return (FtSearchScorer)(c)
}

func (c FtSearchSlop) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchSlop) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchSlop) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchSlop) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchSlop) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchSlop) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchSlop) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchSortbyOrderAsc Completed

func (c FtSearchSortbyOrderAsc) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchSortbyOrderAsc) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchSortbyOrderAsc) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchSortbyOrderAsc) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchSortbyOrderDesc Completed

func (c FtSearchSortbyOrderDesc) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchSortbyOrderDesc) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchSortbyOrderDesc) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchSortbyOrderDesc) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchSortbySortby Completed

func (c FtSearchSortbySortby) Asc() FtSearchSortbyOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (FtSearchSortbyOrderAsc)(c)
}

func (c FtSearchSortbySortby) Desc() FtSearchSortbyOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (FtSearchSortbyOrderDesc)(c)
}

func (c FtSearchSortbySortby) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchSortbySortby) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchSortbySortby) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchSortbySortby) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchSummarizeFieldsField Completed

func (c FtSearchSummarizeFieldsField) Field(field ...string) FtSearchSummarizeFieldsField {
	c.cs.s = append(c.cs.s, field...)
	return c
}

func (c FtSearchSummarizeFieldsField) Frags(num int64) FtSearchSummarizeFrags {
	c.cs.s = append(c.cs.s, "FRAGS", strconv.FormatInt(num, 10))
	return (FtSearchSummarizeFrags)(c)
}

func (c FtSearchSummarizeFieldsField) Len(fragsize int64) FtSearchSummarizeLen {
	c.cs.s = append(c.cs.s, "LEN", strconv.FormatInt(fragsize, 10))
	return (FtSearchSummarizeLen)(c)
}

func (c FtSearchSummarizeFieldsField) Separator(separator string) FtSearchSummarizeSeparator {
	c.cs.s = append(c.cs.s, "SEPARATOR", separator)
	return (FtSearchSummarizeSeparator)(c)
}

func (c FtSearchSummarizeFieldsField) Highlight() FtSearchHighlightHighlight {
	c.cs.s = append(c.cs.s, "HIGHLIGHT")
	return (FtSearchHighlightHighlight)(c)
}

func (c FtSearchSummarizeFieldsField) Slop(slop int64) FtSearchSlop {
	c.cs.s = append(c.cs.s, "SLOP", strconv.FormatInt(slop, 10))
	return (FtSearchSlop)(c)
}

func (c FtSearchSummarizeFieldsField) Timeout(timeout int64) FtSearchTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (FtSearchTimeout)(c)
}

func (c FtSearchSummarizeFieldsField) Inorder() FtSearchTagsInorder {
	c.cs.s = append(c.cs.s, "INORDER")
	return (FtSearchTagsInorder)(c)
}

func (c FtSearchSummarizeFieldsField) Language(language string) FtSearchLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", language)
	return (FtSearchLanguage)(c)
}

func (c FtSearchSummarizeFieldsField) Expander(expander string) FtSearchExpander {
	c.cs.s = append(c.cs.s, "EXPANDER", expander)
	return (FtSearchExpander)(c)
}

func (c FtSearchSummarizeFieldsField) Scorer(scorer string) FtSearchScorer {
	c.cs.s = append(c.cs.s, "SCORER", scorer)
	return (FtSearchScorer)(c)
}

func (c FtSearchSummarizeFieldsField) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchSummarizeFieldsField) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchSummarizeFieldsField) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchSummarizeFieldsField) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchSummarizeFieldsField) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchSummarizeFieldsField) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchSummarizeFieldsField) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchSummarizeFieldsFields Completed

func (c FtSearchSummarizeFieldsFields) Field(field ...string) FtSearchSummarizeFieldsField {
	c.cs.s = append(c.cs.s, field...)
	return (FtSearchSummarizeFieldsField)(c)
}

type FtSearchSummarizeFrags Completed

func (c FtSearchSummarizeFrags) Len(fragsize int64) FtSearchSummarizeLen {
	c.cs.s = append(c.cs.s, "LEN", strconv.FormatInt(fragsize, 10))
	return (FtSearchSummarizeLen)(c)
}

func (c FtSearchSummarizeFrags) Separator(separator string) FtSearchSummarizeSeparator {
	c.cs.s = append(c.cs.s, "SEPARATOR", separator)
	return (FtSearchSummarizeSeparator)(c)
}

func (c FtSearchSummarizeFrags) Highlight() FtSearchHighlightHighlight {
	c.cs.s = append(c.cs.s, "HIGHLIGHT")
	return (FtSearchHighlightHighlight)(c)
}

func (c FtSearchSummarizeFrags) Slop(slop int64) FtSearchSlop {
	c.cs.s = append(c.cs.s, "SLOP", strconv.FormatInt(slop, 10))
	return (FtSearchSlop)(c)
}

func (c FtSearchSummarizeFrags) Timeout(timeout int64) FtSearchTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (FtSearchTimeout)(c)
}

func (c FtSearchSummarizeFrags) Inorder() FtSearchTagsInorder {
	c.cs.s = append(c.cs.s, "INORDER")
	return (FtSearchTagsInorder)(c)
}

func (c FtSearchSummarizeFrags) Language(language string) FtSearchLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", language)
	return (FtSearchLanguage)(c)
}

func (c FtSearchSummarizeFrags) Expander(expander string) FtSearchExpander {
	c.cs.s = append(c.cs.s, "EXPANDER", expander)
	return (FtSearchExpander)(c)
}

func (c FtSearchSummarizeFrags) Scorer(scorer string) FtSearchScorer {
	c.cs.s = append(c.cs.s, "SCORER", scorer)
	return (FtSearchScorer)(c)
}

func (c FtSearchSummarizeFrags) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchSummarizeFrags) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchSummarizeFrags) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchSummarizeFrags) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchSummarizeFrags) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchSummarizeFrags) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchSummarizeFrags) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchSummarizeLen Completed

func (c FtSearchSummarizeLen) Separator(separator string) FtSearchSummarizeSeparator {
	c.cs.s = append(c.cs.s, "SEPARATOR", separator)
	return (FtSearchSummarizeSeparator)(c)
}

func (c FtSearchSummarizeLen) Highlight() FtSearchHighlightHighlight {
	c.cs.s = append(c.cs.s, "HIGHLIGHT")
	return (FtSearchHighlightHighlight)(c)
}

func (c FtSearchSummarizeLen) Slop(slop int64) FtSearchSlop {
	c.cs.s = append(c.cs.s, "SLOP", strconv.FormatInt(slop, 10))
	return (FtSearchSlop)(c)
}

func (c FtSearchSummarizeLen) Timeout(timeout int64) FtSearchTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (FtSearchTimeout)(c)
}

func (c FtSearchSummarizeLen) Inorder() FtSearchTagsInorder {
	c.cs.s = append(c.cs.s, "INORDER")
	return (FtSearchTagsInorder)(c)
}

func (c FtSearchSummarizeLen) Language(language string) FtSearchLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", language)
	return (FtSearchLanguage)(c)
}

func (c FtSearchSummarizeLen) Expander(expander string) FtSearchExpander {
	c.cs.s = append(c.cs.s, "EXPANDER", expander)
	return (FtSearchExpander)(c)
}

func (c FtSearchSummarizeLen) Scorer(scorer string) FtSearchScorer {
	c.cs.s = append(c.cs.s, "SCORER", scorer)
	return (FtSearchScorer)(c)
}

func (c FtSearchSummarizeLen) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchSummarizeLen) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchSummarizeLen) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchSummarizeLen) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchSummarizeLen) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchSummarizeLen) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchSummarizeLen) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchSummarizeSeparator Completed

func (c FtSearchSummarizeSeparator) Highlight() FtSearchHighlightHighlight {
	c.cs.s = append(c.cs.s, "HIGHLIGHT")
	return (FtSearchHighlightHighlight)(c)
}

func (c FtSearchSummarizeSeparator) Slop(slop int64) FtSearchSlop {
	c.cs.s = append(c.cs.s, "SLOP", strconv.FormatInt(slop, 10))
	return (FtSearchSlop)(c)
}

func (c FtSearchSummarizeSeparator) Timeout(timeout int64) FtSearchTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (FtSearchTimeout)(c)
}

func (c FtSearchSummarizeSeparator) Inorder() FtSearchTagsInorder {
	c.cs.s = append(c.cs.s, "INORDER")
	return (FtSearchTagsInorder)(c)
}

func (c FtSearchSummarizeSeparator) Language(language string) FtSearchLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", language)
	return (FtSearchLanguage)(c)
}

func (c FtSearchSummarizeSeparator) Expander(expander string) FtSearchExpander {
	c.cs.s = append(c.cs.s, "EXPANDER", expander)
	return (FtSearchExpander)(c)
}

func (c FtSearchSummarizeSeparator) Scorer(scorer string) FtSearchScorer {
	c.cs.s = append(c.cs.s, "SCORER", scorer)
	return (FtSearchScorer)(c)
}

func (c FtSearchSummarizeSeparator) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchSummarizeSeparator) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchSummarizeSeparator) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchSummarizeSeparator) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchSummarizeSeparator) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchSummarizeSeparator) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchSummarizeSeparator) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchSummarizeSummarize Completed

func (c FtSearchSummarizeSummarize) Fields(count string) FtSearchSummarizeFieldsFields {
	c.cs.s = append(c.cs.s, "FIELDS", count)
	return (FtSearchSummarizeFieldsFields)(c)
}

func (c FtSearchSummarizeSummarize) Frags(num int64) FtSearchSummarizeFrags {
	c.cs.s = append(c.cs.s, "FRAGS", strconv.FormatInt(num, 10))
	return (FtSearchSummarizeFrags)(c)
}

func (c FtSearchSummarizeSummarize) Len(fragsize int64) FtSearchSummarizeLen {
	c.cs.s = append(c.cs.s, "LEN", strconv.FormatInt(fragsize, 10))
	return (FtSearchSummarizeLen)(c)
}

func (c FtSearchSummarizeSummarize) Separator(separator string) FtSearchSummarizeSeparator {
	c.cs.s = append(c.cs.s, "SEPARATOR", separator)
	return (FtSearchSummarizeSeparator)(c)
}

func (c FtSearchSummarizeSummarize) Highlight() FtSearchHighlightHighlight {
	c.cs.s = append(c.cs.s, "HIGHLIGHT")
	return (FtSearchHighlightHighlight)(c)
}

func (c FtSearchSummarizeSummarize) Slop(slop int64) FtSearchSlop {
	c.cs.s = append(c.cs.s, "SLOP", strconv.FormatInt(slop, 10))
	return (FtSearchSlop)(c)
}

func (c FtSearchSummarizeSummarize) Timeout(timeout int64) FtSearchTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (FtSearchTimeout)(c)
}

func (c FtSearchSummarizeSummarize) Inorder() FtSearchTagsInorder {
	c.cs.s = append(c.cs.s, "INORDER")
	return (FtSearchTagsInorder)(c)
}

func (c FtSearchSummarizeSummarize) Language(language string) FtSearchLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", language)
	return (FtSearchLanguage)(c)
}

func (c FtSearchSummarizeSummarize) Expander(expander string) FtSearchExpander {
	c.cs.s = append(c.cs.s, "EXPANDER", expander)
	return (FtSearchExpander)(c)
}

func (c FtSearchSummarizeSummarize) Scorer(scorer string) FtSearchScorer {
	c.cs.s = append(c.cs.s, "SCORER", scorer)
	return (FtSearchScorer)(c)
}

func (c FtSearchSummarizeSummarize) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchSummarizeSummarize) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchSummarizeSummarize) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchSummarizeSummarize) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchSummarizeSummarize) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchSummarizeSummarize) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchSummarizeSummarize) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchTagsInorder Completed

func (c FtSearchTagsInorder) Language(language string) FtSearchLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", language)
	return (FtSearchLanguage)(c)
}

func (c FtSearchTagsInorder) Expander(expander string) FtSearchExpander {
	c.cs.s = append(c.cs.s, "EXPANDER", expander)
	return (FtSearchExpander)(c)
}

func (c FtSearchTagsInorder) Scorer(scorer string) FtSearchScorer {
	c.cs.s = append(c.cs.s, "SCORER", scorer)
	return (FtSearchScorer)(c)
}

func (c FtSearchTagsInorder) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchTagsInorder) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchTagsInorder) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchTagsInorder) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchTagsInorder) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchTagsInorder) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchTagsInorder) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchTimeout Completed

func (c FtSearchTimeout) Inorder() FtSearchTagsInorder {
	c.cs.s = append(c.cs.s, "INORDER")
	return (FtSearchTagsInorder)(c)
}

func (c FtSearchTimeout) Language(language string) FtSearchLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", language)
	return (FtSearchLanguage)(c)
}

func (c FtSearchTimeout) Expander(expander string) FtSearchExpander {
	c.cs.s = append(c.cs.s, "EXPANDER", expander)
	return (FtSearchExpander)(c)
}

func (c FtSearchTimeout) Scorer(scorer string) FtSearchScorer {
	c.cs.s = append(c.cs.s, "SCORER", scorer)
	return (FtSearchScorer)(c)
}

func (c FtSearchTimeout) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchTimeout) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchTimeout) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchTimeout) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchTimeout) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchTimeout) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchTimeout) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchVerbatim Completed

func (c FtSearchVerbatim) Nostopwords() FtSearchNostopwords {
	c.cs.s = append(c.cs.s, "NOSTOPWORDS")
	return (FtSearchNostopwords)(c)
}

func (c FtSearchVerbatim) Withscores() FtSearchWithscores {
	c.cs.s = append(c.cs.s, "WITHSCORES")
	return (FtSearchWithscores)(c)
}

func (c FtSearchVerbatim) Withpayloads() FtSearchWithpayloads {
	c.cs.s = append(c.cs.s, "WITHPAYLOADS")
	return (FtSearchWithpayloads)(c)
}

func (c FtSearchVerbatim) Withsortkeys() FtSearchWithsortkeys {
	c.cs.s = append(c.cs.s, "WITHSORTKEYS")
	return (FtSearchWithsortkeys)(c)
}

func (c FtSearchVerbatim) Filter(numericField string) FtSearchFilterFilter {
	c.cs.s = append(c.cs.s, "FILTER", numericField)
	return (FtSearchFilterFilter)(c)
}

func (c FtSearchVerbatim) Geofilter(geoField string) FtSearchGeoFilterGeofilter {
	c.cs.s = append(c.cs.s, "GEOFILTER", geoField)
	return (FtSearchGeoFilterGeofilter)(c)
}

func (c FtSearchVerbatim) Inkeys(count string) FtSearchInKeysInkeys {
	c.cs.s = append(c.cs.s, "INKEYS", count)
	return (FtSearchInKeysInkeys)(c)
}

func (c FtSearchVerbatim) Infields(count string) FtSearchInFieldsInfields {
	c.cs.s = append(c.cs.s, "INFIELDS", count)
	return (FtSearchInFieldsInfields)(c)
}

func (c FtSearchVerbatim) Return(count string) FtSearchReturnReturn {
	c.cs.s = append(c.cs.s, "RETURN", count)
	return (FtSearchReturnReturn)(c)
}

func (c FtSearchVerbatim) Summarize() FtSearchSummarizeSummarize {
	c.cs.s = append(c.cs.s, "SUMMARIZE")
	return (FtSearchSummarizeSummarize)(c)
}

func (c FtSearchVerbatim) Highlight() FtSearchHighlightHighlight {
	c.cs.s = append(c.cs.s, "HIGHLIGHT")
	return (FtSearchHighlightHighlight)(c)
}

func (c FtSearchVerbatim) Slop(slop int64) FtSearchSlop {
	c.cs.s = append(c.cs.s, "SLOP", strconv.FormatInt(slop, 10))
	return (FtSearchSlop)(c)
}

func (c FtSearchVerbatim) Timeout(timeout int64) FtSearchTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (FtSearchTimeout)(c)
}

func (c FtSearchVerbatim) Inorder() FtSearchTagsInorder {
	c.cs.s = append(c.cs.s, "INORDER")
	return (FtSearchTagsInorder)(c)
}

func (c FtSearchVerbatim) Language(language string) FtSearchLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", language)
	return (FtSearchLanguage)(c)
}

func (c FtSearchVerbatim) Expander(expander string) FtSearchExpander {
	c.cs.s = append(c.cs.s, "EXPANDER", expander)
	return (FtSearchExpander)(c)
}

func (c FtSearchVerbatim) Scorer(scorer string) FtSearchScorer {
	c.cs.s = append(c.cs.s, "SCORER", scorer)
	return (FtSearchScorer)(c)
}

func (c FtSearchVerbatim) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchVerbatim) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchVerbatim) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchVerbatim) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchVerbatim) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchVerbatim) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchVerbatim) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchWithpayloads Completed

func (c FtSearchWithpayloads) Withsortkeys() FtSearchWithsortkeys {
	c.cs.s = append(c.cs.s, "WITHSORTKEYS")
	return (FtSearchWithsortkeys)(c)
}

func (c FtSearchWithpayloads) Filter(numericField string) FtSearchFilterFilter {
	c.cs.s = append(c.cs.s, "FILTER", numericField)
	return (FtSearchFilterFilter)(c)
}

func (c FtSearchWithpayloads) Geofilter(geoField string) FtSearchGeoFilterGeofilter {
	c.cs.s = append(c.cs.s, "GEOFILTER", geoField)
	return (FtSearchGeoFilterGeofilter)(c)
}

func (c FtSearchWithpayloads) Inkeys(count string) FtSearchInKeysInkeys {
	c.cs.s = append(c.cs.s, "INKEYS", count)
	return (FtSearchInKeysInkeys)(c)
}

func (c FtSearchWithpayloads) Infields(count string) FtSearchInFieldsInfields {
	c.cs.s = append(c.cs.s, "INFIELDS", count)
	return (FtSearchInFieldsInfields)(c)
}

func (c FtSearchWithpayloads) Return(count string) FtSearchReturnReturn {
	c.cs.s = append(c.cs.s, "RETURN", count)
	return (FtSearchReturnReturn)(c)
}

func (c FtSearchWithpayloads) Summarize() FtSearchSummarizeSummarize {
	c.cs.s = append(c.cs.s, "SUMMARIZE")
	return (FtSearchSummarizeSummarize)(c)
}

func (c FtSearchWithpayloads) Highlight() FtSearchHighlightHighlight {
	c.cs.s = append(c.cs.s, "HIGHLIGHT")
	return (FtSearchHighlightHighlight)(c)
}

func (c FtSearchWithpayloads) Slop(slop int64) FtSearchSlop {
	c.cs.s = append(c.cs.s, "SLOP", strconv.FormatInt(slop, 10))
	return (FtSearchSlop)(c)
}

func (c FtSearchWithpayloads) Timeout(timeout int64) FtSearchTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (FtSearchTimeout)(c)
}

func (c FtSearchWithpayloads) Inorder() FtSearchTagsInorder {
	c.cs.s = append(c.cs.s, "INORDER")
	return (FtSearchTagsInorder)(c)
}

func (c FtSearchWithpayloads) Language(language string) FtSearchLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", language)
	return (FtSearchLanguage)(c)
}

func (c FtSearchWithpayloads) Expander(expander string) FtSearchExpander {
	c.cs.s = append(c.cs.s, "EXPANDER", expander)
	return (FtSearchExpander)(c)
}

func (c FtSearchWithpayloads) Scorer(scorer string) FtSearchScorer {
	c.cs.s = append(c.cs.s, "SCORER", scorer)
	return (FtSearchScorer)(c)
}

func (c FtSearchWithpayloads) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchWithpayloads) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchWithpayloads) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchWithpayloads) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchWithpayloads) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchWithpayloads) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchWithpayloads) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchWithscores Completed

func (c FtSearchWithscores) Withpayloads() FtSearchWithpayloads {
	c.cs.s = append(c.cs.s, "WITHPAYLOADS")
	return (FtSearchWithpayloads)(c)
}

func (c FtSearchWithscores) Withsortkeys() FtSearchWithsortkeys {
	c.cs.s = append(c.cs.s, "WITHSORTKEYS")
	return (FtSearchWithsortkeys)(c)
}

func (c FtSearchWithscores) Filter(numericField string) FtSearchFilterFilter {
	c.cs.s = append(c.cs.s, "FILTER", numericField)
	return (FtSearchFilterFilter)(c)
}

func (c FtSearchWithscores) Geofilter(geoField string) FtSearchGeoFilterGeofilter {
	c.cs.s = append(c.cs.s, "GEOFILTER", geoField)
	return (FtSearchGeoFilterGeofilter)(c)
}

func (c FtSearchWithscores) Inkeys(count string) FtSearchInKeysInkeys {
	c.cs.s = append(c.cs.s, "INKEYS", count)
	return (FtSearchInKeysInkeys)(c)
}

func (c FtSearchWithscores) Infields(count string) FtSearchInFieldsInfields {
	c.cs.s = append(c.cs.s, "INFIELDS", count)
	return (FtSearchInFieldsInfields)(c)
}

func (c FtSearchWithscores) Return(count string) FtSearchReturnReturn {
	c.cs.s = append(c.cs.s, "RETURN", count)
	return (FtSearchReturnReturn)(c)
}

func (c FtSearchWithscores) Summarize() FtSearchSummarizeSummarize {
	c.cs.s = append(c.cs.s, "SUMMARIZE")
	return (FtSearchSummarizeSummarize)(c)
}

func (c FtSearchWithscores) Highlight() FtSearchHighlightHighlight {
	c.cs.s = append(c.cs.s, "HIGHLIGHT")
	return (FtSearchHighlightHighlight)(c)
}

func (c FtSearchWithscores) Slop(slop int64) FtSearchSlop {
	c.cs.s = append(c.cs.s, "SLOP", strconv.FormatInt(slop, 10))
	return (FtSearchSlop)(c)
}

func (c FtSearchWithscores) Timeout(timeout int64) FtSearchTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (FtSearchTimeout)(c)
}

func (c FtSearchWithscores) Inorder() FtSearchTagsInorder {
	c.cs.s = append(c.cs.s, "INORDER")
	return (FtSearchTagsInorder)(c)
}

func (c FtSearchWithscores) Language(language string) FtSearchLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", language)
	return (FtSearchLanguage)(c)
}

func (c FtSearchWithscores) Expander(expander string) FtSearchExpander {
	c.cs.s = append(c.cs.s, "EXPANDER", expander)
	return (FtSearchExpander)(c)
}

func (c FtSearchWithscores) Scorer(scorer string) FtSearchScorer {
	c.cs.s = append(c.cs.s, "SCORER", scorer)
	return (FtSearchScorer)(c)
}

func (c FtSearchWithscores) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchWithscores) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchWithscores) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchWithscores) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchWithscores) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchWithscores) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchWithscores) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchWithsortkeys Completed

func (c FtSearchWithsortkeys) Filter(numericField string) FtSearchFilterFilter {
	c.cs.s = append(c.cs.s, "FILTER", numericField)
	return (FtSearchFilterFilter)(c)
}

func (c FtSearchWithsortkeys) Geofilter(geoField string) FtSearchGeoFilterGeofilter {
	c.cs.s = append(c.cs.s, "GEOFILTER", geoField)
	return (FtSearchGeoFilterGeofilter)(c)
}

func (c FtSearchWithsortkeys) Inkeys(count string) FtSearchInKeysInkeys {
	c.cs.s = append(c.cs.s, "INKEYS", count)
	return (FtSearchInKeysInkeys)(c)
}

func (c FtSearchWithsortkeys) Infields(count string) FtSearchInFieldsInfields {
	c.cs.s = append(c.cs.s, "INFIELDS", count)
	return (FtSearchInFieldsInfields)(c)
}

func (c FtSearchWithsortkeys) Return(count string) FtSearchReturnReturn {
	c.cs.s = append(c.cs.s, "RETURN", count)
	return (FtSearchReturnReturn)(c)
}

func (c FtSearchWithsortkeys) Summarize() FtSearchSummarizeSummarize {
	c.cs.s = append(c.cs.s, "SUMMARIZE")
	return (FtSearchSummarizeSummarize)(c)
}

func (c FtSearchWithsortkeys) Highlight() FtSearchHighlightHighlight {
	c.cs.s = append(c.cs.s, "HIGHLIGHT")
	return (FtSearchHighlightHighlight)(c)
}

func (c FtSearchWithsortkeys) Slop(slop int64) FtSearchSlop {
	c.cs.s = append(c.cs.s, "SLOP", strconv.FormatInt(slop, 10))
	return (FtSearchSlop)(c)
}

func (c FtSearchWithsortkeys) Timeout(timeout int64) FtSearchTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (FtSearchTimeout)(c)
}

func (c FtSearchWithsortkeys) Inorder() FtSearchTagsInorder {
	c.cs.s = append(c.cs.s, "INORDER")
	return (FtSearchTagsInorder)(c)
}

func (c FtSearchWithsortkeys) Language(language string) FtSearchLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", language)
	return (FtSearchLanguage)(c)
}

func (c FtSearchWithsortkeys) Expander(expander string) FtSearchExpander {
	c.cs.s = append(c.cs.s, "EXPANDER", expander)
	return (FtSearchExpander)(c)
}

func (c FtSearchWithsortkeys) Scorer(scorer string) FtSearchScorer {
	c.cs.s = append(c.cs.s, "SCORER", scorer)
	return (FtSearchScorer)(c)
}

func (c FtSearchWithsortkeys) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchWithsortkeys) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchWithsortkeys) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchWithsortkeys) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchWithsortkeys) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchWithsortkeys) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchWithsortkeys) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSpellcheck Completed

func (b Builder) FtSpellcheck() (c FtSpellcheck) {
	c = FtSpellcheck{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FT.SPELLCHECK")
	return c
}

func (c FtSpellcheck) Index(index string) FtSpellcheckIndex {
	c.cs.s = append(c.cs.s, index)
	return (FtSpellcheckIndex)(c)
}

type FtSpellcheckDialect Completed

func (c FtSpellcheckDialect) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSpellcheckDistance Completed

func (c FtSpellcheckDistance) TermsInclude() FtSpellcheckTermsTermsInclude {
	c.cs.s = append(c.cs.s, "TERMS", "INCLUDE")
	return (FtSpellcheckTermsTermsInclude)(c)
}

func (c FtSpellcheckDistance) TermsExclude() FtSpellcheckTermsTermsExclude {
	c.cs.s = append(c.cs.s, "TERMS", "EXCLUDE")
	return (FtSpellcheckTermsTermsExclude)(c)
}

func (c FtSpellcheckDistance) Dialect(dialect int64) FtSpellcheckDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSpellcheckDialect)(c)
}

func (c FtSpellcheckDistance) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSpellcheckIndex Completed

func (c FtSpellcheckIndex) Query(query string) FtSpellcheckQuery {
	c.cs.s = append(c.cs.s, query)
	return (FtSpellcheckQuery)(c)
}

type FtSpellcheckQuery Completed

func (c FtSpellcheckQuery) Distance(distance int64) FtSpellcheckDistance {
	c.cs.s = append(c.cs.s, "DISTANCE", strconv.FormatInt(distance, 10))
	return (FtSpellcheckDistance)(c)
}

func (c FtSpellcheckQuery) TermsInclude() FtSpellcheckTermsTermsInclude {
	c.cs.s = append(c.cs.s, "TERMS", "INCLUDE")
	return (FtSpellcheckTermsTermsInclude)(c)
}

func (c FtSpellcheckQuery) TermsExclude() FtSpellcheckTermsTermsExclude {
	c.cs.s = append(c.cs.s, "TERMS", "EXCLUDE")
	return (FtSpellcheckTermsTermsExclude)(c)
}

func (c FtSpellcheckQuery) Dialect(dialect int64) FtSpellcheckDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSpellcheckDialect)(c)
}

func (c FtSpellcheckQuery) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSpellcheckTermsDictionary Completed

func (c FtSpellcheckTermsDictionary) Terms(terms ...string) FtSpellcheckTermsTerms {
	c.cs.s = append(c.cs.s, terms...)
	return (FtSpellcheckTermsTerms)(c)
}

func (c FtSpellcheckTermsDictionary) Dialect(dialect int64) FtSpellcheckDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSpellcheckDialect)(c)
}

func (c FtSpellcheckTermsDictionary) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSpellcheckTermsTerms Completed

func (c FtSpellcheckTermsTerms) Terms(terms ...string) FtSpellcheckTermsTerms {
	c.cs.s = append(c.cs.s, terms...)
	return c
}

func (c FtSpellcheckTermsTerms) Dialect(dialect int64) FtSpellcheckDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSpellcheckDialect)(c)
}

func (c FtSpellcheckTermsTerms) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSpellcheckTermsTermsExclude Completed

func (c FtSpellcheckTermsTermsExclude) Dictionary(dictionary string) FtSpellcheckTermsDictionary {
	c.cs.s = append(c.cs.s, dictionary)
	return (FtSpellcheckTermsDictionary)(c)
}

type FtSpellcheckTermsTermsInclude Completed

func (c FtSpellcheckTermsTermsInclude) Dictionary(dictionary string) FtSpellcheckTermsDictionary {
	c.cs.s = append(c.cs.s, dictionary)
	return (FtSpellcheckTermsDictionary)(c)
}

type FtSyndump Completed

func (b Builder) FtSyndump() (c FtSyndump) {
	c = FtSyndump{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FT.SYNDUMP")
	return c
}

func (c FtSyndump) Index(index string) FtSyndumpIndex {
	c.cs.s = append(c.cs.s, index)
	return (FtSyndumpIndex)(c)
}

type FtSyndumpIndex Completed

func (c FtSyndumpIndex) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSynupdate Completed

func (b Builder) FtSynupdate() (c FtSynupdate) {
	c = FtSynupdate{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FT.SYNUPDATE")
	return c
}

func (c FtSynupdate) Index(index string) FtSynupdateIndex {
	c.cs.s = append(c.cs.s, index)
	return (FtSynupdateIndex)(c)
}

type FtSynupdateIndex Completed

func (c FtSynupdateIndex) SynonymGroupId(synonymGroupId string) FtSynupdateSynonymGroupId {
	c.cs.s = append(c.cs.s, synonymGroupId)
	return (FtSynupdateSynonymGroupId)(c)
}

type FtSynupdateSkipinitialscan Completed

func (c FtSynupdateSkipinitialscan) Term(term ...string) FtSynupdateTerm {
	c.cs.s = append(c.cs.s, term...)
	return (FtSynupdateTerm)(c)
}

type FtSynupdateSynonymGroupId Completed

func (c FtSynupdateSynonymGroupId) Skipinitialscan() FtSynupdateSkipinitialscan {
	c.cs.s = append(c.cs.s, "SKIPINITIALSCAN")
	return (FtSynupdateSkipinitialscan)(c)
}

func (c FtSynupdateSynonymGroupId) Term(term ...string) FtSynupdateTerm {
	c.cs.s = append(c.cs.s, term...)
	return (FtSynupdateTerm)(c)
}

type FtSynupdateTerm Completed

func (c FtSynupdateTerm) Term(term ...string) FtSynupdateTerm {
	c.cs.s = append(c.cs.s, term...)
	return c
}

func (c FtSynupdateTerm) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtTagvals Completed

func (b Builder) FtTagvals() (c FtTagvals) {
	c = FtTagvals{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FT.TAGVALS")
	return c
}

func (c FtTagvals) Index(index string) FtTagvalsIndex {
	c.cs.s = append(c.cs.s, index)
	return (FtTagvalsIndex)(c)
}

type FtTagvalsFieldName Completed

func (c FtTagvalsFieldName) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtTagvalsIndex Completed

func (c FtTagvalsIndex) FieldName(fieldName string) FtTagvalsFieldName {
	c.cs.s = append(c.cs.s, fieldName)
	return (FtTagvalsFieldName)(c)
}
