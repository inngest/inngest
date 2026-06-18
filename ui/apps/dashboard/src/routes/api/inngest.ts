import { serve } from 'inngest/edge';
import { inngest } from '@/lib/inngest/client';
import { runInsightsAgent } from '@/lib/inngest/functions/run-insights';
import { insightsJudgeScorer } from '@/lib/inngest/functions/scoring/insights-judge';
import { scoreInsightsFeedback } from '@/lib/inngest/functions/scoring/score-insights-feedback';
import { createFileRoute } from '@tanstack/react-router';

const handler = serve({
  client: inngest,
  functions: [runInsightsAgent, insightsJudgeScorer, scoreInsightsFeedback],
});

export const Route = createFileRoute('/api/inngest')({
  server: {
    handlers: {
      GET: async ({ request }) => handler(request),
      POST: async ({ request }) => handler(request),
      PUT: async ({ request }) => handler(request),
    },
  },
});
