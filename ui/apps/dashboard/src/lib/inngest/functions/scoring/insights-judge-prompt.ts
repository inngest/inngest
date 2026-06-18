import { z } from 'zod';

import judgeSystemPrompt from './insights-judge-system.md?raw';

export interface JudgeInput {
  question: string;
  sql: string;
  summary: string;
}

export const JUDGE_SYSTEM_PROMPT = judgeSystemPrompt;

export const SubmitScoreParams = z.object({
  relevance: z
    .number()
    .describe(
      'How well the response answers the question, from 0 (irrelevant/wrong) to 1 (fully answers).',
    ),
  reasoning: z.string().describe('One-sentence justification for the score.'),
});

export const submitScoreTool = {
  name: 'submit_score' as const,
  description:
    'Record the relevance score for the AI response. Call exactly once.',
  input_schema: z.toJSONSchema(SubmitScoreParams) as {
    type: 'object';
    [k: string]: unknown;
  },
};

export function buildJudgeUserPrompt({
  question,
  sql,
  summary,
}: JudgeInput): string {
  return [
    "User's question:",
    question,
    '',
    'SQL the assistant produced:',
    '```sql',
    sql,
    '```',
    '',
    "Assistant's summary:",
    summary,
  ].join('\n');
}

function clamp01(n: number): number {
  return Math.min(1, Math.max(0, n));
}

interface MaybeToolUse {
  type?: string;
  name?: string;
  input?: { relevance?: unknown };
}

/**
 * Returns null (not 0) on malformed/absent output,
 * so the scorer no-ops rather than writing a misleading score.
 */
export function parseJudgeRelevance(
  response: { content?: unknown } | null | undefined,
): number | null {
  const blocks = response?.content;
  if (!Array.isArray(blocks)) return null;
  const toolUse = (blocks as MaybeToolUse[]).find(
    (b) => b?.type === 'tool_use' && b?.name === 'submit_score',
  );
  const relevance = toolUse?.input?.relevance;
  if (typeof relevance !== 'number' || Number.isNaN(relevance)) return null;
  return clamp01(relevance);
}
