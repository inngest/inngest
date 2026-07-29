import { clickhouse, formatDialect } from 'sql-formatter';

export function getCanRunQuery(query: string, isRunning: boolean): boolean {
  return query.trim() !== '' && !isRunning;
}

export function formatSQL(sql: string): string {
  try {
    return formatDialect(sql, {
      dialect: clickhouse,
      tabWidth: 2,
      linesBetweenQueries: 2,
    });
  } catch (error) {
    console.error('SQL formatting error:', error);
    return sql; // Return original SQL if formatting fails
  }
}
