import { useCallback, useEffect, useMemo, useState } from 'react';
import { EventDetails } from '@inngest/components/EventDetails';
import { RunDetails } from '@inngest/components/RunDetails';
import { SlideOver } from '@inngest/components/SlideOver';
import type { NavigateToRunFn } from '@inngest/components/Timeline';
import { HistoryParser } from '@inngest/components/utils/historyParser';
import { useClient } from 'urql';

import LoadingIcon from '@/icons/LoadingIcon';
import { getHistoryItemOutput } from './getHistoryItemOutput';
import { useEvent } from './useEvent';
import { useRun } from './useRun';

type Props = {
  envID: string;
  eventID: string | undefined;
  onClose: () => void;
  navigateToRun: NavigateToRunFn;
};

export function Details({ envID, eventID, onClose, navigateToRun }: Props) {
  const [selectedRun, setSelectedRun] = useState<{ functionID: string; runID: string } | undefined>(
    undefined
  );
  const client = useClient();

  const eventRes = useEvent({ envID, eventID });
  if (eventRes.error) {
    throw eventRes.error;
  }

  useEffect(() => {
    const firstRun = eventRes.data?.runs[0];
    if (firstRun) {
      setSelectedRun({
        functionID: firstRun.functionID,
        runID: firstRun.id,
      });
    } else {
      setSelectedRun(undefined);
    }
  }, [eventRes.data?.runs]);

  const runRes = useRun({ envID, functionID: selectedRun?.functionID, runID: selectedRun?.runID });
  if (runRes.error) {
    throw runRes.error;
  }

  const getOutput = useMemo(() => {
    return (historyItemID: string) => {
      if (!selectedRun) {
        throw new Error('missing selected run');
      }

      return getHistoryItemOutput({
        client,
        envID,
        functionID: selectedRun.functionID,
        historyItemID,
        runID: selectedRun.runID,
      });
    };
  }, [client, envID, selectedRun]);

  const onFunctionRunClick = useCallback(
    (runID: string) => {
      if (!eventRes.data?.runs) {
        throw new Error('missing run data');
      }

      for (const run of eventRes.data.runs) {
        if (run.id === runID) {
          setSelectedRun({ functionID: run.functionID, runID });
          return;
        }
      }

      throw new Error(`could not find function ID for run ${runID}`);
    },
    [eventRes.data]
  );

  let eventDetails;
  if (eventRes.isLoading) {
    eventDetails = <Loading />;
  } else if (eventRes.isSkipped) {
    eventDetails = null;
  } else {
    eventDetails = (
      <EventDetails
        events={[eventRes.data.event]}
        functionRuns={eventRes.data.runs}
        onFunctionRunClick={onFunctionRunClick}
        selectedRunID={selectedRun?.runID}
      />
    );
  }

  let runDetails;
  if (runRes.isLoading) {
    runDetails = <Loading />;
  } else if (runRes.isSkipped) {
    runDetails = <></>;
  } else {
    runDetails = (
      <RunDetails
        func={runRes.data.func}
        functionVersion={runRes.data.functionVersion}
        getHistoryItemOutput={getOutput}
        history={new HistoryParser(runRes.data.run.history)}
        run={runRes.data.run}
        navigateToRun={navigateToRun}
      />
    );
  }

  return (
    <>
      {eventID && (
        <SlideOver onClose={onClose} size="large">
          <div className={'text-basis grid h-full grid-cols-2'}>
            {eventDetails}
            {runDetails}
          </div>
        </SlideOver>
      )}
    </>
  );
}

function Loading() {
  return (
    <div className="flex h-full w-full items-center justify-center">
      <div className="flex flex-col items-center justify-center gap-2">
        <LoadingIcon />
        <div>Loading</div>
      </div>
    </div>
  );
}
