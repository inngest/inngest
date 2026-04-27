/**
 * SQL string literal quoting: SQL escapes a single quote by doubling it.
 * See https://www.postgresql.org/docs/current/sql-syntax-lexical.html.
 */
function quoteSqlString(value: string): string {
  return `'${value.replace(/'/g, "''")}'`;
}

/**
 * URL to open the Insights explorer pre-populated with a query for the given
 * experiment's steps. Kept as a helper so the SQL escaping lives in one place.
 */
export function experimentInsightsUrl(
  envSlug: string,
  experimentName: string,
): string {
  const sql = `SELECT * FROM isteps WHERE \`inngest.experiment.values.experiment_name\` = ${quoteSqlString(
    experimentName,
  )} ORDER BY started_at DESC`;
  const params = new URLSearchParams({ sql });
  return `/env/${encodeURIComponent(envSlug)}/insights?${params.toString()}`;
}
