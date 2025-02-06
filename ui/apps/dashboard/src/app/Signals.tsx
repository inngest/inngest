'use client';

import { SignalsProvider, type SignalHandlers } from '@inngest/components/Signals/SignalsContext';

import { useInvokeRun } from '@/queries/useInvokeRun';
import { useRerunFromStep } from '@/queries/useRerunFromStep';

export const Signals = ({ children }: { children: React.ReactNode }) => {
  const rerunFromStep = useRerunFromStep();
  const invokeRun = useInvokeRun();

  const handlers: Partial<SignalHandlers> = {
    invokeRun,
    rerunFromStep,
  };

  return <SignalsProvider handlers={handlers}>{children}</SignalsProvider>;
};
