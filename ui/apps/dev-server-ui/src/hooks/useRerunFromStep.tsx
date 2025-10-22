import { useCallback } from 'react';
import type {
  RerunFromStepPayload,
  RerunFromStepResult,
} from '@inngest/components/SharedContext/useRerunFromStep';

import { useRerunFromStepMutation } from '@/store/generated';
import { pathCreator } from '@/utils/pathCreator';

export const useRerunFromStep = () => {
  const [rerunFromStep] = useRerunFromStepMutation();

  return useCallback(
    async (payload: RerunFromStepPayload): Promise<RerunFromStepResult> => {
      try {
        const res = await rerunFromStep(payload);
        if ('error' in res) {
          throw res.error;
        }

        return {
          ...res,
          redirect: res.data?.rerun ? pathCreator.runPopout({ runID: res.data.rerun }) : undefined,
        };
      } catch (error) {
        console.error('error rerunning from step', error);
        return {
          error: error instanceof Error ? error : new Error('Error rerunning from step'),
          data: undefined,
        };
      }
    },
    [rerunFromStep]
  );
};
