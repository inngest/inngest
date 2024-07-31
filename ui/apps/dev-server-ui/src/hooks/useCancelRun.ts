import { useCallback } from 'react';

import { useCancelRunMutation } from '@/store/generated';

export function useCancelRun() {
  const [cancelRun] = useCancelRunMutation();

  return useCallback(
    async (runID: string) => {
      const res = await cancelRun({ runID });
      if ('error' in res) {
        // Throw error so that the modal can catch and display it
        throw res.error;
      }
    },
    [cancelRun]
  );
}
