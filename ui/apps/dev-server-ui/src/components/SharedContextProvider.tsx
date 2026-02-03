import {
  SharedProvider,
  type SharedHandlers,
} from '@inngest/components/SharedContext/SharedContext';

import { useBooleanFlag } from '@/hooks/useBooleanFlag';
import { useCancelRun } from '@/hooks/useCancelRun';
import { useCreateDebugSession } from '@/hooks/useCreateDebugSession';
import { useGetDebugRun } from '@/hooks/useGetDebugRun';
import { useGetDebugSession } from '@/hooks/useGetDebugSession';
import { useGetRun } from '@/hooks/useGetRun';
import { useGetRunTrace } from '@/hooks/useGetRunTrace';
import { useGetTraceResult } from '@/hooks/useGetTraceResult';
import { useInvokeRun } from '@/hooks/useInvokeRun';
import { useRerun } from '@/hooks/useRerun';
import { useRerunFromStep } from '@/hooks/useRerunFromStep';
import { pathCreator } from '@/utils/pathCreator';

export const SharedContextProvider = ({
  children,
}: {
  children: React.ReactNode;
}) => {
  const invokeRun = useInvokeRun();
  const rerunFromStep = useRerunFromStep();
  const rerun = useRerun();
  const cancelRun = useCancelRun();
  const getRun = useGetRun();
  const getRunTrace = useGetRunTrace();
  const getTraceResult = useGetTraceResult();
  const createDebugSession = useCreateDebugSession();
  const getDebugRun = useGetDebugRun();
  const getDebugSession = useGetDebugSession();

  //
  // No-op streamRun for dev-server (streaming not supported)
  const streamRun = () => () => {};

  const handlers: Partial<SharedHandlers> = {
    invokeRun,
    rerun,
    rerunFromStep,
    cancelRun,
    pathCreator,
    booleanFlag: useBooleanFlag,
    cloud: false,
    getRun,
    getRunTrace,
    inngestStatus: null,
    getTraceResult,
    getDebugRun,
    getDebugSession,
    createDebugSession,
    streamRun,
  };

  return <SharedProvider handlers={handlers}>{children}</SharedProvider>;
};
