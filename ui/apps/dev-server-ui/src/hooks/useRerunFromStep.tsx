import { useCallback } from 'react';
import { Link } from '@inngest/components/Link';
import { toast } from 'sonner';

import { useRerunFromStepMutation, useRerunMutation } from '@/store/generated';
import { pathCreator } from '@/utils/pathCreator';

export function useRerunFromStep() {
  const [rerunFromStep] = useRerunFromStepMutation();

  return useCallback(
    async ({ runID, fromStep }: { runID: string; fromStep: { stepID: string; input: string } }) => {
      return await rerunFromStep({ runID, fromStep });
    },
    [rerunFromStep]
  );
}
