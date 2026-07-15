import { RiEqualizerLine, RiGroupLine, RiLineChartLine, RiScalesLine } from '@remixicon/react';
import type { Meta, StoryObj } from '@storybook/react';

import { FeatureEmptyState } from './FeatureEmptyState';

const meta = {
  title: 'Components/FeatureEmptyState',
  component: FeatureEmptyState,
  parameters: {
    layout: 'fullscreen',
  },
} satisfies Meta<typeof FeatureEmptyState>;

export default meta;

type Story = StoryObj<typeof FeatureEmptyState>;

const VALUE_PROPS = [
  {
    icon: RiScalesLine,
    title: 'Compare models',
    description: 'Run two models or prompts side by side on live data.',
  },
  {
    icon: RiLineChartLine,
    title: 'Canary a rewrite',
    description: 'Ship a refactor to 1% of runs, then ramp it up.',
  },
  {
    icon: RiEqualizerLine,
    title: 'Tune costly operations',
    description: 'Trial a batch size or temperature per run.',
  },
  {
    icon: RiGroupLine,
    title: 'Migrate a cohort',
    description: 'Keep an account on one experience through a rollout.',
  },
];

const PROMPT = `Read the docs about Inngest experiments @https://www.inngest.com/docs/features/inngest-functions/steps-workflows/step-experiments and show me how to declare variants with group.experiment() using weighted, bucket, custom, or fixed selection, then help me find a function where I could safely trial a change and compare outcomes across variants.`;

const SINGLE_EXAMPLE = `const charge = await group.experiment("payments-provider", {
  variants: {
    stripe: () => step.run("charge-stripe", () => chargeStripe(event.data.order)),
    adyen: () => step.run("charge-adyen", () => chargeAdyen(event.data.order)),
  },
  select: experiment.bucket(event.data.accountId, { weights: { stripe: 90, adyen: 10 } }),
});`;

export const Default: Story = {
  args: {
    title: 'Experiments',
    description:
      'Experiments let you compare different versions of logic on real traffic without building a separate A/B testing system.',
    docsUrl:
      'https://www.inngest.com/docs/features/inngest-functions/steps-workflows/step-experiments',
    valueProps: VALUE_PROPS,
    prompt: {
      description: 'Copy this prompt to learn about this feature and implement experiments',
      content: PROMPT,
    },
    example: {
      description: 'add group.experiment() to any function',
      tabs: [{ title: 'Code', content: SINGLE_EXAMPLE, readOnly: true, language: 'typescript' }],
    },
  },
};

export const WithTabs: Story = {
  args: {
    ...Default.args,
    example: {
      tabs: [
        { title: 'weighted ( )', content: SINGLE_EXAMPLE, readOnly: true, language: 'typescript' },
        { title: 'bucket ( )', content: SINGLE_EXAMPLE, readOnly: true, language: 'typescript' },
        { title: 'custom ( )', content: SINGLE_EXAMPLE, readOnly: true, language: 'typescript' },
        { title: 'fixed ( )', content: SINGLE_EXAMPLE, readOnly: true, language: 'typescript' },
      ],
      height: 280,
    },
  },
};
