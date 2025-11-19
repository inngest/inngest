export function getCanRunQuery(query: string, isRunning: boolean): boolean {
  return query.trim() !== "" && !isRunning;
}
