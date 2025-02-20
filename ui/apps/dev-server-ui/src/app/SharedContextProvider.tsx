import { SharedProvider, type SharedHandlers } from '@inngest/components/Shared/SharedContext';
import { legacyTraceToggle } from '@inngest/components/Shared/useLegacyTrace';

import { useInvokeRun } from '@/hooks/useInvokeRun';
import { useRerunFromStep } from '@/hooks/useRerunFromStep';
import { convertError } from '@/store/error';

export const SharedContextProvider = ({ children }: { children: React.ReactNode }) => {
  const invokeRun = useInvokeRun();
  const rerunFromStep = useRerunFromStep();
  const legacyTrace = legacyTraceToggle();
  const handlers: Partial<SharedHandlers> = {
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
    legacyTrace,
  };

  return <SharedProvider handlers={handlers}>{children}</SharedProvider>;
};
