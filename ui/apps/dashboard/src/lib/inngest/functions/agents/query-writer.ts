import {
  createAgent,
  createTool,
  openai,
  type AnyZodType,
} from "@inngest/agent-kit";
import { z } from "zod";

import type { InsightsAgentState } from "./types";

const queryGrammar = `
  QueryAST = "SELECT" SelectClause "FROM" From ("WHERE" Expression)? ("GROUP" "BY" GroupBy)? ("ORDER" "BY" OrderBy)? ("LIMIT" <number>)? ("OFFSET" <number>)? ";"? .
  SelectClause = "DISTINCT"? ("*" | (AliasedColumnExpression ("," AliasedColumnExpression)*)) .
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

const insightsDocs = `
# Insights

Inngest Insights allows you to query and analyze your event data using SQL directly within the Inngest platform. Every event sent to Inngest contains valuable information, and Insights gives you the power to extract meaningful patterns and analytics from that data.

<Info>
  Insights support is currently in Public Beta. Some details including SQL syntax and feature availability are still subject to change during this period. Read more about the [Public Beta release phase here](/docs/release-phases#public-beta) and the [roadmap here](#roadmap).
</Info>

## Overview

Insights provides an in-app SQL editor and query interface where you can:
- Query event data using familiar SQL syntax
- Save and reuse common queries
- Analyze patterns in your event triggers
- Extract business intelligence from your workflows

Currently, you can **only query events**. Support for querying function runs will be added in future releases.

## Getting Started

Access Insights through the Inngest dashboard by clicking on the "Insights" tab in the left navigation. 

![Getting Started Dashboard View](/assets/docs/platform/monitor/insights/insights_dashboard_view.png)

We have several pre-built query templates to help you get started exploring your data.

![Getting Started Templates View](/assets/docs/platform/monitor/insights/insights_template_view.png)

## SQL Editor

The Insights interface includes a full-featured SQL editor where you can:

- Write and execute SQL queries against your event data
- Save frequently used queries for later access
- View query results in an organized table format
- Access query history and templates from the sidebar

![Sql Editor View](/assets/docs/platform/monitor/insights/insights_sql_editor.png)

### Available Columns

When querying events, you have access to the following columns:

| Column | Type | Description |
|--------|------|-------------|
| id | String | Unique identifier for the event |
| name | String | The name/type of the event |
| **data** | **JSON** | **The event payload data - users can send any JSON structure here** |
| ts | Unix timestamp (ms) | Unix timestamp in milliseconds when the event occurred - [reference](https://www.unixtimestamp.com/) |
| v | String | Event format version |

For more details on the event format, see the [Inngest Event Format documentation](/docs/features/events-triggers/event-format).

### Data Retention

Refer to [pricing plans](/pricing) for data retention limits.

### Result Limits

- Current page limit: **1000 rows**
- Future updates will support larger result sets through async data exports

## SQL Support

Insights is built on ClickHouse, which provides powerful SQL capabilities with some differences from traditional SQL databases.

![Sql Editor View](/assets/docs/platform/monitor/insights/insights_query_results.png)

### Supported Functions

#### Arithmetic Functions
Basic mathematical operations and calculations.
[View ClickHouse arithmetic functions documentation](https://clickhouse.com/docs/sql-reference/functions/arithmetic-functions)

#### String Functions
String manipulation and search capabilities.
- [String search functions](https://clickhouse.com/docs/sql-reference/functions/string-search-functions)
- [String manipulation functions](https://clickhouse.com/docs/sql-reference/functions/string-functions)

#### JSON Functions
Essential for working with events.data payloads.
[View ClickHouse JSON functions documentation](https://clickhouse.com/docs/sql-reference/functions/json-functions)

#### Date/Time Functions
For analyzing event timing and patterns.
[View ClickHouse date/time functions documentation](https://clickhouse.com/docs/sql-reference/functions/date-time-functions)

#### Other Supported Function Categories
- [Logical functions](https://clickhouse.com/docs/sql-reference/functions/logical-functions)
- [Rounding functions](https://clickhouse.com/docs/sql-reference/functions/rounding-functions)
- [Type conversion functions](https://clickhouse.com/docs/sql-reference/functions/type-conversion-functions)
- [Functions for nulls](https://clickhouse.com/docs/sql-reference/functions/functions-for-nulls)
- [ULID functions](https://clickhouse.com/docs/sql-reference/functions/ulid-functions)

### Aggregate Functions

The following aggregate functions are supported:

| Function | Description |
|----------|-------------|
| ARRAY_AGG() | [Aggregates values into an array](https://clickhouse.com/docs/sql-reference/aggregate-functions/reference/grouparray)  * |
| AVG() | [Calculates average](https://clickhouse.com/docs/sql-reference/aggregate-functions/reference/avg) |
| COUNT() | [Counts rows](https://clickhouse.com/docs/sql-reference/aggregate-functions/reference/count) |
| MAX() | [Finds maximum value](https://clickhouse.com/docs/sql-reference/aggregate-functions/reference/max) |
| MIN() | [Finds minimum value](https://clickhouse.com/docs/sql-reference/aggregate-functions/reference/min) |
| STDDEV_POP() | [Population standard deviation](https://clickhouse.com/docs/sql-reference/aggregate-functions/reference/stddevpop) |
| STDDEV_SAMP() | [Sample standard deviation](https://clickhouse.com/docs/sql-reference/aggregate-functions/reference/stddevsamp) |
| SUM() | [Calculates sum](https://clickhouse.com/docs/sql-reference/aggregate-functions/reference/sum) |
| VAR_POP() | [Population variance](https://clickhouse.com/docs/en/sql-reference/aggregate-functions/reference/varPop) |
| VAR_SAMP() | [Sample variance](https://clickhouse.com/docs/sql-reference/aggregate-functions/reference/varSamp) |
| median() | [Finds median value](https://clickhouse.com/docs/sql-reference/aggregate-functions/reference/median) |



### SQL Syntax Limitations

Some SQL features are not yet supported but are planned for future releases:

- **CTEs (Common Table Expressions)** using WITH
- **IS operator**
- **NOT operator**


## Working with Event Data

### Event-Specific Schema

Within **events.data**, users can send any JSON they want, so the structure and available fields will be specific to their payloads. You can use ClickHouse's JSON functions to extract and query specific fields within your event data.

### Example Queries

#### Basic Event Filtering

SELECT count(*)
FROM events
WHERE name = 'inngest/function.failed'
AND simpleJSONExtractString(data, 'function_id') = 'generate-report'
AND ts > toUnixTimestamp(addHours(now(), -1)) * 1000;


#### Extracting JSON Data and Aggregating

SELECT simpleJSONExtractString(data, 'user_id') as user_id, count(*) 
FROM events
WHERE name = 'order.created'
GROUP BY user_id
ORDER BY count(*) DESC
LIMIT 10;

## Saved Queries

You can save frequently used queries for quick access. Queries are only saved private for you to use; they are not shared across your Inngest organization.

## Need Help?

If you encounter issues or have questions about Insights:
1. Check this documentation for common solutions
2. Review the [ClickHouse SQL reference](https://clickhouse.com/docs/sql-reference/) for advanced function usage
3. Contact support through the Inngest platform

`;

const GenerateSqlParams = z.object({
  sql: z
    .string()
    .min(1)
    .describe(
      "A single valid SELECT statement. Do not include DDL/DML or multiple statements.",
    ),
  title: z
    .string()
    .min(1)
    .describe("Short 20-30 character title for this query"),
  reasoning: z
    .string()
    .min(1)
    .describe(
      "Brief 1-2 sentence explanation of how this query addresses the request",
    ),
});

export const generateSqlTool = createTool({
  name: "generate_sql",
  description:
    "Provide the final SQL SELECT statement for ClickHouse based on the selected events and schemas.",
  parameters: GenerateSqlParams as unknown as AnyZodType, // (ted): need to update to latest version of zod + agent-kit
  handler: ({ sql, title, reasoning }: z.infer<typeof GenerateSqlParams>) => {
    return {
      sql,
      title,
      reasoning,
    };
  },
});

export const queryWriterAgent = createAgent<InsightsAgentState>({
  name: "Insights Query Writer",
  description:
    "Generates a safe, read-only SQL SELECT statement for ClickHouse.",
  system: async ({ network }) => {
    const selected =
      network?.state.data.selectedEvents?.map((e) => e.event_name) ?? [];
    return [
      "You write ClickHouse-compatible SQL for analytics.",
      `You MUST follow these rules ${insightsDocs}`,
      `You MUST follow this grammar ${queryGrammar}`,
      selected.length
        ? `Target the following events if relevant: ${selected.join(", ")}`
        : "If events were selected earlier, incorporate them appropriately.",
      "",
      "When ready, call the generate_sql tool with the final SQL and a short 20-30 character title.",
      "Few rules to be aware of AT ALL TIMES:",
      '- Do NOT under any circumstances prefix table names or column names with "events_"',
    ].join("\n");
  },
  model: openai({ model: "gpt-4.1-2025-04-14" }),
  tools: [generateSqlTool],
  tool_choice: "generate_sql",
});
