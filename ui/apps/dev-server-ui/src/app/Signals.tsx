import { SignalsProvider, type SignalHandlers } from '@inngest/components/Signals/SignalsContext';

import { useInvokeRun } from '@/hooks/useInvokeRun';
import { useRerunFromStep } from '@/hooks/useRerunFromStep';
import { convertError } from '@/store/error';

export const Signals = ({ children }: { children: React.ReactNode }) => {
  const invokeRun = useInvokeRun();
  const rerunFromStep = useRerunFromStep();
  const handlers: Partial<SignalHandlers> = {
    invokeRun,
    rerunFromStep: async (payload) => {
      //
      // Translate redux query specific type to generic error for upstream
      const result = await rerunFromStep(payload);
      if ('error' in result) {
        return { error: convertError('Failed to rerun from step', result.error) };
      }
      return result;
    },
  };

  return <SignalsProvider handlers={handlers}>{children}</SignalsProvider>;
};
