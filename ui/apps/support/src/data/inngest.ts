import { createServerFn } from "@tanstack/react-start";

// Inngest account plan types
export type AccountPlanInfo = {
  planId?: string;
  planName?: string;
  amount?: number;
  isEnterprise: boolean;
  isPaid: boolean;
  error?: boolean;
};

export const getAccountPlanInfo = createServerFn({ method: "GET" })
  .inputValidator(() => ({}))
  .handler(async (): Promise<AccountPlanInfo> => {
    try {
      // Import the Inngest GraphQL API client
      const { inngestGQLAPI } = await import("./gqlApi");

      const query = `
        query GetAccountSupportInfo {
          account {
            id
            plan {
              id
              name
              amount
            }
          }
        }
      `;

      const data = await inngestGQLAPI.request<{
        account: {
          id: string;
          plan: {
            id: string;
            name: string;
            amount: number;
          } | null;
        };
      }>(query);

      const plan = data?.account?.plan;

      if (!plan) {
        // If no plan found, treat as paid (graceful fallback)
        return {
          isEnterprise: false,
          isPaid: true,
          error: false,
        };
      }

      // Check if plan name starts with "Enterprise" (case-insensitive)
      const isEnterprise = Boolean(plan.name?.match(/^Enterprise/i));
      // isPaid if amount > 0 or is enterprise
      const isPaid = (plan.amount || 0) > 0 || isEnterprise;

      return {
        planId: plan.id,
        planName: plan.name,
        amount: plan.amount,
        isEnterprise,
        isPaid,
        error: false,
      };
    } catch (error) {
      console.error("Error fetching account plan info:", error);
      // On error, gracefully treat as paid
      return {
        isEnterprise: false,
        isPaid: true,
        error: true,
      };
    }
  });
