import { InlineCode } from '@inngest/components/Code';
import { FeatureEmptyState } from '@inngest/components/FeatureEmptyState';
import {
  RiCheckboxCircleLine,
  RiFilterLine,
  RiLineChartLine,
  RiQuestionLine,
} from '@remixicon/react';

import {
  trackScoreEmptyStateDocsLinkOpened,
  trackScoreEmptyStateExampleCopied,
  trackScoreEmptyStatePromptCopied,
  trackScoreEmptyStateViewed,
} from './tracking';

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

const valueProps = [
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
  return (
    <FeatureEmptyState
      title="Scores"
      description="Use scores to track and evaluate custom metrics from inside your functions. Record numeric or boolean scores on any run - eval pass/fail, confidence intervals, latency, tool use. Use Inngest to measure quality and performance trends over time."
      docsUrl={DOCS_URL}
      onDocsLinkClick={trackScoreEmptyStateDocsLinkOpened}
      valueProps={valueProps}
      prompt={{
        description:
          'Copy this prompt to learn about this feature and implement scores',
        content: prompt,
        onCopy: trackScoreEmptyStatePromptCopied,
      }}
      example={{
        description: (
          <>
            add <InlineCode>inngest.score()</InlineCode> to any function
          </>
        ),
        tabs: [
          {
            title: 'Code',
            content: example,
            readOnly: true,
            language: 'typescript',
          },
        ],
        onCopy: trackScoreEmptyStateExampleCopied,
      }}
      onViewed={trackScoreEmptyStateViewed}
    />
  );
}
