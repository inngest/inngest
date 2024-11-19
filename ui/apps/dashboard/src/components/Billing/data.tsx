import 'server-only';
import { graphql } from '@/gql';
import {
  type EntitlementUsageQuery,
  type GetBillingDetailsQuery,
  type GetCurrentPlanQuery,
} from '@/gql/graphql';
import graphqlAPI from '@/queries/graphqlAPI';

export const entitlementUsageDocument = graphql(`
  query EntitlementUsage {
    account {
      id
      entitlementUsage {
        runCount {
          current
          limit
          overageAllowed
        }
        stepCount {
          current
          limit
          overageAllowed
        }
        accountConcurrencyLimitHits
      }
    }
  }
`);

export const entitlementUsage = async () => {
  try {
    const res = await graphqlAPI.request<EntitlementUsageQuery>(entitlementUsageDocument);
    return res.account.entitlementUsage;
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
        features
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
