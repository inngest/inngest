import type { InvokeRunPayload } from '@inngest/components/SharedContext/useInvokeRun';

import { useInvokeFunctionMutation } from '@/store/generated';

export const useInvokeRun = () => {
  const [invokeFunction] = useInvokeFunctionMutation();

  return async ({ functionSlug, data, user }: InvokeRunPayload) => {
    return await invokeFunction({
      data,
      functionSlug,
      user,
    });
  };
};
