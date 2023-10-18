'use client';

import { Badge } from '@inngest/components/Badge';
import { ContentCard } from '@inngest/components/ContentCard';
import { FunctionRunStatusIcon } from '@inngest/components/FunctionRunStatusIcon';
import { MetadataGrid } from '@inngest/components/Metadata';
import { OutputCard } from '@inngest/components/OutputCard';
import { Timeline } from '@inngest/components/Timeline';
import { IconClock } from '@inngest/components/icons/Clock';
import type { Function } from '@inngest/components/types/function';
import type { FunctionRun } from '@inngest/components/types/functionRun';
import type { HistoryParser } from '@inngest/components/utils/historyParser';
import { type OutputType } from '@inngest/components/utils/outputRenderer';

import { SleepingSummary } from './SleepingSummary';
import { WaitingSummary } from './WaitingSummary';
import { renderRunMetadata } from './runMetadataRenderer';

interface Props {
  func: Pick<Function, 'name' | 'triggers'>;
  getHistoryItemOutput: (historyItemID: string) => Promise<string | undefined>;
  history: HistoryParser;
  run: Pick<FunctionRun, 'endedAt' | 'id' | 'output' | 'startedAt' | 'status'>;
}

export function RunDetails({ func, getHistoryItemOutput, history, run }: Props) {
  const firstTrigger = func.triggers[0] ?? null;
  const cron = firstTrigger && firstTrigger.type === 'CRON';

  const metadataItems = renderRunMetadata(run, history);
  let type: OutputType | undefined;
  if (run.status === 'COMPLETED') {
    type = 'completed';
  } else if (run.status === 'FAILED') {
    type = 'failed';
  }

  return (
    <ContentCard
      active
      title={func.name}
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
        {run.status && run.endedAt && run.output && type && (
          <OutputCard content={run.output} type={type} />
        )}

        <WaitingSummary history={history.groups} />
        <SleepingSummary history={history.groups} />
      </div>

      <hr className="mt-8 border-slate-800/50" />
      <div className="px-5 pt-4">
        <h3 className="py-4 text-sm text-slate-400">Timeline</h3>
        <Timeline getOutput={getHistoryItemOutput} history={history.groups} />
      </div>
    </ContentCard>
  );
}
