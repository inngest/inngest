import { Banner } from '@inngest/components/Banner/Banner';
import { Button } from '@inngest/components/Button';
import * as Sentry from '@sentry/tanstackstart-react';

import { useSQLEditorInstance } from '../../InsightsSQLEditor/SQLEditorContext';
import { useInsightsAIHelper } from '../../InsightsAIHelperContext';
import { useInsightsStateMachineContext } from '../../InsightsStateMachineContext/InsightsStateMachineContext';
import type { InsightsFetchResult } from '../../InsightsStateMachineContext/types';
import { type SQLEditorInstance } from '@inngest/components/SQLEditor/SQLEditor';

const FALLBACK_ERROR = 'Something went wrong. Please try again.';

export function ErrorState() {
  const { error, query, data } = useInsightsStateMachineContext();
  const aiHelper = useInsightsAIHelper();
  const editorInstance = useSQLEditorInstance();
  if (!editorInstance) {
    throw new Error('InsightsSQLEditor must be used within ErrorState');
  }

  const { editorRef } = editorInstance;

  const errorMessage = error
    ? pruneGraphQLError(error)
    : data?.diagnostics.find((x) => x.severity === 'ERROR') !== undefined
    ? formatDiagnosticMessage(editorRef.current, data.diagnostics)
    : FALLBACK_ERROR;

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
      <div className="whitespace-pre-wrap">{errorMessage}</div>
    </Banner>
  );
}

function pruneGraphQLError(error: Error) {
  return error.message.replace(/^\[GraphQL\] /, '');
}

function formatDiagnosticMessage(
  editor: SQLEditorInstance | null,
  diagnostics: InsightsFetchResult['diagnostics'],
) {
  const model = editor?.getModel();
  return diagnostics
    .filter((d) => d.severity === 'ERROR')
    .map((diag) => {
      if (!diag.position || !model) {
        return `${diag.message}`;
      }

      const startPos = model.getPositionAt(diag.position.start);

      return `${startPos.lineNumber}:${startPos.column}\t  ${diag.message} (${diag.position.context})`;
    })
    .join('\n');
}
