import { RiSparkling2Line } from '@remixicon/react';

import { Button } from '../Button/Button';
import { InlineCode } from '../Code';
import { useBooleanFlag } from '../SharedContext/useBooleanFlag';
import { usePathCreator } from '../SharedContext/usePathCreator';
import { KindInngestAI } from '../generated';
import { traceWalk } from './runDetailsUtils';
import type { Trace } from './types';

const AI_METADATA_DOCS_URL =
  'https://www.inngest.com/docs/examples/ai-metadata-quickstart?ref=app-run-metadata';

const CUSTOM_METADATA_DOCS_URL =
  'https://www.inngest.com/docs/reference/typescript/v4/functions/metadata?ref=app-run-metadata';

const docsLinks = [
  { label: 'Set up AI metadata', href: AI_METADATA_DOCS_URL },
  { label: 'Add custom metadata', href: CUSTOM_METADATA_DOCS_URL },
];

export const traceHasAIMetadata = (trace: Trace | undefined): boolean => {
  if (!trace) {
    return false;
  }

  let found = false;
  traceWalk(trace, (span) => {
    if (span.metadata?.some((md) => md.kind === KindInngestAI)) {
      found = true;
    }
  });

  return found;
};

export const AIMetadataNudge = () => {
  const { booleanFlag } = useBooleanFlag();
  const { pathCreator } = usePathCreator();
  const { value: aiOverviewEnabled } = booleanFlag('ai-overview-dashboard', false);
  const aiOverviewPath = aiOverviewEnabled ? pathCreator.aiOverview?.() : null;

  return (
    <div className="px-4 pb-4 pt-2">
      <div className="border-subtle bg-canvasBase flex items-start gap-3 rounded-md border p-4">
        <RiSparkling2Line className="text-muted mt-0.5 h-4 w-4 shrink-0" />
        <div className="flex min-w-0 grow flex-col gap-1">
          <p className="text-basis text-sm font-medium">Get more from metadata</p>
          <p className="text-muted text-sm leading-relaxed">
            With OpenTelemetry enabled, Inngest captures model, tokens, cost, and latency from AI
            calls automatically. You can also attach your own key/values with{' '}
            <InlineCode>step.metadata()</InlineCode>.
          </p>
          <div className="mt-2 flex flex-wrap items-center gap-2">
            {docsLinks.map((link) => (
              <Button
                key={link.href}
                kind="secondary"
                appearance="outlined"
                size="small"
                label={link.label}
                href={link.href}
                target="_blank"
                rel="noopener noreferrer"
              />
            ))}
            {aiOverviewPath && (
              <Button
                kind="secondary"
                appearance="outlined"
                size="small"
                label="View AI Overview"
                href={aiOverviewPath}
              />
            )}
          </div>
        </div>
      </div>
    </div>
  );
};
