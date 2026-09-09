package insights

import (
	"maps"
	"slices"
)

// FunctionSchema is one allowed function's exported metadata -- used by
// cmd/gen-insights-schema's JSON dump, the same way TableSchema
// (describe.go) is for tables/columns.
type FunctionSchema struct {
	Name        string
	Description string
	DocsURL     string
}

// AllFunctionSchemas returns every allowed function's schema, sorted by
// name.
func AllFunctionSchemas() []FunctionSchema {
	names := slices.Sorted(maps.Keys(allowedFunctions))
	out := make([]FunctionSchema, len(names))
	for i, name := range names {
		info := allowedFunctions[name]
		out[i] = FunctionSchema{Name: name, Description: info.description, DocsURL: info.docsURL}
	}
	return out
}

// functionInfo is one allowed function's metadata: a short description and
// a link to its official DuckDB documentation, both surfaced by
// cmd/gen-insights-schema's JSON dump (embedded into the SQL editor UI) and
// by DESCRIBE-adjacent tooling.
//
// docsURL points at the function's own anchor on its DuckDB docs page
// (e.g. ".../aggregates#sumarg"), extracted by hand from that page's real
// HTML (curl + grep for the doc site's own "<a href="#anchor"><code>
// name(...)" self-links -- not guessed, and not taken from an LLM's
// unverified summary of the page, which was tried first and produced
// fabricated anchors that don't exist). A small number of functions have no
// per-function anchor in DuckDB's own docs at all (verified by the same
// method) -- those fall back to the closest page or subsection instead of
// a fabricated deep link: MOD, NOW, TO_TIMESTAMP have no anchor on their
// page; JSON_ARRAY/JSON_OBJECT/JSON_QUOTE/JSON_MERGE_PATCH/TO_JSON share
// the "Creating JSON" page with no per-function anchors there either.
// Captured against DuckDB's docs as published when this was written --
// not re-verified on every build, so a doc site restructure could stale
// these without this package noticing.
type functionInfo struct {
	description string
	docsURL     string
}

// allowedFunctions is the conservative starting function allowlist. Keys
// are upper-cased, dot-joined function names. No table-function or
// filesystem/catalog-touching surface (read_csv, ATTACH, etc.) is ever
// included here. functionReturnType (typecheck.go) has a case for every
// key here — if you add a function, add its return-type rule too.
//
// Deliberately excludes COALESCE, NULLIF, TRIM, and SUBSTRING even though
// they're ordinary DuckDB functions: pkg/duckdb/parser doesn't implement
// their grammar production (DuckDB gives these their own special
// keyword-based syntax instead of routing through the generic function-call
// rule), so ParseString errors on them unconditionally — confirmed
// empirically. Including them here would be dead code. SUBSTR/LTRIM/RTRIM
// (this package's plain-function-call equivalents) are unaffected.
//
// Descriptions are DuckDB's own (from duckdb_functions().description against
// the vendored DuckDB version, or hand-written for the handful of entries
// that table leaves NULL -- IF/IFNULL/NVL aren't in the catalog at all under
// those names, and the JSON extension's functions don't populate
// description in the build this was captured against). Not auto-synced --
// if a re-vendored DuckDB changes a function's behavior, these won't notice.
var allowedFunctions = map[string]functionInfo{
	"abs":                   {description: "Absolute value.", docsURL: "https://duckdb.org/docs/current/sql/functions/numeric#absx"},
	"age":                   {description: "Subtract arguments, resulting in the time difference between the two timestamps.", docsURL: "https://duckdb.org/docs/current/sql/functions/timestamp#agetimestamp-timestamp"},
	"any_value":             {description: "Returns the first non-NULL value from arg. This function is affected by ordering.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates#any_valuearg"},
	"approx_count_distinct": {description: "Computes the approximate count of distinct elements using HyperLogLog.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates#approximate-aggregates"},
	"approx_quantile":       {description: "Computes the approximate quantile using T-Digest.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates#approximate-aggregates"},
	"arg_max":               {description: "Finds the row with the maximum val. Calculates the non-NULL arg expression at that row.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates#arg_maxarg-val"},
	"arg_min":               {description: "Finds the row with the minimum val. Calculates the non-NULL arg expression at that row.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates#arg_minarg-val"},
	"array_agg":             {description: "Returns a LIST containing all the values of a column.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates#listarg"},
	"array_aggregate":       {description: "Executes the aggregate function function_name on the elements of list.", docsURL: "https://duckdb.org/docs/current/sql/functions/list#list_aggregatelist-function_name-"},
	"array_contains":        {description: "Returns true if the list contains the element.", docsURL: "https://duckdb.org/docs/current/sql/functions/list#list_containslist-element"},
	"array_length":          {description: "Returns the length of the list.", docsURL: "https://duckdb.org/docs/current/sql/functions/list#lengthlist"},
	"array_position":        {description: "Returns the index of the element if the list contains the element. If the element is not found, it returns NULL.", docsURL: "https://duckdb.org/docs/current/sql/functions/list#list_positionlist-element"},
	"array_to_string":       {description: "Concatenates list/array elements using an optional delimiter.", docsURL: "https://duckdb.org/docs/current/sql/functions/list#array_to_stringlist-delimiter"},
	"ascii":                 {description: "Returns an integer that represents the Unicode code point of the first character of the string.", docsURL: "https://duckdb.org/docs/current/sql/functions/text#asciistring"},
	"avg":                   {description: "Calculates the average value for all tuples in x.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates#avgarg"},
	"bit_and":               {description: "Returns the bitwise AND of all bits in a given expression.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates#bit_andarg"},
	"bit_or":                {description: "Returns the bitwise OR of all bits in a given expression.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates#bit_orarg"},
	"bit_xor":               {description: "Returns the bitwise XOR of all bits in a given expression.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates#bit_xorarg"},
	"bool_and":              {description: "Returns TRUE if every input value is TRUE, otherwise FALSE.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates#bool_andarg"},
	"bool_or":               {description: "Returns TRUE if any input value is TRUE, otherwise FALSE.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates#bool_orarg"},
	"cbrt":                  {description: "Returns the cube root of x.", docsURL: "https://duckdb.org/docs/current/sql/functions/numeric#cbrtx"},
	"ceil":                  {description: "Rounds the number up.", docsURL: "https://duckdb.org/docs/current/sql/functions/numeric#ceilx"},
	"ceiling":               {description: "Rounds the number up.", docsURL: "https://duckdb.org/docs/current/sql/functions/numeric#ceilingx"},
	"chr":                   {description: "Returns a character which is corresponding the ASCII code value or Unicode code point.", docsURL: "https://duckdb.org/docs/current/sql/functions/text#chrcode_point"},
	"concat":                {description: "Concatenates multiple strings or lists. NULL inputs are skipped. See also operator ||.", docsURL: "https://duckdb.org/docs/current/sql/functions/text#concatvalue-"},
	"contains":              {description: "Returns true if search_string is found within string.", docsURL: "https://duckdb.org/docs/current/sql/functions/text#containsstring-search_string"},
	"corr":                  {description: "Returns the correlation coefficient for non-NULL pairs in a group.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates#corry-x"},
	"count":                 {description: "Returns the number of non-NULL values in arg.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates#count"},
	"covar_pop":             {description: "Returns the population covariance of input values.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates#covar_popy-x"},
	"covar_samp":            {description: "Returns the sample covariance for non-NULL pairs in a group.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates#covar_sampy-x"},
	"cume_dist":             {description: "The number of partition rows preceding or peer with current row / total partition rows.", docsURL: "https://duckdb.org/docs/current/sql/functions/window_functions#cume_distorder-by-ordering"},
	"date_add":              {description: "Adds an interval to a date, time, or timestamp.", docsURL: "https://duckdb.org/docs/current/sql/functions/date#date_adddate-interval"},
	"date_diff":             {description: "The number of partition boundaries between the timestamps.", docsURL: "https://duckdb.org/docs/current/sql/functions/date#date_diffpart-startdate-enddate"},
	"date_part":             {description: "Get subfield (equivalent to extract).", docsURL: "https://duckdb.org/docs/current/sql/functions/date#date_partpart-date"},
	"date_sub":              {description: "The number of complete partitions between the timestamps.", docsURL: "https://duckdb.org/docs/current/sql/functions/date#date_subpart-startdate-enddate"},
	"date_trunc":            {description: "Truncate to specified precision.", docsURL: "https://duckdb.org/docs/current/sql/functions/date#date_truncpart-date"},
	"day":                   {description: "Extract the day component from a date or timestamp.", docsURL: "https://duckdb.org/docs/current/sql/functions/datepart#daydate"},
	"dayofweek":             {description: "Extract the dayofweek component from a date or timestamp.", docsURL: "https://duckdb.org/docs/current/sql/functions/datepart#dayofweekdate"},
	"dayofyear":             {description: "Extract the dayofyear component from a date or timestamp.", docsURL: "https://duckdb.org/docs/current/sql/functions/datepart#dayofyeardate"},
	"dense_rank":            {description: "The rank of the current row without gaps.", docsURL: "https://duckdb.org/docs/current/sql/functions/window_functions#dense_rank"},
	"ends_with":             {description: "Returns true if string ends with search_string.", docsURL: "https://duckdb.org/docs/current/sql/functions/text#suffixstring-search_string"},
	"entropy":               {description: "Returns the log-2 entropy of count input-values.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates#entropyx"},
	"epoch":                 {description: "Extract the epoch component from a temporal type.", docsURL: "https://duckdb.org/docs/current/sql/functions/timestamp#epochtimestamp"},
	"exp":                   {description: "Computes e to the power of x.", docsURL: "https://duckdb.org/docs/current/sql/functions/numeric#expx"},
	"first":                 {description: "Returns the first value (NULL or non-NULL) from arg. This function is affected by ordering.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates#firstarg"},
	"first_value":           {description: "The first value of expr in the frame. Can IGNORE or RESPECT NULLS.", docsURL: "https://duckdb.org/docs/current/sql/functions/window_functions#first_valueexpr-order-by-ordering-ignore-nulls"},
	"flatten":               {description: "Flattens a nested list by one level.", docsURL: "https://duckdb.org/docs/current/sql/functions/list#flattennested_list"},
	"floor":                 {description: "Rounds the number down.", docsURL: "https://duckdb.org/docs/current/sql/functions/numeric#floorx"},
	"format":                {description: "Formats a string using the fmt syntax.", docsURL: "https://duckdb.org/docs/current/sql/functions/text#formatformat-"},
	"greatest":              {description: "Returns the largest value. For strings lexicographical ordering is used. Note that lowercase characters are considered \"larger\" than uppercase characters and collations are not supported.", docsURL: "https://duckdb.org/docs/current/sql/functions/text#greatestarg1-"},
	"hash":                  {description: "Returns a UBIGINT with the hash of the value. Note that this is not a cryptographic hash.", docsURL: "https://duckdb.org/docs/current/sql/functions/utility#hashvalue"},
	"hour":                  {description: "Extract the hour component from a date or timestamp.", docsURL: "https://duckdb.org/docs/current/sql/functions/datepart#hourdate"},
	"if":                    {description: "Returns then if condition evaluates to true, otherwise returns the else value.", docsURL: "https://duckdb.org/docs/current/sql/functions/utility#ifa-b-c"},
	"ifnull":                {description: "Returns expr1 if it's not NULL, otherwise expr2.", docsURL: "https://duckdb.org/docs/current/sql/functions/utility#ifnullexpr-other"},
	"instr":                 {description: "Returns location of first occurrence of search_string in string, counting from 1. Returns 0 if no match found.", docsURL: "https://duckdb.org/docs/current/sql/functions/text#instrstring-search_string"},
	"isodow":                {description: "Extract the isodow component from a date or timestamp.", docsURL: "https://duckdb.org/docs/current/sql/functions/datepart#isodowdate"},
	"isoyear":               {description: "Extract the isoyear component from a date or timestamp.", docsURL: "https://duckdb.org/docs/current/sql/functions/datepart#isoyeardate"},
	"jaccard":               {description: "The Jaccard similarity between two strings. Characters of different cases (e.g., a and A) are considered different. Returns a number between 0 and 1.", docsURL: "https://duckdb.org/docs/current/sql/functions/text#jaccards1-s2"},
	"json_array":            {description: "Creates a JSON array from a list of arguments.", docsURL: "https://duckdb.org/docs/current/data/json/creating_json"},
	"json_array_length":     {description: "Returns the number of elements in a JSON array, or 0 if not an array.", docsURL: "https://duckdb.org/docs/current/data/json/json_functions#json-scalar-functions"},
	"json_contains":         {description: "Returns true if a JSON value contains the specified search value.", docsURL: "https://duckdb.org/docs/current/data/json/json_functions#json-scalar-functions"},
	"json_exists":           {description: "Returns true if a JSON path exists in the JSON value.", docsURL: "https://duckdb.org/docs/current/data/json/json_functions#json-extraction-functions"},
	"json_extract":          {description: "Extracts the JSON value at the given path or key.", docsURL: "https://duckdb.org/docs/current/data/json/json_functions#json-extraction-functions"},
	"json_extract_path":     {description: "Extracts the JSON value at the given path (alias for json_extract).", docsURL: "https://duckdb.org/docs/current/data/json/json_functions#json-extraction-functions"},
	"json_extract_string":   {description: "Extracts the value at the given path or key as a string.", docsURL: "https://duckdb.org/docs/current/data/json/json_functions#json-extraction-functions"},
	"json_group_array":      {description: "Aggregates values into a JSON array.", docsURL: "https://duckdb.org/docs/current/data/json/json_functions#json-aggregate-functions"},
	"json_group_object":     {description: "Aggregates key/value pairs into a JSON object.", docsURL: "https://duckdb.org/docs/current/data/json/json_functions#json-aggregate-functions"},
	"json_keys":             {description: "Returns the keys of a JSON object as a list.", docsURL: "https://duckdb.org/docs/current/data/json/json_functions#json-scalar-functions"},
	"json_merge_patch":      {description: "Merges two JSON documents using RFC 7396 JSON Merge Patch semantics.", docsURL: "https://duckdb.org/docs/current/data/json/creating_json"},
	"json_object":           {description: "Creates a JSON object from a list of key/value argument pairs.", docsURL: "https://duckdb.org/docs/current/data/json/creating_json"},
	"json_quote":            {description: "Converts a value to a quoted JSON value (alias for to_json).", docsURL: "https://duckdb.org/docs/current/data/json/creating_json"},
	"json_structure":        {description: "Returns the structure (a JSON schema-like description) of a JSON value.", docsURL: "https://duckdb.org/docs/current/data/json/json_functions#json-scalar-functions"},
	"json_type":             {description: "Returns the type of a JSON value as a string.", docsURL: "https://duckdb.org/docs/current/data/json/json_functions#json-scalar-functions"},
	"json_valid":            {description: "Returns true if the string is valid JSON.", docsURL: "https://duckdb.org/docs/current/data/json/json_functions#json-scalar-functions"},
	"json_value":            {description: "Extracts a JSON scalar value at the given path as a string.", docsURL: "https://duckdb.org/docs/current/data/json/json_functions#json-extraction-functions"},
	"kurtosis":              {description: "Returns the excess kurtosis (Fisher's definition) of all input values, with a bias correction according to the sample size.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates#kurtosisx"},
	"lag":                   {description: "The value of expr n rows after the current row, or the default. Can IGNORE or RESPECT NULLS.", docsURL: "https://duckdb.org/docs/current/sql/functions/window_functions#lagexpr-offset-default-order-by-ordering-ignore-nulls"},
	"last":                  {description: "Returns the last value of a column. This function is affected by ordering.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates#lastarg"},
	"last_day":              {description: "Returns the last day of the month.", docsURL: "https://duckdb.org/docs/current/sql/functions/date#last_daydate"},
	"last_value":            {description: "The last value of expr in the frame. Can IGNORE or RESPECT NULLS.", docsURL: "https://duckdb.org/docs/current/sql/functions/window_functions#last_valueexpr-order-by-ordering-ignore-nulls"},
	"lead":                  {description: "The value of expr n rows after the current row, or the default. Can IGNORE or RESPECT NULLS.", docsURL: "https://duckdb.org/docs/current/sql/functions/window_functions#leadexpr-offset-default-order-by-ordering-ignore-nulls"},
	"least":                 {description: "Returns the smallest value. For strings lexicographical ordering is used. Note that uppercase characters are considered \"smaller\" than lowercase characters, and collations are not supported.", docsURL: "https://duckdb.org/docs/current/sql/functions/text#leastarg1-"},
	"left":                  {description: "Extracts the left-most count characters.", docsURL: "https://duckdb.org/docs/current/sql/functions/text#leftstring-count"},
	"len":                   {description: "Number of characters in string.", docsURL: "https://duckdb.org/docs/current/sql/functions/text#lengthstring"},
	"length":                {description: "Number of characters in string.", docsURL: "https://duckdb.org/docs/current/sql/functions/text#lengthstring"},
	"levenshtein":           {description: "The minimum number of single-character edits (insertions, deletions or substitutions) required to change one string to the other. Characters of different cases (e.g., a and A) are considered different.", docsURL: "https://duckdb.org/docs/current/sql/functions/text#levenshteins1-s2"},
	"list":                  {description: "Returns a LIST containing all the values of a column.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates#listarg"},
	"list_aggregate":        {description: "Executes the aggregate function function_name on the elements of list.", docsURL: "https://duckdb.org/docs/current/sql/functions/list#list_aggregatelist-function_name-"},
	"list_concat":           {description: "Concatenates lists. NULL inputs are skipped. See also operator ||.", docsURL: "https://duckdb.org/docs/current/sql/functions/list#list_concatlist_1--list_n"},
	"list_contains":         {description: "Returns true if the list contains the element.", docsURL: "https://duckdb.org/docs/current/sql/functions/list#list_containslist-element"},
	"list_distinct":         {description: "Removes all duplicates and NULL values from a list. Does not preserve the original order.", docsURL: "https://duckdb.org/docs/current/sql/functions/list#list_distinctlist"},
	"list_has_all":          {description: "Returns true if all elements of list2 are in list1. NULLs are ignored.", docsURL: "https://duckdb.org/docs/current/sql/functions/list#list_has_alllist1-list2"},
	"list_has_any":          {description: "Returns true if the lists have any element in common. NULLs are ignored.", docsURL: "https://duckdb.org/docs/current/sql/functions/list#list_has_anylist1-list2"},
	"list_intersect":        {description: "Returns a list containing the distinct elements that are present in both list1 and list2.", docsURL: "https://duckdb.org/docs/current/sql/functions/list#list_intersectlist1-list2"},
	"list_position":         {description: "Returns the index of the element if the list contains the element. If the element is not found, it returns NULL.", docsURL: "https://duckdb.org/docs/current/sql/functions/list#list_positionlist-element"},
	"list_reverse_sort":     {description: "Sorts the elements of the list in reverse order.", docsURL: "https://duckdb.org/docs/current/sql/functions/list#list_reverse_sortlist-col1"},
	"list_slice":            {description: "Extracts a sublist or substring using slice conventions. Negative values are accepted.", docsURL: "https://duckdb.org/docs/current/sql/functions/list#list_slicelist-begin-end"},
	"list_sort":             {description: "Sorts the elements of the list.", docsURL: "https://duckdb.org/docs/current/sql/functions/list#list_sortlist-col1-col2"},
	"list_unique":           {description: "Counts the unique elements of a list.", docsURL: "https://duckdb.org/docs/current/sql/functions/list#list_uniquelist"},
	"list_value":            {description: "Creates a LIST containing the argument values.", docsURL: "https://duckdb.org/docs/current/sql/functions/list#list_valuearg-"},
	"ln":                    {description: "Computes the natural logarithm of x.", docsURL: "https://duckdb.org/docs/current/sql/functions/numeric#lnx"},
	"log":                   {description: "Computes the logarithm of x to base b. b may be omitted, in which case the default 10.", docsURL: "https://duckdb.org/docs/current/sql/functions/numeric#logx"},
	"log10":                 {description: "Computes the 10-log of x.", docsURL: "https://duckdb.org/docs/current/sql/functions/numeric#log10x"},
	"log2":                  {description: "Computes the 2-log of x.", docsURL: "https://duckdb.org/docs/current/sql/functions/numeric#log2x"},
	"lower":                 {description: "Converts string to lower case.", docsURL: "https://duckdb.org/docs/current/sql/functions/text#lowerstring"},
	"lpad":                  {description: "Pads the string with the character on the left until it has count characters. Truncates the string on the right if it has more than count characters.", docsURL: "https://duckdb.org/docs/current/sql/functions/text#lpadstring-count-character"},
	"ltrim":                 {description: "Removes any occurrences of any of the characters from the left side of the string. characters defaults to space.", docsURL: "https://duckdb.org/docs/current/sql/functions/text#ltrimstring-characters"},
	"make_date":             {description: "The date for the given parts.", docsURL: "https://duckdb.org/docs/current/sql/functions/date#make_dateyear-month-day"},
	"make_timestamp":        {description: "The timestamp for the given parts.", docsURL: "https://duckdb.org/docs/current/sql/functions/timestamp#make_timestampbigint-bigint-bigint-bigint-bigint-double"},
	"max":                   {description: "Returns the maximum value present in arg.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates#maxarg"},
	"md5":                   {description: "Returns the MD5 hash of the string as a VARCHAR.", docsURL: "https://duckdb.org/docs/current/sql/functions/utility#md5string"},
	"median":                {description: "Returns the middle value of the set. NULL values are ignored. For even value counts, interpolate-able types (numeric, date/time) return the average of the two middle values. Non-interpolate-able types (everything else) return the lower of the two middle values.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates#medianx"},
	"min":                   {description: "Returns the minimum value present in arg.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates#minarg"},
	"minute":                {description: "Extract the minute component from a date or timestamp.", docsURL: "https://duckdb.org/docs/current/sql/functions/datepart#minutedate"},
	"mod":                   {description: "The remainder of x divided by y.", docsURL: "https://duckdb.org/docs/current/sql/functions/numeric"},
	"mode":                  {description: "Returns the most frequent value for the values within x. NULL values are ignored.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates#modex"},
	"month":                 {description: "Extract the month component from a date or timestamp.", docsURL: "https://duckdb.org/docs/current/sql/functions/datepart#monthdate"},
	"now":                   {description: "Returns the current timestamp.", docsURL: "https://duckdb.org/docs/current/sql/functions/date"},
	"nth_value":             {description: "The nth value of expr in the frame. Can IGNORE or RESPECT NULLS.", docsURL: "https://duckdb.org/docs/current/sql/functions/window_functions#nth_valueexpr-nth-order-by-ordering-ignore-nulls"},
	"ntile":                 {description: "The row bucket in a window partition for a given bucket count.", docsURL: "https://duckdb.org/docs/current/sql/functions/window_functions#ntilenum_buckets-order-by-ordering"},
	"nvl":                   {description: "Returns expr1 if it's not NULL, otherwise expr2 (alias for ifnull).", docsURL: "https://duckdb.org/docs/current/sql/functions/utility#ifnullexpr-other"},
	"percent_rank":          {description: "The relative rank of the current row as a fraction.", docsURL: "https://duckdb.org/docs/current/sql/functions/window_functions#percent_rankorder-by-ordering"},
	"pi":                    {description: "Returns the value of pi.", docsURL: "https://duckdb.org/docs/current/sql/functions/numeric#pi"},
	"pow":                   {description: "Computes x to the power of y.", docsURL: "https://duckdb.org/docs/current/sql/functions/numeric#powx-y"},
	"power":                 {description: "Computes x to the power of y.", docsURL: "https://duckdb.org/docs/current/sql/functions/numeric#powerx-y"},
	"printf":                {description: "Formats a string using printf syntax.", docsURL: "https://duckdb.org/docs/current/sql/functions/text#printfformat-"},
	"product":               {description: "Calculates the product of all tuples in arg.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates#productarg"},
	"quantile_cont":         {description: "Returns the interpolated quantile number between 0 and 1 . If pos is a LIST of FLOATs, then the result is a LIST of the corresponding interpolated quantiles.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates#quantile_contx-pos"},
	"quantile_disc":         {description: "Returns the exact quantile number between 0 and 1 . If pos is a LIST of FLOATs, then the result is a LIST of the corresponding exact quantiles.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates#quantile_discx-pos"},
	"quarter":               {description: "Extract the quarter component from a date or timestamp.", docsURL: "https://duckdb.org/docs/current/sql/functions/datepart#quarterdate"},
	"rank":                  {description: "The rank of the current row with gaps.", docsURL: "https://duckdb.org/docs/current/sql/functions/window_functions#rankorder-by-ordering"},
	"regexp_extract":        {description: "If string contains the regex pattern, returns the capturing group specified by optional parameter group; otherwise, returns the empty string. The group must be a constant value. If no group is given, it defaults to 0. A set of optional regex options can be set.", docsURL: "https://duckdb.org/docs/current/sql/functions/regular_expressions#regexp_extractstring-pattern-group--0-options"},
	"regexp_full_match":     {description: "Returns true if the entire string matches the regex. A set of optional regex options can be set.", docsURL: "https://duckdb.org/docs/current/sql/functions/regular_expressions#regexp_full_matchstring-regex-options"},
	"regexp_matches":        {description: "Returns true if string contains the regex, false otherwise. A set of optional regex options can be set.", docsURL: "https://duckdb.org/docs/current/sql/functions/regular_expressions#regexp_matchesstring-pattern-options"},
	"regexp_replace":        {description: "If string contains the regex, replaces the matching part with replacement. A set of optional regex options can be set.", docsURL: "https://duckdb.org/docs/current/sql/functions/regular_expressions#regexp_replacestring-pattern-replacement-options"},
	"regexp_split_to_array": {description: "Splits the string along the regex. A set of optional regex options can be set.", docsURL: "https://duckdb.org/docs/current/sql/functions/regular_expressions#regexp_split_to_arraystring-regex-options"},
	"repeat":                {description: "Repeats the string count number of times.", docsURL: "https://duckdb.org/docs/current/sql/functions/text#repeatstring-count"},
	"replace":               {description: "Replaces any occurrences of the source with target in string.", docsURL: "https://duckdb.org/docs/current/sql/functions/text#replacestring-source-target"},
	"reverse":               {description: "Reverses the string.", docsURL: "https://duckdb.org/docs/current/sql/functions/text#reversestring"},
	"right":                 {description: "Extract the right-most count characters.", docsURL: "https://duckdb.org/docs/current/sql/functions/text#rightstring-count"},
	"round":                 {description: "Rounds x to s decimal places.", docsURL: "https://duckdb.org/docs/current/sql/functions/numeric#roundv-numeric-s-integer"},
	"row_number":            {description: "The row number in a window partition.", docsURL: "https://duckdb.org/docs/current/sql/functions/window_functions#row_numberorder-by-ordering"},
	"rpad":                  {description: "Pads the string with the character on the right until it has count characters. Truncates the string on the right if it has more than count characters.", docsURL: "https://duckdb.org/docs/current/sql/functions/text#rpadstring-count-character"},
	"rtrim":                 {description: "Removes any occurrences of any of the characters from the right side of the string. characters defaults to space.", docsURL: "https://duckdb.org/docs/current/sql/functions/text#rtrimstring-characters"},
	"second":                {description: "Extract the second component from a date or timestamp.", docsURL: "https://duckdb.org/docs/current/sql/functions/datepart#seconddate"},
	"sha256":                {description: "Returns a VARCHAR with the SHA-256 hash of the value.", docsURL: "https://duckdb.org/docs/current/sql/functions/utility#sha256string"},
	"sign":                  {description: "Returns the sign of x as -1, 0 or 1.", docsURL: "https://duckdb.org/docs/current/sql/functions/numeric#signx"},
	"skewness":              {description: "Returns the skewness of all input values.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates#skewnessx"},
	"split_part":            {description: "Splits the string along the separator and returns the data at the (1-based) index.", docsURL: "https://duckdb.org/docs/current/sql/functions/text#split_partstring-separator-index"},
	"sqrt":                  {description: "Returns the square root of x.", docsURL: "https://duckdb.org/docs/current/sql/functions/numeric#sqrtx"},
	"starts_with":           {description: "Returns true if string begins with search_string.", docsURL: "https://duckdb.org/docs/current/sql/functions/text#starts_withstring-search_string"},
	"stddev":                {description: "Returns the sample standard deviation.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates#stddev_sampx"},
	"stddev_pop":            {description: "Returns the population standard deviation.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates#stddev_popx"},
	"stddev_samp":           {description: "Returns the sample standard deviation.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates#stddev_sampx"},
	"strftime":              {description: "Converts a date to a string according to the format string.", docsURL: "https://duckdb.org/docs/current/sql/functions/date#strftimedate-format"},
	"string_agg":            {description: "Concatenates the column string values with an optional separator.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates#string_aggarg-sep"},
	"string_split":          {description: "Splits the string along the separator.", docsURL: "https://duckdb.org/docs/current/sql/functions/text#string_splitstring-separator"},
	"strip_accents":         {description: "Strips accents from string.", docsURL: "https://duckdb.org/docs/current/sql/functions/text#strip_accentsstring"},
	"strptime":              {description: "Converts the string text to timestamp according to the format string. Throws an error on failure. To return NULL on failure, use try_strptime.", docsURL: "https://duckdb.org/docs/current/sql/functions/timestamp#strptimetext-format-list"},
	"substr":                {description: "Extracts substring starting from character start up to the end of the string. If optional argument length is set, extracts a substring of length characters instead. Note that a start value of 1 refers to the first character of the string.", docsURL: "https://duckdb.org/docs/current/sql/functions/text#substringstring-start-length"},
	"sum":                   {description: "Calculates the sum value for all tuples in arg.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates#sumarg"},
	"timezone":              {description: "Extract the timezone component from a date or timestamp.", docsURL: "https://duckdb.org/docs/current/sql/functions/datepart#timezonedate"},
	"to_json":               {description: "Converts a value to JSON.", docsURL: "https://duckdb.org/docs/current/data/json/creating_json"},
	"to_timestamp":          {description: "Converts secs since epoch to a timestamp with time zone.", docsURL: "https://duckdb.org/docs/current/sql/functions/timestamp"},
	"translate":             {description: "Replaces each character in string that matches a character in the from set with the corresponding character in the to set. If from is longer than to, occurrences of the extra characters in from are deleted.", docsURL: "https://duckdb.org/docs/current/sql/functions/text#translatestring-from-to"},
	"trunc":                 {description: "Truncates the number.", docsURL: "https://duckdb.org/docs/current/sql/functions/numeric#truncx"},
	"typeof":                {description: "Returns the name of the data type of the result of the expression.", docsURL: "https://duckdb.org/docs/current/sql/functions/utility#typeofexpression"},
	"unicode":               {description: "Returns an INTEGER representing the unicode codepoint of the first character in the string.", docsURL: "https://duckdb.org/docs/current/sql/functions/text#unicodestring"},
	"unnest":                {description: "Unnests a list or struct by one level, turning a single row's list into one row per element.", docsURL: "https://duckdb.org/docs/current/sql/functions/list#unnestlist"},
	"upper":                 {description: "Converts string to upper case.", docsURL: "https://duckdb.org/docs/current/sql/functions/text#upperstring"},
	"var_pop":               {description: "Returns the population variance.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates#var_popx"},
	"var_samp":              {description: "Returns the sample variance of all input values.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates#var_sampx"},
	"variance":              {description: "Returns the sample variance of all input values.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates#var_sampx"},
	"week":                  {description: "Extract the week component from a date or timestamp.", docsURL: "https://duckdb.org/docs/current/sql/functions/datepart#weekdate"},
	"year":                  {description: "Extract the year component from a date or timestamp.", docsURL: "https://duckdb.org/docs/current/sql/functions/datepart#yeardate"}}
