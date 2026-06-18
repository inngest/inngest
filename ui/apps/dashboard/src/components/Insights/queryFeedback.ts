export interface QueryFeedbackPayload {
  runId: string;
  executedOk?: boolean;
  rowCount?: number;
  userEdited?: boolean;
  saved?: boolean;
  fixWithAi?: boolean;
}

function normalizeSqlForCompare(sql: string): string {
  return sql.replace(/;\s*$/, '').replace(/\s+/g, ' ').trim().toLowerCase();
}

// Heuristic: the editor SQL is formatSQL-formatted while the suggestion is raw,
// so compare on a normalized form. No suggestion → treat as edited.
export function sqlWasEdited(
  suggested: string | undefined,
  executed: string,
): boolean {
  if (suggested === undefined) return true;
  return normalizeSqlForCompare(suggested) !== normalizeSqlForCompare(executed);
}

export async function postQueryFeedback(
  payload: QueryFeedbackPayload,
): Promise<void> {
  try {
    await fetch('/api/query-feedback', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    });
  } catch {
    // Best-effort: feedback scoring must never disrupt the user.
  }
}
