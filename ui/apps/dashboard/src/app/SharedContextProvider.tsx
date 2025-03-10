'use client';

import {
  SharedProvider,
  type SharedHandlers,
} from '@inngest/components/SharedContext/SharedContext';
import { legacyTraceToggle } from '@inngest/components/SharedContext/useLegacyTrace';

import { useInvokeRun } from '@/queries/useInvokeRun';
import { useRerun } from '@/queries/useRerun';
import { useRerunFromStep } from '@/queries/useRerunFromStep';

export const SharedContextProvider = ({ children }: { children: React.ReactNode }) => {
  const rerunFromStep = useRerunFromStep();
  const invokeRun = useInvokeRun();
  const legacyTrace = legacyTraceToggle();
  const rerun = useRerun();

  const handlers: Partial<SharedHandlers> = {
    invokeRun,
    rerunFromStep,
    legacyTrace,
    rerun,
  };

  return <SharedProvider handlers={handlers}>{children}</SharedProvider>;
};
