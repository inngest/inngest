import 'server-only';
import { graphql } from '@/gql';
import {
  type EntitlementUsageQuery,
  type GetBillingDetailsQuery,
  type GetCurrentPlanQuery,
  type GetPlansQuery,
} from '@/gql/graphql';
import graphqlAPI from '@/queries/graphqlAPI';

export const entitlementUsageDocument = graphql(`
  query EntitlementUsage {
    account {
      id
      addons {
        concurrency {
          available
          baseValue
          maxValue
          name
          price
          purchaseCount
          quantityPer
        }
        userCount {
          available
          baseValue
          maxValue
          name
          price
          purchaseCount
          quantityPer
        }
      }
      entitlements {
        runCount {
          usage
          limit
          overageAllowed
        }
        stepCount {
          usage
          limit
          overageAllowed
        }
        concurrency {
          usage
          limit
        }
        eventSize {
          limit
        }
        history {
          limit
        }
        userCount {
          usage
          limit
        }
        hipaa {
          enabled
        }
        metricsExport {
          enabled
        }
        metricsExportFreshness {
          limit
        }
        metricsExportGranularity {
          limit
        }
      }
      plan {
        name
      }
    }
  }
`);

export const entitlementUsage = async () => {
  try {
    const res = await graphqlAPI.request<EntitlementUsageQuery>(entitlementUsageDocument);

    // TODO: Replace this with a proper programmatic check. Relying on the plan
    // name is fragile.
    const isCustomPlan = (res.account.plan?.name ?? '').toLowerCase().includes('enterprise');

    return {
      ...res.account,
      isCustomPlan,
    };
  } catch (error) {
    console.error('Error fetching entitlement usage:', error);
    throw new Error('Failed to fetch entitlement usage');
  }
};

export const currentPlanDocument = graphql(`
  query GetCurrentPlan {
    account {
      plan {
        id
        name
        amount
        billingPeriod
        entitlements {
          concurrency {
            limit
          }
          eventSize {
            limit
          }
          history {
            limit
          }
          runCount {
            limit
          }
          stepCount {
            limit
          }
          userCount {
            limit
          }
        }
        addons {
          concurrency {
            available
            price
            purchaseCount
            quantityPer
          }
          userCount {
            available
            price
            purchaseCount
            quantityPer
          }
        }
      }
      subscription {
        nextInvoiceDate
      }
    }
  }
`);

export const currentPlan = async () => {
  try {
    const res = await graphqlAPI.request<GetCurrentPlanQuery>(currentPlanDocument);
    return res.account;
  } catch (error) {
    console.error('Error fetching current plan:', error);
    throw new Error('Failed to fetch current plan');
  }
};

export const billingDetailsDocument = graphql(`
  query GetBillingDetails {
    account {
      billingEmail
      name
      paymentMethods {
        brand
        last4
        expMonth
        expYear
        createdAt
        default
      }
    }
  }
`);

export const billingDetails = async () => {
  try {
    const res = await graphqlAPI.request<GetBillingDetailsQuery>(billingDetailsDocument);
    return res.account;
  } catch (error) {
    console.error('Error fetching billing details:', error);
    throw new Error('Failed to fetch billing details');
  }
};

export const plansDocument = graphql(`
  query GetPlans {
    plans {
      id
      name
      amount
      billingPeriod
      entitlements {
        concurrency {
          limit
        }
        eventSize {
          limit
        }
        history {
          limit
        }
        runCount {
          limit
        }
        stepCount {
          limit
        }
      }
    }
  }
`);

export const plans = async () => {
  try {
    const res = await graphqlAPI.request<GetPlansQuery>(plansDocument);
    return res.plans;
  } catch (error) {
    console.error('Error fetching plans:', error);
    throw new Error('Failed to fetch plans');
  }
};
