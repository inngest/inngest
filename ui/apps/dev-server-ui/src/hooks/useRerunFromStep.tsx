import { useCallback } from 'react';
import { Link } from '@inngest/components/Link';
import { toast } from 'sonner';

import { useRerunFromStepMutation, useRerunMutation } from '@/store/generated';
import { pathCreator } from '@/utils/pathCreator';

export function useRerunFromStep() {
  const [rerunFromStep] = useRerunFromStepMutation();

  return useCallback(
    async ({ runID, fromStep }: { runID: string; fromStep: { stepID: string; input: string } }) => {
      try {
        const res = await rerunFromStep({ runID, fromStep });
        if ('error' in res) {
          throw res.error;
        }

        return res.data.rerun;
      } catch (e) {
        toast.error('Failed to queue rerun');
        throw e;
      }
    },
    [rerunFromStep]
  );
}
