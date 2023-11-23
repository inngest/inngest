'use client';

import { useMemo } from 'react';
import { type Route } from 'next';
import { useRouter } from 'next/navigation';
import { EventDetails } from '@inngest/components/EventDetails';
import { RunDetails } from '@inngest/components/RunDetails';
import { SlideOver } from '@inngest/components/SlideOver';
import { useParsedHistory } from '@inngest/components/hooks/useParsedHistory';
import type { Environment } from '@inngest/components/types/environment';
import type { Event } from '@inngest/components/types/event';
import type { Function } from '@inngest/components/types/function';
import type { FunctionRun } from '@inngest/components/types/functionRun';
import type { FunctionVersion } from '@inngest/components/types/functionVersion';
import { classNames } from '@inngest/components/utils/classNames';
import { type RawHistoryItem } from '@inngest/components/utils/historyParser';
import { useClient } from 'urql';

import RerunButton from './RerunButton';
import { getHistoryItemOutput } from './getHistoryItemOutput';

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
  const router = useRouter();

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
  const parentURL = `/env/${environment.slug}/functions/${encodeURIComponent(
    func.slug
  )}/logs` as Route;

  let rerunButton: React.ReactNode | undefined;
  if (run.canRerun) {
    rerunButton = <RerunButton environment={environment} func={func} functionRunID={run.id} />;
  }

  return (
    <SlideOver size={event ? 'large' : 'small'} onClose={() => router.push(parentURL)}>
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
    </SlideOver>
  );
}
