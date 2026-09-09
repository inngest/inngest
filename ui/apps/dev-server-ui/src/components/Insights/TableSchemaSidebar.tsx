import { useMemo, useState } from 'react';
import { Search } from '@inngest/components/Forms/Search';
import { Link } from '@inngest/components/Link';
import { SchemaViewer } from '@inngest/components/SchemaViewer/SchemaViewer';
import type {
  SchemaNode,
  TypedNode,
} from '@inngest/components/SchemaViewer/types';
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@inngest/components/Tooltip/Tooltip';
import { RiExternalLinkLine, RiInformationLine } from '@remixicon/react';

import {
  INSIGHTS_FUNCTIONS,
  INSIGHTS_TABLES,
  type InsightsSchemaFunction,
  type InsightsSchemaTable,
} from './insightsSchema';

function toSchemaNode(table: InsightsSchemaTable): SchemaNode {
  return {
    kind: 'table',
    name: table.name,
    path: table.name,
    children: table.columns.map((column) => ({
      kind: 'value',
      name: column.name,
      path: `${table.name}.${column.name}`,
      type: column.type,
    })),
  };
}

// Both tables and columns are addressed by their SchemaNode path ("table"
// or "table.column"), which is exactly how toSchemaNode builds it above --
// this map is the only place the sidebar knows a description exists at
// all, since SchemaNode itself (a shared, backend-agnostic type) has no
// description field of its own.
function buildDescriptions(
  tables: readonly InsightsSchemaTable[],
): Map<string, string> {
  const descriptions = new Map<string, string>();
  for (const table of tables) {
    descriptions.set(table.name, table.description);
    for (const column of table.columns) {
      descriptions.set(`${table.name}.${column.name}`, column.description);
    }
  }
  return descriptions;
}

// The SQL editor's schema explorer -- lists every table Insights queries
// can reference, expandable to each table's columns, both sourced from the
// generated insightsSchema.generated.json (make insights-schema) rather
// than fetched live: this is a fixed, build-time schema, not per-tenant
// data. Mirrors the Cloud dashboard Insights feature's own schema explorer
// (InsightsHelperPanel/features/SchemaExplorer), simplified to this
// backend's flat (no nested-JSON) column model.
export function TableSchemaSidebar() {
  const descriptions = useMemo(() => buildDescriptions(INSIGHTS_TABLES), []);

  const renderAdornment = (node: TypedNode) => {
    const description = descriptions.get(node.path);
    if (!description) return null;
    return (
      <Tooltip>
        <TooltipTrigger asChild>
          {/* align-middle overrides ValueRow/TableRow's own wrapping span
              (align-baseline, shared component -- not ours to edit), which
              otherwise sinks an SVG icon below the row's text baseline;
              -translate-y-px nudges it up the rest of the way to visually
              center against the row's type label. */}
          <span className="text-light inline-flex -translate-y-px cursor-help items-center align-middle">
            <RiInformationLine className="h-3.5 w-3.5" />
          </span>
        </TooltipTrigger>
        <TooltipContent side="left" className="max-w-xs">
          {description}
        </TooltipContent>
      </Tooltip>
    );
  };

  return (
    <div className="flex h-full flex-col gap-1 overflow-auto p-2">
      {INSIGHTS_TABLES.map((table) => (
        <SchemaViewer
          key={table.name}
          node={toSchemaNode(table)}
          renderAdornment={renderAdornment}
        />
      ))}
    </div>
  );
}

// A searchable list of every function the SQL editor accepts (functions.go's
// allowedFunctions) -- its own vertical tab (InsightsPage.tsx), at the same
// level as Schema, since there's no natural tree structure to a flat
// function list the way there is for a table's columns.
export function FunctionsSidebar() {
  const [search, setSearch] = useState('');

  const matches = useMemo(() => {
    const term = search.trim().toLowerCase();
    if (!term) return INSIGHTS_FUNCTIONS;
    return INSIGHTS_FUNCTIONS.filter((fn) =>
      fn.name.toLowerCase().includes(term),
    );
  }, [search]);

  return (
    <div className="flex h-full min-h-0 flex-col gap-2 p-2">
      <Search
        inngestSize="base"
        onUpdate={setSearch}
        placeholder="Search functions"
        value={search}
      />
      <div className="flex min-h-0 flex-1 flex-col gap-0.5 overflow-auto">
        {matches.map((fn) => (
          <FunctionRow key={fn.name} fn={fn} />
        ))}
        {matches.length === 0 && (
          <div className="text-light p-2 text-sm">No functions match.</div>
        )}
      </div>
    </div>
  );
}

function FunctionRow({ fn }: { fn: InsightsSchemaFunction }) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <div className="hover:bg-canvasSubtle flex cursor-help items-center justify-between gap-2 rounded px-1 py-0.5">
          <span className="text-subtle overflow-hidden text-ellipsis whitespace-nowrap font-mono text-sm">
            {fn.name}
          </span>
          <Link
            href={fn.docsUrl}
            target="_blank"
            onClick={(e) => e.stopPropagation()}
          >
            <RiExternalLinkLine className="h-3.5 w-3.5" />
          </Link>
        </div>
      </TooltipTrigger>
      <TooltipContent side="left" className="max-w-xs">
        {fn.description}
      </TooltipContent>
    </Tooltip>
  );
}
