'use client';

import { SharedProvider, type SharedHandlers } from '@inngest/components/Shared/SharedContext';

import { useInvokeRun } from '@/queries/useInvokeRun';
import { useRerunFromStep } from '@/queries/useRerunFromStep';

export const SharedDataProvider = ({ children }: { children: React.ReactNode }) => {
  const rerunFromStep = useRerunFromStep();
  const invokeRun = useInvokeRun();

  const handlers: Partial<SharedHandlers> = {
    invokeRun,
    rerunFromStep,
  };

  return <SharedProvider handlers={handlers}>{children}</SharedProvider>;
};
