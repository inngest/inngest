import {
  ContextualBanner,
  type Severity,
} from '@inngest/components/Banner/Banner';
import { Button } from '@inngest/components/Button';

import { useSQLEditorInstance } from '../InsightsSQLEditor/SQLEditorContext';
import { useInsightsAIHelper } from '../InsightsAIHelperContext';
import { useInsightsStateMachineContext } from '../InsightsStateMachineContext/InsightsStateMachineContext';
import type {
  InsightsDiagnostic,
  InsightsDiagnosticSeverity,
} from '../InsightsStateMachineContext/types';
import { type SQLEditorModel } from '@inngest/components/SQLEditor/SQLEditor';
import { handleFixWithAI } from './ai';

export function DiagnosticsBanner() {
  const { query, data } = useInsightsStateMachineContext();
  const aiHelper = useInsightsAIHelper();
  const editorInstance = useSQLEditorInstance();
  if (!editorInstance) {
    throw new Error('InsightsSQLEditor must be used within ErrorState');
  }

  const { editorRef } = editorInstance;

  const diagnostics = data?.diagnostics ?? [];
  const model = editorRef.current?.getModel() ?? null;

  const maxSeverity = maxDiagnosticsSeverity(diagnostics);

  return (
    <ContextualBanner
      title={`${formatDiagnosticSeverity(maxSeverity)} executing query`}
      cta={
        <div className="flex items-center gap-2">
          {aiHelper && (
            <Button
              appearance="solid"
              kind="secondary"
              label="Fix with AI"
              onClick={handleFixWithAI(
                aiHelper,
                diagnostics.map((d) => d.message).join('\n'),
                query,
              )}
            />
          )}
        </div>
      }
      severity={severityToBannerSeverity(maxSeverity)}
      bodySeverity="none"
    >
      <div className="flex flex-col gap-1 pb-1">
        {diagnostics.map((diag, index) => (
          <DiagnosticBannerEntry key={index} model={model} diag={diag} />
        ))}
      </div>
    </ContextualBanner>
  );
}

function DiagnosticBannerEntry({
  model,
  diag,
}: {
  model: SQLEditorModel | null;
  diag: InsightsDiagnostic;
}) {
  return (
    <div className="px-4 pt-1">
      <span>{formatDiagnosticMessage(model, diag)}</span>
    </div>
  );
}

function formatDiagnosticMessage(
  model: SQLEditorModel | null,
  diag: InsightsDiagnostic,
) {
  if (!diag.position || !model) {
    return `${diag.message}`;
  }

  const startPos = model.getPositionAt(diag.position.start);

  return `${diag.severity} ${startPos.lineNumber}:${startPos.column} ${diag.message} (${diag.position.context})`;
}

function maxDiagnosticsSeverity(
  diagnostics: InsightsDiagnostic[],
): InsightsDiagnosticSeverity {
  let maxSeverity: Severity = 'info';

  diagnostics.forEach((diag) => {
    if (diag.severity === 'error') {
      maxSeverity = 'error';
    } else if (diag.severity === 'warning' && maxSeverity !== 'error') {
      maxSeverity = 'warning';
    }
  });

  return maxSeverity;
}

function formatDiagnosticSeverity(
  severity: InsightsDiagnosticSeverity,
): string {
  switch (severity) {
    case 'error':
      return 'Error';
    case 'warning':
      return 'Warning';
    case 'info':
    default:
      return 'Info';
  }
}

function severityToBannerSeverity(
  severity: InsightsDiagnosticSeverity,
): Severity {
  switch (severity) {
    case 'error':
      return 'error';
    case 'warning':
      return 'warning';
    case 'info':
    default:
      return 'info';
  }
}
