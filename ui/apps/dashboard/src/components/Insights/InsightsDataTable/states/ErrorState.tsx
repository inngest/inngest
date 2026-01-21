import { Banner } from '@inngest/components/Banner/Banner';
import { Button } from '@inngest/components/Button';
import * as Sentry from '@sentry/tanstackstart-react';

import { useInsightsAIHelper } from '../../InsightsAIHelperContext';
import { useInsightsStateMachineContext } from '../../InsightsStateMachineContext/InsightsStateMachineContext';

const FALLBACK_ERROR = 'Something went wrong. Please try again.';

export function ErrorState() {
  const { error, query } = useInsightsStateMachineContext();
  const aiHelper = useInsightsAIHelper();

  const errorMessage = error ? pruneGraphQLError(error) : FALLBACK_ERROR;

  const handleFixWithAI = async () => {
    if (!aiHelper) return;

    try {
      const prompt =
        'The query execution failed with the following error:\n\n' +
        errorMessage +
        '\n\nThe query that failed was:\n\n```sql\n' +
        query +
        '\n```\n\nPlease provide a corrected version of the query that fixes this error.';
      await aiHelper.openAIHelperWithPrompt(prompt);
    } catch (error) {
      Sentry.captureException(error);
      // Error is logged but doesn't prevent user interaction
    }
  };

  return (
    <Banner
      cta={
        <div className="flex items-center gap-2">
          {aiHelper && (
            <Button
              appearance="solid"
              kind="danger"
              label="Fix with AI"
              onClick={handleFixWithAI}
            />
          )}
        </div>
      }
      severity="error"
    >
      {errorMessage}
    </Banner>
  );
}

function pruneGraphQLError(error: Error) {
  return error.message.replace(/^\[GraphQL\] /, '');
}
