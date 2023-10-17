'use client';

import { useMemo } from 'react';
import { EventDetails } from '@inngest/components/EventDetails';
import { RunDetails } from '@inngest/components/RunDetails';
import { useParsedHistory } from '@inngest/components/hooks/useParsedHistory';
import type { Event } from '@inngest/components/types/event';
import type { Function } from '@inngest/components/types/function';
import type { FunctionRun } from '@inngest/components/types/functionRun';
import { classNames } from '@inngest/components/utils/classNames';
import { type RawHistoryItem } from '@inngest/components/utils/historyParser';
import { Client, useClient } from 'urql';

import { graphql } from '@/gql';

type Props = {
  envID: string;
  event?: Pick<Event, 'id' | 'name' | 'payload' | 'receivedAt'>;
  func: Pick<Function, 'id' | 'name' | 'triggers'>;
  rawHistory: RawHistoryItem[];
  run: Pick<FunctionRun, 'endedAt' | 'id' | 'output' | 'startedAt' | 'status'>;
};

export function StreamDetails({ envID, event, func, rawHistory, run }: Props) {
  const client = useClient();

  const getOutput = useMemo(() => {
    return (historyItemID: string) => {
      return getHistoryItemOutput({
        client,
        envID,
        functionID: func.id,
        historyItemID,
        runID: run.id,
      });
    };
  }, [client, envID, func.id, run.id]);

  const history = useParsedHistory(rawHistory);

  return (
    <div className={classNames('grid h-full text-white', event ? 'grid-cols-2' : 'grid-cols-1')}>
      {event && (
        <EventDetails
          event={event}
          // TODO
          onReplayEvent={console.log}
        />
      )}

      <RunDetails func={func} getHistoryItemOutput={getOutput} history={history} run={run} />
    </div>
  );
}

const getHistoryItemOutputDocument = graphql(`
  query GetHistoryItemOutput($envID: ID!, $functionID: ID!, $historyItemID: ULID!, $runID: ULID!) {
    environment: workspace(id: $envID) {
      function: workflow(id: $functionID) {
        run(id: $runID) {
          historyItemOutput(id: $historyItemID)
        }
      }
    }
  }
`);

async function getHistoryItemOutput({
  client,
  envID,
  functionID,
  historyItemID,
  runID,
}: {
  client: Client;
  envID: string;
  functionID: string;
  historyItemID: string;
  runID: string;
}): Promise<string | undefined> {
  // TODO: How to get type annotations? It returns `any`.
  const res = await client
    .query(getHistoryItemOutputDocument, {
      envID,
      functionID,
      historyItemID,
      runID,
    })
    .toPromise();
  if (res.error) {
    throw res.error;
  }

  const { historyItemOutput } = res.data?.environment.function?.run ?? {};
  if (!historyItemOutput) {
    throw new Error('invalid response');
  }
  return historyItemOutput;
}
