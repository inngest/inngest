import { useCallback } from 'react';
import type { RerunResult } from '@inngest/components/SharedContext/useRerun';

import { useRerunMutation } from '@/store/generated';
import { pathCreator } from '@/utils/pathCreator';

export const useRerun = () => {
  const [rerun, { error }] = useRerunMutation();

  return useCallback(
    async ({ runID }: { runID?: string }): Promise<RerunResult> => {
      try {
        const res = await rerun({ runID });

        if ('error' in res) {
          throw res.error;
        }

        const newRunID = res.data?.rerun;
        return {
          data: { newRunID },
          redirect: pathCreator.runPopout({ runID: newRunID }),
        };
      } catch (error) {
        console.error('error rerunning function', error);
        return {
          error: error instanceof Error ? error : new Error('Error re-running function'),
          data: undefined,
        };
      }
    },
    [rerun]
  );
};
