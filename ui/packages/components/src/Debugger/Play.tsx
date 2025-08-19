import { RiPlayLine } from '@remixicon/react';

import { useStepSelection } from '../RunDetailsV3/utils';
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
  const { selectedStep } = useStepSelection(runID);
  const { rerun } = useRerunFromStep();

  const handleRerun = async () => {
    if (selectedStep?.trace.stepID && runID) {
      const result = await rerun({
        runID,
        fromStep: {
          stepID: selectedStep.trace.stepID,
          input: '[{}]',
        },
        debugRunID,
        debugSessionID,
      });
      console.log('play result', result);
    }
  };

  return (
    <RiPlayLine
      className="text-muted hover:bg-canvasSubtle h-6 w-6 cursor-pointer rounded-md p-1"
      onClick={handleRerun}
    />
  );
};
