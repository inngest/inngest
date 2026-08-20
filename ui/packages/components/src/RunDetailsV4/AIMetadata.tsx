import type { ReactNode } from 'react';
import { RiSparkling2Line } from '@remixicon/react';

import { Button } from '../Button/Button';
import { InlineCode } from '../Code';
import { ElementWrapper, TimeElement } from '../DetailsCard/Element';
import { Pill } from '../Pill/Pill';
import { useBooleanFlag } from '../SharedContext/useBooleanFlag';
import { usePathCreator } from '../SharedContext/usePathCreator';
import { KindInngestAI } from '../generated';
import { traceWalk } from './runDetailsUtils';
import type { SpanMetadataInngestAISummary, Trace } from './types';

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

const formatTokenCount = (count: number): string => count.toLocaleString('en-US');

// Costs are USD (see ModelPricing in the backend extractor). Extra fraction
// digits keep sub-cent per-run costs from rounding to $0.00.
const formatEstimatedCost = (cost: number): string =>
  new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: 2,
    maximumFractionDigits: 6,
  }).format(cost);

const pillList = (items: string[]) => (
  <div className="flex flex-wrap gap-1">
    {items.map((item) => (
      <Pill key={item} appearance="outlined">
        {item}
      </Pill>
    ))}
  </div>
);

const aiSummaryRows = (
  values: SpanMetadataInngestAISummary['values']
): { label: string; value: ReactNode }[] => {
  const rows: { label: string; value: ReactNode }[] = [
    { label: 'Total tokens', value: formatTokenCount(values.total_tokens) },
    { label: 'Input tokens', value: formatTokenCount(values.input_tokens) },
    { label: 'Output tokens', value: formatTokenCount(values.output_tokens) },
  ];

  if (values.cache_read_tokens !== undefined) {
    rows.push({ label: 'Cache read tokens', value: formatTokenCount(values.cache_read_tokens) });
  }
  if (values.cache_creation_tokens !== undefined) {
    rows.push({
      label: 'Cache creation tokens',
      value: formatTokenCount(values.cache_creation_tokens),
    });
  }
  if (values.reasoning_tokens !== undefined) {
    rows.push({ label: 'Reasoning tokens', value: formatTokenCount(values.reasoning_tokens) });
  }
  if (values.estimated_cost !== undefined) {
    rows.push({ label: 'Estimated cost', value: formatEstimatedCost(values.estimated_cost) });
  }
  if (values.models?.length) {
    rows.push({ label: 'Models', value: pillList(values.models) });
  }
  if (values.providers?.length) {
    rows.push({ label: 'Providers', value: pillList(values.providers) });
  }

  return rows;
};

/** Displays the run-level AI usage summary the backend aggregates from every AI call in the run. */
export const AISummaryAttrs = ({ metadata }: { metadata: SpanMetadataInngestAISummary }) => {
  const rows = aiSummaryRows(metadata.values);

  return (
    <div className="flex flex-col justify-start gap-2">
      <div className="flex h-11 w-full flex-row items-center justify-between border-none px-4 pt-2">
        <span className="text-basis flex items-center gap-2 text-sm font-medium">AI Usage</span>
      </div>
      <div className="flex flex-row flex-wrap items-center justify-start gap-x-10 gap-y-4 px-4">
        <ElementWrapper label="Updated at">
          <TimeElement date={new Date(metadata.updatedAt)} />
        </ElementWrapper>
      </div>
      <div className="border-muted mt-2 flex max-h-full flex-col gap-2 overflow-hidden border-b pb-4">
        <div className="text-muted bg-canvasSubtle sticky top-0 flex flex-row px-4 py-2 text-xs font-medium leading-tight">
          <div className="w-48">Key</div>
          <div className="">Value</div>
        </div>
        {rows.map(({ label, value }) => (
          <div
            key={`ai-summary-${label}`}
            className="flex flex-row items-start overflow-hidden px-4 pb-2"
          >
            <div className="text-muted w-48 shrink-0 text-sm font-normal leading-tight">
              {label}
            </div>
            <div className="text-basis min-w-0 whitespace-pre-wrap text-sm font-normal leading-tight [overflow-wrap:anywhere]">
              {value}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
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
