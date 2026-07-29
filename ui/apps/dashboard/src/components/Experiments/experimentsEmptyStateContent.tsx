import type { ComponentType } from 'react';
import {
  RiEqualizerLine,
  RiGroupLine,
  RiLineChartLine,
  RiScalesLine,
} from '@remixicon/react';

import type { TabsProps } from '@inngest/components/CodeBlock/CommandBlock';

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
  {
    title: 'weighted ( )',
    content: WEIGHTED,
    language: 'typescript',
    readOnly: true,
  },
  {
    title: 'bucket ( )',
    content: BUCKET,
    language: 'typescript',
    readOnly: true,
  },
  {
    title: 'custom ( )',
    content: CUSTOM,
    language: 'typescript',
    readOnly: true,
  },
  {
    title: 'fixed ( )',
    content: FIXED,
    language: 'typescript',
    readOnly: true,
  },
];

export const PROMPT = `Read the docs about Inngest experiments @https://www.inngest.com/docs/features/inngest-functions/steps-workflows/step-experiments and show me how to declare variants with group.experiment() using weighted, bucket, custom, or fixed selection, then help me find a function where I could safely trial a change and compare outcomes across variants.`;
