import { type InvokeFunctionMutationVariables } from "@/gql/graphql";
import { getProductionEnvironment } from "@/queries/server-only/getEnvironment";
import {
  getInvokeFunctionLookups,
  invokeFn,
  preloadInvokeFunctionLookups,
} from "./data";

export async function invokeFunction({
  functionSlug,
  user,
  data,
}: Pick<InvokeFunctionMutationVariables, "data" | "functionSlug" | "user">) {
  try {
    await invokeFn({ functionSlug, user, data });

    return {
      success: true,
    };
  } catch (error) {
    console.error("Error invoking function:", error);

    if (error instanceof Error) {
      return {
        success: false,
        error: error.message,
      };
    }

    return {
      success: false,
      error: "Unknown error occurred while invoking function",
    };
  }
}

export async function prefetchFunctions() {
  const environment = await getProductionEnvironment();

  preloadInvokeFunctionLookups(environment.slug);
  const {
    envBySlug: {
      workflows: { data: functions },
    },
  } = await getInvokeFunctionLookups(environment.slug);

  return functions;
}
