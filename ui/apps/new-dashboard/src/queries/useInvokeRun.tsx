import type { InvokeRunPayload } from "@inngest/components/SharedContext/useInvokeRun";
import { useMutation } from "urql";

import { useEnvironment } from "@/components/Environments/environment-context";
// TANSTACK TODO Re-enable InvokeFunctionDocument import once tsx components are migrated
// import { InvokeFunctionDocument } from "@/gql/graphql";

export const useInvokeRun = () => {
  const env = useEnvironment();
  const [, invokeFunction] = useMutation<any>(null as any);

  return async ({ functionSlug, data, user, envID }: InvokeRunPayload) => {
    return await invokeFunction({
      envID: envID ? envID : env.id,
      data,
      functionSlug,
      user,
    });
  };
};
