import { SignalsProvider, type SignalHandlers } from '@inngest/components/Signals/SignalsContext';

import { useInvokeRun } from '@/hooks/useInvokeRun';

export const Signals = ({ children }: { children: React.ReactNode }) => {
  const invokeRun = useInvokeRun();

  const handlers: Partial<SignalHandlers> = {
    invokeRun,
  };

  return <SignalsProvider handlers={handlers}>{children}</SignalsProvider>;
};
