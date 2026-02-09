import * as Sentry from '@sentry/tanstackstart-react';
import { type InsightsAIHelperContextValue } from '../InsightsAIHelperContext';
export function handleFixWithAI(
  aiHelper: InsightsAIHelperContextValue | null,
  fixMessage: string,
  query: string,
) {
  return async () => {
    if (!aiHelper) return;

    try {
      const prompt =
        'The query execution failed with the following error:\n\n' +
        fixMessage +
        '\n\nThe query that failed was:\n\n```sql\n' +
        query +
        '\n```\n\nPlease provide a corrected version of the query that fixes this error.';
      await aiHelper.openAIHelperWithPrompt(prompt);
    } catch (error) {
      Sentry.captureException(error);
      // Error is logged but doesn't prevent user interaction
    }
  };
}
