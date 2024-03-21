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

import { CancelRunButton } from '../CancelRunButton';
import { CancellationSummary } from './CancellationSummary';
import { SleepingSummary } from './SleepingSummary';
import { WaitingSummary } from './WaitingSummary';
import { renderRunMetadata } from './runMetadataRenderer';

type FuncProps = {
  cancelRun?: () => Promise<unknown>;
  functionVersion?: Pick<FunctionVersion, 'url' | 'version'>;
  // TODO: Replace this with an imported component.
  rerunButton?: React.ReactNode;
};
type LoadingRun = {
  func?: Pick<Function, 'name' | 'triggers'>;
  loading: true;
  history?: undefined;
  run?: Pick<FunctionRun, 'endedAt' | 'id' | 'output' | 'startedAt' | 'status'>;
  getHistoryItemOutput?: (historyItemID: string) => Promise<string | undefined>;
  navigateToRun?: undefined;
};

type WithRun = {
  func: Pick<Function, 'name' | 'triggers'>;
  loading?: false;
  history: HistoryParser;
  getHistoryItemOutput: (historyItemID: string) => Promise<string | undefined>;
  run: Pick<FunctionRun, 'endedAt' | 'id' | 'output' | 'startedAt' | 'status'>;
  navigateToRun: React.ComponentProps<typeof Timeline>['navigateToRun'];
};

type Props = FuncProps & (WithRun | LoadingRun);

export function RunDetails({
  cancelRun,
  func,
  functionVersion,
  getHistoryItemOutput,
  history,
  rerunButton,
  run,
  navigateToRun,
  loading = false,
}: Props) {
  const firstTrigger = (func?.triggers && func.triggers[0]) ?? null;
  const cron = firstTrigger && firstTrigger.type === 'CRON';

  const metadataItems = renderRunMetadata({
    functionRun: run,
    functionVersion,
    history,
  });

  const isSuccess = run?.status === 'COMPLETED';

  return (
    <ContentCard
      active
      button={
        !loading && (
          <div className="flex gap-2">
            {cancelRun && <CancelRunButton disabled={Boolean(run?.endedAt)} onClick={cancelRun} />}
            {rerunButton}
          </div>
        )
      }
      title={func?.name || '...'}
      icon={run?.status && <FunctionRunStatusIcon status={run?.status} className="h-5 w-5" />}
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
          <MetadataGrid metadataItems={metadataItems} loading={loading} />
        </div>
      }
    >
      {run && history && getHistoryItemOutput && (
        <>
          <div className="px-5 pt-4">
            {run.status && run.endedAt && run.output && (
              <OutputCard content={run.output} isSuccess={isSuccess} />
            )}
            <CancellationSummary history={history} />
            <WaitingSummary history={history} />
            <SleepingSummary history={history} />
          </div>

          <hr className="mt-8 border-slate-800/50" />
          <div className="px-5 pt-4">
            <h3 className="py-4 text-sm text-slate-400">Timeline</h3>
            <Timeline
              getOutput={getHistoryItemOutput}
              history={history}
              navigateToRun={navigateToRun}
            />
          </div>
        </>
      )}
    </ContentCard>
  );
}
