import type { InvokeRunPayload } from "@inngest/components/SharedContext/useInvokeRun";
import { useMutation } from "urql";

import { useEnvironment } from "@/components/Environments/environment-context";
import { InvokeFunctionOnboardingDocument } from "@/gql/graphql";

export const useInvokeRun = () => {
  const env = useEnvironment();
  const [, invokeFunction] = useMutation(InvokeFunctionOnboardingDocument);

  return async ({ functionSlug, data, user, envID }: InvokeRunPayload) => {
    return await invokeFunction({
      envID: envID ? envID : env.id,
      data,
      functionSlug,
      user,
    });
  };
};
