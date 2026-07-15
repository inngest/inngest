import { useEffect, useRef } from 'react';
import { InlineCode } from '@inngest/components/Code';
import CommandBlock from '@inngest/components/CodeBlock/CommandBlock';
import { Link } from '@inngest/components/Link';
import {
  RiCheckboxCircleLine,
  RiFilterLine,
  RiLineChartLine,
  RiQuestionLine,
} from '@remixicon/react';

import { analytics } from '@/utils/segment';

const DOCS_URL =
  'https://www.inngest.com/docs/features/inngest-functions/steps-workflows/scoring?ref=app-empty-scores';

const example = `import { Inngest } from 'inngest';
import { scoreMiddleware } from 'inngest/experimental';

const inngest = new Inngest({
  id: 'my-app',
  middleware: [scoreMiddleware()],
});

export default inngest.createFunction(
  { id: 'grade-response' },
  { event: 'ai/response.generated' },
  async ({ event, step }) => {
    const result = await step.run('agent-result', () => {/* ... */});

    // Numeric and boolean scores both chart on the Scores tab
    await step.score('quality', { name: 'response_quality', value: result.qualityScore });
    await step.score('budget', { name: 'in_budget', value: result.tokens_used < TOKEN_BUDGET });
  },
);`;

const prompt = `Read the docs about Inngest scores @https://www.inngest.com/docs-markdown/features/inngest-functions/steps-workflows/scoring and show me how to implement scores using scoreMiddleware(), step.score(), and inngest.score() and explore what scores might be ideal for my existing functions including quality, performance, tool use.`;

const benefits = [
  {
    icon: RiCheckboxCircleLine,
    title: 'Grade AI & evals',
    description: 'Record boolean or numeric scores to track quality.',
  },
  {
    icon: RiLineChartLine,
    title: 'Trend any metric',
    description: 'Chart latency, eval results, or confidence over time.',
  },
  {
    icon: RiQuestionLine,
    title: 'Query with Insights',
    description: 'Extract datasets by querying score data directly.',
  },
  {
    icon: RiFilterLine,
    title: 'Slice by function',
    description: 'Filter scores per function to spot regressions fast.',
  },
];

export function ScoresEmptyState() {
  // Fire once on view. The ref guards against React 18 StrictMode's
  // double-invoke so we don't double-count.
  const tracked = useRef(false);
  useEffect(() => {
    if (tracked.current) return;
    tracked.current = true;
    analytics.track('Empty State Viewed', { feature: 'scores' });
  }, []);

  return (
    <div className="bg-canvasBase flex flex-1 flex-col items-center overflow-auto px-6 py-12">
      <div className="mx-auto flex w-full max-w-[800px] flex-col gap-10">
        <div className="flex flex-col gap-2">
          <h1 className="text-basis text-2xl">Scores</h1>
          <p className="text-subtle text-sm leading-relaxed">
            Use scores to track and evaluate custom metrics from inside your
            functions. Record numeric or boolean scores on any run - eval
            pass/fail, confidence intervals, latency, tool use. Use Inngest to
            measure quality and performance trends over time.
          </p>
          <Link href={DOCS_URL} target="_blank">
            Learn more about scores
          </Link>
        </div>

        <div className="grid grid-cols-2 gap-x-8 gap-y-6">
          {benefits.map(({ icon: Icon, title, description }) => (
            <div key={title} className="flex items-start gap-3">
              <div className="border-subtle bg-canvasSubtle text-basis flex h-10 w-10 shrink-0 items-center justify-center rounded-md border">
                <Icon className="h-5 w-5" />
              </div>
              <div className="flex flex-col gap-0.5">
                <p className="text-basis text-sm font-medium">{title}</p>
                <p className="text-muted text-sm leading-relaxed">
                  {description}
                </p>
              </div>
            </div>
          ))}
        </div>

        <hr className="border-subtle" />

        <div className="flex flex-col gap-6">
          <h2 className="text-basis text-lg">Get started</h2>

          <CommandBlock.Wrapper>
            <CommandBlock.Header className="flex items-center justify-between px-4 py-2.5">
              <p className="text-subtle text-sm">
                Copy this prompt to learn about this feature and implement
                scores
              </p>
              <CommandBlock.CopyButton
                content={prompt}
                onCopy={() => {
                  analytics.track('Empty State Prompt Copied', {
                    feature: 'scores',
                  });
                }}
              />
            </CommandBlock.Header>
            <CommandBlock
              height={124}
              currentTabContent={{
                title: 'Code',
                content: prompt,
                readOnly: true,
                language: 'shell',
                wordWrap: 'on',
              }}
            />
          </CommandBlock.Wrapper>

          <CommandBlock.Wrapper>
            <CommandBlock.Header className="flex items-center justify-between px-4 py-2.5">
              <p className="text-subtle text-sm">
                add <InlineCode>inngest.score()</InlineCode> to any function
              </p>
              <CommandBlock.CopyButton content={example} />
            </CommandBlock.Header>
            <CommandBlock
              currentTabContent={{
                title: 'Code',
                content: example,
                readOnly: true,
                language: 'typescript',
              }}
            />
          </CommandBlock.Wrapper>
        </div>
      </div>
    </div>
  );
}
