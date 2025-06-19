'use client';

import {
  SharedProvider,
  type SharedHandlers,
} from '@inngest/components/SharedContext/SharedContext';

import { useBooleanFlag } from '@/components/FeatureFlags/hooks';
import { useCancelRun } from '@/queries/useCancelRun';
import { useGetRun } from '@/queries/useGetRun';
import { useInvokeRun } from '@/queries/useInvokeRun';
import { useRerun } from '@/queries/useRerun';
import { useRerunFromStep } from '@/queries/useRerunFromStep';
import { usePathCreator } from '@/utils/usePathCreator';

export const SharedContextProvider = ({ children }: { children: React.ReactNode }) => {
  const rerunFromStep = useRerunFromStep();
  const invokeRun = useInvokeRun();
  const rerun = useRerun();
  const cancelRun = useCancelRun();
  const pathCreator = usePathCreator();
  const getRun = useGetRun();

  const handlers: Partial<SharedHandlers> = {
    invokeRun,
    rerunFromStep,
    rerun,
    cancelRun,
    cloud: true,
    booleanFlag: useBooleanFlag,
    pathCreator,
    getRun,
  };

  return <SharedProvider handlers={handlers}>{children}</SharedProvider>;
};
