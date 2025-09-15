import { useEffect } from 'react';
import { RiPlayLine } from '@remixicon/react';

import { useStepSelection } from '../RunDetailsV3/utils';
import { useRerun } from '../SharedContext/useRerun';
import { useRerunFromStep } from '../SharedContext/useRerunFromStep';

export const Play = ({
  runID,
  debugRunID,
  debugSessionID,
}: {
  runID?: string;
  debugRunID?: string;
  debugSessionID?: string;
}) => {
  const { selectedStep } = useStepSelection({
    debugRunID,
    runID,
  });
  const { rerun: rerunFromStep } = useRerunFromStep();
  const { rerun } = useRerun();

  const handleRerun = async () => {
    if (selectedStep?.trace.stepID && runID) {
      const result = await rerunFromStep({
        runID,
        fromStep: {
          stepID: selectedStep.trace.stepID,
          input: '[{}]',
        },
        debugRunID,
        debugSessionID,
      });
    } else if (runID) {
      const result = await rerun({
        runID,
        debugRunID,
        debugSessionID,
      });
    }
  };

  return (
    <RiPlayLine
      className="text-subtle hover:bg-canvasSubtle h-8 w-8 cursor-pointer rounded-md p-1"
      onClick={handleRerun}
    />
  );
};
