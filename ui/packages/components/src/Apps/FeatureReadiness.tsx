import { RiExternalLinkLine } from '@remixicon/react';

import { cn } from '../utils/classNames';

export type FeatureReadinessKind = 'aiMetadata' | 'extendedTraces';

export const featureReadinessTooltipClassName =
  'border-subtle bg-canvasBase text-basis shadow-primary w-[26rem] max-w-[calc(100vw-2rem)] rounded-md border p-0 text-left';

const readinessLabels: Record<FeatureReadinessKind, Record<number, string>> = {
  aiMetadata: {
    1: 'Ready',
    2: 'Disabled in app',
    3: 'OTel provider missing',
    4: 'OTel span processor not added',
  },
  extendedTraces: {
    1: 'Ready',
    2: 'Not enabled in app',
    3: 'Disabled in app',
    4: 'OTel provider missing',
    5: 'OTel span processor not added',
    6: 'OTel provider creation failed',
  },
};

const featureTooltips: Record<
  FeatureReadinessKind,
  { body: string; docsURL: string; title: string }
> = {
  aiMetadata: {
    title: 'AI OpenTelemetry',
    body: "If you're building an AI product, AI OTel can improve the debugging experience.",
    docsURL:
      'https://www.inngest.com/docs/examples/open-telemetry#extract-ai-metadata-from-open-telemetry-spans',
  },
  extendedTraces: {
    title: 'Extended Traces',
    body: 'Extended Traces can add OTel spans to function run traces for deeper debugging.',
    docsURL: 'https://www.inngest.com/docs/platform/monitor/traces#extended-traces',
  },
};

export function featureReadinessLabel(kind: FeatureReadinessKind, reason: number): string {
  return readinessLabels[kind][reason] ?? `Unknown (${reason})`;
}

export function FeatureReadinessDetail({
  kind,
  reason,
}: {
  kind: FeatureReadinessKind;
  reason: number;
}) {
  const isReady = reason === 1;

  return (
    <span className="inline-flex min-w-0 items-center gap-1.5">
      <span
        className={cn(
          'h-2.5 w-2.5 shrink-0 rounded-full',
          isReady ? 'bg-success' : 'bg-surfaceMuted',
        )}
      />
      <span className="truncate">{featureReadinessLabel(kind, reason)}</span>
    </span>
  );
}

export function FeatureReadinessTooltip({ kind }: { kind: FeatureReadinessKind }) {
  const { body, docsURL, title } = featureTooltips[kind];

  return (
    <div>
      <div className="space-y-3 px-4 py-4">
        <div className="text-basis text-base font-medium">{title}</div>
        <p className="text-muted text-sm leading-6">{body}</p>
      </div>
      <a
        className="border-subtle text-link flex items-center gap-2 border-t px-4 py-3 text-sm font-medium"
        href={docsURL}
        target="_blank"
        rel="noreferrer"
      >
        View docs
        <RiExternalLinkLine className="h-4 w-4" />
      </a>
    </div>
  );
}
