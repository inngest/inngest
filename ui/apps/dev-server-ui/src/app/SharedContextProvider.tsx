import {
  SharedProvider,
  type SharedHandlers,
} from '@inngest/components/SharedContext/SharedContext';
import { legacyTraceToggle } from '@inngest/components/SharedContext/useLegacyTrace';

import { useCancelRun } from '@/hooks/useCancelRun';
import { useInvokeRun } from '@/hooks/useInvokeRun';
import { useRerun } from '@/hooks/useRerun';
import { useRerunFromStep } from '@/hooks/useRerunFromStep';

export const SharedContextProvider = ({ children }: { children: React.ReactNode }) => {
  const invokeRun = useInvokeRun();
  const rerunFromStep = useRerunFromStep();
  const rerun = useRerun();
  const legacyTrace = legacyTraceToggle();
  const cancelRun = useCancelRun();

  const handlers: Partial<SharedHandlers> = {
    invokeRun,
    rerun,
    rerunFromStep,
    legacyTrace,
    cancelRun,
    cloud: false,
  };

  return <SharedProvider handlers={handlers}>{children}</SharedProvider>;
};
