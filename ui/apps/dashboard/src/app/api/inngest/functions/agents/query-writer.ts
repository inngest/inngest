import { createAgent, createTool, openai, type AnyZodType } from '@inngest/agent-kit';
import { z } from 'zod';

import type { GenerateSqlResult, InsightsAgentState as InsightsState } from './types';

const queryGrammar = `
  QueryAST = "SELECT" SelectClause "FROM" From ("WHERE" Expression)? ("GROUP" "BY" GroupBy)? ("ORDER" "BY" OrderBy)? ("LIMIT" <number>)? ("OFFSET" <number>)? ";"? .
  SelectClause = "*" | (AliasedColumnExpression ("," AliasedColumnExpression)*) .
  AliasedColumnExpression = (FunctionCall | ColumnReference) ("AS" <ident>)? .
  FunctionCall = <ident> "(" (FunctionArgument ("," FunctionArgument)*)? ")" .
  FunctionArgument = FunctionCall | Operand .
  Operand = Summand ("|" "|" Summand)* .
  Summand = Factor (("+" | "-") Factor)? .
  Factor = Term (("*" | "/" | "%") Term)? .
  Term = ValueType | FunctionCall | ColumnReference .
  ValueType = ("*" | <number> | <string> | ("TRUE" | "FALSE") | "NULL") .
  ColumnReference = (<ident> ".")? <ident> .
  From = TableExpression .
  TableExpression = <ident> ("AS" <ident>)? .
  Expression = OrCondition ("OR" OrCondition)* .
  OrCondition = Condition ("AND" Condition)* .
  Condition = ConditionOperand .
  ConditionOperand = Operand ConditionRHS? .
  ConditionRHS = Compare .
  Compare = ("<>" | "<=" | ">=" | "=" | "<" | ">" | "!=") Operand .
  GroupBy = Expression ("," Expression)* .
  OrderBy = OrderExpression ("," OrderExpression)* .
  OrderExpression = Expression ("ASC" | "DESC")? .
`;

const queryRules = `
Inngest Insights allows you to query and analyze your event data using SQL directly within the Inngest platform. Every event sent to Inngest contains valuable information, and Insights gives you the power to extract meaningful patterns and analytics from that data.

Overview
Insights provides an in-app SQL editor and query interface where you can:

Query event data using familiar SQL syntax
Save and reuse common queries
Analyze patterns in your event streams
Extract business intelligence from your workflows
Currently, you can query events only. Support for querying function runs will be added in future releases.

Getting Started
Available Columns
When querying events, you have access to the following columns:

Column	Type	Description
id	String	Unique identifier for the event
name	String	The name/type of the event
data	JSON	The event payload data - users can send any JSON structure here
ts	DateTime	Timestamp when the event occurred
v	String	Event format version
For more details on the event format, see the Inngest Event Format documentation.

Data Retention
You can query events from up to 30 days in the past. Older events are not available through Insights.

Result Limits
Current page limit: 1000 rows
Future updates will support larger result sets through async data exports
Pagination support is planned for future releases
SQL Support
Insights is built on ClickHouse, which provides powerful SQL capabilities with some differences from traditional SQL databases.

Supported Functions
Arithmetic Functions
Basic mathematical operations and calculations. View ClickHouse arithmetic functions documentation

String Functions
String manipulation and search capabilities.

String search functions
String manipulation functions
JSON Functions
Essential for working with data payloads. View ClickHouse JSON functions documentation

Date/Time Functions
For analyzing event timing and patterns. View ClickHouse date/time functions documentation

Other Supported Function Categories
Logical functions
Rounding functions
Type conversion functions
Functions for nulls
ULID functions
Aggregate Functions
The following aggregate functions are supported:

Function	Description
ARRAY_AGG()	Aggregates values into an array*
AVG()	Calculates average
COUNT()	Counts rows
MAX()	Finds maximum value
MIN()	Finds minimum value
STDDEV_POP()	Population standard deviation
STDDEV_SAMP()	Sample standard deviation
SUM()	Calculates sum
VAR_POP()	Population variance
VAR_SAMP()	Sample variance
median()	Finds median value
*Note on ARRAY_AGG: Due to a current serialization bug, you need to convert arrays to strings using toString(ARRAY_AGG(column_name)).

View complete ClickHouse aggregate functions documentation

SQL Syntax Limitations
Some SQL features are not yet supported but are planned for future releases:

CTEs (Common Table Expressions) using WITH
IS operator
NOT operator
Working with Event Data
Common Schema vs Event-Specific Schema
Common Schema: These columns are available for every user and every event:

id
name
ts
v
data
Event-Specific Schema: Within data, users can send any JSON they want, so the structure and available fields will be specific to their payloads. You can use ClickHouse's JSON functions to extract and query specific fields within your event data.

Example Queries
Basic Event Filtering

Copy
Copied
SELECT count(*)
FROM events
WHERE name = 'inngest/function.failed'
AND simpleJSONExtractString(data, 'function_id') = 'generate-report'
AND ts > toUnixTimestamp(addHours(now(), -1)) * 1000;
Extracting JSON Data and Aggregating

Copy
Copied
SELECT simpleJSONExtractString(data, 'user_id') as user_id, count(*) 
FROM events
WHERE name = 'order.created'
GROUP BY user_id
ORDER BY count(*) DESC
LIMIT 10;
Saved Queries
You can save frequently used queries for quick access. Currently, saved queries are stored in your browser's local storage.

Note: Saved queries are only available on the device and browser where you created them. Cloud synchronization of saved queries is planned for a future release.

Common Errors and Troubleshooting
Array Serialization Issues
When using ARRAY_AGG(), you must convert the result to a string:

Correct:


Copy
Copied
SELECT toString(ARRAY_AGG(id)) as ids FROM events
Incorrect:


Copy
Copied
SELECT ARRAY_AGG(id) as ids FROM events  -- This will cause serialization errors
Roadmap
Coming Soon
Query support for function runs
received_at column for tracking event receipt time
Pagination for large result sets
Async data exports for results larger than 1000 rows
Future Enhancements
Support for CTEs (Common Table Expressions)
IS and NOT operators
Cloud synchronization for saved queries
Advanced visualization capabilities

`;

const GenerateSqlParams = z.object({
  sql: z
    .string()
    .min(1)
    .describe('A single valid SELECT statement. Do not include DDL/DML or multiple statements.'),
  title: z.string().min(1).describe('Short 20-30 character title for this query'),
  reasoning: z
    .string()
    .min(1)
    .describe('Brief 1-2 sentence explanation of how this query addresses the request'),
});

export const generateSqlTool = createTool({
  name: 'generate_sql',
  description:
    'Provide the final SQL SELECT statement for ClickHouse based on the selected events and schemas.',
  parameters: GenerateSqlParams as unknown as AnyZodType, // (ted): need to update to latest version of zod + agent-kit
  handler: ({ sql, title, reasoning }: z.infer<typeof GenerateSqlParams>) => {
    return {
      sql,
      title,
      reasoning,
    } as GenerateSqlResult;
  },
});

export const queryWriterAgent = createAgent<InsightsState>({
  name: 'Insights Query Writer',
  description: 'Generates a safe, read-only SQL SELECT statement for ClickHouse.',
  system: async ({ network }) => {
    const selected = network?.state.data.selectedEvents?.map((e) => e.event_name) ?? [];
    return [
      'You write ClickHouse-compatible SQL for analytics.',
      `You MUST follow these rules ${queryRules}`,
      `You MUST follow this grammar ${queryGrammar}`,
      selected.length
        ? `Target the following events if relevant: ${selected.join(', ')}`
        : 'If events were selected earlier, incorporate them appropriately.',
      '',
      'When ready, call the generate_sql tool with the final SQL and a short 20-30 character title.',
      'Few rules to be aware of AT ALL TIMES:',
      '- Do NOT under any circumstances prefix table names or column names with "events_"',
    ].join('\n');
  },
  model: openai({ model: 'gpt-4.1-2025-04-14' }),
  tools: [generateSqlTool],
  tool_choice: 'generate_sql',
});
