import {
  Fragment,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type KeyboardEvent,
} from 'react';
import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button';
import { Pill, PillContent } from '@inngest/components/Pill';
import { Resizable } from '@inngest/components/Resizable/Resizable';
import {
  SQLEditor,
  type SQLEditorInstance,
  type SQLEditorMarkerData,
} from '@inngest/components/SQLEditor/SQLEditor';
import type { SQLCompletionConfig } from '@inngest/components/SQLEditor/types';
import {
  usePathCreator,
  type PathCreator,
} from '@inngest/components/SharedContext/usePathCreator';
import { LinkCell, Table, TextCell, TimeCell } from '@inngest/components/Table';
import { RiCodeBlock, RiPlayLine, RiStopLine } from '@remixicon/react';
import type { Monaco } from '@monaco-editor/react';
import type { ColumnDef } from '@tanstack/react-table';
import { duckdb } from 'sql-formatter';

import {
  InsightsColumnHint,
  InsightsColumnType,
  InsightsDiagnosticSeverity,
  useLazyExecuteInsightsQueryQuery,
  type ExecuteInsightsQueryQuery,
} from '@/store/generated';
import type { CellDetailData } from './CellDetail';
import { NoResultsState, QueryEmptyState } from './EmptyStates';
import { badgeHintHref, idHintHref } from './hintLinks';
import { INSIGHTS_FUNCTIONS, INSIGHTS_TABLES } from './insightsSchema';
import { rootHint } from './pathHints';
import { Section } from './Section';

// Sourced from the generated schema (make insights-schema) rather than
// hardcoded, so autocomplete can't drift from pkg/duckdb/insights' real
// table/column registry the way a hand-maintained list eventually would.
const TABLES = INSIGHTS_TABLES.map((table) => table.name);

// Deduped across all six tables -- SQLCompletionConfig.columns is a flat
// list with no table association, so e.g. run_id (shared by several
// tables) only needs to appear once. Keeps the first table's own wording
// for a shared column's description and type (they're consistent enough
// across tables -- e.g. run_id is always a "The run this ... belongs to"
// STRING -- that picking one over another doesn't matter).
const COLUMNS = (() => {
  const byName = new Map<string, { description: string; type: string }>();
  for (const table of INSIGHTS_TABLES) {
    for (const column of table.columns) {
      if (!byName.has(column.name)) {
        byName.set(column.name, {
          description: column.description,
          type: column.type,
        });
      }
    }
  }
  return Array.from(byName, ([name, { description, type }]) => ({
    name,
    description,
    type,
  }));
})();

// `signature` here is deliberately just a generic "$1" snippet placeholder
// between parens, not fn.signature's own real parameter names -- it only
// needs to land the cursor in the right place after accepting the
// suggestion, and a fixed shape works the same for every function
// regardless of its real arity. fn.signature (the real, docs-scraped call
// signature, e.g. "concat_ws(separator, string, ...)") is shown as the
// suggestion's own `detail` instead, the same way a column's type is.
const FUNCTIONS = INSIGHTS_FUNCTIONS.map((fn) => ({
  name: fn.name,
  signature: `${fn.name}($1)`,
  description: fn.description,
  detail: fn.signature,
}));

// COUNT/SUM/AVG/MIN/MAX etc. are real entries in FUNCTIONS now (sourced
// from the same allowedFunctions registry) -- keeping them here too would
// suggest each one twice.
const KEYWORDS = [
  'SELECT',
  'FROM',
  'WHERE',
  'JOIN',
  'LEFT',
  'RIGHT',
  'INNER',
  'ON',
  'GROUP',
  'BY',
  'ORDER',
  'LIMIT',
  'OFFSET',
  'AS',
  'AND',
  'OR',
  'NOT',
  'IN',
  'IS',
  'NULL',
  'DISTINCT',
  'WITH',
  'UNION',
  'ALL',
  'HAVING',
  'CASE',
  'WHEN',
  'THEN',
  'ELSE',
  'END',
  'ASC',
  'DESC',
];

const completionConfig: SQLCompletionConfig = {
  columns: COLUMNS,
  keywords: KEYWORDS,
  functions: FUNCTIONS,
  tables: TABLES,
};

// Mirrors the dashboard Insights table's per-type column sizing
// (InsightsDataTable/states/ResultsState/useColumns.tsx).
const COLUMN_SIZE_BY_TYPE: Partial<Record<InsightsColumnType, number>> = {
  [InsightsColumnType.Datetime]: 200,
  [InsightsColumnType.Number]: 150,
  [InsightsColumnType.String]: 320,
};
const DEFAULT_COLUMN_SIZE = 340;

type ResultRow = { id: string; values: unknown[] };

// A hinted ID column (event_ids, parent_run_ids) can carry a whole
// VARCHAR[]/JSON array rather than a single ID -- the hint describes each
// element (see pkg/duckdb/insights/tables.go's event_ids comment). Cap how
// many of those we link so one wide array can't blow out a row's height.
const MAX_HINTED_LIST_ITEMS = 1;

// App/function ID columns actually carry the app's name / function's slug
// (see pkg/duckdb/insights/tables.go), so they render as a clickable badge
// linking to that app/function's page, matching the badges used elsewhere
// (e.g. the Functions table's App column).
function renderBadgeCell(
  value: string,
  hint: InsightsColumnHint.AppId | InsightsColumnHint.FunctionId,
  pathCreator: PathCreator,
) {
  return (
    <Pill href={badgeHintHref(hint, pathCreator, value)} appearance="outlined">
      <PillContent
        type={hint === InsightsColumnHint.AppId ? 'APP' : 'FUNCTION'}
      >
        {value}
      </PillContent>
    </Pill>
  );
}

// Comma-separated on one line; only the first MAX_HINTED_LIST_ITEMS render,
// with a trailing ellipsis standing in for the rest.
function renderHintedIdList(
  values: unknown[],
  hint: InsightsColumnHint.RunId | InsightsColumnHint.EventId,
  pathCreator: PathCreator,
) {
  const visible = values.slice(0, MAX_HINTED_LIST_ITEMS);
  return (
    <TextCell className="font-mono">
      {visible.map((value, i) => (
        <Fragment key={i}>
          {i > 0 && ', '}
          <LinkCell href={idHintHref(hint, pathCreator, String(value))}>
            {String(value)}
          </LinkCell>
        </Fragment>
      ))}
      {values.length > MAX_HINTED_LIST_ITEMS && ', …'}
    </TextCell>
  );
}

// Mirrors the dashboard Insights table's cell rendering (same file) --
// TextCell/TimeCell for consistent look, extended with a JSON case since
// this backend (unlike Cloud's, at the time that hook was written) already
// reports JSON columns. Run/event ID columns render as links, and app/
// function ID columns render as clickable badges (per the backend's
// column-hint inference), ahead of the type-based cases.
export function renderCell(
  value: unknown,
  type: InsightsColumnType,
  hint: InsightsColumnHint | null | undefined,
  pathCreator: PathCreator,
) {
  if (value === null || value === undefined) {
    return <TextCell>ᴺᵁᴸᴸ</TextCell>;
  }
  if (
    hint === InsightsColumnHint.RunId ||
    hint === InsightsColumnHint.EventId
  ) {
    if (Array.isArray(value)) {
      return renderHintedIdList(value, hint, pathCreator);
    }
    return (
      <LinkCell
        className="font-mono"
        href={idHintHref(hint, pathCreator, String(value))}
      >
        {String(value)}
      </LinkCell>
    );
  }
  if (
    hint === InsightsColumnHint.AppId ||
    hint === InsightsColumnHint.FunctionId
  ) {
    return renderBadgeCell(String(value), hint, pathCreator);
  }
  switch (type) {
    case InsightsColumnType.Datetime:
      return <TimeCell date={new Date(String(value))} />;
    case InsightsColumnType.Json:
      return <TextCell>{JSON.stringify(value)}</TextCell>;
    default:
      return <TextCell>{String(value)}</TextCell>;
  }
}

// graphql-request's ClientError.message is "<gql error message>: {JSON dump
// of the full request/response}" -- strip the dump so the banner only shows
// the human-readable part.
function getErrorMessage(error: unknown): string {
  if (error && typeof error === 'object' && 'message' in error) {
    const message = String((error as { message: unknown }).message);
    return message.split(': {"response"')[0] ?? message;
  }
  return 'Something went wrong running this query.';
}

type InsightsDiagnostic =
  ExecuteInsightsQueryQuery['insights']['diagnostics'][number];

const MARKER_SEVERITY: Record<
  InsightsDiagnosticSeverity,
  (monaco: Monaco) => number
> = {
  [InsightsDiagnosticSeverity.Error]: (monaco) => monaco.MarkerSeverity.Error,
  [InsightsDiagnosticSeverity.Warning]: (monaco) =>
    monaco.MarkerSeverity.Warning,
  [InsightsDiagnosticSeverity.Info]: (monaco) => monaco.MarkerSeverity.Info,
};

// Converts one diagnostic into a Monaco marker (squiggle). Widens a
// zero-width range (start === end, e.g. a rejected-query diagnostic built
// from a single-position *ValidationError -- see the resolver's
// Diagnostic() doc comment) by one column so it's actually visible.
function toMarker(
  monaco: Monaco,
  diagnostic: InsightsDiagnostic,
): SQLEditorMarkerData {
  const { start, end } = diagnostic;
  const endColumn =
    start.line === end.line && start.column === end.column
      ? end.column + 1
      : end.column;
  return {
    severity: MARKER_SEVERITY[diagnostic.severity](monaco),
    startLineNumber: start.line,
    startColumn: start.column,
    endLineNumber: end.line,
    endColumn,
    message: diagnostic.message,
    code: diagnostic.code,
  };
}

// Cancelling the in-flight request via .abort() resolves the query state
// with a synthesized {name: 'AbortError'} error (Redux Toolkit's generic
// createAsyncThunk cancellation behavior) -- not a real failure, so it
// shouldn't render as one.
function isAbortError(error: unknown): boolean {
  return Boolean(
    error &&
      typeof error === 'object' &&
      (error as { name?: unknown }).name === 'AbortError',
  );
}

export function InsightsQueryTab({
  initialSql,
  onSqlChange,
  selectedCell,
  onSelectedCellChange,
}: {
  initialSql: string;
  onSqlChange: (sql: string) => void;
  // Selection lives in the parent (InsightsPage) rather than local state,
  // so CellDetail can render as a sibling of the schema sidebar -- both at
  // the outermost layout level -- instead of nested inside this component.
  selectedCell: CellDetailData | null;
  onSelectedCellChange: (cell: CellDetailData | null) => void;
}) {
  const [sql, setSql] = useState(initialSql);
  const { pathCreator } = usePathCreator();
  const [runQuery, { data, error, isFetching }] =
    useLazyExecuteInsightsQueryQuery();
  const tableContainerRef = useRef<HTMLDivElement>(null);
  const activeQueryRef = useRef<ReturnType<typeof runQuery> | null>(null);
  const editorRef = useRef<SQLEditorInstance | null>(null);
  const monacoRef = useRef<Monaco | null>(null);

  const handleRunClick = () => {
    if (isFetching) {
      activeQueryRef.current?.abort();
      return;
    }
    activeQueryRef.current = runQuery({ sql });
  };

  const handleFormatClick = () => {
    editorRef.current?.getAction('editor.action.formatDocument')?.run();
  };

  const result = data?.insights;

  const columns: ColumnDef<ResultRow, unknown>[] = useMemo(
    () =>
      (result?.columns ?? []).map((column, i) => ({
        id: column.name,
        header: column.name,
        accessorFn: (row: ResultRow) => row.values[i],
        cell: (info) =>
          renderCell(
            info.getValue(),
            column.type,
            rootHint(column.pathHints),
            pathCreator,
          ),
        minSize: COLUMN_SIZE_BY_TYPE[column.type] ?? DEFAULT_COLUMN_SIZE,
      })),
    [result?.columns, pathCreator],
  );

  const rows: ResultRow[] = useMemo(
    () => (result?.rows ?? []).map((values, i) => ({ id: String(i), values })),
    [result?.rows],
  );

  const handleCellClick = useCallback(
    (rowIndex: number, columnId: string, value: unknown) => {
      const colIndex =
        result?.columns.findIndex((c) => c.name === columnId) ?? -1;
      const col = colIndex === -1 ? undefined : result?.columns[colIndex];
      if (!col) return;
      onSelectedCellChange({
        rowIndex,
        columnId,
        columnType: col.type,
        columnPathHints: col.pathHints,
        value,
      });
    },
    [result, onSelectedCellChange],
  );

  // Keyboard navigation: arrow keys move between cells, Escape deselects --
  // mirrors the dashboard Insights table's ResultsTable.tsx.
  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (!selectedCell || !result) return;

      const colNames = result.columns.map((c) => c.name);
      const colIndex = colNames.indexOf(selectedCell.columnId);
      if (colIndex === -1) return;

      let nextRow = selectedCell.rowIndex;
      let nextColIndex = colIndex;

      switch (e.key) {
        case 'ArrowUp':
          nextRow = Math.max(0, nextRow - 1);
          break;
        case 'ArrowDown':
          nextRow = Math.min(rows.length - 1, nextRow + 1);
          break;
        case 'ArrowLeft':
          nextColIndex = Math.max(0, nextColIndex - 1);
          break;
        case 'ArrowRight':
          nextColIndex = Math.min(colNames.length - 1, nextColIndex + 1);
          break;
        case 'Escape':
          onSelectedCellChange(null);
          return;
        default:
          return;
      }

      e.preventDefault();

      const nextColumnId = colNames[nextColIndex];
      const nextCol = result.columns[nextColIndex];
      if (!nextColumnId || !nextCol) return;

      const value = rows[nextRow]?.values[nextColIndex] ?? null;
      onSelectedCellChange({
        rowIndex: nextRow,
        columnId: nextColumnId,
        columnType: nextCol.type,
        columnPathHints: nextCol.pathHints,
        value,
      });
    },
    [selectedCell, result, rows, onSelectedCellChange],
  );

  useEffect(() => {
    if (!selectedCell) return;
    tableContainerRef.current
      ?.querySelector('td[data-selected="true"]')
      ?.scrollIntoView({ block: 'nearest', inline: 'nearest' });
  }, [selectedCell]);

  const diagnostics = result?.diagnostics ?? [];
  const fatalDiagnostic = diagnostics.find(
    (d) => d.severity === InsightsDiagnosticSeverity.Error,
  );
  const noteDiagnostics = diagnostics.filter(
    (d) => d.severity !== InsightsDiagnosticSeverity.Error,
  );

  // Renders each diagnostic as a Monaco squiggle at its exact position,
  // in addition to the plain-text list below -- mirrors the dashboard
  // Insights editor's own diagnostics-as-markers treatment.
  useEffect(() => {
    const editor = editorRef.current;
    const monaco = monacoRef.current;
    const model = editor?.getModel();
    if (!monaco || !model) return;

    monaco.editor.setModelMarkers(
      model,
      'insights-diagnostics',
      diagnostics.map((d) => toMarker(monaco, d)),
    );
  }, [diagnostics]);

  const resultsTable = (
    <div
      ref={tableContainerRef}
      tabIndex={0}
      onKeyDown={handleKeyDown}
      className="h-full overflow-auto outline-none"
    >
      <Table
        data={rows}
        columns={columns}
        isLoading={isFetching}
        enableColumnDynamicSizing
        selectedCell={
          selectedCell
            ? {
                rowIndex: selectedCell.rowIndex,
                columnId: selectedCell.columnId,
              }
            : null
        }
        onCellClick={handleCellClick}
        cellClassName="[&:not(:first-child)]:border-l [&:not(:first-child)]:border-light box-border align-top px-4 py-2.5 overflow-hidden text-ellipsis whitespace-nowrap"
        blankState={data ? <NoResultsState /> : <QueryEmptyState />}
      />
    </div>
  );

  const tabContent = (
    <Resizable
      orientation="vertical"
      defaultSplitPercentage={37.5}
      minSplitPercentage={20}
      maxSplitPercentage={80}
      splitKey="insights-tab-panel-split-vertical"
      first={
        <Section
          className="h-full"
          title="Query"
          actions={
            <>
              <Button
                appearance="outlined"
                kind="secondary"
                icon={<RiCodeBlock />}
                tooltip="Format query"
                onClick={handleFormatClick}
                disabled={!sql.trim()}
              />
              <Button
                label={isFetching ? 'Cancel' : 'Run query'}
                icon={isFetching ? <RiStopLine /> : <RiPlayLine />}
                iconSide="left"
                onClick={handleRunClick}
                disabled={!isFetching && !sql.trim()}
              />
            </>
          }
        >
          <SQLEditor
            completionConfig={completionConfig}
            content={sql}
            dialect={duckdb}
            onChange={(value) => {
              setSql(value);
              onSqlChange(value);
            }}
            onMount={(editor, monaco) => {
              editorRef.current = editor;
              monacoRef.current = monaco;
            }}
          />
        </Section>
      }
      second={
        <Section
          className="border-subtle h-full border-t"
          title={<span className="uppercase">Results</span>}
          actions={
            <>
              {isFetching && (
                <span className="text-muted text-xs">Running query...</span>
              )}
              {result?.info.limited && (
                <span className="text-muted text-xs">
                  Results limited to {rows.length.toLocaleString()} rows
                </span>
              )}
            </>
          }
        >
          {error && !isAbortError(error) ? (
            <div className="p-3">
              <Alert severity="error">{getErrorMessage(error)}</Alert>
            </div>
          ) : fatalDiagnostic ? (
            <div className="p-3">
              <Alert severity="error">{fatalDiagnostic.message}</Alert>
            </div>
          ) : (
            <div className="flex h-full min-h-0 flex-col">
              {noteDiagnostics.length > 0 && (
                <ul className="border-subtle flex flex-col gap-1 border-b px-4 py-2">
                  {noteDiagnostics.map((d, i) => (
                    <li key={i} className="text-muted text-xs">
                      {d.message}
                    </li>
                  ))}
                </ul>
              )}
              <div className="min-h-0 flex-1">{resultsTable}</div>
            </div>
          )}
        </Section>
      }
    />
  );

  return <div className="flex h-full min-h-0 flex-col">{tabContent}</div>;
}
