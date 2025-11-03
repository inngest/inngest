import { RiPlayLine } from '@remixicon/react';
import { useNavigate } from '@tanstack/react-router';
import { toast } from 'sonner';
import { ulid } from 'ulid';

import { useStepSelection } from '../RunDetailsV3/utils';
import { usePathCreator } from '../SharedContext/usePathCreator';
import { useRerun } from '../SharedContext/useRerun';
import { useRerunFromStep } from '../SharedContext/useRerunFromStep';

type PlayProps = {
  functionSlug: string;
  runID?: string;
  debugRunID?: string;
  debugSessionID?: string;
};

export const Play = ({ functionSlug, runID, debugRunID, debugSessionID }: PlayProps) => {
  const { pathCreator } = usePathCreator();
  const navigate = useNavigate();
  const { selectedStep } = useStepSelection({
    // TODO: add debug run id
    runID,
  });
  const { rerun: rerunFromStep } = useRerunFromStep();
  const { rerun } = useRerun();
  const newDebugRunID = ulid();

  const handleRerun = async () => {
    if (!runID) {
      console.error('runID is currently required');
      return;
    }

    const result = selectedStep?.trace.stepID
      ? await rerunFromStep({
          runID,
          fromStep: {
            stepID: selectedStep.trace.stepID,
            input: '[{}]',
          },
          debugRunID: debugRunID ?? newDebugRunID,
          debugSessionID,
        })
      : await rerun({
          runID,
          debugSessionID,
          debugRunID: ulid(),
        });

    if (result.error) {
      console.error('error running debugger', result.error);
      toast.error(`Error running debugger, see console for more details.`);
      return;
    }

    //
    // if this is our first debug run, send them there
    if (!debugRunID) {
      navigate({
        to: pathCreator.debugger({
          functionSlug,
          runID,
          debugRunID: newDebugRunID,
          debugSessionID,
        }),
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
