import { useMemo } from 'react';
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
import { RiInformationLine } from '@remixicon/react';

import { INSIGHTS_TABLES, type InsightsSchemaTable } from './insightsSchema';
import { Section } from './Section';

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
          <span className="text-light inline-flex cursor-help items-center">
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
    <Section className="h-full" title="Schema">
      <div className="flex h-full flex-col gap-1 overflow-auto p-2">
        {INSIGHTS_TABLES.map((table) => (
          <SchemaViewer
            key={table.name}
            node={toSchemaNode(table)}
            renderAdornment={renderAdornment}
          />
        ))}
      </div>
    </Section>
  );
}
