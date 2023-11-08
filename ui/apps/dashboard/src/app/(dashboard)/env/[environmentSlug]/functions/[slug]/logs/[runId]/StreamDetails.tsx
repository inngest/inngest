'use client';

import { useMemo } from 'react';
import { EventDetails } from '@inngest/components/EventDetails';
import { RunDetails } from '@inngest/components/RunDetails';
import { useParsedHistory } from '@inngest/components/hooks/useParsedHistory';
import type { Environment } from '@inngest/components/types/environment';
import type { Event } from '@inngest/components/types/event';
import type { Function } from '@inngest/components/types/function';
import type { FunctionRun } from '@inngest/components/types/functionRun';
import type { FunctionVersion } from '@inngest/components/types/functionVersion';
import { classNames } from '@inngest/components/utils/classNames';
import { type RawHistoryItem } from '@inngest/components/utils/historyParser';
import { Client, useClient } from 'urql';

import { graphql } from '@/gql';
import RerunButton from './(side-card)/(timeline)/RerunButton';

type Props = {
  environment: Pick<Environment, 'id' | 'slug'>;
  event?: Pick<Event, 'id' | 'name' | 'payload' | 'receivedAt'>;
  func: Pick<Function, 'id' | 'name' | 'slug' | 'triggers'>;
  functionVersion?: Pick<FunctionVersion, 'url' | 'version'>;
  rawHistory: RawHistoryItem[];
  run: Pick<FunctionRun, 'canRerun' | 'endedAt' | 'id' | 'output' | 'startedAt' | 'status'>;
};

export function StreamDetails({
  environment,
  event,
  func,
  functionVersion,
  rawHistory,
  run,
}: Props) {
  const client = useClient();

  const getOutput = useMemo(() => {
    return (historyItemID: string) => {
      return getHistoryItemOutput({
        client,
        envID: environment.id,
        functionID: func.id,
        historyItemID,
        runID: run.id,
      });
    };
  }, [client, environment.id, func.id, run.id]);

  const history = useParsedHistory(rawHistory);

  let rerunButton: React.ReactNode | undefined;
  if (run.canRerun) {
    rerunButton = <RerunButton environment={environment} func={func} functionRunID={run.id} />;
  }

  return (
    <div
      className={classNames('dark grid h-full text-white', event ? 'grid-cols-2' : 'grid-cols-1')}
    >
      {event && <EventDetails event={event} />}

      <RunDetails
        func={func}
        functionVersion={functionVersion}
        getHistoryItemOutput={getOutput}
        history={history}
        rerunButton={rerunButton}
        run={run}
      />
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

  return res.data?.environment.function?.run.historyItemOutput ?? undefined;
}
