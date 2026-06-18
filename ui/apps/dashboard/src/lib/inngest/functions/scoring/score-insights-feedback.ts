import { inngest } from '../../client';
import { buildFeedbackScores, type QueryFeedback } from './insights-feedback';

export const scoreInsightsFeedback = inngest.createFunction(
  {
    id: 'score-insights-feedback',
    name: 'Insights AI feedback scorer',
    triggers: [{ event: 'insights-agent/query.feedback' }],
  },
  async ({ event, step }) => {
    const feedback = event.data as QueryFeedback;
    const scores = buildFeedbackScores(feedback);
    if (scores.length === 0) {
      return { runId: feedback.runId, scoresWritten: 0 };
    }

    await step.run('write-feedback-scores', async () => {
      for (const { name, value } of scores) {
        await inngest.score({ runId: feedback.runId, name, value });
      }
    });

    return { runId: feedback.runId, scoresWritten: scores.length };
  },
);
