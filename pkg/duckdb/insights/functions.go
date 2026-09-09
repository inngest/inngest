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
// a link to its official DuckDB documentation page, both surfaced by
// cmd/gen-insights-schema's JSON dump (embedded into the SQL editor UI) and
// by DESCRIBE-adjacent tooling. docsURL points at the function's category
// page (e.g. all list functions share one page), not a per-function
// anchor -- DuckDB's docs don't expose stable per-function anchors, and a
// fabricated one risks landing on the wrong section entirely.
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
	"ABS":                   {description: "Absolute value.", docsURL: "https://duckdb.org/docs/current/sql/functions/numeric"},
	"AGE":                   {description: "Subtract arguments, resulting in the time difference between the two timestamps.", docsURL: "https://duckdb.org/docs/current/sql/functions/date"},
	"ANY_VALUE":             {description: "Returns the first non-NULL value from arg. This function is affected by ordering.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates"},
	"APPROX_COUNT_DISTINCT": {description: "Computes the approximate count of distinct elements using HyperLogLog.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates"},
	"APPROX_QUANTILE":       {description: "Computes the approximate quantile using T-Digest.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates"},
	"ARG_MAX":               {description: "Finds the row with the maximum val. Calculates the non-NULL arg expression at that row.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates"},
	"ARG_MIN":               {description: "Finds the row with the minimum val. Calculates the non-NULL arg expression at that row.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates"},
	"ARRAY_AGG":             {description: "Returns a LIST containing all the values of a column.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates"},
	"ARRAY_AGGREGATE":       {description: "Executes the aggregate function function_name on the elements of list.", docsURL: "https://duckdb.org/docs/current/sql/functions/list"},
	"ARRAY_CONTAINS":        {description: "Returns true if the list contains the element.", docsURL: "https://duckdb.org/docs/current/sql/functions/list"},
	"ARRAY_LENGTH":          {description: "Returns the length of the list.", docsURL: "https://duckdb.org/docs/current/sql/functions/list"},
	"ARRAY_POSITION":        {description: "Returns the index of the element if the list contains the element. If the element is not found, it returns NULL.", docsURL: "https://duckdb.org/docs/current/sql/functions/list"},
	"ARRAY_TO_STRING":       {description: "Concatenates list/array elements using an optional delimiter.", docsURL: "https://duckdb.org/docs/current/sql/functions/list"},
	"ASCII":                 {description: "Returns an integer that represents the Unicode code point of the first character of the string.", docsURL: "https://duckdb.org/docs/current/sql/functions/text"},
	"AVG":                   {description: "Calculates the average value for all tuples in x.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates"},
	"BIT_AND":               {description: "Returns the bitwise AND of all bits in a given expression.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates"},
	"BIT_OR":                {description: "Returns the bitwise OR of all bits in a given expression.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates"},
	"BIT_XOR":               {description: "Returns the bitwise XOR of all bits in a given expression.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates"},
	"BOOL_AND":              {description: "Returns TRUE if every input value is TRUE, otherwise FALSE.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates"},
	"BOOL_OR":               {description: "Returns TRUE if any input value is TRUE, otherwise FALSE.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates"},
	"CBRT":                  {description: "Returns the cube root of x.", docsURL: "https://duckdb.org/docs/current/sql/functions/numeric"},
	"CEIL":                  {description: "Rounds the number up.", docsURL: "https://duckdb.org/docs/current/sql/functions/numeric"},
	"CEILING":               {description: "Rounds the number up.", docsURL: "https://duckdb.org/docs/current/sql/functions/numeric"},
	"CHR":                   {description: "Returns a character which is corresponding the ASCII code value or Unicode code point.", docsURL: "https://duckdb.org/docs/current/sql/functions/text"},
	"CONCAT":                {description: "Concatenates multiple strings or lists. NULL inputs are skipped. See also operator ||.", docsURL: "https://duckdb.org/docs/current/sql/functions/text"},
	"CONTAINS":              {description: "Returns true if search_string is found within string.", docsURL: "https://duckdb.org/docs/current/sql/functions/text"},
	"CORR":                  {description: "Returns the correlation coefficient for non-NULL pairs in a group.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates"},
	"COUNT":                 {description: "Returns the number of non-NULL values in arg.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates"},
	"COVAR_POP":             {description: "Returns the population covariance of input values.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates"},
	"COVAR_SAMP":            {description: "Returns the sample covariance for non-NULL pairs in a group.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates"},
	"CUME_DIST":             {description: "The number of partition rows preceding or peer with current row / total partition rows.", docsURL: "https://duckdb.org/docs/current/sql/functions/window_functions"},
	"DATE_ADD":              {description: "Adds an interval to a date, time, or timestamp.", docsURL: "https://duckdb.org/docs/current/sql/functions/date"},
	"DATE_DIFF":             {description: "The number of partition boundaries between the timestamps.", docsURL: "https://duckdb.org/docs/current/sql/functions/datepart"},
	"DATE_PART":             {description: "Get subfield (equivalent to extract).", docsURL: "https://duckdb.org/docs/current/sql/functions/datepart"},
	"DATE_SUB":              {description: "The number of complete partitions between the timestamps.", docsURL: "https://duckdb.org/docs/current/sql/functions/date"},
	"DATE_TRUNC":            {description: "Truncate to specified precision.", docsURL: "https://duckdb.org/docs/current/sql/functions/date"},
	"DAY":                   {description: "Extract the day component from a date or timestamp.", docsURL: "https://duckdb.org/docs/current/sql/functions/datepart"},
	"DAYOFWEEK":             {description: "Extract the dayofweek component from a date or timestamp.", docsURL: "https://duckdb.org/docs/current/sql/functions/datepart"},
	"DAYOFYEAR":             {description: "Extract the dayofyear component from a date or timestamp.", docsURL: "https://duckdb.org/docs/current/sql/functions/datepart"},
	"DENSE_RANK":            {description: "The rank of the current row without gaps.", docsURL: "https://duckdb.org/docs/current/sql/functions/window_functions"},
	"ENDS_WITH":             {description: "Returns true if string ends with search_string.", docsURL: "https://duckdb.org/docs/current/sql/functions/text"},
	"ENTROPY":               {description: "Returns the log-2 entropy of count input-values.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates"},
	"EPOCH":                 {description: "Extract the epoch component from a temporal type.", docsURL: "https://duckdb.org/docs/current/sql/functions/date"},
	"EXP":                   {description: "Computes e to the power of x.", docsURL: "https://duckdb.org/docs/current/sql/functions/numeric"},
	"FIRST":                 {description: "Returns the first value (NULL or non-NULL) from arg. This function is affected by ordering.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates"},
	"FIRST_VALUE":           {description: "The first value of expr in the frame. Can IGNORE or RESPECT NULLS.", docsURL: "https://duckdb.org/docs/current/sql/functions/window_functions"},
	"FLATTEN":               {description: "Flattens a nested list by one level.", docsURL: "https://duckdb.org/docs/current/sql/functions/list"},
	"FLOOR":                 {description: "Rounds the number down.", docsURL: "https://duckdb.org/docs/current/sql/functions/numeric"},
	"FORMAT":                {description: "Formats a string using the fmt syntax.", docsURL: "https://duckdb.org/docs/current/sql/functions/text"},
	"GREATEST":              {description: "Returns the largest value. For strings lexicographical ordering is used. Note that lowercase characters are considered \"larger\" than uppercase characters and collations are not supported.", docsURL: "https://duckdb.org/docs/current/sql/functions/text"},
	"HASH":                  {description: "Returns a UBIGINT with the hash of the value. Note that this is not a cryptographic hash.", docsURL: "https://duckdb.org/docs/current/sql/functions/utility"},
	"HOUR":                  {description: "Extract the hour component from a date or timestamp.", docsURL: "https://duckdb.org/docs/current/sql/functions/datepart"},
	"IF":                    {description: "Returns then if condition evaluates to true, otherwise returns the else value.", docsURL: "https://duckdb.org/docs/current/sql/functions/utility"},
	"IFNULL":                {description: "Returns expr1 if it's not NULL, otherwise expr2.", docsURL: "https://duckdb.org/docs/current/sql/functions/utility"},
	"INSTR":                 {description: "Returns location of first occurrence of search_string in string, counting from 1. Returns 0 if no match found.", docsURL: "https://duckdb.org/docs/current/sql/functions/text"},
	"ISODOW":                {description: "Extract the isodow component from a date or timestamp.", docsURL: "https://duckdb.org/docs/current/sql/functions/datepart"},
	"ISOYEAR":               {description: "Extract the isoyear component from a date or timestamp.", docsURL: "https://duckdb.org/docs/current/sql/functions/datepart"},
	"JACCARD":               {description: "The Jaccard similarity between two strings. Characters of different cases (e.g., a and A) are considered different. Returns a number between 0 and 1.", docsURL: "https://duckdb.org/docs/current/sql/functions/text"},
	"JSON_ARRAY":            {description: "Creates a JSON array from a list of arguments.", docsURL: "https://duckdb.org/docs/current/data/json/json_functions"},
	"JSON_ARRAY_LENGTH":     {description: "Returns the number of elements in a JSON array, or 0 if not an array.", docsURL: "https://duckdb.org/docs/current/data/json/json_functions"},
	"JSON_CONTAINS":         {description: "Returns true if a JSON value contains the specified search value.", docsURL: "https://duckdb.org/docs/current/data/json/json_functions"},
	"JSON_EXISTS":           {description: "Returns true if a JSON path exists in the JSON value.", docsURL: "https://duckdb.org/docs/current/data/json/json_functions"},
	"JSON_EXTRACT":          {description: "Extracts the JSON value at the given path or key.", docsURL: "https://duckdb.org/docs/current/data/json/json_functions"},
	"JSON_EXTRACT_PATH":     {description: "Extracts the JSON value at the given path (alias for json_extract).", docsURL: "https://duckdb.org/docs/current/data/json/json_functions"},
	"JSON_EXTRACT_STRING":   {description: "Extracts the value at the given path or key as a string.", docsURL: "https://duckdb.org/docs/current/data/json/json_functions"},
	"JSON_GROUP_ARRAY":      {description: "Aggregates values into a JSON array.", docsURL: "https://duckdb.org/docs/current/data/json/json_functions"},
	"JSON_GROUP_OBJECT":     {description: "Aggregates key/value pairs into a JSON object.", docsURL: "https://duckdb.org/docs/current/data/json/json_functions"},
	"JSON_KEYS":             {description: "Returns the keys of a JSON object as a list.", docsURL: "https://duckdb.org/docs/current/data/json/json_functions"},
	"JSON_MERGE_PATCH":      {description: "Merges two JSON documents using RFC 7396 JSON Merge Patch semantics.", docsURL: "https://duckdb.org/docs/current/data/json/json_functions"},
	"JSON_OBJECT":           {description: "Creates a JSON object from a list of key/value argument pairs.", docsURL: "https://duckdb.org/docs/current/data/json/json_functions"},
	"JSON_QUOTE":            {description: "Converts a value to a quoted JSON value (alias for to_json).", docsURL: "https://duckdb.org/docs/current/data/json/json_functions"},
	"JSON_STRUCTURE":        {description: "Returns the structure (a JSON schema-like description) of a JSON value.", docsURL: "https://duckdb.org/docs/current/data/json/json_functions"},
	"JSON_TYPE":             {description: "Returns the type of a JSON value as a string.", docsURL: "https://duckdb.org/docs/current/data/json/json_functions"},
	"JSON_VALID":            {description: "Returns true if the string is valid JSON.", docsURL: "https://duckdb.org/docs/current/data/json/json_functions"},
	"JSON_VALUE":            {description: "Extracts a JSON scalar value at the given path as a string.", docsURL: "https://duckdb.org/docs/current/data/json/json_functions"},
	"KURTOSIS":              {description: "Returns the excess kurtosis (Fisher's definition) of all input values, with a bias correction according to the sample size.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates"},
	"LAG":                   {description: "The value of expr n rows after the current row, or the default. Can IGNORE or RESPECT NULLS.", docsURL: "https://duckdb.org/docs/current/sql/functions/window_functions"},
	"LAST":                  {description: "Returns the last value of a column. This function is affected by ordering.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates"},
	"LAST_DAY":              {description: "Returns the last day of the month.", docsURL: "https://duckdb.org/docs/current/sql/functions/date"},
	"LAST_VALUE":            {description: "The last value of expr in the frame. Can IGNORE or RESPECT NULLS.", docsURL: "https://duckdb.org/docs/current/sql/functions/window_functions"},
	"LEAD":                  {description: "The value of expr n rows after the current row, or the default. Can IGNORE or RESPECT NULLS.", docsURL: "https://duckdb.org/docs/current/sql/functions/window_functions"},
	"LEAST":                 {description: "Returns the smallest value. For strings lexicographical ordering is used. Note that uppercase characters are considered \"smaller\" than lowercase characters, and collations are not supported.", docsURL: "https://duckdb.org/docs/current/sql/functions/text"},
	"LEFT":                  {description: "Extracts the left-most count characters.", docsURL: "https://duckdb.org/docs/current/sql/functions/text"},
	"LEN":                   {description: "Number of characters in string.", docsURL: "https://duckdb.org/docs/current/sql/functions/text"},
	"LENGTH":                {description: "Number of characters in string.", docsURL: "https://duckdb.org/docs/current/sql/functions/text"},
	"LEVENSHTEIN":           {description: "The minimum number of single-character edits (insertions, deletions or substitutions) required to change one string to the other. Characters of different cases (e.g., a and A) are considered different.", docsURL: "https://duckdb.org/docs/current/sql/functions/text"},
	"LIST":                  {description: "Returns a LIST containing all the values of a column.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates"},
	"LIST_AGGREGATE":        {description: "Executes the aggregate function function_name on the elements of list.", docsURL: "https://duckdb.org/docs/current/sql/functions/list"},
	"LIST_CONCAT":           {description: "Concatenates lists. NULL inputs are skipped. See also operator ||.", docsURL: "https://duckdb.org/docs/current/sql/functions/list"},
	"LIST_CONTAINS":         {description: "Returns true if the list contains the element.", docsURL: "https://duckdb.org/docs/current/sql/functions/list"},
	"LIST_DISTINCT":         {description: "Removes all duplicates and NULL values from a list. Does not preserve the original order.", docsURL: "https://duckdb.org/docs/current/sql/functions/list"},
	"LIST_HAS_ALL":          {description: "Returns true if all elements of list2 are in list1. NULLs are ignored.", docsURL: "https://duckdb.org/docs/current/sql/functions/list"},
	"LIST_HAS_ANY":          {description: "Returns true if the lists have any element in common. NULLs are ignored.", docsURL: "https://duckdb.org/docs/current/sql/functions/list"},
	"LIST_INTERSECT":        {description: "Returns a list containing the distinct elements that are present in both list1 and list2.", docsURL: "https://duckdb.org/docs/current/sql/functions/list"},
	"LIST_POSITION":         {description: "Returns the index of the element if the list contains the element. If the element is not found, it returns NULL.", docsURL: "https://duckdb.org/docs/current/sql/functions/list"},
	"LIST_REVERSE_SORT":     {description: "Sorts the elements of the list in reverse order.", docsURL: "https://duckdb.org/docs/current/sql/functions/list"},
	"LIST_SLICE":            {description: "Extracts a sublist or substring using slice conventions. Negative values are accepted.", docsURL: "https://duckdb.org/docs/current/sql/functions/list"},
	"LIST_SORT":             {description: "Sorts the elements of the list.", docsURL: "https://duckdb.org/docs/current/sql/functions/list"},
	"LIST_UNIQUE":           {description: "Counts the unique elements of a list.", docsURL: "https://duckdb.org/docs/current/sql/functions/list"},
	"LIST_VALUE":            {description: "Creates a LIST containing the argument values.", docsURL: "https://duckdb.org/docs/current/sql/functions/list"},
	"LN":                    {description: "Computes the natural logarithm of x.", docsURL: "https://duckdb.org/docs/current/sql/functions/numeric"},
	"LOG":                   {description: "Computes the logarithm of x to base b. b may be omitted, in which case the default 10.", docsURL: "https://duckdb.org/docs/current/sql/functions/numeric"},
	"LOG10":                 {description: "Computes the 10-log of x.", docsURL: "https://duckdb.org/docs/current/sql/functions/numeric"},
	"LOG2":                  {description: "Computes the 2-log of x.", docsURL: "https://duckdb.org/docs/current/sql/functions/numeric"},
	"LOWER":                 {description: "Converts string to lower case.", docsURL: "https://duckdb.org/docs/current/sql/functions/text"},
	"LPAD":                  {description: "Pads the string with the character on the left until it has count characters. Truncates the string on the right if it has more than count characters.", docsURL: "https://duckdb.org/docs/current/sql/functions/text"},
	"LTRIM":                 {description: "Removes any occurrences of any of the characters from the left side of the string. characters defaults to space.", docsURL: "https://duckdb.org/docs/current/sql/functions/text"},
	"MAKE_DATE":             {description: "The date for the given parts.", docsURL: "https://duckdb.org/docs/current/sql/functions/date"},
	"MAKE_TIMESTAMP":        {description: "The timestamp for the given parts.", docsURL: "https://duckdb.org/docs/current/sql/functions/date"},
	"MAX":                   {description: "Returns the maximum value present in arg.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates"},
	"MD5":                   {description: "Returns the MD5 hash of the string as a VARCHAR.", docsURL: "https://duckdb.org/docs/current/sql/functions/utility"},
	"MEDIAN":                {description: "Returns the middle value of the set. NULL values are ignored. For even value counts, interpolate-able types (numeric, date/time) return the average of the two middle values. Non-interpolate-able types (everything else) return the lower of the two middle values.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates"},
	"MIN":                   {description: "Returns the minimum value present in arg.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates"},
	"MINUTE":                {description: "Extract the minute component from a date or timestamp.", docsURL: "https://duckdb.org/docs/current/sql/functions/datepart"},
	"MOD":                   {description: "The remainder of x divided by y.", docsURL: "https://duckdb.org/docs/current/sql/functions/numeric"},
	"MODE":                  {description: "Returns the most frequent value for the values within x. NULL values are ignored.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates"},
	"MONTH":                 {description: "Extract the month component from a date or timestamp.", docsURL: "https://duckdb.org/docs/current/sql/functions/datepart"},
	"NOW":                   {description: "Returns the current timestamp.", docsURL: "https://duckdb.org/docs/current/sql/functions/date"},
	"NTH_VALUE":             {description: "The nth value of expr in the frame. Can IGNORE or RESPECT NULLS.", docsURL: "https://duckdb.org/docs/current/sql/functions/window_functions"},
	"NTILE":                 {description: "The row bucket in a window partition for a given bucket count.", docsURL: "https://duckdb.org/docs/current/sql/functions/window_functions"},
	"NVL":                   {description: "Returns expr1 if it's not NULL, otherwise expr2 (alias for ifnull).", docsURL: "https://duckdb.org/docs/current/sql/functions/utility"},
	"PERCENT_RANK":          {description: "The relative rank of the current row as a fraction.", docsURL: "https://duckdb.org/docs/current/sql/functions/window_functions"},
	"PI":                    {description: "Returns the value of pi.", docsURL: "https://duckdb.org/docs/current/sql/functions/numeric"},
	"POW":                   {description: "Computes x to the power of y.", docsURL: "https://duckdb.org/docs/current/sql/functions/numeric"},
	"POWER":                 {description: "Computes x to the power of y.", docsURL: "https://duckdb.org/docs/current/sql/functions/numeric"},
	"PRINTF":                {description: "Formats a string using printf syntax.", docsURL: "https://duckdb.org/docs/current/sql/functions/text"},
	"PRODUCT":               {description: "Calculates the product of all tuples in arg.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates"},
	"QUANTILE_CONT":         {description: "Returns the interpolated quantile number between 0 and 1 . If pos is a LIST of FLOATs, then the result is a LIST of the corresponding interpolated quantiles.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates"},
	"QUANTILE_DISC":         {description: "Returns the exact quantile number between 0 and 1 . If pos is a LIST of FLOATs, then the result is a LIST of the corresponding exact quantiles.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates"},
	"QUARTER":               {description: "Extract the quarter component from a date or timestamp.", docsURL: "https://duckdb.org/docs/current/sql/functions/datepart"},
	"RANK":                  {description: "The rank of the current row with gaps.", docsURL: "https://duckdb.org/docs/current/sql/functions/window_functions"},
	"REGEXP_EXTRACT":        {description: "If string contains the regex pattern, returns the capturing group specified by optional parameter group; otherwise, returns the empty string. The group must be a constant value. If no group is given, it defaults to 0. A set of optional regex options can be set.", docsURL: "https://duckdb.org/docs/current/sql/functions/regular_expressions"},
	"REGEXP_FULL_MATCH":     {description: "Returns true if the entire string matches the regex. A set of optional regex options can be set.", docsURL: "https://duckdb.org/docs/current/sql/functions/regular_expressions"},
	"REGEXP_MATCHES":        {description: "Returns true if string contains the regex, false otherwise. A set of optional regex options can be set.", docsURL: "https://duckdb.org/docs/current/sql/functions/regular_expressions"},
	"REGEXP_REPLACE":        {description: "If string contains the regex, replaces the matching part with replacement. A set of optional regex options can be set.", docsURL: "https://duckdb.org/docs/current/sql/functions/regular_expressions"},
	"REGEXP_SPLIT_TO_ARRAY": {description: "Splits the string along the regex. A set of optional regex options can be set.", docsURL: "https://duckdb.org/docs/current/sql/functions/regular_expressions"},
	"REPEAT":                {description: "Repeats the string count number of times.", docsURL: "https://duckdb.org/docs/current/sql/functions/text"},
	"REPLACE":               {description: "Replaces any occurrences of the source with target in string.", docsURL: "https://duckdb.org/docs/current/sql/functions/text"},
	"REVERSE":               {description: "Reverses the string.", docsURL: "https://duckdb.org/docs/current/sql/functions/text"},
	"RIGHT":                 {description: "Extract the right-most count characters.", docsURL: "https://duckdb.org/docs/current/sql/functions/text"},
	"ROUND":                 {description: "Rounds x to s decimal places.", docsURL: "https://duckdb.org/docs/current/sql/functions/numeric"},
	"ROW_NUMBER":            {description: "The row number in a window partition.", docsURL: "https://duckdb.org/docs/current/sql/functions/window_functions"},
	"RPAD":                  {description: "Pads the string with the character on the right until it has count characters. Truncates the string on the right if it has more than count characters.", docsURL: "https://duckdb.org/docs/current/sql/functions/text"},
	"RTRIM":                 {description: "Removes any occurrences of any of the characters from the right side of the string. characters defaults to space.", docsURL: "https://duckdb.org/docs/current/sql/functions/text"},
	"SECOND":                {description: "Extract the second component from a date or timestamp.", docsURL: "https://duckdb.org/docs/current/sql/functions/datepart"},
	"SHA256":                {description: "Returns a VARCHAR with the SHA-256 hash of the value.", docsURL: "https://duckdb.org/docs/current/sql/functions/utility"},
	"SIGN":                  {description: "Returns the sign of x as -1, 0 or 1.", docsURL: "https://duckdb.org/docs/current/sql/functions/numeric"},
	"SKEWNESS":              {description: "Returns the skewness of all input values.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates"},
	"SPLIT_PART":            {description: "Splits the string along the separator and returns the data at the (1-based) index.", docsURL: "https://duckdb.org/docs/current/sql/functions/text"},
	"SQRT":                  {description: "Returns the square root of x.", docsURL: "https://duckdb.org/docs/current/sql/functions/numeric"},
	"STARTS_WITH":           {description: "Returns true if string begins with search_string.", docsURL: "https://duckdb.org/docs/current/sql/functions/text"},
	"STDDEV":                {description: "Returns the sample standard deviation.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates"},
	"STDDEV_POP":            {description: "Returns the population standard deviation.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates"},
	"STDDEV_SAMP":           {description: "Returns the sample standard deviation.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates"},
	"STRFTIME":              {description: "Converts a date to a string according to the format string.", docsURL: "https://duckdb.org/docs/current/sql/functions/date"},
	"STRING_AGG":            {description: "Concatenates the column string values with an optional separator.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates"},
	"STRING_SPLIT":          {description: "Splits the string along the separator.", docsURL: "https://duckdb.org/docs/current/sql/functions/text"},
	"STRIP_ACCENTS":         {description: "Strips accents from string.", docsURL: "https://duckdb.org/docs/current/sql/functions/text"},
	"STRPTIME":              {description: "Converts the string text to timestamp according to the format string. Throws an error on failure. To return NULL on failure, use try_strptime.", docsURL: "https://duckdb.org/docs/current/sql/functions/date"},
	"SUBSTR":                {description: "Extracts substring starting from character start up to the end of the string. If optional argument length is set, extracts a substring of length characters instead. Note that a start value of 1 refers to the first character of the string.", docsURL: "https://duckdb.org/docs/current/sql/functions/text"},
	"SUM":                   {description: "Calculates the sum value for all tuples in arg.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates"},
	"TIMEZONE":              {description: "Extract the timezone component from a date or timestamp.", docsURL: "https://duckdb.org/docs/current/sql/functions/date"},
	"TO_JSON":               {description: "Converts a value to JSON.", docsURL: "https://duckdb.org/docs/current/data/json/json_functions"},
	"TO_TIMESTAMP":          {description: "Converts secs since epoch to a timestamp with time zone.", docsURL: "https://duckdb.org/docs/current/sql/functions/date"},
	"TRANSLATE":             {description: "Replaces each character in string that matches a character in the from set with the corresponding character in the to set. If from is longer than to, occurrences of the extra characters in from are deleted.", docsURL: "https://duckdb.org/docs/current/sql/functions/text"},
	"TRUNC":                 {description: "Truncates the number.", docsURL: "https://duckdb.org/docs/current/sql/functions/numeric"},
	"TYPEOF":                {description: "Returns the name of the data type of the result of the expression.", docsURL: "https://duckdb.org/docs/current/sql/functions/utility"},
	"UNICODE":               {description: "Returns an INTEGER representing the unicode codepoint of the first character in the string.", docsURL: "https://duckdb.org/docs/current/sql/functions/text"},
	"UNNEST":                {description: "Unnests a list or struct by one level, turning a single row's list into one row per element.", docsURL: "https://duckdb.org/docs/current/sql/functions/nested"},
	"UPPER":                 {description: "Converts string to upper case.", docsURL: "https://duckdb.org/docs/current/sql/functions/text"},
	"VAR_POP":               {description: "Returns the population variance.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates"},
	"VAR_SAMP":              {description: "Returns the sample variance of all input values.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates"},
	"VARIANCE":              {description: "Returns the sample variance of all input values.", docsURL: "https://duckdb.org/docs/current/sql/functions/aggregates"},
	"WEEK":                  {description: "Extract the week component from a date or timestamp.", docsURL: "https://duckdb.org/docs/current/sql/functions/datepart"},
	"YEAR":                  {description: "Extract the year component from a date or timestamp.", docsURL: "https://duckdb.org/docs/current/sql/functions/datepart"}}
