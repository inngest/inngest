import type { ComponentType } from 'react';
import { RiEqualizerLine, RiGroupLine, RiLineChartLine, RiScalesLine } from '@remixicon/react';

import type { TabsProps } from '../CodeBlock/CommandBlock';

export const DOCS_URL =
  'https://www.inngest.com/docs/features/inngest-functions/steps-workflows/step-experiments';

export const INTRO_DESCRIPTION =
  'Experiments let you compare different versions of logic on real traffic without building a separate A/B testing system.';

type UseCase = {
  Icon: ComponentType<{ className?: string }>;
  title: string;
  description: string;
};

export const USE_CASES: UseCase[] = [
  {
    Icon: RiScalesLine,
    title: 'Compare models',
    description: 'Run two models or prompts side by side on live data.',
  },
  {
    Icon: RiLineChartLine,
    title: 'Canary a rewrite',
    description: 'Ship a refactor to 1% of runs, then ramp it up.',
  },
  {
    Icon: RiEqualizerLine,
    title: 'Tune costly operations',
    description: 'Trial a batch size or temperature per run.',
  },
  {
    Icon: RiGroupLine,
    title: 'Migrate a cohort',
    description: 'Keep an account on one experience through a rollout.',
  },
];

const WEIGHTED = `import { experiment } from "inngest";
import { inngest } from "./client";
export default inngest.createFunction(
  {
    id: "generate-invoice",
    triggers: { event: "billing/invoice.requested" },
  },
  async ({ event, step, group }) => {
    return await group.experiment("invoice-engine", {
      variants: {
        current: () =>
          step.run("generate-current", () =>
            generateInvoiceV1(event.data)
          ),
        rewrite: () =>
          step.run("generate-rewrite", () =>
            generateInvoiceV2(event.data)
          ),
      },
      select: experiment.weighted({ current: 99, rewrite: 1 }),
    });
  }
);`;

const BUCKET = `const charge = await group.experiment("payments-provider", {
  variants: {
    stripe: () =>
      step.run("charge-stripe", () =>
        chargeStripe(event.data.order)
      ),
    adyen: () =>
      step.run("charge-adyen", () =>
        chargeAdyen(event.data.order)
      ),
  },
  select: experiment.bucket(event.data.accountId, {
    weights: { stripe: 90, adyen: 10 },
  }),
});`;

const CUSTOM = `const invoice = await group.experiment("invoice-engine", {
  variants: {
    current: () =>
      step.run("generate-current", () =>
        generateInvoiceV1(event.data)
      ),
    rewrite: () =>
      step.run("generate-rewrite", () =>
        generateInvoiceV2(event.data)
      ),
  },
  select: experiment.custom(async () => {
    const assignment = await rolloutAssignments.get(event.data.accountId);
    return assignment ?? "current";
  }),
});`;

const FIXED = `import { experiment } from "inngest";
import { inngest } from "./client";
export default inngest.createFunction(
  {
    id: "generate-invoice",
    triggers: { event: "billing/invoice.requested" },
  },
  async ({ event, step, group }) => {
    return await group.experiment("invoice-engine", {
      variants: {
        current: () =>
          step.run("generate-current", () =>
            generateInvoiceV1(event.data)
          ),
        rewrite: () =>
          step.run("generate-rewrite", () =>
            generateInvoiceV2(event.data)
          ),
      },
      select: experiment.fixed("rewrite"),
    });
  }
);`;

export const VARIANT_TABS: TabsProps[] = [
  { title: 'weighted ( )', content: WEIGHTED, language: 'typescript', readOnly: true },
  { title: 'bucket ( )', content: BUCKET, language: 'typescript', readOnly: true },
  { title: 'custom ( )', content: CUSTOM, language: 'typescript', readOnly: true },
  { title: 'fixed ( )', content: FIXED, language: 'typescript', readOnly: true },
];

const TRACK_OUTCOME = `const outcome = await group.experiment("email-copy", {
  variants: {
    short: () => step.run("short-copy", () => generateShortCopy(event.data)),
    detailed: () =>
      step.run("detailed-copy", () => generateDetailedCopy(event.data)),
  },
  select: experiment.bucket(event.data.userId, {
    weights: { short: 50, detailed: 50 },
  }),
  withVariant: true,
});
await step.run("track-selected-variant", () =>
  analytics.track("experiment.variant_selected", {
    experiment: "email-copy",
    variant: outcome.variant,
    userId: event.data.userId,
  })
);`;

export const TRACK_OUTCOME_TAB: TabsProps = {
  title: 'track-outcome.ts',
  content: TRACK_OUTCOME,
  language: 'typescript',
  readOnly: true,
};

export const STEPS = {
  one: {
    title: 'Declare your variants inside an inngest function and choose how runs are selected',
    description:
      'Wrap the paths you’re comparing in group.experiment( ). Each variant calls a step so the work stays durable. Every strategy returns exactly one variant per run.',
  },
  two: {
    title: 'Track the outcome',
    description:
      'Compare latency, errors and traces. You can also emit your own score from inside the selected variant.',
  },
};
