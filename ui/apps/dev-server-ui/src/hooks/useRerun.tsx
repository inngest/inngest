import { useCallback } from 'react';
import type { RerunPayload, RerunResult } from '@inngest/components/SharedContext/useRerun';

import { useRerunMutation } from '@/store/generated';
import { pathCreator } from '@/utils/pathCreator';

export const useRerun = () => {
  const [rerun] = useRerunMutation();

  return useCallback(
    async (payload: RerunPayload): Promise<RerunResult> => {
      try {
        const res = await rerun(payload);

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
