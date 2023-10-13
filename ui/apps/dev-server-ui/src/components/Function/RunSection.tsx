import { useCallback, useEffect, useMemo, useState } from 'react';
import { Badge } from '@inngest/components/Badge';
import { ContentCard } from '@inngest/components/ContentCard';
import { FunctionRunStatusIcon } from '@inngest/components/FunctionRunStatusIcon';
import { MetadataGrid } from '@inngest/components/Metadata';

import { type OutputType } from '@/components/Function/OutputRenderer';
import { IconClock } from '@/icons';
import { client } from '@/store/baseApi';
import {
  FunctionRunStatus,
  FunctionTriggerTypes,
  GetHistoryItemOutputDocument,
  useGetFunctionRunQuery,
} from '../../store/generated';
import { BlankSlate } from '../Blank';
import { Timeline } from '../Timeline';
import { useParsedHistory } from '../Timeline/historyParser';
import OutputCard from './Output';
import renderRunMetadata from './RunMetadataRenderer';
import { SleepingSummary } from './SleepingSummary';
import { WaitingSummary } from './WaitingSummary';

interface FunctionRunSectionProps {
  runId: string | null | undefined;
}

export const FunctionRunSection = ({ runId }: FunctionRunSectionProps) => {
  const [pollingInterval, setPollingInterval] = useState(1000);
  const query = useGetFunctionRunQuery(
    { id: runId || '' },
    { pollingInterval, skip: !runId, refetchOnMountOrArgChange: true }
  );
  const run = useMemo(() => query.data?.functionRun, [query.data?.functionRun]);
  const history = useParsedHistory(run?.history ?? []);
  const firstTrigger = run?.function?.triggers?.[0] ?? null;
  const cron = firstTrigger && firstTrigger.type === FunctionTriggerTypes.Cron;

  useEffect(() => {
    if (!run?.event?.id) {
      return;
    }
  }, [run?.event?.id]);

  const getOutput = useCallback(
    (historyItemID: string) => {
      if (!runId) {
        // Should be unreachable.
        return new Promise<string>((resolve) => resolve(''));
      }

      return getHistoryItemOutput({ historyItemID, runID: runId });
    },
    [runId]
  );

  if (query.isLoading) {
    return (
      <ContentCard>
        <div className="flex h-full w-full items-center justify-center p-8">
          <div className="italic opacity-75">Loading...</div>
        </div>
      </ContentCard>
    );
  }

  if (!run || !runId) {
    return (
      <ContentCard>
        <BlankSlate
          imageUrl="/images/no-results.png"
          title="No functions called"
          subtitle="Read our documentation to learn how to write functions"
          link={{
            text: 'Writing Functions',
            url: 'https://www.inngest.com/docs/functions',
          }}
        />
      </ContentCard>
    );
  }
  const metadataItems = renderRunMetadata(run);
  let type: OutputType | undefined;
  if (run.status === FunctionRunStatus.Completed) {
    type = 'completed';
  } else if (run.status === FunctionRunStatus.Failed) {
    type = 'failed';
  }

  return (
    <ContentCard
      title={run.name || 'Unknown'}
      icon={run.status && <FunctionRunStatusIcon status={run.status} className="h-5 w-5" />}
      type="run"
      badge={
        cron ? (
          <div className="py-2">
            <Badge className="bg-orange-400/10 text-orange-400" kind="solid">
              <IconClock />
              {firstTrigger.value}
            </Badge>
          </div>
        ) : null
      }
      metadata={
        <div className="pt-8">
          <MetadataGrid metadataItems={metadataItems} />
        </div>
      }
    >
      <div className="px-5 pt-4">
        {run.status && run.finishedAt && run.output && type && (
          <OutputCard content={run.output} type={type} />
        )}

        <WaitingSummary history={history} />
        <SleepingSummary history={history} />
      </div>

      <hr className="mt-8 border-slate-800/50" />
      <div className="px-5 pt-4">
        <h3 className="py-4 text-sm text-slate-400">Timeline</h3>
        <Timeline getOutput={getOutput} history={history} />
      </div>
    </ContentCard>
  );
};

async function getHistoryItemOutput({
  historyItemID,
  runID,
}: {
  historyItemID: string;
  runID: string;
}): Promise<string | undefined> {
  // TODO: How to get type annotations? It returns `any`.
  const res: unknown = await client.request(GetHistoryItemOutputDocument, {
    historyItemID,
    runID,
  });

  if (typeof res !== 'object' || res === null || !('functionRun' in res)) {
    throw new Error('invalid response');
  }
  const { functionRun } = res;

  if (
    typeof functionRun !== 'object' ||
    functionRun === null ||
    !('historyItemOutput' in functionRun)
  ) {
    throw new Error('invalid response');
  }
  const { historyItemOutput } = functionRun;

  if (historyItemOutput === null) {
    return undefined;
  }
  if (typeof historyItemOutput !== 'string') {
    throw new Error('invalid response');
  }

  return historyItemOutput;
}
