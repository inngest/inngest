import type { SchemaNode } from '@inngest/components/SchemaViewer/types';

import type {
  InsightsColumn,
  InsightsSchemaCatalog,
  InsightsSchemaMetadata,
  InsightsTable,
} from './types';

export type InsightsSchemaExplorerRender = {
  entries: Array<{ key: string; node: SchemaNode }>;
  metadataByPath: Record<string, InsightsSchemaMetadata>;
};

export function renderInsightsSchemaForExplorer(
  catalog: InsightsSchemaCatalog,
): InsightsSchemaExplorerRender {
  const metadataByPath: Record<string, InsightsSchemaMetadata> = {};
  const entries = catalog.tables.map((table) => {
    const node = tableToNode(table);
    metadataByPath[node.path] = tableToMetadata(table);
    addColumnMetadata(metadataByPath, table.name, table.columns);

    return {
      key: table.name,
      node,
    };
  });

  return { entries, metadataByPath };
}

function tableToNode(table: InsightsTable): SchemaNode {
  return {
    kind: 'table',
    name: table.name,
    path: table.name,
    children: table.columns.map((column) => columnToNode(table.name, column)),
  };
}

function columnToNode(parentPath: string, column: InsightsColumn): SchemaNode {
  const path = `${parentPath}.${column.name}`;

  if (column.children?.length) {
    return {
      kind: 'object',
      name: column.name,
      path,
      type: column.type,
      children: column.children.map((child) => columnToNode(path, child)),
    };
  }

  return {
    kind: 'value',
    name: column.name,
    path,
    type: column.type,
  };
}

function tableToMetadata(table: InsightsTable): InsightsSchemaMetadata {
  return {
    title: table.name,
    description: table.description,
    notes: table.notes,
  };
}

function addColumnMetadata(
  metadataByPath: Record<string, InsightsSchemaMetadata>,
  parentPath: string,
  columns: InsightsColumn[],
): void {
  for (const column of columns) {
    const path = `${parentPath}.${column.name}`;
    metadataByPath[path] = columnToMetadata(column);
    if (column.children?.length) {
      addColumnMetadata(metadataByPath, path, column.children);
    }
  }
}

function columnToMetadata(column: InsightsColumn): InsightsSchemaMetadata {
  return {
    title: column.name,
    description: column.description,
    notes: column.notes,
    examples: column.examples,
  };
}
