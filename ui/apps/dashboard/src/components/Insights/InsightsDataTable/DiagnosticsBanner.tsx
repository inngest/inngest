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
import {
  RiCircleFill,
  RiCloseCircleLine,
  RiErrorWarningLine,
  RiInformationLine,
  RiSparkling2Line,
  type RemixiconComponentType,
} from '@remixicon/react';
import { cn } from '@inngest/components/utils/classNames';

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
      title={bannerTitleForSeverity(maxSeverity)}
      cta={
        <div className="flex items-center gap-2">
          {aiHelper && (
            <Button
              appearance="outlined"
              kind="secondary"
              icon={<RiSparkling2Line />}
              iconSide="left"
              label="Fix with AI"
              onClick={handleFixWithAI(
                aiHelper,
                diagnostics
                  .map((diag) => formatDiagnosticMessage(model, diag))
                  .join('\n'),
                query,
              )}
            />
          )}
        </div>
      }
      severity="none"
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
  const Icon = diagnosticIconForSeverity(diag.severity);
  return (
    <div
      className={cn(
        colorForSeverity(diag.severity),
        'px-4 pt-1 flex flex-row gap-1 pb-1 items-center',
      )}
    >
      <Icon className="h-4" />
      <span className="font-mono">{formatDiagnosticMessage(model, diag)}</span>
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

  return `[${startPos.lineNumber}:${
    startPos.column
  }] ${formatDiagnosticSeverity(diag.severity)}: ${diag.message} (${
    diag.position.context
  })`;
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
      return 'ERROR';
    case 'warning':
      return 'WARNING';
    case 'info':
    default:
      return 'INFO';
  }
}

function bannerTitleForSeverity(
  severity: InsightsDiagnosticSeverity,
): React.ReactNode {
  return (
    <div className="flex items-center gap-1">
      <RiCircleFill className={cn(colorForSeverity(severity), 'h-4')} />
      <span className="font-mono">{bannerLabelForSeverity(severity)}</span>
    </div>
  );
}

function bannerLabelForSeverity(severity: InsightsDiagnosticSeverity): string {
  switch (severity) {
    case 'error':
      return 'QUERY FAILED TO RUN';
    case 'warning':
      return 'QUERY RAN WITH WARNINGS';
    default:
      return 'QUERY RAN WITH INFO';
  }
}

function colorForSeverity(severity: InsightsDiagnosticSeverity): string {
  switch (severity) {
    case 'error':
      return 'text-red-400 dark:text-red-500';
    case 'warning':
      return 'text-amber-600 dark:text-amber-500';
    case 'info':
    default:
      return 'text-blue-400 dark:text-blue-500';
  }
}

function diagnosticIconForSeverity(
  severity: InsightsDiagnosticSeverity,
): RemixiconComponentType {
  switch (severity) {
    case 'error':
      return RiCloseCircleLine;
    case 'warning':
      return RiErrorWarningLine;
    case 'info':
    default:
      return RiInformationLine;
  }
}
