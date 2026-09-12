import { getPeriodAbbreviation } from '@inngest/components/utils/date';
import * as Sentry from '@sentry/tanstackstart-react';

import type {
  BillingPlan,
  EntitlementConcurrency,
  EntitlementExecutions,
  EntitlementInt,
  EntitlementRunCount,
  GetPlansQuery,
} from '@/gql/graphql';
import { pathCreator } from '@/utils/urls';

export type Plan = Omit<
  BillingPlan,
  'entitlements' | 'features' | 'availableAddons' | 'addons'
> & {
  entitlements: {
    concurrency: Pick<EntitlementConcurrency, 'limit'>;
    executions?: Pick<EntitlementExecutions, 'limit'>;
    runCount: Pick<EntitlementRunCount, 'limit'>;
    history: Pick<EntitlementInt, 'limit'>;
  };
};

export type SelfServePlan = GetPlansQuery['plans'][number];

// Under normal circumstances, we should only have one active + visible plan for
// Hobby and Pro, but the backend doesn't guarantee that for us. Pick one here
// using an arbitrary rule.
export function pickSelfServePlans(plans?: SelfServePlan[]): {
  hobby: SelfServePlan | null;
  pro: SelfServePlan | null;
} {
  const selectablePlans = plans ?? [];

  // Multiple matches indicate that an old plan was not retired correctly, so
  // report that via Sentry. Keep checkout available by picking a plan arbitrarily.
  const pickPlan = (name: string, isFree: boolean) => {
    const candidates = selectablePlans.filter(
      (plan) => plan.isFree === isFree && !plan.isLegacy && plan.name === name,
    );

    if (candidates.length > 1) {
      Sentry.captureMessage(
        `Multiple active, visible ${name} billing plans returned`,
        {
          level: 'error',
          extra: {
            plans: candidates.map(({ amount, slug }) => ({ amount, slug })),
          },
        },
      );
    }

    if (plans && candidates.length === 0) {
      Sentry.captureMessage(`Missing active, visible ${name} billing plan`, {
        level: 'error',
      });
    }

    return candidates.reduce<SelfServePlan | null>(
      (selected, candidate) =>
        !selected || candidate.amount > selected.amount ? candidate : selected,
      null,
    );
  };

  const selfServePlans = {
    hobby: pickPlan('Hobby', true),
    pro: pickPlan('Pro', false),
  };

  return selfServePlans;
}

export enum PlanNames {
  Free = 'Free Tier',
  Basic = 'Basic',
  Pro = 'Pro',
  Hobby = 'Hobby - Free',
  Enterprise = 'Enterprise',
}

export function processPlan(plan: Plan) {
  const { name, amount, billingPeriod, entitlements, slug } = plan;

  const featureDescriptions = getFeatureDescriptions(name, entitlements);

  const priceLabel =
    name === PlanNames.Enterprise || amount === Infinity
      ? 'Contact us'
      : new Intl.NumberFormat('en-US', {
          style: 'currency',
          currency: 'USD',
          maximumFractionDigits: 0,
        }).format(amount / 100);

  return {
    name,
    price: priceLabel,
    billingPeriod:
      typeof billingPeriod === 'string'
        ? getPeriodAbbreviation(billingPeriod)
        : 'mo',
    features: featureDescriptions,
    slug,
  };
}

function getFeatureDescriptions(
  planName: string,
  entitlements: Plan['entitlements'],
): (string | React.ReactNode)[] {
  const numberFormatter = new Intl.NumberFormat('en-US', {
    notation: 'compact',
    compactDisplay: 'short',
  });
  const executionLimit = entitlements.executions?.limit;
  const runLimit = executionLimit ?? entitlements.runCount.limit;

  switch (planName) {
    case PlanNames.Free:
      return [
        ...(runLimit ? [`${numberFormatter.format(runLimit)} runs/mo`] : []),
        `${numberFormatter.format(
          entitlements.concurrency.limit,
        )} concurrent steps`,
        'Unlimited branch and staging envs',
        'Logs, traces, and observability',
        'Basic alerting',
        'Community support',
      ];

    case PlanNames.Basic:
      return [
        ...(runLimit
          ? [`Starts at ${numberFormatter.format(runLimit)} runs/mo`]
          : []),
        `Starts at ${numberFormatter.format(
          entitlements.concurrency.limit,
        )} concurrent steps`,
        `${entitlements.history.limit} day trace and history retention`,
        'Unlimited functions and apps',
        'No event rate limit',
        'Basic email and ticketing support',
      ];

    case PlanNames.Pro:
      return [
        'Starts at 1,000,000+ executions',
        '100+ concurrent steps',
        '1,000+ realtime connections',
        <>
          10 users included
          <a
            className="hover:underline"
            href={pathCreator.billing({
              ref: 'app-billing-plans-pro-addons',
              highlight: 'users',
            })}
          >
            (add-ons available)
          </a>
        </>,
        'Granular metrics',
        'Higher usage limits',
      ];

    case PlanNames.Enterprise:
      return [
        'Custom executions',
        '500+ concurrent steps',
        '1,000+ realtime connections',
        '50+ users',
        'SAML, RBAC, and audit trails',
        'Exportable observability',
        'Dedicated slack channel',
      ];

    case PlanNames.Hobby:
      return [
        '100k executions, hard limited',
        '5 concurrent steps',
        '3 realtime connections',
        '3 users',
        'Basic alerting',
        'Community support',
      ];

    // "runs/mo" is legacy copy, but we still need to support users whose
    // current plan predates execution-based billing.
    default:
      return [
        ...(runLimit
          ? [
              executionLimit == null
                ? `${numberFormatter.format(runLimit)} runs/mo`
                : `${executionLimit.toLocaleString('en-US')} executions`,
            ]
          : []),
        `${numberFormatter.format(
          entitlements.concurrency.limit,
        )}  concurrent steps`,
        `${entitlements.history.limit} day trace and history retention`,
      ];
  }
}

export function isActive(
  currentPlan: Plan | (Partial<BillingPlan> & { name: string }),
  cardPlan: Plan | (Partial<BillingPlan> & { name: string }),
): boolean {
  return (
    currentPlan.name === cardPlan.name ||
    (cardPlan.name === PlanNames.Enterprise && isEnterprisePlan(currentPlan))
  );
}

// TODO: Return these from the API
export function isEnterprisePlan(
  plan: Plan | (Partial<BillingPlan> & { name: string }),
): boolean {
  return Boolean(plan.name.match(/^Enterprise/i));
}

export function isTrialPlan(
  plan: Plan | (Partial<BillingPlan> & { name: string }),
): boolean {
  return Boolean(plan.name.match(/Trial/i));
}

export function isHobbyFreePlan(
  plan: Plan | (Partial<BillingPlan> & { name: string }),
): boolean {
  if (!plan.slug) return false;
  return plan.slug.startsWith('hobby-free-');
}

export function isHobbyPlan(
  plan: Plan | (Partial<BillingPlan> & { name: string }),
): boolean {
  return isHobbyFreePlan(plan);
}
