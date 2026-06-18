import { describe, expect, it } from 'vitest';

import { parseJudgeRelevance } from './insights-judge-prompt';

const toolUse = (input: unknown) => ({
  content: [{ type: 'tool_use', name: 'submit_score', input }],
});

describe('parseJudgeRelevance', () => {
  it('reads relevance from the submit_score tool call', () => {
    expect(parseJudgeRelevance(toolUse({ relevance: 0.8 }))).toBe(0.8);
  });

  it('clamps to [0, 1]', () => {
    expect(parseJudgeRelevance(toolUse({ relevance: 1.5 }))).toBe(1);
    expect(parseJudgeRelevance(toolUse({ relevance: -0.4 }))).toBe(0);
  });

  it('returns null when there is no submit_score tool call', () => {
    expect(
      parseJudgeRelevance({ content: [{ type: 'text', text: 'hi' }] }),
    ).toBe(null);
  });

  it('returns null when relevance is missing or not a number', () => {
    expect(parseJudgeRelevance(toolUse({ reasoning: 'x' }))).toBe(null);
    expect(parseJudgeRelevance(toolUse({ relevance: 'high' }))).toBe(null);
    expect(parseJudgeRelevance(toolUse({ relevance: NaN }))).toBe(null);
  });

  it('returns null defensively for empty or malformed responses', () => {
    expect(parseJudgeRelevance({ content: [] })).toBe(null);
    expect(parseJudgeRelevance(null)).toBe(null);
    expect(parseJudgeRelevance(undefined)).toBe(null);
  });
});
