import Anthropic from '@anthropic-ai/sdk';
import { createScorer } from 'inngest/experimental';
import { z } from 'zod';

import { inngest } from '../../client';
import {
  buildJudgeUserPrompt,
  JUDGE_SYSTEM_PROMPT,
  parseJudgeRelevance,
  submitScoreTool,
} from './insights-judge-prompt';

// Runs off the critical path (deferred), so a capable model is fine.
const JUDGE_MODEL = 'claude-sonnet-4-5';

export const insightsJudgeSchema = z.object({
  question: z.string(),
  sql: z.string(),
  summary: z.string(),
});

export const insightsJudgeScorer = createScorer(
  inngest,
  {
    id: 'score-insights-judge',
    name: 'Insights AI judge',
    schema: insightsJudgeSchema,
  },
  async ({ event, step }) => {
    const { question, sql, summary } = event.data;

    const relevance = await step.run('judge', async () => {
      const client = new Anthropic();
      const response = await client.messages.create({
        model: JUDGE_MODEL,
        max_tokens: 1024,
        system: JUDGE_SYSTEM_PROMPT,
        messages: [
          {
            role: 'user',
            content: buildJudgeUserPrompt({ question, sql, summary }),
          },
        ],
        tools: [submitScoreTool],
        tool_choice: { type: 'tool' as const, name: 'submit_score' },
      });
      return parseJudgeRelevance(response);
    });

    if (relevance === null) return null; // nullish return → createScorer writes no score
    return { name: 'insights_judge_relevance', value: relevance };
  },
);
