import { useCallback } from 'react';

import { useRerunMutation } from '@/store/generated';

export function useRerun() {
  const [rerun] = useRerunMutation();

  return useCallback(
    async ({ runID }: { fnID: string; runID: string }) => {
      const res = await rerun({ runID });
      if ('error' in res) {
        // Throw error so that the modal can catch and display it
        throw res.error;
      }
    },
    [rerun]
  );
}
