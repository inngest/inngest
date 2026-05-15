import { useMutation } from '@tanstack/react-query';

import { useShared } from './SharedContext';

export type CreateDebugSessionPayload = {
  functionSlug: string;
  runID?: string;
};

export type CreateDebugSessionResult = {
  loading: boolean;
  error?: Error;
  data?: {
    debugSessionID: string;
    debugRunID: string;
  };
  isSuccess: boolean;
  isError: boolean;
};

export const useCreateDebugSession = () => {
  const shared = useShared();

  const mutation = useMutation({
    mutationFn: async (payload: CreateDebugSessionPayload) => {
      const result = await shared.createDebugSession(payload);
      if (result.error) {
        throw result.error;
      }
      return result.data;
    },
    onError: (error) => {
      console.error('Error creating debug session:', error);
    },
  });

  return {
    createDebugSession: mutation.mutate,
    data: mutation.data,
    loading: mutation.isPending,
    error: mutation.error,
    isSuccess: mutation.isSuccess,
    isError: mutation.isError,
  };
};
