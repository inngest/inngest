import type { SQLCompletionConfig } from '@inngest/components/SQLEditor/types';

import { SUPPORTED_FUNCTIONS } from './functions';

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

const TABLES = ['events'] as const;

export const SQL_COMPLETION_CONFIG: SQLCompletionConfig = {
  columns: [],
  keywords: KEYWORDS,
  functions: SUPPORTED_FUNCTIONS,
  tables: TABLES,
};
