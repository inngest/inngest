import { useCallback } from 'react';

import { useRerunFromStepMutation } from '@/store/generated';

export function useRerunFromStep() {
  const [rerunFromStep] = useRerunFromStepMutation();

  return useCallback(
    async ({ runID, fromStep }: { runID: string; fromStep: { stepID: string; input: string } }) => {
      return await rerunFromStep({ runID, fromStep });
    },
    [rerunFromStep]
  );
}
