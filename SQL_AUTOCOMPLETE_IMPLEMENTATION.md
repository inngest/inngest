# SQL Autocomplete Implementation Overview

## Summary

This changeset adds intelligent, context-aware autocomplete to the SQL Editor for the Insights feature. The autocomplete system provides suggestions for:

1. **Event names** - When typing `name = '` in WHERE clauses
2. **Data properties** - When typing `data.` in SELECT clauses
3. **ClickHouse functions** - All 542 available ClickHouse functions
4. **Table names and columns** - `events` table with `name` and `data` columns
5. **SQL keywords** - Standard SQL keywords (SELECT, WHERE, FROM, etc.)

## Architecture Overview

### Data Sources

The implementation **piggybacks on existing GraphQL queries** used by the Schema Explorer rather than creating new endpoints:

#### 1. Event Names (`GetAllEventNames`)
- **Source**: `ui/apps/dashboard/src/components/EventTypes/useEventTypes.ts:207-218`
- **Hook**: `useAllEventTypes()`
- **Caching**: React Query with 5-minute stale time
- **Returns**: Array of `{ name: string }` for all event types
- **Used for**: Suggesting event names after typing `name = '`

#### 2. Event Schemas (`GetEventTypeSchemas`)
- **Source**: `ui/apps/dashboard/src/components/Insights/InsightsTabManager/InsightsHelperPanel/features/SchemaExplorer/SchemasContext/useEventTypeSchemas.ts:10-33`
- **Hook**: `useEventTypeSchemas()`
- **Fetching Strategy**: Dynamically fetched based on event names detected in the current query
- **Returns**: JSON Schema for each event type's `data` object
- **Used for**: Extracting property names and types for `data.*` autocomplete

#### 3. ClickHouse Functions
- **Source**: `ui/packages/components/src/SQLEditor/hooks/availableClickhouseFunctions.ts`
- **Type**: Static array of 542 function names
- **Used for**: Function autocomplete in SELECT clauses

### Query Parsing Strategy

The implementation reuses the **same regex pattern** from the existing Schema Explorer (`useSchemasInUse.ts:50`):

```javascript
const POSSIBLE_EVENT_NAME_REGEX = /name\s*=\s*'([^']+)'/gi;
```

This means:
- **No duplicate logic** - Uses proven, existing pattern matching
- **Consistent behavior** - Schema Explorer and autocomplete stay in sync
- **Automatic schema loading** - Schemas are fetched when event names are detected in the query

## How Autocomplete Decides What to Show

The autocomplete uses **context detection** based on the text before the cursor position. Here's the decision flow:

### 1. Event Name Context
**Pattern**: `/name\s*=\s*'[^']*$/i`

**Triggers when**:
```sql
SELECT * FROM events WHERE name = '|
                                   ↑ cursor here
```

**Shows**: All available event names from `GetAllEventNames`

**Example**:
```sql
WHERE name = 'user.signup'
WHERE name = 'payment.succeeded'
```

### 2. Data Property Context
**Pattern**: `/\bdata\.[a-zA-Z_]*$/i`

**Triggers when**:
```sql
SELECT data.| FROM events WHERE name = 'user.signup'
            ↑ cursor here
```

**Shows**: Properties extracted from event schemas (only when schemas are available)

**Process**:
1. Parse query to find event names using `POSSIBLE_EVENT_NAME_REGEX`
2. Fetch schemas for those event names
3. Extract properties from `schema.properties.data.properties`
4. Show union of all properties if multiple events are referenced

**Example**:
```sql
-- After detecting 'user.signup' in query, shows:
data.email
data.first_name
data.account_id
```

### 3. Default Context (Everything Else)

**Shows** (in priority order):
1. **Columns**: `name`, `data`
2. **Functions**: All 542 ClickHouse functions (JSONType, concat, toDateTime, etc.)
3. **Keywords**: SQL keywords (SELECT, WHERE, FROM, AND, OR, etc.)
4. **Tables**: `events`

## Implementation Details

### File Structure

```
ui/
├── packages/components/src/SQLEditor/
│   ├── types.ts                          # SQLCompletionConfig interface
│   ├── hooks/
│   │   ├── useSQLCompletions.ts          # Context-aware autocomplete logic
│   │   └── availableClickhouseFunctions.ts # 542 ClickHouse function names
│   └── SQLEditor.tsx                     # Editor component
│
└── apps/dashboard/src/components/Insights/
    └── InsightsSQLEditor/
        ├── InsightsSQLEditor.tsx         # Insights-specific editor wrapper
        ├── constants.ts                  # Static SQL constants (keywords, tables)
        └── hooks/
            └── useSQLCompletionConfig.ts # Dynamic config builder

```

### Key Files Changed

#### 1. `ui/packages/components/src/SQLEditor/types.ts`
**What changed**: Added optional fields to `SQLCompletionConfig`
```typescript
export interface SQLCompletionConfig {
  columns: readonly string[];
  keywords: readonly string[];
  functions: readonly { name: string; signature: string }[];
  tables: readonly string[];
  eventNames?: readonly string[];        // NEW
  dataProperties?: readonly { name: string; type: string }[]; // NEW
}
```

#### 2. `ui/packages/components/src/SQLEditor/hooks/useSQLCompletions.ts`
**What changed**: Added context detection logic

**Key additions**:
- Text-before-cursor analysis using Monaco's `getValueInRange()`
- Pattern matching for `name = '` and `data.`
- Early returns for context-specific suggestions
- Different Monaco completion item kinds for different suggestion types

**Monaco Integration**:
```typescript
// Get text before cursor for context detection
const textBeforeCursor = model.getValueInRange({
  startLineNumber: position.lineNumber,
  startColumn: 1,
  endLineNumber: position.lineNumber,
  endColumn: position.column,
});

// Check context and provide specific suggestions
if (isAfterNameEquals) {
  // Only show event names
  return { suggestions: eventNameSuggestions };
}
```

#### 3. `ui/apps/dashboard/src/components/Insights/InsightsSQLEditor/hooks/useSQLCompletionConfig.ts`
**What changed**: New file - dynamic configuration builder

**Responsibilities**:
- Fetch all event types using `useAllEventTypes()`
- Parse current query to detect event names
- Fetch schemas for detected events
- Extract `data.*` properties from JSON schemas
- Build and memoize complete autocomplete config

**Key features**:
- **React Query caching**: Event types cached for 5 minutes
- **Debounced schema fetching**: Only fetches when query changes
- **Union of properties**: If multiple events in query, shows all properties
- **Graceful error handling**: Continues if schema parsing fails

**Example flow**:
```typescript
User types: "SELECT * FROM events WHERE name = 'user.signup'"
           ↓
useSQLCompletionConfig detects 'user.signup' via regex
           ↓
Fetches schema for 'user.signup' from GetEventTypeSchemas
           ↓
Parses schema.properties.data.properties
           ↓
Returns config with dataProperties: [
  { name: 'email', type: 'string' },
  { name: 'first_name', type: 'string' },
  ...
]
```

#### 4. `ui/apps/dashboard/src/components/Insights/InsightsSQLEditor/InsightsSQLEditor.tsx`
**What changed**: Switched from static to dynamic config

**Before**:
```typescript
<SQLEditor completionConfig={SQL_COMPLETION_CONFIG} ... />
```

**After**:
```typescript
const completionConfig = useSQLCompletionConfig();
<SQLEditor completionConfig={completionConfig} ... />
```

## Extensibility

### Adding New Columns
Edit `useSQLCompletionConfig.ts`:
```typescript
const COLUMNS = ['name', 'data', 'timestamp', 'version'] as const;
```

### Adding New Keywords
Edit `useSQLCompletionConfig.ts`:
```typescript
const KEYWORDS = [
  'SELECT', 'WHERE', 'FROM',
  'HAVING', 'WINDOW', 'PARTITION BY', // Add more
] as const;
```

### Adding New ClickHouse Functions
Edit `ui/packages/components/src/SQLEditor/hooks/availableClickhouseFunctions.ts`:
```typescript
export const availableClickhouseFunctions = [
  'abs', 'concat', 'JSONType',
  'myNewFunction', // Add here
];
```

### Adding New Context-Aware Rules
Edit `useSQLCompletions.ts` to add new patterns:
```typescript
// Example: Autocomplete table names after FROM
const isAfterFrom = /\bFROM\s+\w*$/i.test(textBeforeCursor);
if (isAfterFrom) {
  // Show only tables
  return { suggestions: tableSuggestions };
}
```

## Performance Considerations

### Caching Strategy
1. **Event names**: Cached with React Query for 5 minutes
2. **Schemas**: Fetched on-demand, limited to 5 events max
3. **ClickHouse functions**: Static array, no fetching needed

### Debouncing
- Query parsing is debounced in the existing `useSchemasInUse` hook
- Schema fetching only triggers when event names actually change

### Memory Usage
- ClickHouse functions list: ~542 items (~8KB)
- Event names: Depends on environment (typically <1000 items)
- Schemas: Only loaded for events referenced in query (max 5)

## Edge Cases Handled

### Multiple Event Names
```sql
SELECT data.id FROM events
WHERE name = 'user.signup' OR name = 'user.login'
```
**Behavior**: Shows union of properties from both schemas

### No Event Name Detected
```sql
SELECT data.| FROM events
```
**Behavior**: No data property suggestions (can't determine which schema)

### Invalid/Missing Schema
**Behavior**: Gracefully skips autocomplete for data properties, other suggestions still work

### String Literals
The regex `/name\s*=\s*'[^']*$/` ensures we only match actual WHERE clauses, not:
```sql
SELECT 'some text with name = ' FROM events  -- Won't trigger
```

## Testing Scenarios

### Scenario 1: Event Name Autocomplete
```sql
SELECT * FROM events WHERE name = '|
```
**Expected**: Shows all event types from GetAllEventNames

### Scenario 2: Data Property Autocomplete
```sql
SELECT data.| FROM events WHERE name = 'user.signup'
```
**Expected**: Shows properties like `email`, `first_name`, `account_id`

### Scenario 3: Function Autocomplete
```sql
SELECT JSONType|
```
**Expected**: Shows ClickHouse functions starting with "JSONType"

### Scenario 4: Column Autocomplete
```sql
SELECT | FROM events
```
**Expected**: Shows `name`, `data`, keywords, and functions

### Scenario 5: Multiple Events
```sql
SELECT data.| FROM events WHERE name = 'a' OR name = 'b'
```
**Expected**: Shows union of properties from both events' schemas

## Benefits of This Approach

1. **No new API endpoints** - Reuses existing GraphQL queries from Schema Explorer
2. **Consistent with UI** - Same regex, same data sources as Schema Explorer
3. **Performant** - Caching, debouncing, and on-demand loading
4. **Type-safe** - Full TypeScript support throughout
5. **Extensible** - Easy to add new columns, functions, or context rules
6. **User-friendly** - Context-aware suggestions reduce noise

## Future Enhancements

### Potential Improvements
1. **Aggregate function detection** - Only show aggregate functions after GROUP BY
2. **Column alias support** - Autocomplete column aliases defined earlier in query
3. **JOIN support** - Autocomplete join conditions
4. **Subquery awareness** - Context-aware autocomplete inside subqueries
5. **Documentation tooltips** - Show ClickHouse function documentation on hover
6. **Fuzzy matching** - Match suggestions even with typos
7. **Snippet templates** - Pre-built query templates (e.g., "common aggregations")

## Migration Notes

### Breaking Changes
None - This is purely additive functionality.

### Backward Compatibility
- Old `SQLCompletionConfig` still works (new fields are optional)
- Components not using `useSQLCompletionConfig` still use static config
- All existing autocomplete features remain unchanged
