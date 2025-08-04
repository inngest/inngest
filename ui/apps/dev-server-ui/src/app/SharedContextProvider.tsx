import {
  SharedProvider,
  type SharedHandlers,
} from '@inngest/components/SharedContext/SharedContext';

import { useBooleanFlag } from '@/hooks/useBooleanFlag';
import { useCancelRun } from '@/hooks/useCancelRun';
import { useGetTraceResult } from '@/hooks/useGetTraceResult';
import { useInvokeRun } from '@/hooks/useInvokeRun';
import { useRerun } from '@/hooks/useRerun';
import { useRerunFromStep } from '@/hooks/useRerunFromStep';
import { useRun } from '@/hooks/useRun';
import { pathCreator } from '@/utils/pathCreator';

export const SharedContextProvider = ({ children }: { children: React.ReactNode }) => {
  const invokeRun = useInvokeRun();
  const rerunFromStep = useRerunFromStep();
  const rerun = useRerun();
  const cancelRun = useCancelRun();
  const getRun = useRun();
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
