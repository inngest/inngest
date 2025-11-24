import { graphql } from "@/gql";
import {
  type EntitlementUsageQuery,
  type GetBillingDetailsQuery,
  type GetCurrentPlanQuery,
  type GetPlansQuery,
} from "@/gql/graphql";
import graphqlAPI from "@/queries/graphqlAPI";
import { createServerFn } from "@tanstack/react-start";

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
        advancedObservability {
          available
          name
          price
          purchased
          entitlements {
            history {
              limit
            }
            metricsExportFreshness {
              limit
            }
            metricsExportGranularity {
              limit
            }
          }
        }
        slackChannel {
          available
          baseValue
          maxValue
          name
          price
          purchaseCount
          quantityPer
        }
        connectWorkers {
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
        # Disable entitlements here as it queries the usage table as well - see monorepo for now
        #executions {
        # DISABLE USAGE FOR NOW - SEE EXE-1011
        #usage
        #  limit
        #  overageAllowed
        #}
        runCount {
          #usage
          limit
          overageAllowed
        }
        stepCount {
          #usage
          limit
          overageAllowed
        }
        concurrency {
          #usage
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
        slackChannel {
          enabled
        }
        connectWorkerConnections {
          limit
        }
      }
      plan {
        name
      }
    }
  }
`);

export const entitlementUsage = createServerFn({
  method: "GET",
}).handler(async () => {
  try {
    const res = await graphqlAPI.request<EntitlementUsageQuery>(
      entitlementUsageDocument,
    );

    // TODO: Replace this with a proper programmatic check. Relying on the plan
    // name is fragile.
    const isCustomPlan = (res.account.plan?.name ?? "")
      .toLowerCase()
      .includes("enterprise");

    return {
      ...res.account,
      isCustomPlan,
    };
  } catch (error) {
    console.error("Error fetching entitlement usage:", error);
    throw new Error("Failed to fetch entitlement usage");
  }
});

export const currentPlanDocument = graphql(`
  query GetCurrentPlan {
    account {
      plan {
        id
        slug
        isLegacy
        isFree
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
          slackChannel {
            available
            price
            purchaseCount
            quantityPer
          }
          connectWorkers {
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

export const currentPlan = createServerFn({
  method: "GET",
  //@ts-expect-error TANSTACK TODO: sort out type error at merge time
}).handler(async () => {
  const res = await graphqlAPI.request<GetCurrentPlanQuery>(
    currentPlanDocument,
  );
  return res.account;
});

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

export const billingDetails = createServerFn({
  method: "GET",
}).handler(async () => {
  const res = await graphqlAPI.request<GetBillingDetailsQuery>(
    billingDetailsDocument,
  );
  return res.account;
});

export const plansDocument = graphql(`
  query GetPlans {
    plans {
      id
      isLegacy
      isFree
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

export const plans = createServerFn({
  method: "GET",
  //@ts-expect-error TANSTACK TODO: sort out type error at merge time
}).handler(async () => {
  try {
    const res = await graphqlAPI.request<GetPlansQuery>(plansDocument);
    return res.plans;
  } catch (error) {
    console.error("Error fetching plans:", error);
    throw new Error("Failed to fetch plans");
  }
});
