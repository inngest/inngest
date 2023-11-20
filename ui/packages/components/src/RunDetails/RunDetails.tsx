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
import type { FunctionVersion } from '@inngest/components/types/functionVersion';
import type { HistoryParser } from '@inngest/components/utils/historyParser';

import { SleepingSummary } from './SleepingSummary';
import { WaitingSummary } from './WaitingSummary';
import { renderRunMetadata } from './runMetadataRenderer';

interface Props {
  func: Pick<Function, 'name' | 'triggers'>;
  functionVersion?: Pick<FunctionVersion, 'url' | 'version'>;
  getHistoryItemOutput: (historyItemID: string) => Promise<string | undefined>;
  history: HistoryParser;

  // TODO: Replace this with an imported component.
  rerunButton?: React.ReactNode;

  run: Pick<FunctionRun, 'endedAt' | 'id' | 'output' | 'startedAt' | 'status'>;
}

export function RunDetails({
  func,
  functionVersion,
  getHistoryItemOutput,
  history,
  rerunButton,
  run,
}: Props) {
  const firstTrigger = func.triggers[0] ?? null;
  const cron = firstTrigger && firstTrigger.type === 'CRON';

  const metadataItems = renderRunMetadata({
    functionRun: run,
    functionVersion,
    history,
  });

  const isSuccess = run.status === 'COMPLETED';

  return (
    <ContentCard
      active
      button={rerunButton}
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
        {run.status && run.endedAt && run.output && isSuccess && (
          <OutputCard content={run.output} isSuccess={isSuccess} />
        )}

        <WaitingSummary history={history} />
        <SleepingSummary history={history} />
      </div>

      <hr className="mt-8 border-slate-800/50" />
      <div className="px-5 pt-4">
        <h3 className="py-4 text-sm text-slate-400">Timeline</h3>
        <Timeline getOutput={getHistoryItemOutput} history={history} />
      </div>
    </ContentCard>
  );
}
