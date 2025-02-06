import { useCallback } from 'react';
import type { RerunFromStepPayload } from '@inngest/components/Shared/useRerunFromStep';

import { useRerunFromStepMutation } from '@/store/generated';

export function useRerunFromStep() {
  const [rerunFromStep] = useRerunFromStepMutation();

  return useCallback(
    async ({ runID, fromStep }: RerunFromStepPayload) => {
      return await rerunFromStep({ runID, fromStep });
    },
    [rerunFromStep]
  );
}
