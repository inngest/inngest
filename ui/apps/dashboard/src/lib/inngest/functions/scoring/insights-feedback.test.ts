import { describe, expect, it } from 'vitest';

import { buildFeedbackScores } from './insights-feedback';

describe('buildFeedbackScores', () => {
  it('maps a successful execution to executed_ok, returned_rows, and user_accepted', () => {
    expect(
      buildFeedbackScores({
        runId: 'r1',
        executedOk: true,
        rowCount: 5,
        userEdited: false,
      }),
    ).toEqual([
      { name: 'insights_executed_ok', value: true },
      { name: 'insights_returned_rows', value: true },
      { name: 'insights_user_accepted', value: true },
    ]);
  });

  it('treats zero rows as returned_rows=false and an edit as not accepted', () => {
    expect(
      buildFeedbackScores({
        runId: 'r1',
        executedOk: false,
        rowCount: 0,
        userEdited: true,
      }),
    ).toEqual([
      { name: 'insights_executed_ok', value: false },
      { name: 'insights_returned_rows', value: false },
      { name: 'insights_user_accepted', value: false },
    ]);
  });

  it('maps an explicit save to query_saved', () => {
    expect(buildFeedbackScores({ runId: 'r1', saved: true })).toEqual([
      { name: 'insights_query_saved', value: true },
    ]);
  });

  it('maps a fix-with-AI click to fix_with_ai_requested', () => {
    expect(buildFeedbackScores({ runId: 'r1', fixWithAi: true })).toEqual([
      { name: 'insights_fix_with_ai_requested', value: true },
    ]);
  });

  it('only emits scores for fields that are present', () => {
    expect(buildFeedbackScores({ runId: 'r1' })).toEqual([]);
  });
});
