// Pure helpers for the immediate (in-run) Insights AI success scores.
// Kept side-effect-free so they're unit-testable; the function handler emits
// the results via `inngest.score()`.

const OPUS_VARIANT = 'claude-opus-4-8';

/** Did the agent emit any SQL at all (vs. a clarification / failure)? */
export function producedSql(sql: string | undefined): boolean {
  return typeof sql === 'string' && sql.trim().length > 0;
}

/**
 * Cheap static check: a single SELECT (or WITH … SELECT CTE) statement.
 * Not a full parse — just enough to flag obviously non-runnable output.
 */
export function isParseableSelect(sql: string | undefined): boolean {
  if (!producedSql(sql)) return false;
  const trimmed = sql!.trim().replace(/;\s*$/, ''); // drop one trailing semicolon
  if (trimmed.includes(';')) return false; // any remaining ; => multiple statements
  return /^(select|with)\b/i.test(trimmed);
}

/** Companion variant tag (boolean, since scores can't carry a string). */
export function isOpusVariant(variant: string): boolean {
  return variant === OPUS_VARIANT;
}

export interface ImmediateScoreInput {
  sql: string | undefined;
  variant: string;
}

/** The run-scoped immediate scores for one Insights AI run. */
export function buildImmediateScores({
  sql,
  variant,
}: ImmediateScoreInput): { name: string; value: boolean }[] {
  return [
    { name: 'insights_produced_sql', value: producedSql(sql) },
    { name: 'insights_sql_parseable', value: isParseableSelect(sql) },
    { name: 'isOpus', value: isOpusVariant(variant) },
  ];
}
