import { getPeriodAbbreviation } from '@inngest/components/utils/date';

import type { BillingPlan, Entitlements } from '@/gql/graphql';

export type Plan = Omit<BillingPlan, 'entitlements' | 'features'> & {
  entitlements: Pick<Entitlements, 'concurrency' | 'runCount' | 'history'>;
};

export enum PlanNames {
  Free = 'Free Tier',
  Basic = 'Basic',
  Pro = 'Pro',
  Enterprise = 'Enterprise',
}

export function processPlan(plan: Plan) {
  const { name, amount, billingPeriod, entitlements } = plan;

  const featureDescriptions = getFeatureDescriptions(name, entitlements);

  return {
    name,
    price: new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
      maximumFractionDigits: 0,
    }).format(amount / 100),
    billingPeriod: typeof billingPeriod === 'string' ? getPeriodAbbreviation(billingPeriod) : 'mo',
    features: featureDescriptions,
  };
}

function getFeatureDescriptions(planName: string, entitlements: Plan['entitlements']): string[] {
  const numberFormatter = new Intl.NumberFormat('en-US', {
    notation: 'compact',
    compactDisplay: 'short',
  });

  switch (planName) {
    case PlanNames.Free:
      return [
        ...(entitlements.runCount.limit
          ? [`${numberFormatter.format(entitlements.runCount.limit)} runs/mo`]
          : []),
        `${numberFormatter.format(entitlements.concurrency.limit)} concurrent steps`,
        'Unlimited branch and staging envs',
        'Logs, traces, and observability',
        'Basic alerting',
        'Community support',
      ];

    case PlanNames.Basic:
      return [
        ...(entitlements.runCount.limit
          ? [`Starts at ${numberFormatter.format(entitlements.runCount.limit)} runs/mo`]
          : []),
        `Starts at ${numberFormatter.format(entitlements.concurrency.limit)} concurrent steps`,
        `${entitlements.history.limit} day trace and history retention`,
        'Unlimited functions and apps',
        'No event rate limit',
        'Basic email and ticketing support',
      ];

    case PlanNames.Pro:
      return [
        ...(entitlements.runCount.limit
          ? [`Starts at ${numberFormatter.format(entitlements.runCount.limit)} runs/mo`]
          : []),
        `Starts at ${numberFormatter.format(entitlements.concurrency.limit)} concurrent steps`,
        `${entitlements.history.limit} day trace and history retention`,
        'Granular metrics',
        'Increased scale and throughput',
        'Higher usage limits',
        'SOC2',
        'HIPAA as a paid addon',
      ];

    case PlanNames.Enterprise:
      return [
        ...(entitlements.runCount.limit
          ? [`From 0-${numberFormatter.format(entitlements.runCount.limit)} runs/mo`]
          : []),
        `From 200 - ${numberFormatter.format(entitlements.concurrency.limit)}  concurrent steps`,
        'SAML, RBAC, and audit trails',
        'Exportable observability',
        'Dedicated infrastructure',
        '99.99% uptime SLAs',
        'Support SLAs',
        'Dedicated slack channel',
      ];

    default:
      return [
        ...(entitlements.runCount.limit
          ? [`${numberFormatter.format(entitlements.runCount.limit)} runs/mo`]
          : []),
        `${numberFormatter.format(entitlements.concurrency.limit)}  concurrent steps`,
        `${entitlements.history.limit} day trace and history retention`,
      ];
  }
}

export function isEnterprisePlan(plan: Plan | Partial<BillingPlan>): boolean {
  return Boolean(plan.name?.match(/^Enterprise/i));
}
