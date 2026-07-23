import { Link } from '@inngest/components/Link';
import {
  RiBarChartBoxLine,
  RiFlowChart,
  RiPriceTag3Line,
  RiTimeLine,
} from '@remixicon/react';

import { FeatureEmptyState } from '@/components/FeatureEmptyState/FeatureEmptyState';
import { trackEmptyStateDocsLinkOpened } from '@/utils/analyticsEvents';

// TODO: point this at an actual step-by-step guide once one exists — this is
// currently the Extended Traces reference doc, not a walkthrough.
const DOCS_URL =
  'https://www.inngest.com/docs/reference/typescript/v4/extended-traces?ref=app-empty-ai-overview';

const example = `import { openai } from '@ai-sdk/openai';
import { generateText } from 'ai';

export default inngest.createFunction(
  { id: 'summarize-doc' },
  { event: 'docs/summarize.requested' },
  async ({ event, step }) => {
    // Once Extended Traces (OpenTelemetry) is enabled, gen_ai.* attributes
    // are captured automatically from AI SDK / AI Gateway calls made inside
    // a step — nothing else to instrument.
    const { text } = await step.run('summarize', () =>
      generateText({
        model: openai('gpt-4o'),
        prompt: \`Summarize: \${event.data.content}\`,
      }),
    );

    return { summary: text };
  },
);`;

const prompt = `Read the docs about Inngest's Extended Traces (OpenTelemetry) @https://www.inngest.com/docs-markdown/reference/typescript/v4/extended-traces and show me how to enable OpenTelemetry in my Inngest functions so gen_ai.* AI call metadata shows up in the AI Overview.`;

const valueProps = [
  {
    icon: RiBarChartBoxLine,
    title: 'AI health at a glance',
    description: 'Calls, tokens, cost, and latency in one view after every deploy.',
  },
  {
    icon: RiPriceTag3Line,
    title: 'See where spend goes',
    description: 'Break down usage by model and by function.',
  },
  {
    icon: RiTimeLine,
    title: 'Watch trends over time',
    description: 'Token and cost trends without building your own dashboard.',
  },
  {
    icon: RiFlowChart,
    title: 'Jump to the source',
    description: 'Drill from any chart into the runs and functions behind it.',
  },
];

export function AIOverviewEmptyState({
  compact = false,
  className,
}: {
  compact?: boolean;
  className?: string;
}) {
  return (
    <FeatureEmptyState
      feature="ai-overview"
      title="Get Started"
      description={
        <>
          To display data here, have your AI function emit metadata using{' '}
          <code className="bg-canvasSubtle rounded px-1 text-xs">gen_ai.*</code> and configure
          OpenTelemetry.{' '}
          <Link
            className="inline"
            href={DOCS_URL}
            target="_blank"
            onClick={() => trackEmptyStateDocsLinkOpened({ feature: 'ai-overview' })}
          >
            Learn more.
          </Link>
        </>
      }
      docsUrl={DOCS_URL}
      onDocsLinkClick={() =>
        trackEmptyStateDocsLinkOpened({ feature: 'ai-overview' })
      }
      compact={compact}
      className={className}
      valueProps={valueProps}
      prompt={{
        description:
          'Copy this prompt to learn about this feature and start emitting AI metadata',
        content: prompt,
      }}
      example={{
        description:
          'Once Extended Traces (OpenTelemetry) is enabled, AI SDK and AI Gateway calls inside a step are captured automatically',
        tabs: [
          {
            title: 'Code',
            content: example,
            readOnly: true,
            language: 'typescript',
          },
        ],
      }}
    />
  );
}
