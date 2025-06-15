import {
  SharedProvider,
  type SharedHandlers,
} from '@inngest/components/SharedContext/SharedContext';

import { useCancelRun } from '@/hooks/useCancelRun';
import { useInvokeRun } from '@/hooks/useInvokeRun';
import { useRerun } from '@/hooks/useRerun';
import { useRerunFromStep } from '@/hooks/useRerunFromStep';
import { useRun } from '@/hooks/useRun';
import { useBooleanFlag } from '@/utils/featureFlags';
import { pathCreator } from '@/utils/pathCreator';

export const SharedContextProvider = ({ children }: { children: React.ReactNode }) => {
  const invokeRun = useInvokeRun();
  const rerunFromStep = useRerunFromStep();
  const rerun = useRerun();
  const cancelRun = useCancelRun();
  const getRun = useRun();

  const handlers: Partial<SharedHandlers> = {
    invokeRun,
    rerun,
    rerunFromStep,
    cancelRun,
    pathCreator,
    booleanFlag: useBooleanFlag,
    cloud: false,
    getRun,
  };

  return <SharedProvider handlers={handlers}>{children}</SharedProvider>;
};
