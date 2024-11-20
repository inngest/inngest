import { getPeriodAbbreviation } from '@inngest/components/utils/date';

import type { BillingPlan } from '@/gql/graphql';

export enum PlanNames {
  Free = 'Free Tier',
  Basic = 'Basic',
  Pro = 'Pro',
  Enterprise = 'Enterprise',
}

export function processPlan(plan: BillingPlan) {
  const { name, amount, billingPeriod, features } = plan;

  const featureDescriptions = getFeatureDescriptions(name, features);

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

function getFeatureDescriptions(planName: string, features: Record<string, any>): string[] {
  const numberFormatter = new Intl.NumberFormat('en-US', {
    notation: 'compact',
    compactDisplay: 'short',
  });

  switch (planName) {
    case PlanNames.Free:
      return [
        `${numberFormatter.format(features.runs)} runs/mo free`,
        `${numberFormatter.format(features.concurrency)} concurrent steps`,
        'Unlimited branch and staging envs',
        'Logs, traces, and observability',
        'Basic alerting',
        'Community support',
      ];

    case PlanNames.Basic:
      return [
        `Starts at ${numberFormatter.format(features.runs)} runs/mo`,
        `Starts at ${numberFormatter.format(features.concurrency)} concurrent steps`,
        `${features.log_retention} day trace and history retention`,
        'Unlimited functions and apps',
        'No event rate limit',
        'Basic email and ticketing support',
      ];

    case PlanNames.Pro:
      return [
        `Starts at ${numberFormatter.format(features.runs)} runs/mo`,
        `Starts at ${numberFormatter.format(features.concurrency)} concurrent steps`,
        `${features.log_retention} day trace and history retention`,
        'Granular metrics',
        'Increased scale and throughput',
        'Higher usage limits',
        'SOC2',
        'HIPAA as a paid addon',
      ];

    case PlanNames.Enterprise:
      return [
        `From 0-${numberFormatter.format(features.runs)} runs/mo`,
        `From 200 - ${numberFormatter.format(features.concurrency)}  concurrent steps`,
        'SAML, RBAC, and audit trails',
        'Exportable observability',
        'Dedicated infrastructure',
        '99.99% uptime SLAs',
        'Support SLAs',
        'Dedicated slack channel',
      ];

    default:
      return [];
  }
}

export function isEnterprisePlan(plan: BillingPlan): boolean {
  return Boolean(plan.name.match(/^Enterprise/i));
}
