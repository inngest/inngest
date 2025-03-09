import { SharedProvider, type SharedHandlers } from '@inngest/components/Shared/SharedContext';
import { legacyTraceToggle } from '@inngest/components/Shared/useLegacyTrace';

import { useInvokeRun } from '@/hooks/useInvokeRun';
import { useRerun } from '@/hooks/useRerun';
import { useRerunFromStep } from '@/hooks/useRerunFromStep';

export const SharedContextProvider = ({ children }: { children: React.ReactNode }) => {
  const invokeRun = useInvokeRun();
  const rerunFromStep = useRerunFromStep();
  const rerun = useRerun();
  const legacyTrace = legacyTraceToggle();
  const handlers: Partial<SharedHandlers> = {
    invokeRun,
    rerun,
    rerunFromStep,
    legacyTrace,
  };

  return <SharedProvider handlers={handlers}>{children}</SharedProvider>;
};
