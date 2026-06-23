export interface QueryFeedback {
  runId: string;
  executedOk?: boolean;
  rowCount?: number;
  userEdited?: boolean;
  saved?: boolean;
  fixWithAi?: boolean;
}

export function buildFeedbackScores(
  feedback: QueryFeedback,
): { name: string; value: boolean }[] {
  const scores: { name: string; value: boolean }[] = [];
  if (feedback.executedOk !== undefined) {
    scores.push({ name: 'insights_executed_ok', value: feedback.executedOk });
  }
  if (feedback.rowCount !== undefined) {
    scores.push({
      name: 'insights_returned_rows',
      value: feedback.rowCount > 0,
    });
  }
  if (feedback.userEdited !== undefined) {
    scores.push({
      name: 'insights_user_accepted',
      value: !feedback.userEdited,
    });
  }
  if (feedback.saved !== undefined) {
    scores.push({ name: 'insights_query_saved', value: feedback.saved });
  }
  if (feedback.fixWithAi !== undefined) {
    scores.push({
      name: 'insights_fix_with_ai_requested',
      value: feedback.fixWithAi,
    });
  }
  return scores;
}
