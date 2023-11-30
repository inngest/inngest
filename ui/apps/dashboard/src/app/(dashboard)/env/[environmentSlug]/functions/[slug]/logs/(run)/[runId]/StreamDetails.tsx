'use client';

import { useMemo } from 'react';
import type { Route } from 'next';
import { EventDetails } from '@inngest/components/EventDetails';
import { Link } from '@inngest/components/Link';
import { RunDetails } from '@inngest/components/RunDetails';
import { useParsedHistory } from '@inngest/components/hooks/useParsedHistory';
import type { Environment } from '@inngest/components/types/environment';
import type { Event } from '@inngest/components/types/event';
import type { Function } from '@inngest/components/types/function';
import type { FunctionRun } from '@inngest/components/types/functionRun';
import type { FunctionVersion } from '@inngest/components/types/functionVersion';
import { classNames } from '@inngest/components/utils/classNames';
import { type RawHistoryItem } from '@inngest/components/utils/historyParser';
import type { NavigateToRunFn } from 'node_modules/@inngest/components/src/Timeline/Timeline';
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

  const navigateToRun: NavigateToRunFn = (opts) => {
    return (
      <Link
        internalNavigation
        href={
          `/env/${environment.slug}/functions/${encodeURIComponent(opts.fnID)}/logs/${
            opts.runID
          }` as Route
        }
      >
        Go to run
      </Link>
    );
  };

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
        navigateToRun={navigateToRun}
      />
    </div>
  );
}
