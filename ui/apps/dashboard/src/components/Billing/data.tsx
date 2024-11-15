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
  return await graphqlAPI.request<EntitlementUsageQuery>(entitlementUsageDocument);
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
  return await graphqlAPI.request<GetCurrentPlanQuery>(currentPlanDocument);
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
  return await graphqlAPI.request<GetBillingDetailsQuery>(billingDetailsDocument);
};
