import type {
  InsightsColumn,
  InsightsSchemaCatalog,
  InsightsTable,
} from './types';

export function renderInsightsSchemaForPrompt(
  catalog: InsightsSchemaCatalog,
): string {
  const lines = [`<insights_tables version="${escapeAttr(catalog.version)}">`];
  for (const table of catalog.tables) {
    lines.push(renderTable(table));
  }
  lines.push('</insights_tables>');

  return lines.join('\n');
}

function renderTable(table: InsightsTable): string {
  const attrs = renderAttrs({
    name: table.name,
    default_time_column: table.defaultTimeColumn,
    description: table.description,
  });

  const lines = [indent(`<table ${attrs}>`, 1)];
  lines.push(renderTextElement('notes', renderSentences(table.notes), 2));

  lines.push(indent('<columns>', 2));
  for (const column of table.columns) {
    lines.push(renderColumn(column, 3));
  }
  lines.push(indent('</columns>', 2));
  lines.push(indent('</table>', 1));

  return lines.join('\n');
}

function renderColumn(column: InsightsColumn, level: number): string {
  const attrs = renderAttrs({
    name: column.name,
    type: column.type,
    description: column.description,
  });

  const hasColumnBody = Boolean(
    column.notes?.length || column.examples?.length || column.children?.length,
  );
  if (!hasColumnBody) {
    return indent(`<column ${attrs} />`, level);
  }

  const lines = [indent(`<column ${attrs}>`, level)];

  if (column.notes?.length) {
    lines.push(
      renderTextElement('notes', renderSentences(column.notes), level + 1),
    );
  }
  if (column.examples?.length) {
    lines.push(
      renderTextElement('examples', joinDelimited(column.examples), level + 1),
    );
  }
  if (column.children?.length) {
    lines.push(indent('<children>', level + 1));
    for (const child of column.children) {
      lines.push(renderColumn(child, level + 2));
    }
    lines.push(indent('</children>', level + 1));
  }

  lines.push(indent('</column>', level));
  return lines.filter(Boolean).join('\n');
}

function renderSentences(items: string[]): string {
  return items.map(ensureSentence).join(' ');
}

function joinDelimited(items: string[]): string {
  return items.join('; ');
}

function ensureSentence(item: string): string {
  const text = item.trim();
  return /[.!?]$/u.test(text) ? text : `${text}.`;
}

function renderTextElement(tag: string, text: string, level: number): string {
  return indent(`<${tag}>${escapeText(text)}</${tag}>`, level);
}

function renderAttrs(attrs: Record<string, string | undefined>): string {
  return Object.entries(attrs)
    .filter((entry): entry is [string, string] => entry[1] !== undefined)
    .map(([key, value]) => `${key}="${escapeAttr(value)}"`)
    .join(' ');
}

function indent(value: string, level: number): string {
  return `${'  '.repeat(level)}${value}`;
}

function escapeText(value: string): string {
  return value
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;');
}

function escapeAttr(value: string): string {
  return escapeText(value).replaceAll('"', '&quot;');
}
