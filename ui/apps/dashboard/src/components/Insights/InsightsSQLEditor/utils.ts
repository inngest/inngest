import { format } from 'sql-formatter';

export function getCanRunQuery(query: string, isRunning: boolean): boolean {
  return query.trim() !== '' && !isRunning;
}

export function formatSQL(sql: string): string {
  try {
    return format(sql, {
      language: 'sql',
      tabWidth: 2,
      keywordCase: 'upper',
      linesBetweenQueries: 2,
    });
  } catch (error) {
    console.error('SQL formatting error:', error);
    return sql; // Return original SQL if formatting fails
  }
}
