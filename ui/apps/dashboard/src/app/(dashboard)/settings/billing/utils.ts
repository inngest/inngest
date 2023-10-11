import type { BillingPlan } from '@/gql/graphql';

export type ExtendedBillingPlan = BillingPlan & {
  isFreeTier: boolean;
  isLowerTierPlan: boolean;
  isActive: boolean;
  isUsagePlan: boolean;
  isPremium: boolean;
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
  const isCurrentPlanEnterprise =
    isEnterprise && currentPlan !== undefined && isEnterprisePlan(currentPlan);

  // Merge the features if the user is on a custom enterprise plan
  const features =
    isEnterprise && isCurrentPlanEnterprise
      ? { ...plan.features, ...currentPlan.features }
      : plan.features;
  const amount = isCurrentPlanEnterprise ? currentPlan.amount : plan.amount;

  let actions: number | undefined = undefined;
  let usagePercentage = 0;
  if (typeof features.actions === 'number') {
    actions = features.actions;
    usagePercentage = (usage / actions) * 100;
  }

  // NOTE - Have to hard code this for now before backend support for this
  let additionalSteps;
  if (isUsagePlan) {
    if (plan.name.toLowerCase() === 'startup') {
      additionalSteps = {
        cost: 1000,
        quantity: '1M',
      };
    } else if (plan.name.toLowerCase() === 'team') {
      additionalSteps = {
        cost: 100,
        quantity: '10k',
      };
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
    isFreeTier: amount === 0,
    isLowerTierPlan: (currentPlan?.amount || 0) > plan.amount,
    isActive: currentPlan?.id === plan.id || isCurrentPlanEnterprise,
    isUsagePlan,
    additionalSteps,
    isPremium: isEnterprise,
    usagePercentage,
  };
}
