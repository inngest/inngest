import { getPeriodAbbreviation } from '@inngest/components/utils/date';
import * as Sentry from '@sentry/tanstackstart-react';

import type {
  BillingPlan,
  EntitlementConcurrencyValue,
  EntitlementInt,
  EntitlementWithOverage,
  GetPlansQuery,
} from '@/gql/graphql';
import { pathCreator } from '@/utils/urls';

export type Plan = Omit<
  BillingPlan,
  'entitlements' | 'ents' | 'features' | 'availableAddons' | 'addons'
> & {
  entitlements: {
    concurrency: Pick<EntitlementConcurrencyValue, 'limit'>;
    executions?: Pick<EntitlementWithOverage, 'limit'>;
    runCount: Pick<EntitlementWithOverage, 'limit'>;
    history: Pick<EntitlementInt, 'limit'>;
  };
};

export type SelfServePlan = GetPlansQuery['plans'][number];

// the billing API does not guarantee one active plan per name.  each card
// needs one plan so checkout has one price.
export function pickSelfServePlans(plans?: SelfServePlan[]): {
  hobby: SelfServePlan | null;
  pro: SelfServePlan | null;
  business: SelfServePlan | null;
} {
  const selectablePlans = plans ?? [];

  // more than one match means an old plan was not retired.  this includes an
  // active Growth plan next to an active Business plan.  report it to Sentry
  // and keep checkout available.
  const pickPlan = (names: string[], isFree: boolean) => {
    const candidates = selectablePlans.filter(
      (plan) =>
        plan.isFree === isFree && !plan.isLegacy && names.includes(plan.name),
    );

    if (candidates.length > 1) {
      Sentry.captureMessage(
        `Multiple active, visible ${names[0]} billing plans returned`,
        {
          level: 'error',
          extra: {
            plans: candidates.map(({ amount, slug }) => ({ amount, slug })),
          },
        },
      );
    }

    if (plans && candidates.length === 0) {
      Sentry.captureMessage(
        `Missing active, visible ${names[0]} billing plan`,
        {
          level: 'error',
        },
      );
    }

    const preferredCandidates = candidates.filter(
      (plan) => plan.name === names[0],
    );
    return (
      preferredCandidates.length ? preferredCandidates : candidates
    ).reduce<SelfServePlan | null>(
      (selected, candidate) =>
        !selected || candidate.amount > selected.amount ? candidate : selected,
      null,
    );
  };

  const selfServePlans = {
    hobby: pickPlan(['Hobby'], true),
    pro: pickPlan(['Pro'], false),
    business: pickPlan([PlanNames.Business, legacyBusinessPlanName], false),
  };

  return selfServePlans;
}

export enum PlanNames {
  Free = 'Free Tier',
  Basic = 'Basic',
  Pro = 'Pro',
  Business = 'Business',
  Hobby = 'Hobby - Free',
  Enterprise = 'Enterprise',
}

// the billing API still returns the Business plan as Growth for some
// accounts.  names such as "Growth Trial" keep their suffix.
const legacyBusinessPlanName = 'Growth';

export function displayPlanName(name: string): string {
  return name.startsWith(legacyBusinessPlanName)
    ? PlanNames.Business + name.slice(legacyBusinessPlanName.length)
    : name;
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
    name: displayPlanName(name),
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
    displayPlanName(currentPlan.name) === displayPlanName(cardPlan.name) ||
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

export function executionUsageDescription(
  plan: Pick<BillingPlan, 'name' | 'slug'>,
  includedExecutions: number | null,
): string {
  if (isHobbyPlan(plan)) {
    return 'The maximum number of executions per month';
  }

  if (isEnterprisePlan(plan) || includedExecutions === null) {
    return 'Additional usage billed at the start of the next billing cycle.';
  }

  // Match executionOverageTiers in monorepo/pkg/billing/invoice/consts.go.
  // Tier boundaries include the account's execution allowance, not just overages.
  const pricePerMillion =
    includedExecutions < 5_000_000
      ? 50
      : includedExecutions < 15_000_000
      ? 25
      : includedExecutions < 50_000_000
      ? 20
      : 15;

  return `Usage beyond ${includedExecutions.toLocaleString(
    'en-US',
  )} is billed at $${pricePerMillion} per 1,000,000 at the start of the next billing cycle.`;
}
