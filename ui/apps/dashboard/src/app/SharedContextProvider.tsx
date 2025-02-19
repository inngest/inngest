'use client';

import { SharedProvider, type SharedHandlers } from '@inngest/components/Shared/SharedContext';
import { legacyTraceToggle } from '@inngest/components/Shared/useLegacyTrace';

import { useInvokeRun } from '@/queries/useInvokeRun';
import { useRerunFromStep } from '@/queries/useRerunFromStep';

export const SharedContextProvider = ({ children }: { children: React.ReactNode }) => {
  const rerunFromStep = useRerunFromStep();
  const invokeRun = useInvokeRun();
  const legacyTrace = legacyTraceToggle();

  const handlers: Partial<SharedHandlers> = {
    invokeRun,
    rerunFromStep,
    legacyTrace,
  };

  return <SharedProvider handlers={handlers}>{children}</SharedProvider>;
};
