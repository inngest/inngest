import { createAgent, createTool, openai, type Network } from '@inngest/agent-kit';
import { z } from 'zod';

import type { InsightsAgentState as InsightsState } from './event-matcher';
import type { GenerateSqlInput, GenerateSqlResult } from './types';

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
event_id	String	Unique identifier for the event
event_name	String	The name/type of the event
event_data	JSON	The event payload data - users can send any JSON structure here
event_ts	DateTime	Timestamp when the event occurred
event_v	String	Event format version
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
Essential for working with event_data payloads. View ClickHouse JSON functions documentation

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

event_id
event_name
event_ts
event_v
event_data
Event-Specific Schema: Within event_data, users can send any JSON they want, so the structure and available fields will be specific to their payloads. You can use ClickHouse's JSON functions to extract and query specific fields within your event data.

Example Queries
Basic Event Filtering

Copy
Copied
SELECT count(*)
FROM events
WHERE event_name = 'inngest/function.failed'
AND simpleJSONExtractString(event_data, 'function_id') = 'generate-report'
AND event_ts > toUnixTimestamp(addHours(now(), -1)) * 1000;
Extracting JSON Data and Aggregating

Copy
Copied
SELECT simpleJSONExtractString(event_data, 'user_id') as user_id, count(*) 
FROM events
WHERE event_name = 'order.created'
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
SELECT toString(ARRAY_AGG(event_id)) as event_ids FROM events
Incorrect:


Copy
Copied
SELECT ARRAY_AGG(event_id) as event_ids FROM events  -- This will cause serialization errors
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

function sanitizeSql(text: string): string {
  const sql = String(text || '').trim();
  // Lightweight guardrail: reject clearly unsafe statements
  const lower = sql.replace(/\s+/g, ' ').toLowerCase();
  const forbidden = [
    'insert ',
    'update ',
    'delete ',
    'drop ',
    'alter ',
    'create ',
    'grant ',
    'revoke ',
    'truncate ',
  ];
  for (const kw of forbidden) {
    if (lower.includes(kw)) {
      throw new Error('Only read-only SELECT queries are allowed');
    }
  }
  if (!/^select\s/i.test(sql)) {
    throw new Error('SQL must start with SELECT');
  }
  return sql;
}

const generateSqlTool = createTool({
  name: 'generate_sql',
  description:
    'Provide the final SQL SELECT statement for ClickHouse based on the selected events and schemas.',
  parameters: z.object({
    sql: z
      .string()
      .min(1)
      .describe('A single valid SELECT statement. Do not include DDL/DML or multiple statements.'),
    title: z.string().min(1).describe('Short 20-30 character title for this query'),
    reasoning: z
      .string()
      .min(1)
      .describe('Brief 1-2 sentence explanation of how this query addresses the request'),
  }) as any, // TODO: zod version mismatch is causing a type error here; need to align zod versions
  handler: ({ sql: rawSql, title, reasoning }: GenerateSqlInput, ctx: any): GenerateSqlResult => {
    const network = ctx?.network as Network<InsightsState> | undefined;
    if (!network) {
      throw new Error('Agent network context is required');
    }
    const raw = String(rawSql);
    const sql = sanitizeSql(raw);
    network.state.data.sql = sql;

    const result: GenerateSqlResult = {
      sql: sql,
      title,
      reasoning,
    };
    return result;
  },
});

export const queryWriterAgent = createAgent<InsightsState>({
  name: 'Insights Query Writer',
  description: 'Generates a safe, read-only SQL SELECT statement for ClickHouse.',
  system: async ({ network }): Promise<string> => {
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
    ].join('\n');
  },
  model: openai({ model: 'gpt-5-nano-2025-08-07' }),
  tools: [generateSqlTool],
  tool_choice: 'generate_sql',
});
