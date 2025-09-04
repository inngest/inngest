import {
  SharedProvider,
  type SharedHandlers,
} from '@inngest/components/SharedContext/SharedContext';

import { useBooleanFlag } from '@/hooks/useBooleanFlag';
import { useCancelRun } from '@/hooks/useCancelRun';
import { useGetRun } from '@/hooks/useGetRun';
import { useGetTraceResult } from '@/hooks/useGetTraceResult';
import { useInvokeRun } from '@/hooks/useInvokeRun';
import { useRerun } from '@/hooks/useRerun';
import { useRerunFromStep } from '@/hooks/useRerunFromStep';
import { pathCreator } from '@/utils/pathCreator';

export const SharedContextProvider = ({ children }: { children: React.ReactNode }) => {
  const invokeRun = useInvokeRun();
  const rerunFromStep = useRerunFromStep();
  const rerun = useRerun();
  const cancelRun = useCancelRun();
  const getRun = useGetRun();
  const getTraceResult = useGetTraceResult();

  const handlers: Partial<SharedHandlers> = {
    invokeRun,
    rerun,
    rerunFromStep,
    cancelRun,
    pathCreator,
    booleanFlag: useBooleanFlag,
    cloud: false,
    getRun,
    inngestStatus: null,
    getTraceResult,
  };

  return <SharedProvider handlers={handlers}>{children}</SharedProvider>;
};
