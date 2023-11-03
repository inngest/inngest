import type { BillingPlan } from '@/gql/graphql';

export type ExtendedBillingPlan = BillingPlan & {
  isFreeTier: boolean;
  isLowerTierPlan: boolean;
  isActive: boolean;
  isUsagePlan: boolean;
  isPremium: boolean;
  isTrial: boolean;
  usagePercentage: number;
  additionalSteps?: {
    cost: number;
    quantity: string;
  };
};

// We should add this to a field in the database, but we ship this interim hack for now
export function isEnterprisePlan(plan: Partial<BillingPlan>): boolean {
  return Boolean(plan.name?.match(/^Enterprise/i));
}
export function isTrialPlan(plan: Partial<BillingPlan>): boolean {
  return Boolean(plan.name?.match(/Trial/i));
}

function getAdditionalStepRate(plan: BillingPlan): { cost: number; quantity: string } {
  if (plan.name.toLowerCase() === 'startup') {
    // $150 plan was add mid 2023
    if (plan.amount === 14900) {
      return {
        cost: 1000,
        quantity: '1M',
      };
    }
    // The current default of the $350 plan
    return {
      cost: 500,
      quantity: '200k',
    };
  } else if (plan.name.toLowerCase() === 'team') {
    return {
      cost: 100,
      quantity: '10k',
    };
  }
  // Default to $10/1M
  return {
    cost: 1000,
    quantity: '1M',
  };
}

export function transformPlan({
  plan,
  currentPlan,
  usage,
}: {
  plan: BillingPlan;
  currentPlan?: BillingPlan;
  usage: number;
}): ExtendedBillingPlan {
  const isUsagePlan = plan.name.toLowerCase() === 'team' || plan.name.toLowerCase() === 'startup';
  const isEnterprise = plan.name.toLowerCase() === 'enterprise';
  const isCurrentPlanEnterprise = currentPlan !== undefined && isEnterprisePlan(currentPlan);
  const isTrial = isEnterprise && isCurrentPlanEnterprise && isTrialPlan(currentPlan);

  // Merge the features if the user is on a custom enterprise plan
  const features =
    isEnterprise && isCurrentPlanEnterprise
      ? { ...plan.features, ...currentPlan.features }
      : plan.features;
  const amount = isEnterprise && isCurrentPlanEnterprise ? currentPlan.amount : plan.amount;

  let actions: number | undefined = undefined;
  let usagePercentage = 0;
  if (typeof features.actions === 'number') {
    actions = features.actions;
    usagePercentage = (usage / actions) * 100;
  }

  // NOTE - Have to hard code this for now before backend support for this
  let additionalSteps;
  if (isUsagePlan) {
    additionalSteps = getAdditionalStepRate(plan);
  }

  // Ensure that if the current plan is enterprise, we always show
  // non-enterprise as "lower tiers" regardless of the plan cost
  let isLowerTierPlan = (currentPlan?.amount || 0) > plan.amount;
  if (isCurrentPlanEnterprise) {
    if (isEnterprise) {
      isLowerTierPlan = false;
    } else {
      isLowerTierPlan = true;
    }
  }

  return {
    ...plan,
    amount,
    features: {
      ...features,
      actions:
        actions?.toLocaleString(undefined, {
          notation: 'compact',
          compactDisplay: 'short',
        }) || (isEnterprise ? 'Custom' : ''),
      events: 'Unlimited',
      users: 'Unlimited',
      support: isEnterprise ? 'Premium support' : isUsagePlan ? 'Discord + Email' : 'Discord',
    },
    // Enterprise evaluation/trial plans cost $0
    isFreeTier: !isTrial && !isEnterprise && amount === 0,
    isLowerTierPlan,
    isActive: currentPlan?.id === plan.id || (isEnterprise && isCurrentPlanEnterprise),
    isUsagePlan,
    additionalSteps,
    isPremium: isEnterprise,
    isTrial,
    usagePercentage,
  };
}
