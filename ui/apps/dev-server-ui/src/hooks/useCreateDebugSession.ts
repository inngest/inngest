import type {
  CreateDebugSessionPayload,
  CreateDebugSessionResult,
} from '@inngest/components/SharedContext/useCreateDebugSession';

import { convertError } from '@/store/error';
import { useCreateDebugSessionMutation } from '@/store/generated';

export const useCreateDebugSession = (): ((
  payload: CreateDebugSessionPayload
) => Promise<CreateDebugSessionResult>) => {
  const [createDebugSession, { isLoading, error, isSuccess, isError }] =
    useCreateDebugSessionMutation();

  return async ({ functionSlug, runID }: CreateDebugSessionPayload) => {
    const result = await createDebugSession({
      input: {
        functionSlug,
        runID,
      },
    });

    if ('error' in result) {
      throw result.error;
    }

    return {
      data: result.data?.createDebugSession,
      loading: isLoading,
      error: error ? convertError('Failed to create debug session', error) : undefined,
      isSuccess,
      isError,
    };
  };
};
