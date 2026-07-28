import { RiSparkling2Line } from '@remixicon/react';

import { Button } from '../Button/Button';
import { InlineCode } from '../Code';
import { KindInngestAI } from '../generated';
import { traceWalk } from './runDetailsUtils';
import type { Trace } from './types';

const AI_METADATA_DOCS_URL =
  'https://www.inngest.com/docs/examples/open-telemetry?ref=app-run-metadata';

const CUSTOM_METADATA_DOCS_URL =
  'https://www.inngest.com/docs/reference/typescript/v4/functions/metadata?ref=app-run-metadata';

const links = [
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

export const AIMetadataNudge = () => (
  <div className="px-4 pb-4 pt-2">
    <div className="border-subtle bg-canvasBase flex items-start gap-3 rounded-md border p-4">
      <RiSparkling2Line className="text-muted mt-0.5 h-4 w-4 shrink-0" />
      <div className="flex min-w-0 grow flex-col gap-1">
        <p className="text-basis text-sm font-medium">Get more from metadata</p>
        <p className="text-muted text-sm leading-relaxed">
          Track models, tokens, and cost with AI metadata, or use{' '}
          <InlineCode>step.metadata()</InlineCode> to attach your own key/values.
        </p>
        <div className="mt-2 flex flex-wrap items-center gap-2">
          {links.map((link) => (
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
        </div>
      </div>
    </div>
  </div>
);
