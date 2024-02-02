'use client';

import { useMemo } from 'react';
import type { Route } from 'next';
import { EventDetails } from '@inngest/components/EventDetails';
import { Link } from '@inngest/components/Link';
import { RunDetails } from '@inngest/components/RunDetails';
import { useParsedHistory } from '@inngest/components/hooks/useParsedHistory';
import { IconCloudArrowDown } from '@inngest/components/icons/CloudArrowDown';
import type { Environment } from '@inngest/components/types/environment';
import type { Event } from '@inngest/components/types/event';
import type { Function } from '@inngest/components/types/function';
import type { FunctionRun } from '@inngest/components/types/functionRun';
import type { FunctionVersion } from '@inngest/components/types/functionVersion';
import { classNames } from '@inngest/components/utils/classNames';
import { type RawHistoryItem } from '@inngest/components/utils/historyParser';
import type { NavigateToRunFn } from 'node_modules/@inngest/components/src/Timeline/Timeline';
import { useClient } from 'urql';

import { devServerURL, useDevServer } from '@/utils/useDevServer';
import RerunButton from './RerunButton';
import { getHistoryItemOutput } from './getHistoryItemOutput';

type Props = {
  environment: Pick<Environment, 'id' | 'slug'>;
  events?: Pick<Event, 'id' | 'name' | 'payload' | 'receivedAt'>[];
  func: Pick<Function, 'id' | 'name' | 'slug' | 'triggers'>;
  functionVersion?: Pick<FunctionVersion, 'url' | 'version'>;
  rawHistory: RawHistoryItem[];
  run: Pick<FunctionRun, 'canRerun' | 'endedAt' | 'id' | 'output' | 'startedAt' | 'status'>;
};

export function StreamDetails({
  environment,
  events,
  func,
  functionVersion,
  rawHistory,
  run,
}: Props) {
  const client = useClient();
  const { isRunning, send } = useDevServer();

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

  let codeBlockActions = undefined;
  if (events?.[0] && events.length === 1) {
    const { payload } = events[0];
    codeBlockActions = [
      {
        label: 'Send to Dev Server',
        title: isRunning
          ? 'Send event payload to running Dev Server'
          : `Dev Server is not running at ${devServerURL}`,
        icon: <IconCloudArrowDown />,
        onClick: () => send(payload),
        disabled: !isRunning,
      },
    ];
  }
  const hasCron =
    Array.isArray(func.triggers) &&
    func.triggers.length > 0 &&
    func.triggers.some((trigger) => trigger.type === 'CRON');
  const hasEventDetails = events && !hasCron;

  return (
    <div
      className={classNames(
        'dark grid h-full text-white',
        hasEventDetails ? 'grid-cols-2' : 'grid-cols-1'
      )}
    >
      {hasEventDetails && <EventDetails events={events} codeBlockActions={codeBlockActions} />}
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
