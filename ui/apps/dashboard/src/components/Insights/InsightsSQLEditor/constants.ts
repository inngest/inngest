import type { SQLCompletionConfig } from '@inngest/components/SQLEditor/createSQLCompletionProvider';

const KEYWORDS = [
  'AND',
  'AS',
  'ASC',
  'BETWEEN',
  'DESC',
  'DISTINCT',
  'FALSE',
  'FROM',
  'GROUP BY',
  'IS',
  'LIKE',
  'LIMIT',
  'NOT',
  'NULL',
  'OFFSET',
  'OR',
  'ORDER BY',
  'SELECT',
  'TRUE',
  'WHERE',
] as const;

const FUNCTIONS = [
  { name: 'AVG', signature: 'AVG(${1:column})' },
  { name: 'COUNT', signature: 'COUNT(${1:column})' },
  { name: 'MAX', signature: 'MAX(${1:column})' },
  { name: 'MIN', signature: 'MIN(${1:column})' },
  { name: 'SUM', signature: 'SUM(${1:column})' },
] as const;

const TABLES = ['events'] as const;

export const SQL_COMPLETION_CONFIG: SQLCompletionConfig = {
  columns: [],
  keywords: KEYWORDS,
  functions: FUNCTIONS,
  tables: TABLES,
};
