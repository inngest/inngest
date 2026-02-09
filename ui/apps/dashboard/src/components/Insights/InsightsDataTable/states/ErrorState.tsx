import { Banner } from '@inngest/components/Banner/Banner';
import { Button } from '@inngest/components/Button';

import { useSQLEditorInstance } from '../../InsightsSQLEditor/SQLEditorContext';
import { useInsightsAIHelper } from '../../InsightsAIHelperContext';
import { useInsightsStateMachineContext } from '../../InsightsStateMachineContext/InsightsStateMachineContext';
import { DiagnosticsBanner } from '../DiagnosticsBanner';
import { handleFixWithAI } from '../ai';
const FALLBACK_ERROR = 'Something went wrong. Please try again.';

export function ErrorState() {
  const { error, query } = useInsightsStateMachineContext();
  const aiHelper = useInsightsAIHelper();
  const editorInstance = useSQLEditorInstance();
  if (!editorInstance) {
    throw new Error('InsightsSQLEditor must be used within ErrorState');
  }

  const errorMessage = error ? pruneGraphQLError(error) : FALLBACK_ERROR;

  if (error) {
    return (
      <Banner
        cta={
          <div className="flex items-center gap-2">
            {aiHelper && (
              <Button
                appearance="solid"
                kind="danger"
                label="Fix with AI"
                onClick={handleFixWithAI(aiHelper, errorMessage, query)}
              />
            )}
          </div>
        }
        severity="error"
      >
        <div className="whitespace-pre-wrap">{errorMessage}</div>
      </Banner>
    );
  } else {
    return <DiagnosticsBanner />;
  }
}

function pruneGraphQLError(error: Error) {
  return error.message.replace(/^\[GraphQL\] /, '');
}
