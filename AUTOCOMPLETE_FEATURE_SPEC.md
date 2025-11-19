# Add Context-Aware SQL Autocomplete for Event Names and Data Properties

## Background
The SQL Editor currently has basic autocomplete for keywords, functions, and tables. We need to enhance it with **context-aware autocomplete** that suggests:
1. **Event names** (in WHERE clauses after `name = '`)
2. **Data properties** (in SELECT clauses after `data.`)

## Current Architecture

### Autocomplete System
- Located in: `ui/packages/components/src/SQLEditor/hooks/useSQLCompletions.ts`
- Uses Monaco's `registerCompletionItemProvider` API
- Currently provides static suggestions for: `columns`, `keywords`, `functions`, `tables`
- Configuration: `SQLCompletionConfig` interface in `ui/packages/components/src/SQLEditor/types.ts`
- Current config: `ui/apps/dashboard/src/components/Insights/InsightsSQLEditor/constants.ts` (static `SQL_COMPLETION_CONFIG`)

### Data Sources

#### 1. GetAllEventNames - Returns all event type names
- GraphQL query: `ui/apps/dashboard/src/components/EventTypes/useEventTypes.ts:207-218`
- Hook: `useAllEventTypes()` at `ui/apps/dashboard/src/components/EventTypes/useEventTypes.ts:220-246`
- Returns: `{ name: string }[]`

#### 2. GetEventTypeSchemas - Returns schemas for specific event names
- GraphQL query: `ui/apps/dashboard/src/components/Insights/InsightsTabManager/InsightsHelperPanel/features/SchemaExplorer/SchemasContext/useEventTypeSchemas.ts:10-33`
- Hook: `useEventTypeSchemas()` at `ui/apps/dashboard/src/components/Insights/InsightsTabManager/InsightsHelperPanel/features/SchemaExplorer/SchemasContext/useEventTypeSchemas.ts:40-73`
- Returns: `{ events: { name: string, schema: string }[] }`
- Schema is JSON Schema format, data properties are extracted via `extractDataProperty()` in `ui/apps/dashboard/src/components/Insights/InsightsTabManager/InsightsHelperPanel/features/SchemaExplorer/SchemasContext/queries.ts:68-73`

### Existing Schema Detection
- `ui/apps/dashboard/src/components/Insights/InsightsTabManager/InsightsHelperPanel/features/SchemaExplorer/useSchemasInUse.ts` already detects event names from query using regex: `/name\s*=\s*'([^']+)'/g` (line 50)
- Automatically fetches schemas for detected event names
- Parses schemas to extract `data.*` properties

## Requirements

### 1. Event Name Autocomplete
**Trigger:** User types `name = '` in a WHERE clause

**Behavior:**
- Show all event names from `GetAllEventNames`
- Example: `SELECT * FROM events WHERE name = '|` ← cursor shows suggestions
- Should work for: `name = '`, `name='`, `name  =  '` (flexible whitespace)
- Should also work in OR clauses: `name = 'event1' OR name = '|`

### 2. Data Property Autocomplete
**Trigger:** User types `data.` in a SELECT clause

**Behavior:**
- Parse query to find event names (using existing regex from `useSchemasInUse.ts:50`)
- Fetch schemas for those event names using `GetEventTypeSchemas`
- Extract properties from `schema.properties.data.properties` (JSON Schema format)
- Show property name + type (e.g., `account_id string`, `amount integer`)
- **Only show when schemas are available** (don't show for unsaved queries)

**Examples:**
```sql
-- After typing this:
SELECT data.| FROM events WHERE name = 'user.signup'
-- Should suggest: id, email, first_name, last_name, account_id, company (if those are in the schema)

-- Multiple events:
SELECT data.| FROM events WHERE name = 'payment.succeeded' OR name = 'payment.failed'
-- Should suggest union of properties from both schemas
```

## Implementation Plan

### 1. Make SQL_COMPLETION_CONFIG dynamic
**File:** `ui/apps/dashboard/src/components/Insights/InsightsSQLEditor/constants.ts`
- Convert from static config to a hook (e.g., `useSQLCompletionConfig()`)
- Fetch event names using `useAllEventTypes()`
- Add event names to a new field in config (e.g., `eventNames: string[]`)

### 2. Enhance useSQLCompletions
**File:** `ui/packages/components/src/SQLEditor/hooks/useSQLCompletions.ts`
- Update `SQLCompletionConfig` type to include `eventNames` and `dataProperties`
- Implement context-aware logic in `provideCompletionItems`:
  - Detect `name = '` pattern → suggest event names
  - Detect `data.` pattern → suggest data properties
- Use Monaco's `model.getValueInRange()` to look backward from cursor position

### 3. Add data property detection
- Reuse regex from `useSchemasInUse.ts:50` to extract event names from current query
- Fetch schemas using `useEventTypeSchemas()`
- Parse JSON schemas to extract `data.properties` keys and types
- Pass to autocomplete config as `dataProperties: Array<{ name: string, type: string }>`

### 4. Update InsightsSQLEditor
**File:** `ui/apps/dashboard/src/components/Insights/InsightsSQLEditor/InsightsSQLEditor.tsx`
- Replace static `SQL_COMPLETION_CONFIG` with dynamic `useSQLCompletionConfig()` hook
- Pass dynamic config to `<SQLEditor completionConfig={config} />`

## Key Files to Modify

1. **ui/packages/components/src/SQLEditor/types.ts** - Add `eventNames` and `dataProperties` to interface
2. **ui/packages/components/src/SQLEditor/hooks/useSQLCompletions.ts** - Add context detection logic
3. **ui/apps/dashboard/src/components/Insights/InsightsSQLEditor/constants.ts** - Convert to dynamic hook
4. **ui/apps/dashboard/src/components/Insights/InsightsSQLEditor/InsightsSQLEditor.tsx** - Use new hook

## Technical Details

- **Monaco Context Detection:** Use `model.getLineContent()` and `position.column` to read text before cursor
- **Event Name Pattern:** Look for `name\s*=\s*'` before cursor position
- **Data Property Pattern:** Look for `data\.` before cursor position
- **Schema Parsing:** JSON Schema format, properties at `schema.properties.data.properties[key].type`
- **Debouncing:** Schema fetching should be debounced (already done in `useSchemasInUse`)

## Edge Cases
- Multiple event names in query → union all data properties
- No event name detected → don't show data properties
- Schema not available → gracefully skip data property suggestions
- User typing in string literal → don't autocomplete event names inside regular strings
